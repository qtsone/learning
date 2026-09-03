package memlab

// Report summarizes a series of float64 samples.
type Report struct {
	Min, Max, Sum float64
	N             int
}

// SummarizeHeap is the "before" specimen, kept for comparison: because it
// returns a pointer, the Report must outlive the stack frame, so every call
// heap-allocates one. Read its escape-analysis line and benchmark it — but
// do not call it from Summarize.
func SummarizeHeap(xs []float64) *Report {
	r := &Report{}
	for i, x := range xs {
		if i == 0 || x < r.Min {
			r.Min = x
		}
		if i == 0 || x > r.Max {
			r.Max = x
		}
		r.Sum += x
		r.N++
	}
	return r
}

// Summarize reports the min, max, sum and count of xs as a value.
// An empty or nil slice yields the zero Report.
//
// TODO: implement it so the result never touches the heap — return the
// Report by value and let it live in the caller's frame (criterion 5).
func Summarize(xs []float64) Report {
	return Report{}
}
