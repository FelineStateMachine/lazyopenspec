package spec

import (
	"strings"
	"testing"
)

// md joins lines with "\n" so table cases stay readable.
func md(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

func TestParseTasks(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want TaskStats
	}{
		{"empty", "", TaskStats{}},
		{"no checkboxes", md("## 1. Group", "", "Some prose."), TaskStats{}},
		{"basic", md("## 1. Group", "", "- [ ] 1.1 a", "- [x] 1.2 b"), TaskStats{Done: 1, Total: 2}},
		{"uppercase X is done", md("- [X] 1.1 a", "- [ ] 1.2 b"), TaskStats{Done: 1, Total: 2}},
		{"star bullets count", md("* [x] a", "* [ ] b"), TaskStats{Done: 1, Total: 2}},
		{"plus bullets ignored", md("+ [x] a", "- [ ] b"), TaskStats{Done: 0, Total: 1}},
		{
			"indented sub-checkboxes ignored",
			md("- [ ] 2.2 parent", "  - [x] sub done", "  - [ ] sub open", "\t- [x] tab sub", "- [x] 2.3 next"),
			TaskStats{Done: 1, Total: 2},
		},
		{
			"checkboxes inside fenced code count (CLI does not mask fences)",
			md("- [x] real", "```markdown", "- [ ] example", "- [x] example done", "```", "~~~", "* [ ] tilde", "~~~"),
			TaskStats{Done: 2, Total: 4},
		},
		{"other marks are not tasks", md("- [~] partial", "- [-] skipped", "- [  ] two spaces", "- [ ] real"), TaskStats{Total: 1}},
		{"no space after bullet", md("-[ ] a", "*[x] b"), TaskStats{}},
		{"several spaces after bullet", md("-   [x] a", "*\t[ ] b"), TaskStats{Done: 1, Total: 2}},
		{"no description still counts", md("- [ ]", "- [x]"), TaskStats{Done: 1, Total: 2}},
		{"no space after bracket", md("- [x]1.1 squashed"), TaskStats{Done: 1, Total: 1}},
		{"numbered list ignored", md("1. [x] a", "2. [ ] b"), TaskStats{}},
		{"CRLF line endings", "- [x] a\r\n- [ ] b\r\n- [X] c\r\n", TaskStats{Done: 2, Total: 3}},
		{"no trailing newline", "- [x] a\n- [ ] b", TaskStats{Done: 1, Total: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseTasks(tt.src); got != tt.want {
				t.Errorf("ParseTasks() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTaskStatsComplete(t *testing.T) {
	tests := []struct {
		in   TaskStats
		want bool
	}{
		{TaskStats{}, false},
		{TaskStats{Done: 0, Total: 3}, false},
		{TaskStats{Done: 2, Total: 3}, false},
		{TaskStats{Done: 3, Total: 3}, true},
	}
	for _, tt := range tests {
		if got := tt.in.Complete(); got != tt.want {
			t.Errorf("%+v.Complete() = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseDelta(t *testing.T) {
	req := func(name string) []string {
		return []string{"### Requirement: " + name, "The system SHALL " + name + ".", "", "#### Scenario: s", "- **WHEN** a", "- **THEN** b", ""}
	}
	join := func(parts ...[]string) string {
		var all []string
		for _, p := range parts {
			all = append(all, p...)
		}
		return md(all...)
	}
	h := func(s ...string) []string { return s }

	tests := []struct {
		name string
		src  string
		want DeltaStats
	}{
		{"empty", "", DeltaStats{}},
		{"added", join(h("## ADDED Requirements", ""), req("A"), req("B")), DeltaStats{Added: 2}},
		{"modified", join(h("## MODIFIED Requirements", ""), req("A")), DeltaStats{Modified: 1}},
		{
			"removed header form",
			md("## REMOVED Requirements", "", "### Requirement: Remember me", "**Reason**: risk", "**Migration**: none"),
			DeltaStats{Removed: 1},
		},
		{
			"removed alt bullet form",
			md("## REMOVED Requirements", "", "- `### Requirement: Remember me`", "- `### Requirement: Legacy tokens`"),
			DeltaStats{Removed: 2},
		},
		{
			"removed alt bullet form without backticks",
			md("## REMOVED Requirements", "- ### Requirement: Remember me"),
			DeltaStats{Removed: 1},
		},
		{
			"renamed with leading dash",
			md("## RENAMED Requirements", "", "- FROM: `### Requirement: Password login`", "- TO: `### Requirement: Password authentication`"),
			DeltaStats{Renamed: 1},
		},
		{
			"renamed without leading dash",
			md("## RENAMED Requirements", "", "FROM: `### Requirement: Password login`", "TO: `### Requirement: Password authentication`"),
			DeltaStats{Renamed: 1},
		},
		{
			"renamed indented, no backticks",
			md("## RENAMED Requirements", "  - FROM: ### Requirement: A", "  - TO: ### Requirement: B"),
			DeltaStats{Renamed: 1},
		},
		{
			"two rename pairs",
			md("## RENAMED Requirements", "- FROM: `### Requirement: A`", "- TO: `### Requirement: B`", "", "- FROM: `### Requirement: C`", "- TO: `### Requirement: D`"),
			DeltaStats{Renamed: 2},
		},
		{
			"case-insensitive section headers",
			join(h("## added requirements"), req("A"), h("## Modified Requirements"), req("B"), h("## removed REQUIREMENTS", "### Requirement: C", "## Renamed Requirements", "- FROM: `### Requirement: D`", "- TO: `### Requirement: E`")),
			DeltaStats{Added: 1, Modified: 1, Removed: 1, Renamed: 1},
		},
		{"section header trailing whitespace", join(h("## ADDED Requirements   "), req("A")), DeltaStats{Added: 1}},
		{"case-insensitive requirement header", md("## ADDED Requirements", "### requirement: lower", "### REQUIREMENT: upper"), DeltaStats{Added: 2}},
		{"no space after ###", md("## ADDED Requirements", "###Requirement: tight"), DeltaStats{Added: 1}},
		{"H4 requirement is not a requirement", md("## ADDED Requirements", "#### Requirement: too deep"), DeltaStats{}},
		{"bullet form does not count under ADDED", md("## ADDED Requirements", "- `### Requirement: A`"), DeltaStats{}},
		{"requirement header under RENAMED is not a rename", md("## RENAMED Requirements", "### Requirement: A"), DeltaStats{}},
		{
			"requirements before any delta section are ignored",
			join(h("# Delta for auth", ""), req("Preamble"), h("## ADDED Requirements"), req("A")),
			DeltaStats{Added: 1},
		},
		{
			"unrelated ## section ends the delta section",
			join(h("## ADDED Requirements"), req("A"), h("## Notes"), req("Stray")),
			DeltaStats{Added: 1},
		},
		{
			"requirement headers inside fenced code do not count",
			join(h("## ADDED Requirements"), req("A"), h("```markdown", "### Requirement: Example only", "## REMOVED Requirements", "### Requirement: Also example", "```", "~~~", "### Requirement: Tilde example", "~~~")),
			DeltaStats{Added: 1},
		},
		{
			"section header inside fence does not switch section",
			md("## MODIFIED Requirements", "```", "## ADDED Requirements", "```", "### Requirement: A"),
			DeltaStats{Modified: 1},
		},
		{
			"all four operations",
			join(h("## ADDED Requirements"), req("A"), h("## MODIFIED Requirements"), req("B"), req("C"),
				h("## REMOVED Requirements", "### Requirement: D", "- `### Requirement: E`", "## RENAMED Requirements", "- FROM: `### Requirement: F`", "- TO: `### Requirement: G`")),
			DeltaStats{Added: 1, Modified: 2, Removed: 2, Renamed: 1},
		},
		{"CRLF line endings", "## ADDED Requirements\r\n### Requirement: A\r\nThe system SHALL a.\r\n", DeltaStats{Added: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDelta(tt.src)
			if got != tt.want {
				t.Errorf("ParseDelta() = %+v, want %+v", got, tt.want)
			}
			if got.Total() != tt.want.Total() {
				t.Errorf("Total() = %d, want %d", got.Total(), tt.want.Total())
			}
		})
	}
}

// The OpenSpec CLI (requirement-blocks.js parseRenamedPairs and
// change-parser.js parseRenames) only counts a rename when a
// "FROM: `### Requirement: X`" line is followed by a matching
// "TO: `### Requirement: Y`" line, and both regexes are case-sensitive and
// require the "### Requirement:" part. Verified: `openspec show <change>
// --json` reports deltaCount 0 for this input.
func TestParseDeltaRenamedNeedsFromToPair(t *testing.T) {
	t.Log("was BUG: parse.go:18 renameFromRe `(?i)^\\s*-?\\s*FROM:` counts every FROM-ish line under RENAMED " +
		"(orphan FROM without TO, prose like \"- From: ...\"), so ParseDelta reports Renamed=2 where the CLI reports 0")
	src := md("## RENAMED Requirements", "",
		"- FROM: `### Requirement: Password login`",
		"- From: the old wording was ambiguous")
	if got := ParseDelta(src); got.Renamed != 0 {
		t.Errorf("Renamed = %d, want 0 (no FROM/TO pair)", got.Renamed)
	}
}

// CommonMark (and the CLI's MarkdownParser.buildCodeFenceMask) only closes a
// fence with the same marker character and at least the same length, so a
// ``` block nested inside a ~~~ (or ````) block stays masked.
func TestParseNestedFenceStaysMasked(t *testing.T) {
	t.Log("was BUG: parse.go:29 lines() toggles inFence on ANY ``` or ~~~ line, so an inner ``` inside a ~~~ " +
		"(or ````) block un-masks the example; ParseSpec/ParseDelta count the fenced \"### Requirement:\" " +
		"(spec: 2 vs CLI `spec list` requirementCount 1)")
	example := []string{"~~~markdown", "```", "### Requirement: Fake inside nested fence", "#### Scenario: fake", "```", "~~~"}
	spec := md(append([]string{"# S", "## Purpose", "p", "## Requirements", "### Requirement: Real", "The system SHALL x.", "#### Scenario: s"}, example...)...)
	if got := ParseSpec(spec); got.Requirements != 1 || got.Scenarios != 1 {
		t.Errorf("ParseSpec() = %+v, want 1 requirement and 1 scenario", got)
	}
	delta := md(append([]string{"## ADDED Requirements", "### Requirement: Real"}, example...)...)
	if got := ParseDelta(delta); got.Added != 1 {
		t.Errorf("ParseDelta().Added = %d, want 1", got.Added)
	}
	four := md("## ADDED Requirements", "### Requirement: Real", "````md", "```", "### Requirement: Fake", "```", "````")
	if got := ParseDelta(four); got.Added != 1 {
		t.Errorf("ParseDelta(```` fence).Added = %d, want 1", got.Added)
	}
}

func TestParseSpec(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want SpecMeta
	}{
		{"empty", "", SpecMeta{}},
		{
			"standard layout",
			md("# Data Export Specification", "", "## Purpose", "", "Lets owners export.", "Second line.", "",
				"## Requirements", "", "### Requirement: CSV export", "The system SHALL export CSV.", "",
				"#### Scenario: Export", "- **WHEN** x", "- **THEN** y", "", "#### Scenario: Escape", "- **WHEN** x", "",
				"### Requirement: JSON export", "The system SHALL export JSON.", "", "#### Scenario: JSON", "- **WHEN** x"),
			SpecMeta{Title: "Data Export Specification", Purpose: "Lets owners export.", Requirements: 2, Scenarios: 3},
		},
		{
			"archive skeleton without blank lines",
			md("# notifications Specification", "", "## Purpose", "TBD - created by archiving change x. Update Purpose after archive.",
				"## Requirements", "### Requirement: A", "The system SHALL a.", "", "#### Scenario: s", "- **WHEN** x"),
			SpecMeta{Title: "notifications Specification", Purpose: "TBD - created by archiving change x. Update Purpose after archive.", Requirements: 1, Scenarios: 1},
		},
		{
			"### Purpose works like ## Purpose",
			md("# T", "## Overview", "### Purpose", "Nested purpose.", "## Requirements", "### Requirement: A"),
			SpecMeta{Title: "T", Purpose: "Nested purpose.", Requirements: 1},
		},
		{
			"purpose header is case-insensitive",
			md("# T", "## PURPOSE", "", "  Indented purpose text.  "),
			SpecMeta{Title: "T", Purpose: "Indented purpose text."},
		},
		{"no purpose section", md("# T", "## Requirements", "### Requirement: A"), SpecMeta{Title: "T", Requirements: 1}},
		{
			"purpose not taken from other sections",
			md("# T", "## Background", "Not the purpose.", "## Purpose", "", "Real purpose."),
			SpecMeta{Title: "T", Purpose: "Real purpose."},
		},
		{"empty purpose section", md("# T", "## Purpose", "", "## Requirements", "Body text."), SpecMeta{Title: "T"}},
		{"only first H1 is the title", md("# First", "# Second"), SpecMeta{Title: "First"}},
		{"no title", md("## Purpose", "p"), SpecMeta{Purpose: "p"}},
		{
			"requirement header is case-insensitive and needs ###",
			md("## Requirements", "### requirement: lower", "###Requirement: tight", "#### Requirement: too deep", "## Requirement: too shallow"),
			SpecMeta{Requirements: 2, Scenarios: 1},
		},
		{
			"H5 is not a scenario",
			md("## Requirements", "### Requirement: A", "#### Scenario: s", "##### Detail"),
			SpecMeta{Requirements: 1, Scenarios: 1},
		},
		{
			"requirement and scenario headers inside fenced code do not count",
			md("# T", "## Purpose", "p", "## Requirements", "### Requirement: Real", "The system SHALL x.", "#### Scenario: s",
				"```markdown", "### Requirement: Example", "#### Scenario: Example", "```",
				"~~~", "### Requirement: Tilde example", "~~~"),
			SpecMeta{Title: "T", Purpose: "p", Requirements: 1, Scenarios: 1},
		},
		{
			"heading inside fence does not become the title or purpose",
			md("```", "# Not a title", "## Purpose", "```", "# Title", "## Purpose", "Real."),
			SpecMeta{Title: "Title", Purpose: "Real."},
		},
		{
			"CRLF line endings",
			"# T\r\n## Purpose\r\nP.\r\n## Requirements\r\n### Requirement: A\r\n#### Scenario: s\r\n",
			SpecMeta{Title: "T", Purpose: "P.", Requirements: 1, Scenarios: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseSpec(tt.src); got != tt.want {
				t.Errorf("ParseSpec() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// The CLI counts requirements as the children of the "Requirements" section
// only (markdown-parser.js parseRequirements); a requirement header under a
// later "## Notes" section is a validation ERROR and is not counted.
// Verified: `openspec spec list --json` reports requirementCount 1 here.
func TestParseSpecIgnoresRequirementsOutsideRequirementsSection(t *testing.T) {
	t.Log("was BUG: parse.go:108 ParseSpec counts every \"### Requirement:\" header in the file regardless of section, " +
		"so a stray header under \"## Notes\" gives 2 where `openspec spec list` reports 1")
	src := md("# T", "## Purpose", "p", "## Requirements", "### Requirement: Real", "The system SHALL x.",
		"## Notes", "### Requirement: Stray", "The system SHALL not count this.")
	if got := ParseSpec(src); got.Requirements != 1 {
		t.Errorf("Requirements = %d, want 1", got.Requirements)
	}
}

func TestParseSummary(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"empty", "", ""},
		{"only headings", md("## Why", "## What Changes"), ""},
		{"line under Why", md("## Why", "", "Because sessions last too long.", "", "## What Changes", "- Shorten"), "Because sessions last too long."},
		{
			"prefers Why over earlier prose",
			md("# Change: Full-text task search", "", "Intro paragraph before why.", "", "## Why", "", "The real reason."),
			"The real reason.",
		},
		{"Why later in the document", md("## What Changes", "- Add search", "", "## Why", "Reason."), "Reason."},
		{"Why header is case-insensitive and any level", md("### WHY", "Reason."), "Reason."},
		{"falls back to first prose line", md("## What Changes", "", "- Add search endpoint", "- Rank results"), "Add search endpoint"},
		{"empty Why falls back", md("## Why", "", "## What Changes", "- Add search"), "Add search"},
		{"only first line of Why paragraph", md("## Why", "First line.", "Second line."), "First line."},
		{"skips HTML comments", md("## Why", "<!-- Explain the motivation -->", "Reason."), "Reason."},
		{"strips bullet and quote markers", md("## Why", "> - quoted bullet"), "quoted bullet"},
		{"strips star bullet", md("## Why", "* starred"), "starred"},
		{"trims indentation", md("## Why", "    indented reason   "), "indented reason"},
		{"ignores fenced code", md("## Why", "```", "code line", "```", "Prose."), "Prose."},
		{"CRLF line endings", "## Why\r\n\r\nReason.\r\n", "Reason."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseSummary(tt.src); got != tt.want {
				t.Errorf("ParseSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSummaryKeepsLeadingBold(t *testing.T) {
	t.Log("was BUG: parse.go:135 strings.TrimLeft(text, \"-*> \") strips a cutset, not a bullet prefix, so a Why line " +
		"starting with bold markdown loses its opening ** and becomes \"Security:** sessions last too long.\"")
	src := md("## Why", "", "**Security:** sessions last too long.")
	if got, want := ParseSummary(src), "**Security:** sessions last too long."; got != want {
		t.Errorf("ParseSummary() = %q, want %q", got, want)
	}
}

func TestParseSchema(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"empty", "", ""},
		{"standard", md("schema: spec-driven", "created: 2026-10-13"), "spec-driven"},
		{"not first line", md("created: 2026-10-13", "schema: workspace-planning"), "workspace-planning"},
		{"double quoted", md(`schema: "spec-driven"`), "spec-driven"},
		{"single quoted", md(`schema: 'spec-driven'`), "spec-driven"},
		{"no space after colon", md("schema:custom"), "custom"},
		{"missing", md("created: 2026-10-13"), ""},
		{"commented out", md("# schema: spec-driven"), ""},
		{"different key with same prefix", md("schemas: x"), ""},
		{"first wins", md("schema: a", "schema: b"), "a"},
		{"CRLF", "schema: spec-driven\r\ncreated: 2026-10-13\r\n", "spec-driven"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseSchema(tt.src); got != tt.want {
				t.Errorf("ParseSchema() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSchemaStripsInlineComment(t *testing.T) {
	t.Log("was BUG: parse.go:153 ParseSchema does not strip YAML inline comments, so " +
		"\"schema: spec-driven # default\" yields \"spec-driven # default\" (the CLI's YAML parser yields \"spec-driven\")")
	if got := ParseSchema("schema: spec-driven # default\n"); got != "spec-driven" {
		t.Errorf("ParseSchema() = %q, want %q", got, "spec-driven")
	}
}
