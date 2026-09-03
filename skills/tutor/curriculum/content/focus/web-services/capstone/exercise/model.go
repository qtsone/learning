package board

import (
	"errors"
	"time"
)

// Role is what a user is. Three of them is deliberate: enough to make the
// policy table interesting, few enough to enumerate in a test matrix.
type Role string

const (
	// RoleMember owns tasks: creates them, and reads and updates their own.
	RoleMember Role = "member"
	// RoleAuditor reads everything and changes nothing.
	RoleAuditor Role = "auditor"
	// RoleAdmin does everything, and is exempt from ownership.
	RoleAdmin Role = "admin"
)

// Action names something a caller might do. Handlers and routes name actions,
// never roles: "which roles may update a task" is a policy decision that must
// be changeable without opening a handler.
type Action string

const (
	ActionTaskList   Action = "task:list"
	ActionTaskRead   Action = "task:read"
	ActionTaskCreate Action = "task:create"
	ActionTaskUpdate Action = "task:update"

	// ActionTaskDelete has a route and deliberately has no rule. Under
	// deny-by-default that is a 403 for everybody, including admins.
	ActionTaskDelete Action = "task:delete"

	ActionRoleSet Action = "role:set"
)

// User is an account. PasswordHash carries json:"-" so that no handler, now or
// in two years, can leak it by marshalling a User.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Role         Role   `json:"role"`
	PasswordHash string `json:"-"`
}

// TaskState is where a task is in its short life.
type TaskState string

const (
	StateOpen  TaskState = "open"
	StateDoing TaskState = "doing"
	StateDone  TaskState = "done"
)

// ValidState reports whether s is one of the three states.
func ValidState(s TaskState) bool {
	return s == StateOpen || s == StateDoing || s == StateDone
}

// Task is the domain object the whole service exists for. OwnerID is the
// attribute every ownership decision compares against, and it is set from the
// authenticated session — never from the request body.
type Task struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Title     string    `json:"title"`
	State     TaskState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Event is one thing that happened, on its way to connected clients.
//
// OwnerID is what makes fan-out compatible with authorization: the hub does not
// know who may see what, so every event carries the attribute the policy needs
// and each stream filters for its own subscriber.
type Event struct {
	// ID is assigned by the hub when the event is published, and is what a
	// reconnecting client sends back as Last-Event-ID.
	ID string `json:"id"`

	// Name is the SSE event type: "task.created", "task.updated",
	// "task.notified".
	Name string `json:"name"`

	// OwnerID is the owner of the task this event is about.
	OwnerID string `json:"owner_id"`

	// Data is the JSON payload written to the data: field.
	Data string `json:"data"`

	// Retry, when set, renders a retry: field instead of a payload. It is used
	// once at the start of a stream to set the client's reconnection delay.
	Retry time.Duration `json:"-"`
}

var (
	// ErrNotFound means the row does not exist. Handlers map it to 404 — after
	// asking the policy, never before.
	ErrNotFound = errors.New("board: not found")

	// ErrBadCredentials is what a wrong password and an unknown username both
	// produce, so neither the caller nor the code can tell them apart.
	ErrBadCredentials = errors.New("board: invalid credentials")

	// ErrBadCursor marks an unreadable pagination cursor: a client error (400),
	// not a server error.
	ErrBadCursor = errors.New("board: malformed cursor")
)
