//go:build unix

package spec

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO in a change dir must not block Load: reading it would hang forever.
func TestLoadIgnoresFIFO(t *testing.T) {
	root := copyFixture(t)
	if err := syscall.Mkfifo(filepath.Join(root, "openspec", "changes", "add-due-dates", "pipe.md"), 0o644); err != nil {
		t.Skip("mkfifo unsupported:", err)
	}
	done := make(chan *Snapshot, 1)
	go func() { done <- mustLoad(t, root) }()
	select {
	case s := <-done:
		if len(s.Changes) != 5 {
			t.Errorf("Changes = %v, want the 5 fixture changes", changeIDs(s.Changes))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Load blocked on a FIFO")
	}
}
