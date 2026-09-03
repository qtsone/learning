package channels

// Generate returns a channel that emits each of nums in order, then closes.
//
// TODO: send the values from a goroutine so Generate returns immediately,
// and close the channel when all values are sent. The starter's close-only
// body is wrong on purpose: it compiles, but emits nothing.
func Generate(nums ...int) <-chan int {
	out := make(chan int)
	close(out)
	return out
}

// Square reads every value from in, squares it, and sends the result on the
// returned channel, closing it when in is exhausted.
//
// TODO: range over in from a goroutine, send v*v, and close the output so
// downstream stages terminate.
func Square(in <-chan int) <-chan int {
	out := make(chan int)
	close(out)
	return out
}

// Sum receives from in until it is closed and returns the total.
//
// TODO: range over in. The loop ends when the sender closes the channel —
// that is the whole reason pipeline stages close their outputs.
func Sum(in <-chan int) int {
	return 0
}

// TryRecv performs a non-blocking receive on ch: a ready value yields
// (value, true); otherwise it returns (0, false) immediately — including
// when ch is nil, or closed and drained.
//
// TODO: use select with a default case, and the comma-ok receive so a
// closed channel's zero values report false. No nil special-case needed:
// a nil channel's case is simply never ready.
func TryRecv(ch <-chan int) (int, bool) {
	return 0, false
}

// Counter returns a channel that emits 1, 2, 3, … until done is closed,
// then closes the returned channel so consumers terminate.
//
// TODO: in a goroutine, select between sending the next number and
// receiving from done; when done is closed, close the output and return.
func Counter(done <-chan struct{}) <-chan int {
	out := make(chan int)
	close(out)
	return out
}

// MergeTwo merges values from a and b onto one channel, closing the result
// only after BOTH inputs are closed. Values may arrive in any order.
//
// TODO: run a select loop in a goroutine with a comma-ok case per input.
// When an input reports ok == false, set that channel variable to nil so
// its case goes dormant; loop while either input is non-nil, then close
// the output. Draining one input before touching the other deadlocks —
// the tests prove it.
func MergeTwo(a, b <-chan int) <-chan int {
	out := make(chan int)
	close(out)
	return out
}
