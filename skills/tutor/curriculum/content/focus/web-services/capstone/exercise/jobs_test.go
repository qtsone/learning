package board

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// enqueue commits one job through the same transaction-only door producers use.
func (e *env) enqueue(j NewJob) {
	e.t.Helper()
	ctx := context.Background()
	tx, err := e.store.DB.BeginTx(ctx, nil)
	if err != nil {
		e.t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	if err := e.store.Enqueue(ctx, tx, j); err != nil {
		e.t.Fatalf("Enqueue(%s): %v", j.ID, err)
	}
	if err := tx.Commit(); err != nil {
		e.t.Fatalf("commit: %v", err)
	}
}

func (e *env) processOne() bool {
	e.t.Helper()
	claimed, err := e.pool.ProcessOne(context.Background(), "worker-test")
	if err != nil {
		e.t.Fatalf("ProcessOne: %v", err)
	}
	return claimed
}

func (e *env) createTask(cookie *http.Cookie, title string) Task {
	e.t.Helper()
	rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: title}, cookie)
	if rec.Code != http.StatusCreated {
		e.t.Fatalf("POST /tasks = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}
	var task Task
	decodeData(e.t, rec, &task)
	return task
}

func (e *env) mustJob(id string) Job {
	e.t.Helper()
	job, err := e.store.JobByID(context.Background(), id)
	if err != nil {
		e.t.Fatalf("JobByID(%s): %v — the create handler must enqueue the work it triggers", id, err)
	}
	return job
}

func TestCreateEnqueuesTheWorkInTheSameTransaction(t *testing.T) {
	e := newEnv(t)
	task := e.createTask(e.login("alice"), "ship it")

	job := e.mustJob("notify:" + task.ID)
	if job.Kind != JobKindNotify || job.Payload != task.ID || job.State != JobReady {
		t.Errorf("job = %+v, want a ready %s job for %s", job, JobKindNotify, task.ID)
	}

	// The job is claimable and the row it refers to is already visible to the
	// worker, because both landed in the same commit.
	if !e.processOne() {
		t.Fatal("nothing was claimable: the job and the task must commit together")
	}
	if n, _ := e.store.CountNotifications(context.Background(), task.ID); n != 1 {
		t.Errorf("notifications = %d, want 1", n)
	}
}

// There is no ordering of "write the row" and "publish the job" that is safe
// when they are two systems — so they are not two systems. If the enqueue
// fails, the row it justified must not exist either, and nobody may have been
// told it did.
func TestCreateWritesNothingWhenTheEnqueueFails(t *testing.T) {
	e := newEnvWith(t, func(c *Config) { c.NewID = func() string { return "fixed-1" } })
	e.enqueue(NewJob{ID: "notify:fixed-1", Kind: JobKindNotify, Payload: "fixed-1"})
	watcher := e.watch()

	rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "doomed"}, e.login("alice"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: the duplicate job id must fail the whole operation", rec.Code)
	}
	if n := countRows(t, e, "tasks"); n != 0 {
		t.Errorf("tasks = %d, want 0: a task that could not enqueue its work must not exist. "+
			"Insert-then-enqueue would have left this row behind with no job", n)
	}
	if n := countRows(t, e, "jobs"); n != 1 {
		t.Errorf("jobs = %d, want only the pre-existing one", n)
	}
	noEvent(t, watcher)
}

func TestNotifyWritesTheNotificationAndAnnouncesIt(t *testing.T) {
	e := newEnv(t)
	task := e.createTask(e.login("alice"), "ship it")
	watcher := e.watch() // after the create, so only the job's event arrives

	if !e.processOne() {
		t.Fatal("nothing was claimable")
	}

	n, err := e.store.CountNotifications(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if n != 1 {
		t.Fatalf("notifications = %d, want 1", n)
	}
	ev := nextEvent(t, watcher)
	if ev.Name != EventTaskNotified || ev.OwnerID != task.OwnerID || !strings.Contains(ev.Data, task.ID) {
		t.Errorf("event = %+v, want a %s for %s owned by %s", ev, EventTaskNotified, task.ID, task.OwnerID)
	}
	if job := e.mustJob("notify:" + task.ID); job.State != JobDone {
		t.Errorf("job state = %q, want %q", job.State, JobDone)
	}
}

// At-least-once is not a bug to be fixed, it is the contract. A worker that
// died between doing the work and recording it leaves a lease that expires, and
// the job comes back. Running it twice must cost nothing and, in a service with
// a live stream, must not tell the owner twice either.
func TestNotifyIsIdempotentUnderRedelivery(t *testing.T) {
	e := newEnv(t)
	task := e.createTask(e.login("alice"), "ship it")
	jobID := "notify:" + task.ID

	if !e.processOne() {
		t.Fatal("nothing was claimable")
	}

	// The queue lost the completion — a crash, a partition — so the row is
	// claimable again with its lease expired.
	if _, err := e.store.DB.Exec(
		`UPDATE jobs SET state = ?, lease_until = 0 WHERE id = ?`, JobRunning, jobID); err != nil {
		t.Fatalf("simulate redelivery: %v", err)
	}
	watcher := e.watch()
	if !e.processOne() {
		t.Fatal("the redelivered job was not claimed")
	}

	n, err := e.store.CountNotifications(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if n != 1 {
		t.Errorf("notifications = %d after a redelivery, want 1: the effect must happen once", n)
	}
	noEvent(t, watcher)
	if job := e.mustJob(jobID); job.State != JobDone {
		t.Errorf("job state = %q, want %q: a duplicate delivery still finishes the job", job.State, JobDone)
	}
}

func TestNotifyForAMissingTaskDeadLettersWithoutBurningItsBudget(t *testing.T) {
	e := newEnv(t)
	watcher := e.watch()
	e.enqueue(NewJob{ID: "notify:ghost", Kind: JobKindNotify, Payload: "ghost"})

	if !e.processOne() {
		t.Fatal("nothing was claimable")
	}

	job := e.mustJob("notify:ghost")
	if job.State != JobDead {
		t.Errorf("job state = %q, want %q: a payload naming a row that does not exist is a failure no retry can fix",
			job.State, JobDead)
	}
	if job.Attempts != 1 {
		t.Errorf("attempts = %d, want 1: a permanent failure must not burn the whole budget and the waits between",
			job.Attempts)
	}
	if job.LastError == "" {
		t.Error("last_error is empty: a dead-lettered job with no cause is a mystery for whoever reads it")
	}
	if n := countRows(t, e, "notifications"); n != 0 {
		t.Errorf("notifications = %d, want 0", n)
	}
	if n := countRows(t, e, "processed_jobs"); n != 0 {
		t.Errorf("processed_jobs = %d, want 0: a marker that survives an effect that failed turns the next retry "+
			"into a silent skip", n)
	}
	noEvent(t, watcher)
}
