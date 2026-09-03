package conf

import (
	"strings"
	"testing"
)

// These benchmarks are for you to run and read, not for the test suite to
// gate on. Timings move with the machine, the load and the race detector;
// the allocation columns are the stable part of the story.
//
//	go test -bench=. -benchmem
//	go test -bench=Decode -benchmem -count=6   # then eyeball the spread

func BenchmarkDecodeReflect(b *testing.B) {
	src := fullSrc()
	b.ReportAllocs()
	for range b.N {
		var c Config
		if err := Decode(src, &c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeGenerated(b *testing.B) {
	src := fullSrc()
	b.ReportAllocs()
	for range b.N {
		var c Config
		if err := c.DecodeMap(src); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLookupZeroCopy(b *testing.B) {
	data := envBlob(200)
	b.ReportAllocs()
	for range b.N {
		if _, ok := Lookup(data, "KEY_199"); !ok {
			b.Fatal("key not found")
		}
	}
}

// BenchmarkLookupCopying is the version you did not write: it converts the
// whole blob before scanning it. Compare B/op.
func BenchmarkLookupCopying(b *testing.B) {
	data := envBlob(200)
	b.ReportAllocs()
	for range b.N {
		if _, _, ok := strings.Cut(string(data), "KEY_199="); !ok {
			b.Fatal("key not found")
		}
	}
}
