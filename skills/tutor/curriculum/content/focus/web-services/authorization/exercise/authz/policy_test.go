package authz_test

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"tutor.local/authorization/authz"
)

func quietPolicy(t *testing.T) *authz.Policy {
	t.Helper()
	return authz.NewPolicy(authz.DefaultRules, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

var (
	anon   = authz.Subject{}
	viewer = authz.Subject{UserID: "alice", Role: authz.RoleViewer}
	editor = authz.Subject{UserID: "bob", Role: authz.RoleEditor}
	admin  = authz.Subject{UserID: "carol", Role: authz.RoleAdmin}
)

func describe(s authz.Subject) string {
	if s.UserID == "" {
		return "anonymous"
	}
	return fmt.Sprintf("%s/%s", s.UserID, s.Role)
}

// ownership is the third axis of the matrix: which object, if any, the caller
// has in hand when it asks.
type ownership int

const (
	noResource ownership = iota
	ownsIt
	someoneElses
)

func (o ownership) String() string {
	switch o {
	case ownsIt:
		return "own resource"
	case someoneElses:
		return "someone else's resource"
	default:
		return "no resource"
	}
}

func resourceFor(o ownership, sub authz.Subject) *authz.Resource {
	switch o {
	case ownsIt:
		return &authz.Resource{ID: "d1", OwnerID: sub.UserID}
	case someoneElses:
		return &authz.Resource{ID: "d2", OwnerID: "mallory"}
	default:
		return nil
	}
}

func assertDecision(t *testing.T, got authz.Decision, wantAllow bool, wantReason, what string) {
	t.Helper()
	if got.Allow != wantAllow || got.Reason != wantReason {
		t.Errorf("%s = {allow:%v reason:%q}, want {allow:%v reason:%q}",
			what, got.Allow, got.Reason, wantAllow, wantReason)
	}
}

// TestCheckOwnershipFreeActions covers the actions whose rule says nothing
// about ownership. The answer must be the same whether or not a resource is
// supplied — a rule that ignores the object must ignore it consistently.
func TestCheckOwnershipFreeActions(t *testing.T) {
	p := quietPolicy(t)
	unknown := authz.Action("doc:frobnicate")

	cases := []struct {
		action     authz.Action
		subject    authz.Subject
		wantAllow  bool
		wantReason string
	}{
		{authz.ActionList, anon, false, authz.ReasonAnonymous},
		{authz.ActionList, viewer, true, authz.ReasonRoleGrants},
		{authz.ActionList, editor, true, authz.ReasonRoleGrants},
		{authz.ActionList, admin, true, authz.ReasonRoleGrants},

		{authz.ActionCreate, anon, false, authz.ReasonAnonymous},
		{authz.ActionCreate, viewer, false, authz.ReasonRoleLacksAction},
		{authz.ActionCreate, editor, true, authz.ReasonRoleGrants},
		{authz.ActionCreate, admin, true, authz.ReasonRoleGrants},

		// A route exists for archiving; no rule does. Nobody gets through,
		// admins included.
		{authz.ActionArchive, anon, false, authz.ReasonAnonymous},
		{authz.ActionArchive, viewer, false, authz.ReasonNoRule},
		{authz.ActionArchive, editor, false, authz.ReasonNoRule},
		{authz.ActionArchive, admin, false, authz.ReasonNoRule},

		{unknown, anon, false, authz.ReasonAnonymous},
		{unknown, viewer, false, authz.ReasonNoRule},
		{unknown, editor, false, authz.ReasonNoRule},
		{unknown, admin, false, authz.ReasonNoRule},
	}

	for _, c := range cases {
		for _, o := range []ownership{noResource, ownsIt, someoneElses} {
			name := fmt.Sprintf("%s/%s/%s", c.action, describe(c.subject), o)
			t.Run(name, func(t *testing.T) {
				got := p.Check(authz.Request{
					Subject:  c.subject,
					Action:   c.action,
					Resource: resourceFor(o, c.subject),
				})
				assertDecision(t, got, c.wantAllow, c.wantReason, "Check("+name+")")
			})
		}
	}
}

// TestCheckOwnedActions is the full role-by-action-by-ownership matrix for the
// actions that require ownership. Read the deny rows as carefully as the allow
// rows: they are the ones an attacker cares about.
func TestCheckOwnedActions(t *testing.T) {
	p := quietPolicy(t)

	cases := []struct {
		action     authz.Action
		subject    authz.Subject
		own        ownership
		wantAllow  bool
		wantReason string
	}{
		// read: every role may read, but only their own — except admins.
		{authz.ActionRead, anon, noResource, false, authz.ReasonAnonymous},
		{authz.ActionRead, anon, ownsIt, false, authz.ReasonAnonymous},
		{authz.ActionRead, anon, someoneElses, false, authz.ReasonAnonymous},
		// No object in hand means an ownership rule cannot be satisfied, so it
		// fails closed — even for an admin.
		{authz.ActionRead, viewer, noResource, false, authz.ReasonMissingResource},
		{authz.ActionRead, viewer, ownsIt, true, authz.ReasonOwner},
		{authz.ActionRead, viewer, someoneElses, false, authz.ReasonNotOwner},
		{authz.ActionRead, editor, noResource, false, authz.ReasonMissingResource},
		{authz.ActionRead, editor, ownsIt, true, authz.ReasonOwner},
		{authz.ActionRead, editor, someoneElses, false, authz.ReasonNotOwner},
		{authz.ActionRead, admin, noResource, false, authz.ReasonMissingResource},
		{authz.ActionRead, admin, ownsIt, true, authz.ReasonRoleOverride},
		{authz.ActionRead, admin, someoneElses, true, authz.ReasonRoleOverride},

		// update: viewers are out on role alone — owning the document does not
		// grant an action your role does not have.
		{authz.ActionUpdate, anon, noResource, false, authz.ReasonAnonymous},
		{authz.ActionUpdate, anon, ownsIt, false, authz.ReasonAnonymous},
		{authz.ActionUpdate, anon, someoneElses, false, authz.ReasonAnonymous},
		{authz.ActionUpdate, viewer, noResource, false, authz.ReasonRoleLacksAction},
		{authz.ActionUpdate, viewer, ownsIt, false, authz.ReasonRoleLacksAction},
		{authz.ActionUpdate, viewer, someoneElses, false, authz.ReasonRoleLacksAction},
		{authz.ActionUpdate, editor, noResource, false, authz.ReasonMissingResource},
		{authz.ActionUpdate, editor, ownsIt, true, authz.ReasonOwner},
		{authz.ActionUpdate, editor, someoneElses, false, authz.ReasonNotOwner},
		{authz.ActionUpdate, admin, noResource, false, authz.ReasonMissingResource},
		{authz.ActionUpdate, admin, ownsIt, true, authz.ReasonRoleOverride},
		{authz.ActionUpdate, admin, someoneElses, true, authz.ReasonRoleOverride},

		// delete: same shape as update. Same rule struct, no new code.
		{authz.ActionDelete, anon, noResource, false, authz.ReasonAnonymous},
		{authz.ActionDelete, anon, ownsIt, false, authz.ReasonAnonymous},
		{authz.ActionDelete, anon, someoneElses, false, authz.ReasonAnonymous},
		{authz.ActionDelete, viewer, noResource, false, authz.ReasonRoleLacksAction},
		{authz.ActionDelete, viewer, ownsIt, false, authz.ReasonRoleLacksAction},
		{authz.ActionDelete, viewer, someoneElses, false, authz.ReasonRoleLacksAction},
		{authz.ActionDelete, editor, noResource, false, authz.ReasonMissingResource},
		{authz.ActionDelete, editor, ownsIt, true, authz.ReasonOwner},
		{authz.ActionDelete, editor, someoneElses, false, authz.ReasonNotOwner},
		{authz.ActionDelete, admin, noResource, false, authz.ReasonMissingResource},
		{authz.ActionDelete, admin, ownsIt, true, authz.ReasonRoleOverride},
		{authz.ActionDelete, admin, someoneElses, true, authz.ReasonRoleOverride},
	}

	for _, c := range cases {
		name := fmt.Sprintf("%s/%s/%s", c.action, describe(c.subject), c.own)
		t.Run(name, func(t *testing.T) {
			got := p.Check(authz.Request{
				Subject:  c.subject,
				Action:   c.action,
				Resource: resourceFor(c.own, c.subject),
			})
			assertDecision(t, got, c.wantAllow, c.wantReason, "Check("+name+")")
		})
	}
}

// TestCheckRouteDefersOwnership pins the coarse, route-level half. It must
// answer the role question exactly as Check does, and it must not pretend to
// have answered the ownership question.
func TestCheckRouteDefersOwnership(t *testing.T) {
	p := quietPolicy(t)

	cases := []struct {
		subject    authz.Subject
		action     authz.Action
		wantAllow  bool
		wantReason string
	}{
		{anon, authz.ActionList, false, authz.ReasonAnonymous},
		{anon, authz.ActionRead, false, authz.ReasonAnonymous},
		{viewer, authz.ActionList, true, authz.ReasonRoleGrants},
		{viewer, authz.ActionCreate, false, authz.ReasonRoleLacksAction},
		{viewer, authz.ActionRead, true, authz.ReasonOwnerCheckPending},
		{viewer, authz.ActionUpdate, false, authz.ReasonRoleLacksAction},
		{viewer, authz.ActionDelete, false, authz.ReasonRoleLacksAction},
		{editor, authz.ActionCreate, true, authz.ReasonRoleGrants},
		{editor, authz.ActionRead, true, authz.ReasonOwnerCheckPending},
		{editor, authz.ActionUpdate, true, authz.ReasonOwnerCheckPending},
		{editor, authz.ActionDelete, true, authz.ReasonOwnerCheckPending},
		{admin, authz.ActionUpdate, true, authz.ReasonOwnerCheckPending},
		{admin, authz.ActionArchive, false, authz.ReasonNoRule},
		{editor, authz.ActionArchive, false, authz.ReasonNoRule},
		// A route registered without an action: the zero Action has no rule.
		{admin, authz.Action(""), false, authz.ReasonNoRule},
		{admin, authz.Action("doc:frobnicate"), false, authz.ReasonNoRule},
	}

	for _, c := range cases {
		name := fmt.Sprintf("%s/%s", describe(c.subject), c.action)
		t.Run(name, func(t *testing.T) {
			got := p.CheckRoute(c.subject, c.action)
			assertDecision(t, got, c.wantAllow, c.wantReason, "CheckRoute("+name+")")
		})
	}
}

// TestAllowsMatchesCheck keeps the filtering helper honest: it is the same
// decision, minus the paperwork.
func TestAllowsMatchesCheck(t *testing.T) {
	p := quietPolicy(t)
	mine := authz.Resource{ID: "d1", OwnerID: editor.UserID}
	theirs := authz.Resource{ID: "d2", OwnerID: "mallory"}

	if !p.Allows(editor, authz.ActionRead, mine) {
		t.Error("Allows(editor, doc:read, own document) = false, want true")
	}
	if p.Allows(editor, authz.ActionRead, theirs) {
		t.Error("Allows(editor, doc:read, someone else's document) = true, want false")
	}
	if !p.Allows(admin, authz.ActionRead, theirs) {
		t.Error("Allows(admin, doc:read, someone else's document) = false, want true")
	}
	if p.Allows(anon, authz.ActionRead, mine) {
		t.Error("Allows(anonymous, doc:read, any document) = true, want false")
	}
}

// TestPolicyRulesAreFixedAtConstruction: a policy that can be widened after
// startup is not a policy. NewPolicy copies the table it is given.
func TestPolicyRulesAreFixedAtConstruction(t *testing.T) {
	rules := map[authz.Action]authz.Rule{
		authz.ActionList: {Roles: authz.Set(authz.RoleViewer)},
	}
	p := authz.NewPolicy(rules, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rules[authz.ActionArchive] = authz.Rule{Roles: authz.Set(authz.RoleViewer)}

	got := p.Check(authz.Request{Subject: viewer, Action: authz.ActionArchive})
	assertDecision(t, got, false, authz.ReasonNoRule,
		"Check(doc:archive) after mutating the caller's rule map")
}
