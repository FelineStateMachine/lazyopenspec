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
	renameFromRe  = regexp.MustCompile("^\\s*-?\\s*FROM:\\s*`?###\\s*Requirement:")
	renameToRe    = regexp.MustCompile("^\\s*-?\\s*TO:\\s*`?###\\s*Requirement:")
	fenceRe       = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	bulletRe      = regexp.MustCompile(`^(?:[-*+]\s+|>\s*)+`)
)

// lines yields the lines of a markdown document outside fenced code blocks.
func lines(src string, fn func(line string)) {
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	fence := "" // opening marker of the current fenced block
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			// CommonMark: a fence closes only with the same character and at
			// least the same length, so ``` inside ~~~ stays masked.
			switch {
			case fence == "":
				fence = m[1]
			case m[1][0] == fence[0] && len(m[1]) >= len(fence) && strings.TrimSpace(line[len(m[0]):]) == "":
				fence = ""
			}
			continue
		}
		if fence == "" {
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
	pendingFrom := false
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
			// Count FROM/TO pairs, like the CLI; a lone FROM is not a rename.
			switch {
			case renameFromRe.MatchString(line):
				pendingFrom = true
			case renameToRe.MatchString(line) && pendingFrom:
				d.Renamed++
				pendingFrom = false
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
	inRequirement := false
	lines(src, func(line string) {
		switch {
		case strings.HasPrefix(line, "# ") && s.Title == "":
			s.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "), strings.HasPrefix(line, "### ") && !requirementRe.MatchString(line):
			section = strings.TrimSpace(strings.TrimLeft(line, "# "))
			inRequirement = false
		case !strings.EqualFold(section, "Requirements"):
			// The CLI only counts requirements under "## Requirements".
			if strings.EqualFold(section, "Purpose") && s.Purpose == "" {
				s.Purpose = strings.TrimSpace(line)
			}
		case requirementRe.MatchString(line):
			s.Requirements++
			inRequirement = true
		case inRequirement && scenarioRe.MatchString(line):
			s.Scenarios++
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
		text = strings.TrimSpace(bulletRe.ReplaceAllString(text, ""))
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
			if i := strings.Index(v, " #"); i >= 0 {
				v = v[:i] // YAML inline comment
			}
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}
