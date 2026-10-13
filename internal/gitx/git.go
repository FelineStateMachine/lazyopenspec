// Package gitx shells out to git for the small amount of repository context
// the viewer shows: the current branch, uncommitted openspec files, and diffs.
package gitx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
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
	// Never take .git/index.lock: status/diff would otherwise refresh the
	// index and make an agent's concurrent `git add`/`git commit` fail.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err != nil && errOut.Len() > 0 {
		err = fmt.Errorf("%w: %s", err, strings.TrimSpace(errOut.String()))
	}
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
		if strings.ContainsAny(code, "RC") {
			i++ // -z puts the rename/copy source in the next field
		}
		s.Dirty[filepath.Join(top, filepath.FromSlash(path))] = code
	}
	return s
}

// Diff returns the working-tree diff of path against HEAD. Untracked files
// are diffed against /dev/null so new artifacts show up in full.
func Diff(root, path string, untracked bool) (string, error) {
	if !untracked {
		if _, err := run(root, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
			untracked = true // no commits yet: show the whole file as added
		}
	}
	if untracked {
		out, err := run(root, "diff", "--no-color", "--no-ext-diff", "--no-index", "--", "/dev/null", path)
		// --no-index exits 1 when files differ.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			err = nil
		}
		return out, err
	}
	return run(root, "diff", "--no-color", "--no-ext-diff", "HEAD", "--", path)
}
