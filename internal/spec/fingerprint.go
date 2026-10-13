package spec

import (
	"io/fs"
	"path/filepath"
	"time"
)

type stamp struct {
	mod  time.Time
	size int64
}

// Fingerprint is a cheap stat-only view of every file under openspec/. It is
// used to poll for changes without holding a file descriptor per watched file.
type Fingerprint map[string]stamp

// Scan stats every file and directory under dir, so a new empty change
// directory is noticed too. A missing dir yields an empty fingerprint.
func Scan(dir string) Fingerprint {
	fp := Fingerprint{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			fp[path] = stamp{} // presence only; dir mtimes churn on temp files
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		fp[path] = stamp{mod: fi.ModTime(), size: fi.Size()}
		return nil
	})
	return fp
}

// Changed returns the paths that were added, removed, or modified between
// two fingerprints.
func (fp Fingerprint) Changed(next Fingerprint) []string {
	var paths []string
	for p, s := range next {
		if old, ok := fp[p]; !ok || old != s {
			paths = append(paths, p)
		}
	}
	for p := range fp {
		if _, ok := next[p]; !ok {
			paths = append(paths, p)
		}
	}
	return paths
}
