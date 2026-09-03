package board

import (
	"maps"
	"testing"
)

// The rule table is the security model, so it gets the treatment a security
// model deserves: every role against every action, with the deny rows written
// out as carefully as the allow rows. If a property is not a row here, it is a
// hope.
func TestDefaultRulesMatrix(t *testing.T) {
	e := newEnv(t)
	policy := e.srv.Policy()

	mine := &Resource{ID: "task-1", OwnerID: e.users["alice"].ID}
	theirs := &Resource{ID: "task-2", OwnerID: e.users["bob"].ID}

	cases := []struct {
		name       string
		subject    Subject
		action     Action
		resource   *Resource
		wantAllow  bool
		wantReason string
	}{
		{"anonymous may not list", Subject{}, ActionTaskList, nil, false, ReasonAnonymous},
		{"member may list", e.subject("alice"), ActionTaskList, nil, true, ReasonRoleGrants},
		{"auditor may list", e.subject("iris"), ActionTaskList, nil, true, ReasonRoleGrants},
		{"admin may list", e.subject("root"), ActionTaskList, nil, true, ReasonRoleGrants},

		{"member reads own", e.subject("alice"), ActionTaskRead, mine, true, ReasonOwner},
		{"member does not read another's", e.subject("alice"), ActionTaskRead, theirs, false, ReasonNotOwner},
		{"member with no object at all", e.subject("alice"), ActionTaskRead, nil, false, ReasonMissingResource},
		{"auditor reads anyone's", e.subject("iris"), ActionTaskRead, theirs, true, ReasonRoleOverride},
		{"admin reads anyone's", e.subject("root"), ActionTaskRead, theirs, true, ReasonRoleOverride},

		{"member creates", e.subject("alice"), ActionTaskCreate, nil, true, ReasonRoleGrants},
		{"auditor does not create", e.subject("iris"), ActionTaskCreate, nil, false, ReasonRoleLacksAction},
		{"admin creates", e.subject("root"), ActionTaskCreate, nil, true, ReasonRoleGrants},

		{"member updates own", e.subject("alice"), ActionTaskUpdate, mine, true, ReasonOwner},
		{"member does not update another's", e.subject("alice"), ActionTaskUpdate, theirs, false, ReasonNotOwner},
		{"auditor does not update, even something it could read", e.subject("iris"), ActionTaskUpdate, theirs, false, ReasonRoleLacksAction},
		{"admin updates anyone's", e.subject("root"), ActionTaskUpdate, theirs, true, ReasonRoleOverride},
		{"admin with no object still fails closed", e.subject("root"), ActionTaskUpdate, nil, false, ReasonMissingResource},

		{"member does not set roles", e.subject("alice"), ActionRoleSet, nil, false, ReasonRoleLacksAction},
		{"auditor does not set roles", e.subject("iris"), ActionRoleSet, nil, false, ReasonRoleLacksAction},
		{"admin sets roles", e.subject("root"), ActionRoleSet, nil, true, ReasonRoleGrants},

		{"nobody deletes: there is no rule", e.subject("alice"), ActionTaskDelete, mine, false, ReasonNoRule},
		{"not even an auditor", e.subject("iris"), ActionTaskDelete, theirs, false, ReasonNoRule},
		{"not even an admin", e.subject("root"), ActionTaskDelete, theirs, false, ReasonNoRule},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.Check(Request{Subject: tc.subject, Action: tc.action, Resource: tc.resource})
			if got.Allow != tc.wantAllow || got.Reason != tc.wantReason {
				t.Errorf("Check(%s, %s) = {allow:%v reason:%q}, want {allow:%v reason:%q}",
					tc.subject.Role, tc.action, got.Allow, got.Reason, tc.wantAllow, tc.wantReason)
			}
		})
	}
}

// The route gate runs before any object is loaded, so it must convert exactly
// one denial — "no object was supplied" — and no other.
func TestCheckRouteDefersOnlyTheOwnerQuestion(t *testing.T) {
	e := newEnv(t)
	policy := e.srv.Policy()

	if d := policy.CheckRoute(e.subject("alice"), ActionTaskUpdate); !d.Allow || d.Reason != ReasonOwnerCheckPending {
		t.Errorf("CheckRoute(member, task:update) = %+v, want a pending allow: the gate cannot know who owns the task yet", d)
	}
	if d := policy.CheckRoute(e.subject("iris"), ActionTaskUpdate); d.Allow {
		t.Errorf("CheckRoute(auditor, task:update) = %+v, want deny: a role denial is not the gate's to defer", d)
	}
	if d := policy.CheckRoute(e.subject("root"), ActionTaskDelete); d.Allow {
		t.Errorf("CheckRoute(admin, task:delete) = %+v, want deny: no rule grants it", d)
	}
}

func TestScopeForFollowsTheRules(t *testing.T) {
	e := newEnv(t)
	policy := e.srv.Policy()

	if got := policy.ScopeFor(e.subject("alice")); got.All || got.OwnerID != e.users["alice"].ID {
		t.Errorf("ScopeFor(member) = %+v, want only their own rows (OwnerID = %q)", got, e.users["alice"].ID)
	}
	for _, name := range []string{"iris", "root"} {
		if got := policy.ScopeFor(e.subject(name)); !got.All {
			t.Errorf("ScopeFor(%s) = %+v, want every row: the rule exempts them from ownership", name, got)
		}
	}
}

// The point of deriving the scope from the policy is that changing the policy
// changes the scope. A handler that says "auditors see everything" in its own
// words is a second copy of the model, and it will not change when the table
// does.
func TestScopeForIsDerivedFromThePolicyNotTheRole(t *testing.T) {
	e := newEnv(t)

	strict := maps.Clone(DefaultRules)
	read := strict[ActionTaskRead]
	read.OwnerExempt = nil // nobody is exempt any more
	strict[ActionTaskRead] = read
	policy := NewPolicy(strict, quietLogger())

	if got := policy.ScopeFor(e.subject("iris")); got.All {
		t.Errorf("ScopeFor(auditor) = %+v under a table with no ownership exemption, want owner-scoped: "+
			"the scope must be read out of the rules, not hard-coded per role", got)
	}

	open := maps.Clone(DefaultRules)
	read = open[ActionTaskRead]
	read.RequireOwner = false // reading is no longer about ownership at all
	open[ActionTaskRead] = read

	if got := NewPolicy(open, quietLogger()).ScopeFor(e.subject("alice")); !got.All {
		t.Errorf("ScopeFor(member) = %+v under a table where reading needs no ownership, want every row", got)
	}
}
