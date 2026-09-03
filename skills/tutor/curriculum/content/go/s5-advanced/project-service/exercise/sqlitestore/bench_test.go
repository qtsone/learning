package sqlitestore

import (
	"context"
	"fmt"
	"testing"

	"tutor.local/project-service/task"
)

// BenchmarkListByStatus is the measurement NOTES.md asks you to make: how
// much does the v2 index on tasks(status) actually buy the listing endpoint?
// Run it as it stands, then comment migration v2 out of migrate.go, run it
// again on a fresh database, and put both numbers in NOTES.md.
//
//	go test -run '^$' -bench BenchmarkListByStatus -benchmem ./sqlitestore
func BenchmarkListByStatus(b *testing.B) {
	path := b.TempDir() + "/bench.db"
	st, err := Open(context.Background(), path)
	if err != nil {
		b.Fatalf("Open = %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	const rows = 5000
	for i := 0; i < rows; i++ {
		status := task.StatusDone
		if i%50 == 0 {
			status = task.StatusOpen
		}
		if _, err := st.Create(ctx, task.Task{
			Title:     fmt.Sprintf("task %d", i),
			Status:    status,
			CreatedAt: stamp,
			UpdatedAt: stamp,
		}); err != nil {
			b.Fatalf("seed Create = %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := st.List(ctx, task.StatusOpen)
		if err != nil {
			b.Fatalf("List = %v", err)
		}
		if len(got) != rows/50 {
			b.Fatalf("List returned %d rows, want %d", len(got), rows/50)
		}
	}
}
