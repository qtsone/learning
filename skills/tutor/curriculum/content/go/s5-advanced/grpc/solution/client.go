package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token)
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	task, err := c.raw.GetTask(ctx, &taskboardpb.GetTaskRequest{Id: id})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("%w: %q", ErrNotFound, id)
		}
		return nil, err
	}
	return task, nil
}
