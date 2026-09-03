// Package authz decides what an authenticated subject may do. It answers one
// question — "may this subject perform this action on this resource?" — in one
// function, from rules held as data, and it attaches a reason to every answer
// so decisions can be logged and tested.
//
// Nothing in here writes to storage or knows what a document is. It takes a
// subject, an action, and (when there is one) the owner of the object.
package authz

import (
	"context"
	"log/slog"
)

// Role is the subject's role, assigned by whatever authenticated them.
type Role string

const (
	RoleViewer Role = "viewer"
	RoleEditor Role = "editor"
	RoleAdmin  Role = "admin"
)

// Subject is the authenticated principal: authentication's output and
// authorization's input. An empty UserID means "nobody is logged in".
type Subject struct {
	UserID string
	Role   Role
}

// Action names an operation on a kind of resource. Actions are the policy's
// vocabulary — routes and handlers name actions, never roles, so adding a role
// never means editing a handler.
type Action string

const (
	ActionList   Action = "doc:list"
	ActionCreate Action = "doc:create"
	ActionRead   Action = "doc:read"
	ActionUpdate Action = "doc:update"
	ActionDelete Action = "doc:delete"

	// ActionArchive is deliberately absent from DefaultRules: the route
	// exists, the rule does not. Deny-by-default has to catch that.
	ActionArchive Action = "doc:archive"
)

// Resource is the object an action targets, reduced to exactly what the policy
// needs to know about it.
type Resource struct {
	ID      string
	OwnerID string
}

// Request is one authorization question. Resource is nil when the caller has
// no object in hand yet — at route level, or for actions like create and list
// that have no single target.
type Request struct {
	Subject  Subject
	Action   Action
	Resource *Resource
}

// Rule says who may perform an action and whether they must own the object.
// It is plain data: you can print it, diff it, and review it without reading
// any handler.
type Rule struct {
	// Roles that may perform the action at all.
	Roles map[Role]bool
	// RequireOwner makes the action an object-level one: the subject must own
	// the resource.
	RequireOwner bool
	// OwnerExempt lists roles that satisfy an ownership requirement without
	// owning anything — the deliberate, named escape hatch (admins).
	OwnerExempt map[Role]bool
}

// Decision is the policy's answer, always with a reason attached. The reason
// is for the log and for tests; it is never sent to the client.
type Decision struct {
	Allow  bool
	Reason string
}

// Reasons. They are constants, not literals, because tests assert on them and
// operators grep for them.
const (
	ReasonAnonymous         = "anonymous subject"
	ReasonNoRule            = "no rule for action"
	ReasonRoleLacksAction   = "role lacks action"
	ReasonMissingResource   = "missing resource"
	ReasonNotOwner          = "not owner"
	ReasonOwner             = "owner"
	ReasonRoleOverride      = "role exempt from ownership"
	ReasonRoleGrants        = "role grants action"
	ReasonOwnerCheckPending = "role grants action, owner check pending"
)

// Set is a small helper so rule tables read like sentences.
func Set(rs ...Role) map[Role]bool {
	m := make(map[Role]bool, len(rs))
	for _, r := range rs {
		m[r] = true
	}
	return m
}

// DefaultRules is the whole access-control model of this service, on one
// screen. Reviewing access control means reviewing this table.
var DefaultRules = map[Action]Rule{
	ActionList:   {Roles: Set(RoleViewer, RoleEditor, RoleAdmin)},
	ActionCreate: {Roles: Set(RoleEditor, RoleAdmin)},
	ActionRead: {
		Roles:        Set(RoleViewer, RoleEditor, RoleAdmin),
		RequireOwner: true,
		OwnerExempt:  Set(RoleAdmin),
	},
	ActionUpdate: {
		Roles:        Set(RoleEditor, RoleAdmin),
		RequireOwner: true,
		OwnerExempt:  Set(RoleAdmin),
	},
	ActionDelete: {
		Roles:        Set(RoleEditor, RoleAdmin),
		RequireOwner: true,
		OwnerExempt:  Set(RoleAdmin),
	},
}

// Policy evaluates requests against a fixed rule set. It is read-only after
// construction, so one Policy is safely shared by every request goroutine.
type Policy struct {
	rules map[Action]Rule
	log   *slog.Logger
}

// NewPolicy copies rules so a caller cannot bolt on a new grant after startup.
// The copy is shallow — the Rule values themselves are treated as immutable.
func NewPolicy(rules map[Action]Rule, log *slog.Logger) *Policy {
	if log == nil {
		log = slog.Default()
	}
	own := make(map[Action]Rule, len(rules))
	for a, r := range rules {
		own[a] = r
	}
	return &Policy{rules: own, log: log}
}

// Check is THE decision function: pure, side-effect free, and the only place
// in the program where an access-control rule is interpreted.
//
// The ladder it walks is in LESSON.md; every rung returns a Decision with a
// reason, and the last word is always an explicit allow.
func (p *Policy) Check(req Request) Decision {
	if req.Subject.UserID == "" {
		return Decision{Reason: ReasonAnonymous}
	}
	rule, ok := p.rules[req.Action]
	if !ok {
		return Decision{Reason: ReasonNoRule}
	}
	if !rule.Roles[req.Subject.Role] {
		return Decision{Reason: ReasonRoleLacksAction}
	}
	if !rule.RequireOwner {
		return Decision{Allow: true, Reason: ReasonRoleGrants}
	}
	// An ownership rule with no object to test cannot be satisfied. Denying
	// here is what turns "the handler forgot to pass the resource" into a 403
	// instead of a silent bypass.
	if req.Resource == nil {
		return Decision{Reason: ReasonMissingResource}
	}
	if rule.OwnerExempt[req.Subject.Role] {
		return Decision{Allow: true, Reason: ReasonRoleOverride}
	}
	if req.Resource.OwnerID != req.Subject.UserID {
		return Decision{Reason: ReasonNotOwner}
	}
	return Decision{Allow: true, Reason: ReasonOwner}
}

// CheckRoute is the coarse, route-level half of enforcement: "may this subject
// ever perform this action?", asked before any object has been loaded.
//
// It must answer the role question exactly as Check does, and it must treat
// "this rule needs an object I do not have yet" as a pass — with a reason that
// says the owner check is still owed. It must never turn a role denial into a
// pass.
func (p *Policy) CheckRoute(sub Subject, action Action) Decision {
	d := p.Check(Request{Subject: sub, Action: action})
	// The only denial a route gate is allowed to soften: "I have no object
	// yet". Everything else — anonymous, no rule, wrong role — stands.
	if !d.Allow && d.Reason == ReasonMissingResource {
		return Decision{Allow: true, Reason: ReasonOwnerCheckPending}
	}
	return d
}

// Allows answers the same question as Check, without a reason and without a
// response, for the one caller that needs it: filtering a collection, where a
// denial is not an error, just an omission.
func (p *Policy) Allows(sub Subject, action Action, res Resource) bool {
	return p.Check(Request{Subject: sub, Action: action, Resource: &res}).Allow
}

// subjectKey is unexported so no other package can collide with it in a
// context — the standard rule for context keys since S3.
type subjectKey struct{}

// WithSubject attaches the authenticated subject to a request context.
// Authentication calls it; authorization reads it back.
func WithSubject(ctx context.Context, sub Subject) context.Context {
	return context.WithValue(ctx, subjectKey{}, sub)
}

// SubjectFrom returns the subject attached to ctx. The zero Subject (no
// UserID) is what an unauthenticated request looks like, and the policy denies
// it — so a caller that ignores ok still fails closed.
func SubjectFrom(ctx context.Context) (Subject, bool) {
	sub, ok := ctx.Value(subjectKey{}).(Subject)
	return sub, ok
}
