package memlab

import "testing"

func TestAppendSampleOutput(t *testing.T) {
	cases := []struct {
		name   string
		metric string
		value  int64
		want   string
	}{
		{"positive", "hits", 42, "hits 42\n"},
		{"zero", "errors", 0, "errors 0\n"},
		{"negative", "clock_drift_ms", -7, "clock_drift_ms -7\n"},
		{"large", "bytes_total", 987654321, "bytes_total 987654321\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AppendSample(nil, c.metric, c.value)
			if string(got) != c.want {
				t.Errorf("AppendSample(nil, %q, %d) = %q, want %q", c.metric, c.value, got, c.want)
			}
		})
	}
}

func TestAppendSampleExtendsExistingBuffer(t *testing.T) {
	buf := []byte("# metrics\n")
	got := AppendSample(buf, "hits", 1)
	want := "# metrics\nhits 1\n"
	if string(got) != want {
		t.Errorf("AppendSample must append to dst, not replace it:\ngot  %q\nwant %q", got, want)
	}
}

func TestAppendSampleDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 0, 128)
	allocs := testing.AllocsPerRun(200, func() {
		buf = AppendSample(buf[:0], "hits", 12345)
	})
	if allocs > 0 {
		t.Errorf("AppendSample allocates %.0f time(s) per call even with a pre-sized buffer, want 0 — everything handed to fmt escapes to the heap; build the line with append and strconv.AppendInt instead", allocs)
	}
}
