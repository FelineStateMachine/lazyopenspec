package spec

import (
	"bufio"
	"regexp"
	"strings"
)

// These mirror the OpenSpec CLI's parsers so counts match `openspec list`.
var (
	// Top-level checkboxes only: indented sub-tasks are not counted, and the
	// CLI matches them even inside fenced code.
	taskRe        = regexp.MustCompile(`^[-*]\s+\[([\sxX])\]`)
	sectionRe     = regexp.MustCompile(`(?i)^##\s+(ADDED|MODIFIED|REMOVED|RENAMED)\s+Requirements\s*$`)
	requirementRe = regexp.MustCompile(`(?i)^###\s*Requirement:`)
	removedAltRe  = regexp.MustCompile("(?i)^\\s*-\\s*`?###\\s*Requirement:")
	scenarioRe    = regexp.MustCompile(`^####\s+`)
	renameFromRe  = regexp.MustCompile(`(?i)^\s*-?\s*FROM:`)
	fenceRe       = regexp.MustCompile("^\\s*(```|~~~)")
)

// lines yields the lines of a markdown document outside fenced code blocks.
func lines(src string, fn func(line string)) {
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	inFence := false
	for sc.Scan() {
		line := sc.Text()
		if fenceRe.MatchString(line) {
			inFence = !inFence
			continue
		}
		if !inFence {
			fn(line)
		}
	}
}

// ParseTasks counts checked and unchecked markdown checkboxes.
func ParseTasks(src string) TaskStats {
	var t TaskStats
	for line := range strings.SplitSeq(src, "\n") {
		m := taskRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		t.Total++
		if strings.EqualFold(m[1], "x") {
			t.Done++
		}
	}
	return t
}

// ParseDelta counts requirement operations in a delta spec.
func ParseDelta(src string) DeltaStats {
	var d DeltaStats
	section := ""
	lines(src, func(line string) {
		if m := sectionRe.FindStringSubmatch(line); m != nil {
			section = strings.ToUpper(m[1])
			return
		}
		if strings.HasPrefix(line, "## ") {
			section = ""
			return
		}
		switch section {
		case "ADDED":
			if requirementRe.MatchString(line) {
				d.Added++
			}
		case "MODIFIED":
			if requirementRe.MatchString(line) {
				d.Modified++
			}
		case "REMOVED":
			if requirementRe.MatchString(line) || removedAltRe.MatchString(line) {
				d.Removed++
			}
		case "RENAMED":
			if renameFromRe.MatchString(line) {
				d.Renamed++
			}
		}
	})
	return d
}

// SpecMeta is the summary information extracted from a main spec.
type SpecMeta struct {
	Title        string
	Purpose      string
	Requirements int
	Scenarios    int
}

// ParseSpec extracts the title, purpose, and requirement counts from a spec.
func ParseSpec(src string) SpecMeta {
	var s SpecMeta
	section := ""
	lines(src, func(line string) {
		switch {
		case strings.HasPrefix(line, "# ") && s.Title == "":
			s.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "), strings.HasPrefix(line, "### ") && !requirementRe.MatchString(line):
			section = strings.TrimSpace(strings.TrimLeft(line, "# "))
		case requirementRe.MatchString(line):
			s.Requirements++
		case scenarioRe.MatchString(line):
			s.Scenarios++
		case strings.EqualFold(section, "Purpose") && s.Purpose == "":
			if text := strings.TrimSpace(line); text != "" {
				s.Purpose = text
			}
		}
	})
	return s
}

// ParseSummary returns the first prose line under "## Why" in a proposal,
// falling back to the first prose line in the document.
func ParseSummary(src string) string {
	var why, first string
	section := ""
	lines(src, func(line string) {
		text := strings.TrimSpace(line)
		if strings.HasPrefix(text, "#") {
			section = strings.TrimSpace(strings.TrimLeft(text, "#"))
			return
		}
		if text == "" || strings.HasPrefix(text, "<!--") {
			return
		}
		text = strings.TrimSpace(strings.TrimLeft(text, "-*> "))
		if first == "" {
			first = text
		}
		if why == "" && strings.EqualFold(section, "Why") {
			why = text
		}
	})
	if why != "" {
		return why
	}
	return first
}

// ParseSchema reads the "schema:" field from a change's .openspec.yaml.
func ParseSchema(src string) string {
	for line := range strings.SplitSeq(src, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "schema:"); ok {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}
