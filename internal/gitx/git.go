// Package gitx shells out to git for the small amount of repository context
// the viewer shows: the current branch, uncommitted openspec files, and diffs.
package gitx

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const timeout = 3 * time.Second

// Status is the repository context shown in the status bar.
type Status struct {
	IsRepo bool
	Branch string
	// Dirty maps absolute paths under the openspec dir to their two-letter
	// porcelain status (e.g. " M", "??", "A ").
	Dirty map[string]string
}

func run(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

// Read collects the branch and uncommitted files under openspecDir.
func Read(root, openspecDir string) Status {
	top, err := run(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return Status{}
	}
	top = strings.TrimSpace(top)
	s := Status{IsRepo: true, Dirty: map[string]string{}}

	if b, err := run(root, "branch", "--show-current"); err == nil {
		s.Branch = strings.TrimSpace(b)
	}
	if s.Branch == "" {
		if h, err := run(root, "rev-parse", "--short", "HEAD"); err == nil {
			s.Branch = "@" + strings.TrimSpace(h)
		}
	}

	out, err := run(root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", openspecDir)
	if err != nil {
		return s
	}
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 {
			continue
		}
		code, path := e[:2], e[3:]
		if code[0] == 'R' || code[0] == 'C' {
			i++ // -z puts the rename source in the next field
		}
		s.Dirty[filepath.Join(top, filepath.FromSlash(path))] = code
	}
	return s
}

// Diff returns the working-tree diff of path against HEAD. Untracked files
// are diffed against /dev/null so new artifacts show up in full.
func Diff(root, path string, untracked bool) (string, error) {
	if untracked {
		out, err := run(root, "diff", "--no-color", "--no-index", "--", "/dev/null", path)
		// --no-index exits 1 when files differ.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			err = nil
		}
		return out, err
	}
	return run(root, "diff", "--no-color", "HEAD", "--", path)
}
