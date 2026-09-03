package board

import (
	"log/slog"
	"net/http"
)

// Rule is what the policy knows about one action. The whole access-control
// model is a table of these: reviewing security means reading data on one
// screen, not grepping handlers for `role ==`.
type Rule struct {
	// Roles may perform the action at all.
	Roles map[Role]bool
	// RequireOwner means the subject must own the object it names.
	RequireOwner bool
	// OwnerExempt are the roles that satisfy ownership without owning.
	OwnerExempt map[Role]bool
}

// Roles builds a role set, so the table below reads like a sentence.
func Set(rs ...Role) map[Role]bool {
	set := make(map[Role]bool, len(rs))
	for _, r := range rs {
		set[r] = true
	}
	return set
}

// DefaultRules is this service's access-control model.
//
// TODO: fill in the table so that it says, exactly:
//
//   - members, auditors and admins may list tasks;
//   - all three may read a task, but a member only their own — auditors and
//     admins are exempt from ownership;
//   - members and admins may create;
//   - members and admins may update, and again only their own unless exempt;
//   - only admins may change a role;
//   - nothing here grants task deletion. The route exists; the rule does not.
//
// A Rule literal looks like:
//
//	ActionTaskUpdate: {
//		Roles:        Set(RoleMember, RoleAdmin),
//		RequireOwner: true,
//		OwnerExempt:  Set(RoleAdmin),
//	},
var DefaultRules = map[Action]Rule{}

// Resource is the object a decision is about — as little of it as the policy
// needs, so that Check stays pure and cheap.
type Resource struct {
	ID      string
	OwnerID string
}

// Request is one question for the policy. A nil Resource means "no object was
// supplied", which is a different answer from "an object owned by nobody".
type Request struct {
	Subject  Subject
	Action   Action
	Resource *Resource
}

// Decision is the answer and the why. A boolean is a bad answer to a security
// question: your tests assert *which* denial fired, and your operators answer
// "why was this user refused?" with a log query instead of a debugger.
type Decision struct {
	Allow  bool
	Reason string
}

// Reasons, as constants, because both tests and log queries depend on them.
const (
	ReasonAnonymous         = "anonymous subject"
	ReasonNoRule            = "no rule for action"
	ReasonRoleLacksAction   = "role lacks action"
	ReasonMissingResource   = "no resource supplied for an ownership rule"
	ReasonNotOwner          = "not owner"
	ReasonRoleGrants        = "role grants action"
	ReasonRoleOverride      = "role exempt from ownership"
	ReasonOwner             = "subject owns resource"
	ReasonOwnerCheckPending = "role grants action, owner check pending"
)

// Policy interprets the rules. It holds them as a field rather than reading the
// package variable, so a test — or a deployment — can hand it a different
// table without touching a line of enforcement code.
type Policy struct {
	Rules map[Action]Rule
	log   *slog.Logger
}

// NewPolicy builds a policy over a rule table.
func NewPolicy(rules map[Action]Rule, logger *slog.Logger) *Policy {
	if logger == nil {
		logger = slog.Default()
	}
	return &Policy{Rules: rules, log: logger}
}

// Check is the decision ladder, and it is pure: subject, action and maybe an
// object in; {Allow, Reason} out. No HTTP, no database, no clock — which is
// what lets a table test enumerate its entire behaviour in milliseconds.
//
// Every exit that is not an explicit grant is a denial, including the ones
// caused by *your* mistakes: an action nobody wrote a rule for, and an
// ownership rule asked to judge an object that was never supplied.
func (p *Policy) Check(req Request) Decision {
	if req.Subject.Anonymous() {
		return Decision{Reason: ReasonAnonymous}
	}
	rule, ok := p.Rules[req.Action]
	if !ok {
		return Decision{Reason: ReasonNoRule}
	}
	if !rule.Roles[req.Subject.Role] {
		return Decision{Reason: ReasonRoleLacksAction}
	}
	if !rule.RequireOwner {
		return Decision{Allow: true, Reason: ReasonRoleGrants}
	}
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

// CheckRoute is the coarse half, for the gate that runs before any object has
// been loaded. It answers everything Check can answer without an object, and
// converts only the "no resource supplied" denial into a pending allow. It
// never converts a role denial.
func (p *Policy) CheckRoute(sub Subject, action Action) Decision {
	d := p.Check(Request{Subject: sub, Action: action})
	if !d.Allow && d.Reason == ReasonMissingResource {
		return Decision{Allow: true, Reason: ReasonOwnerCheckPending}
	}
	return d
}

// Allows reports Check's verdict for a subject/action/object triple, with no
// response written and nothing logged. It is the form the streaming filter and
// the listing scope use.
func (p *Policy) Allows(sub Subject, action Action, res Resource) bool {
	return p.Check(Request{Subject: sub, Action: action, Resource: &res}).Allow
}

// ScopeFor reports how much of the task table a subject may read.
//
// The listing and the event stream both need this, and neither may answer it
// with a role literal: "auditors see everything" written in a handler is a
// second copy of the policy that will not change when the table does.
//
// TODO: derive the scope from the rules. A subject who may read a task owned by
// somebody else sees everything; anybody else sees only their own. Ask the
// policy the question — do not re-implement the ladder.
func (p *Policy) ScopeFor(sub Subject) Scope {
	return Scope{}
}

// Audit records a decision. Allows are info, denials are warn, and the reason
// goes here and never into a response body: "not owner" in a response confirms
// that the object exists and belongs to someone else.
//
// Logging lives at the enforcement points, not inside Check, so the decision
// function stays pure and nothing is logged twice.
func (p *Policy) Audit(r *http.Request, req Request, d Decision) {
	attrs := []any{
		"user", req.Subject.UserID,
		"role", string(req.Subject.Role),
		"action", string(req.Action),
		"method", r.Method,
		"path", r.URL.Path,
		"allow", d.Allow,
		"reason", d.Reason,
	}
	if req.Resource != nil {
		attrs = append(attrs, "resource", req.Resource.ID, "owner", req.Resource.OwnerID)
	}
	if d.Allow {
		p.log.Info("authorization", attrs...)
		return
	}
	p.log.Warn("authorization", attrs...)
}
