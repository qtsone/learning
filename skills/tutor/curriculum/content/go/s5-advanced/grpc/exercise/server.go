package tasks

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"tutor.local/grpc-tasks/taskboardpb"
)

// Server implements taskboard.v1.TaskService over a fixed, read-only task
// list. Tasks are set once at construction and never mutated, so concurrent
// RPCs need no locking. (A real server with writes would need one — see the
// databases lesson next.)
type Server struct {
	taskboardpb.UnimplementedTaskServiceServer

	tasks []*taskboardpb.Task
}

// NewServer returns a Server seeded with the given tasks, preserving order.
func NewServer(seed ...*taskboardpb.Task) *Server {
	s := &Server{}
	s.tasks = append(s.tasks, seed...)
	return s
}

// GetTask returns the task with the requested id.
//
// Contract (see taskboard.proto and the tests):
//   - empty id            -> codes.InvalidArgument
//   - id not in the store -> codes.NotFound
//   - otherwise           -> the task, nil error
func (s *Server) GetTask(ctx context.Context, req *taskboardpb.GetTaskRequest) (*taskboardpb.Task, error) {
	// TODO: validate req.GetId(), look the task up in s.tasks, and return
	// status errors with the codes the contract demands.
	return nil, status.Error(codes.Unimplemented, "GetTask not implemented yet")
}

// ListTasks streams every task whose status matches the filter, in insertion
// order. STATUS_UNSPECIFIED means "no filter": send everything.
func (s *Server) ListTasks(req *taskboardpb.ListTasksRequest, stream taskboardpb.TaskService_ListTasksServer) error {
	// TODO: loop over s.tasks, skip non-matching statuses, stream.Send the
	// rest. Return the first Send error you hit; return nil when done.
	return status.Error(codes.Unimplemented, "ListTasks not implemented yet")
}
