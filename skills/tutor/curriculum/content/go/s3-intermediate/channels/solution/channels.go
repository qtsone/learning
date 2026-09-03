package channels

// Generate returns a channel that emits each of nums in order, then closes.
func Generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

// Square reads every value from in, squares it, and sends the result on the
// returned channel, closing it when in is exhausted.
func Square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- v * v
		}
	}()
	return out
}

// Sum receives from in until it is closed and returns the total.
func Sum(in <-chan int) int {
	total := 0
	for v := range in {
		total += v
	}
	return total
}

// TryRecv performs a non-blocking receive on ch: a ready value yields
// (value, true); otherwise it returns (0, false) immediately — including
// when ch is nil, or closed and drained.
func TryRecv(ch <-chan int) (int, bool) {
	select {
	case v, ok := <-ch:
		if !ok {
			return 0, false
		}
		return v, true
	default:
		return 0, false
	}
}

// Counter returns a channel that emits 1, 2, 3, … until done is closed,
// then closes the returned channel so consumers terminate.
func Counter(done <-chan struct{}) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := 1; ; n++ {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}()
	return out
}

// MergeTwo merges values from a and b onto one channel, closing the result
// only after BOTH inputs are closed. Values may arrive in any order.
func MergeTwo(a, b <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for a != nil || b != nil {
			select {
			case v, ok := <-a:
				if !ok {
					a = nil
					continue
				}
				out <- v
			case v, ok := <-b:
				if !ok {
					b = nil
					continue
				}
				out <- v
			}
		}
	}()
	return out
}
