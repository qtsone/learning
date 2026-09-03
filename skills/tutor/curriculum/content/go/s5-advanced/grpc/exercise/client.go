package tasks

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	"tutor.local/grpc-tasks/taskboardpb"
)

// ErrNotFound is returned by Client.FetchTask when the server reports
// codes.NotFound, so callers can use errors.Is instead of gRPC status codes.
var ErrNotFound = errors.New("task not found")

// Client wraps the generated TaskServiceClient with the policies every call
// site would otherwise repeat: auth metadata, a default deadline, and
// translation of gRPC status codes into ordinary Go errors.
type Client struct {
	raw     taskboardpb.TaskServiceClient
	token   string
	timeout time.Duration
}

// NewClient builds a Client. timeout is the default deadline applied to calls
// whose context does not already carry one.
func NewClient(conn grpc.ClientConnInterface, token string, timeout time.Duration) *Client {
	return &Client{
		raw:     taskboardpb.NewTaskServiceClient(conn),
		token:   token,
		timeout: timeout,
	}
}

// FetchTask fetches one task by id.
//
// Contract (see the tests):
//   - every call carries "authorization: Bearer <token>" metadata
//   - if ctx has no deadline, apply c.timeout; if it has one, keep it
//   - codes.NotFound becomes an error satisfying errors.Is(err, ErrNotFound)
//   - any other error is returned as-is
func (c *Client) FetchTask(ctx context.Context, id string) (*taskboardpb.Task, error) {
	// TODO: attach the bearer token to the *outgoing* metadata.
	// TODO: apply the default deadline only when ctx has none.
	// TODO: translate codes.NotFound into ErrNotFound (wrap it, keep the id).
	return c.raw.GetTask(ctx, &taskboardpb.GetTaskRequest{Id: id})
}
