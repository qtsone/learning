package tasks

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"tutor.local/grpc-tasks/taskboardpb"
)

// startServer runs impl on an in-memory bufconn listener — no ports, no
// network — and returns a client connection to it. Server and connection are
// torn down automatically when the test ends.
func startServer(t *testing.T, impl taskboardpb.TaskServiceServer, opts ...grpc.ServerOption) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(opts...)
	taskboardpb.RegisterTaskServiceServer(srv, impl)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func seedTasks() []*taskboardpb.Task {
	return []*taskboardpb.Task{
		{Id: "t1", Title: "write the proto", Status: taskboardpb.Status_STATUS_DONE},
		{Id: "t2", Title: "implement GetTask", Status: taskboardpb.Status_STATUS_OPEN},
		{Id: "t3", Title: "implement ListTasks", Status: taskboardpb.Status_STATUS_OPEN},
	}
}

func TestGetTask_ReturnsTask(t *testing.T) {
	conn := startServer(t, NewServer(seedTasks()...))
	client := taskboardpb.NewTaskServiceClient(conn)

	got, err := client.GetTask(context.Background(), &taskboardpb.GetTaskRequest{Id: "t2"})
	if err != nil {
		t.Fatalf("GetTask(t2): unexpected error: %v", err)
	}
	if got.GetId() != "t2" {
		t.Errorf("GetTask(t2).Id = %q, want %q", got.GetId(), "t2")
	}
	if got.GetTitle() != "implement GetTask" {
		t.Errorf("GetTask(t2).Title = %q, want %q", got.GetTitle(), "implement GetTask")
	}
	if got.GetStatus() != taskboardpb.Status_STATUS_OPEN {
		t.Errorf("GetTask(t2).Status = %v, want %v", got.GetStatus(), taskboardpb.Status_STATUS_OPEN)
	}
}

func TestGetTask_StatusCodes(t *testing.T) {
	conn := startServer(t, NewServer(seedTasks()...))
	client := taskboardpb.NewTaskServiceClient(conn)

	cases := []struct {
		name string
		id   string
		want codes.Code
	}{
		{"unknown id is NotFound", "nope", codes.NotFound},
		{"empty id is InvalidArgument", "", codes.InvalidArgument},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := client.GetTask(context.Background(), &taskboardpb.GetTaskRequest{Id: c.id})
			if err == nil {
				t.Fatalf("GetTask(%q): want error code %v, got nil error", c.id, c.want)
			}
			if got := status.Code(err); got != c.want {
				t.Errorf("GetTask(%q): status code = %v, want %v", c.id, got, c.want)
			}
		})
	}
}

func TestListTasks_Streams(t *testing.T) {
	conn := startServer(t, NewServer(seedTasks()...))
	client := taskboardpb.NewTaskServiceClient(conn)

	cases := []struct {
		name    string
		filter  taskboardpb.Status
		wantIDs []string
	}{
		{"unspecified streams all, in order", taskboardpb.Status_STATUS_UNSPECIFIED, []string{"t1", "t2", "t3"}},
		{"filter by OPEN", taskboardpb.Status_STATUS_OPEN, []string{"t2", "t3"}},
		{"filter by DONE", taskboardpb.Status_STATUS_DONE, []string{"t1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stream, err := client.ListTasks(context.Background(), &taskboardpb.ListTasksRequest{Status: c.filter})
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			var gotIDs []string
			for {
				task, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("stream.Recv: %v", err)
				}
				gotIDs = append(gotIDs, task.GetId())
			}
			if len(gotIDs) != len(c.wantIDs) {
				t.Fatalf("streamed ids = %v, want %v", gotIDs, c.wantIDs)
			}
			for i := range gotIDs {
				if gotIDs[i] != c.wantIDs[i] {
					t.Fatalf("streamed ids = %v, want %v", gotIDs, c.wantIDs)
				}
			}
		})
	}
}

func TestAuthUnaryInterceptor(t *testing.T) {
	conn := startServer(t, NewServer(seedTasks()...),
		grpc.UnaryInterceptor(AuthUnaryInterceptor("secret")))
	client := taskboardpb.NewTaskServiceClient(conn)

	cases := []struct {
		name     string
		ctx      context.Context
		wantCode codes.Code
	}{
		{"no metadata is rejected", context.Background(), codes.Unauthenticated},
		{
			"wrong token is rejected",
			metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer wrong"),
			codes.Unauthenticated,
		},
		{
			"correct token passes through",
			metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer secret"),
			codes.OK,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := client.GetTask(c.ctx, &taskboardpb.GetTaskRequest{Id: "t1"})
			if got := status.Code(err); got != c.wantCode {
				t.Errorf("GetTask with %s: status code = %v (err: %v), want %v",
					c.name, got, err, c.wantCode)
			}
		})
	}
}

// captureServer records what the server actually saw on the wire: the auth
// metadata and the deadline the client's context propagated.
type captureServer struct {
	taskboardpb.UnimplementedTaskServiceServer

	mu          sync.Mutex
	auth        string
	hadDeadline bool
	remaining   time.Duration
}

func (c *captureServer) GetTask(ctx context.Context, req *taskboardpb.GetTaskRequest) (*taskboardpb.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("authorization"); len(vals) > 0 {
			c.auth = vals[0]
		}
	}
	var deadline time.Time
	deadline, c.hadDeadline = ctx.Deadline()
	if c.hadDeadline {
		c.remaining = time.Until(deadline)
	}
	return &taskboardpb.Task{Id: req.GetId()}, nil
}

func (c *captureServer) snapshot() (string, bool, time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.auth, c.hadDeadline, c.remaining
}

func TestFetchTask_AttachesTokenAndDefaultDeadline(t *testing.T) {
	capture := &captureServer{}
	conn := startServer(t, capture)
	client := NewClient(conn, "secret", 30*time.Second)

	if _, err := client.FetchTask(context.Background(), "t1"); err != nil {
		t.Fatalf("FetchTask: %v", err)
	}
	auth, hadDeadline, remaining := capture.snapshot()
	if want := "Bearer secret"; auth != want {
		t.Errorf("server saw authorization = %q, want %q (attach it to the outgoing metadata)", auth, want)
	}
	if !hadDeadline {
		t.Fatal("server saw no deadline: FetchTask must apply the default timeout when ctx has none")
	}
	if remaining > 31*time.Second {
		t.Errorf("server saw %v left on the deadline, want <= the client's 30s default", remaining)
	}
}

func TestFetchTask_KeepsCallerDeadline(t *testing.T) {
	capture := &captureServer{}
	conn := startServer(t, capture)
	client := NewClient(conn, "secret", 5*time.Second)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
	defer cancel()
	if _, err := client.FetchTask(ctx, "t1"); err != nil {
		t.Fatalf("FetchTask: %v", err)
	}
	_, hadDeadline, remaining := capture.snapshot()
	if !hadDeadline {
		t.Fatal("server saw no deadline: the caller's deadline must propagate")
	}
	if remaining < 30*time.Minute {
		t.Errorf("server saw %v left on the deadline; the caller set ~1h, so FetchTask must not shrink it to its 5s default", remaining)
	}
}

func TestFetchTask_NotFoundSentinel(t *testing.T) {
	conn := startServer(t, NewServer(seedTasks()...),
		grpc.UnaryInterceptor(AuthUnaryInterceptor("secret")))
	client := NewClient(conn, "secret", 30*time.Second)

	got, err := client.FetchTask(context.Background(), "t1")
	if err != nil {
		t.Fatalf("FetchTask(t1) through the auth interceptor: %v", err)
	}
	if got.GetId() != "t1" {
		t.Errorf("FetchTask(t1).Id = %q, want %q", got.GetId(), "t1")
	}

	_, err = client.FetchTask(context.Background(), "nope")
	if err == nil {
		t.Fatal("FetchTask(nope): want an error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FetchTask(nope): errors.Is(err, ErrNotFound) = false; err = %v — translate codes.NotFound", err)
	}
}
