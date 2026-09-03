package heaps

// Task is a unit of scheduled work: lower Priority values run sooner.
type Task struct {
	Name     string
	Priority int
}

// TaskQueue implements heap.Interface over a slice of tasks ordered by
// ascending Priority. Use it only through the container/heap package
// functions (heap.Init, heap.Push, heap.Pop) — the five methods below are
// plumbing that the package calls for you.
type TaskQueue []Task

// Len reports the number of tasks in the queue.
func (q TaskQueue) Len() int {
	// TODO: implement.
	return 0
}

// Less reports whether the task at index i is more urgent than the task
// at index j: smaller Priority wins.
func (q TaskQueue) Less(i, j int) bool {
	// TODO: implement.
	return false
}

// Swap exchanges the tasks at indexes i and j.
func (q TaskQueue) Swap(i, j int) {
	// TODO: implement.
}

// Push appends x to the queue. Called by heap.Push, which sifts it up
// afterwards. The pointer receiver lets the slice grow.
func (q *TaskQueue) Push(x any) {
	// TODO: assert x back to a Task with x.(Task), then append it.
}

// Pop removes and returns the LAST task. Called by heap.Pop after it has
// already swapped the most urgent task into the last position.
func (q *TaskQueue) Pop() any {
	// TODO: take the last task, shrink the slice, return the task.
	return nil
}
