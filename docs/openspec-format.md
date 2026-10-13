# OpenSpec 1.4.1 on-disk format (for lazyopenspec)

Source of truth: the `@fission-ai/openspec` npm package (v1.4.1, Node CLI, `dist/` is plain readable JS).
Everything below was confirmed by reading `dist/` and running the CLI against the fixture.

Fixture repo: `testdata/fixture` (copied without its git history).

---

## 1. `openspec init`

`openspec init --tools none .` (non-interactive) creates **only empty dirs**:

```
openspec/
openspec/specs/
openspec/changes/
openspec/changes/archive/
```

`config.yaml` is skipped in non-interactive mode ("Config: skipped (non-interactive mode)") unless you pass `--force`
(or run interactively). `openspec init --tools none --force .` adds `openspec/config.yaml`:

```yaml
schema: spec-driven

# Project context (optional)
# ... (commented examples for `context: |` and `rules: {proposal: [...], tasks: [...]}`)
```

With tools (`--tools claude` etc.) it also writes agent skills/commands outside `openspec/` (e.g. `.claude/skills/openspec-*/`,
`.claude/commands/opsx/*`). Not relevant to a viewer. **1.4.1 does not create `project.md` or `openspec/AGENTS.md`**; those are
0.x legacy files (init only prints a migration hint for `project.md`, which is superseded by `config.yaml` `context:`).

Note: git does not track empty dirs, so a fresh clone may lack `changes/archive/` (or even `specs/`). Handle missing dirs.

## 2. Directory tree (fixture)

```
openspec/
  config.yaml                                   # schema + optional context/rules
  specs/                                        # "main" (source-of-truth) specs
    data-export/spec.md
    notifications/spec.md                       # created by archive (Purpose still "TBD - ...")
    task-management/spec.md
    user-auth/spec.md
  changes/
    add-workspace-invites/                      # scaffold only (just ran `openspec new change`)
      .openspec.yaml
    add-due-dates/                              # proposal only (+ README.md from --description)
      .openspec.yaml  README.md  proposal.md
    add-task-search/                            # proposal + delta specs (one new capability, one modified)
      .openspec.yaml  proposal.md
      specs/search/spec.md                      # ADDED (new capability)
      specs/task-management/spec.md             # MODIFIED
    harden-session-security/                    # proposal+design+specs+tasks, tasks 4/10
      .openspec.yaml  proposal.md  design.md  tasks.md
      specs/user-auth/spec.md                   # ADDED + MODIFIED + REMOVED + RENAMED
    add-csv-export-filters/                     # proposal+specs+tasks (8/8), NO design.md
      .openspec.yaml  proposal.md  tasks.md
      specs/data-export/spec.md                 # ADDED + MODIFIED
    archive/
      2026-08-19-add-user-auth/                 # full change dir moved verbatim
        .openspec.yaml proposal.md design.md tasks.md specs/user-auth/spec.md
      2026-09-23-add-email-notifications/
        .openspec.yaml proposal.md tasks.md specs/{notifications,task-management}/spec.md
```

Optional extra dirs a viewer may see: `openspec/schemas/<name>/schema.yaml` (+ `templates/*.md`) for project-local workflow
schemas; `openspec/config.yml` (alt extension); legacy `openspec/project.md`, `openspec/AGENTS.md`.

## 3. Per-change files

### `.openspec.yaml` (written by `openspec new change`)
```yaml
schema: spec-driven
created: 2026-10-13          # YYYY-MM-DD, optional
# optional, workspace/initiative features only:
goal: "..."                  # string
affected_areas: [web, api]   # string[]
initiative: { store: my-store, id: my-initiative }   # kebab-case ids
```
Zod schema in `dist/core/change-metadata/schema.js`. `schema` must name an existing schema or the CLI errors.
Schema resolution for a change: explicit `--schema` > `.openspec.yaml schema` > `config.yaml schema` > `spec-driven`.
The file may be missing (hand-made or pre-1.x changes): treat as optional.

### `README.md`
Only created by `openspec new change <name> --description "..."`: `# <name>\n\n<description>\n`. Not an artifact.

### Change names
`new change` enforces `/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/`. Archived dirs (`2026-...`) fail this, which is why
`status`/`show` cannot address archived changes (see section 9).

## 4. Default workflow schema `spec-driven`

`schemas/spec-driven/schema.yaml` (also `workspace-planning`, same 4 artifact ids, used for workspaces):

| artifact | generates | requires |
|---|---|---|
| proposal | `proposal.md` | - |
| specs | `specs/**/*.md` (glob) | proposal |
| design | `design.md` | proposal |
| tasks | `tasks.md` | specs, design |

`apply: { requires: [tasks], tracks: tasks.md }`. Templates live in `schemas/spec-driven/templates/{proposal,spec,design,tasks}.md`
(`openspec templates --json` prints resolved paths). `openspec schemas --json` lists `{name, description, artifacts[], source: package|user|project}`.

Schema lookup order: `<repo>/openspec/schemas/<name>/` > `$XDG_DATA_HOME/openspec/schemas/<name>/` (default `~/.local/share/openspec/schemas`) > package.

### How `status` decides an artifact is done (`dist/core/artifact-graph/{state,outputs}.js`)
- **done** = the `generates` path exists as a file. Pure existence; empty file counts. For globs, fast-glob
  `specs/**/*.md` relative to the change dir (dotfiles excluded) must match >= 1 file. Note `specs/foo.md` (no capability dir)
  counts as done for status even though the delta parser only reads `specs/<cap>/spec.md`.
- **ready** = not done and every `requires` id is done.
- **blocked** = otherwise; `missingDeps` = unmet requires (sorted).
- Dependencies do NOT gate "done": `add-csv-export-filters` has tasks.md but no design.md, so status is
  proposal=done, design=ready, specs=done, tasks=done, `isComplete:false`.
- `isComplete` = every artifact done. It has nothing to do with task checkboxes.
- Artifact order in output = topological build order with alphabetical tie-break: `proposal, design, specs, tasks`.

## 5. Markdown formats

### proposal.md (template; no H1)
```
## Why
## What Changes          (bullets; **BREAKING** marker convention)
## Capabilities
### New Capabilities     - `<kebab-name>`: description
### Modified Capabilities
## Impact
```
Change **title** (used by `show`/`change list` JSON) = first match of `/^#\s+(?:Change:\s+)?(.+)$/im` in proposal.md, else the
change id. The template has no H1, so most titles fall back to the id. Fixture: `add-task-search` has `# Change: Full-text task search`;
`add-email-notifications` (archived) also has an H1; others don't.
Parsing a change for `show --json` throws unless `## Why` and `## What Changes` exist (case-insensitive header match, any level).

### design.md
`## Context`, `## Goals / Non-Goals` (with `**Goals:**`/`**Non-Goals:**`), `## Decisions`, `## Risks / Trade-offs`,
optional `## Migration Plan`, `## Open Questions`. Free-form; never parsed by the CLI.

### tasks.md and task progress
```
## 1. Group name

- [ ] 1.1 Task description
- [x] 1.2 Done task
```
Counting (`dist/utils/task-progress.js`, used by `list`, `view`, `archive`, `change list`):
```js
TASK_PATTERN           = /^[-*]\s+\[[\sx]\]/i   // tested against the raw line, NOT trimmed
COMPLETED_TASK_PATTERN = /^[-*]\s+\[x\]/i
```
- Only column-0 `-`/`*` bullets count. **Indented (nested) checkboxes are ignored entirely.** `+` bullets ignored.
- `[x]` and `[X]` both complete. `[~]`, `[-]` etc. are not tasks.
- Code fences are NOT excluded (checkboxes inside ``` blocks count).
- Missing tasks.md => `{total:0, completed:0}`.
- `list --json` status: `total==0 -> "no-tasks"`, `completed==total -> "complete"`, else `"in-progress"`.
`instructions apply --json` uses a slightly different regex, `/^[-*]\s*\[([ xX])\]\s*(.+)\s*$/` (space after bullet optional,
needs text), and returns per-task `{id: "<1-based ordinal>", description, done}` plus `state: blocked|all_done|ready`.
Fixture `harden-session-security/tasks.md` exercises this: 10 top-level tasks (4 done incl. one `[X]`) plus 2 indented
sub-checkboxes that the CLI ignores (=> `4/10`; counting nested would give `5/12`).

### Main spec `openspec/specs/<cap>/spec.md`
```
# <Title> Specification           (H1 is free text; the CLI never reads it)

## Purpose
<text>                            (required; < 50 chars => WARNING)

## Requirements
### Requirement: <name>
<first body line MUST contain SHALL or MUST>

#### Scenario: <name>
- **WHEN** ...
- **THEN** ...
- **AND** ...
```
Key regexes (`dist/core/parsers/*.js`):
```js
/^(#{1,6})\s+(.+)$/                          // generic header; fenced code (``` or ~~~) is masked out
/^##\s+Requirements\s*$/i                    // requirements section (ends at next /^##\s+/)
/^###\s*Requirement:\s*(.+)\s*$/i            // requirement header; name = trim(capture)
/^####\s+/                                   // scenario count (any H4 under a requirement)
/^##\s+(ADDED|MODIFIED|REMOVED|RENAMED)\s+Requirements\s*$/i   // forbidden in main specs (validation ERROR)
```
Section lookup for `Purpose`/`Requirements`/`Why`/`What Changes` is case-insensitive on the title and searches nested
sections, so `## Purpose` vs `### Purpose` both work. Requirement "text" = first non-empty line after the header (before any `#`).

Archive-created specs use this skeleton (note: archive also strips blank lines around `## Requirements` in specs it rewrites):
```
# <cap-id> Specification

## Purpose
TBD - created by archiving change <change-name>. Update Purpose after archive.
## Requirements
### Requirement: ...
```
Fixture `notifications` keeps the TBD purpose; `task-management` shows the blank-line stripping after a MODIFIED merge.

### Delta spec `openspec/changes/<change>/specs/<cap>/spec.md`
No H1/Purpose. Top-level `##` sections (case-insensitive):
```
## ADDED Requirements
### Requirement: Session revocation
Users SHALL ...                               (SHALL/MUST on first body line; `**Key**:` metadata lines skipped)
#### Scenario: ...                            (>= 1 required for ADDED/MODIFIED)

## MODIFIED Requirements
### Requirement: Session expiry               (name must match the main spec header; FULL replacement block)
...

## REMOVED Requirements
### Requirement: Remember me
**Reason**: ...
**Migration**: ...
   (alt form also accepted: - `### Requirement: Name`)

## RENAMED Requirements
- FROM: `### Requirement: Password login`
- TO: `### Requirement: Password authentication`
```
RENAMED regexes: `/^\s*-?\s*FROM:\s*`?###\s*Requirement:\s*(.+?)`?\s*$/` and same with `TO:`; pairs are FROM followed by TO.
A requirement block runs from its `### Requirement:` line to the next requirement header or `## ` line.
Only `specs/<dir>/spec.md` files are read for deltas (one level deep, file name must be `spec.md`).

### Archive (`openspec archive <name> -y`)
Applies deltas into `openspec/specs/<cap>/spec.md` (creating the skeleton for new caps), validates the rebuilt spec,
then moves the whole change dir to `openspec/changes/archive/<YYYY-MM-DD>-<name>/` (UTC date from `toISOString()`).
Archived dirs keep `.openspec.yaml`, all artifacts and the delta specs. Fixture archive dirs were renamed to backdated
dates after a real archive run.

## 6. Discovery rules differ per command (important for parity)

| command | what counts as a change | what counts as a spec | root |
|---|---|---|---|
| `list` (changes) | any **dir** in `changes/` except `archive` (dot-dirs included, proposal NOT required) | - | `./openspec` (cwd only) |
| `list --specs` | - | any dir in `specs/` (missing/unparseable spec.md => 0 requirements) | cwd only |
| `spec list` | - | any dir in `specs/` that has `spec.md` (dot-dirs included) | cwd only |
| `show`, `validate`, `change list` | dir not starting with `.`, not `archive`, **has proposal.md** | dir not starting with `.`, has `spec.md` | cwd only |
| `status`, `instructions`, `new change` | dir under `changesDir` matching kebab name | - | walks up ancestors to the nearest dir containing `openspec/` (or a workspace) |

So `add-workspace-invites` (no proposal.md) appears in `list --json` but not in `change list`, `show`, or `validate`.
`list`/`show`/`validate` fail from a subdirectory ("No OpenSpec changes directory found"); run them with cwd = repo root.

## 7. "last modified" / `--sort recent`

`dist/core/list.js getLastModified(dir)`: recursive walk of the change dir, max `stat.mtime` over **files** (incl.
`.openspec.yaml`, nested `specs/**`); if no files, the dir's own mtime. Output as `toISOString()` (UTC, ms). `--sort recent`
(default) = desc by that; `--sort name` = `localeCompare`. Git checkout resets mtimes, so on a fresh clone all changes tie.
(The fixture's mtimes were set with `touch -t` locally to give a meaningful order; they won't survive a clone.)

## 8. Validation (`openspec validate`)

- Change validation only looks at **delta specs** (`validateChangeDeltaSpecs`), never at proposal.md content.
  ERRORs: zero deltas total ("Change must have at least one delta..."), a `specs/<cap>/spec.md` with no delta headers,
  delta headers with no parsed entries, ADDED/MODIFIED without SHALL/MUST (`/\b(SHALL|MUST)\b/` on first body line) or without
  an `####` scenario, duplicates within a section, same name in MODIFIED+REMOVED / MODIFIED+ADDED / ADDED+REMOVED,
  MODIFIED using a RENAMED old name, RENAMED TO colliding with ADDED.
- Spec validation: needs `## Purpose` + `## Requirements`, >=1 requirement, each with SHALL/MUST (substring, case-sensitive) and
  >=1 non-empty scenario; delta headers or requirement headers outside `## Requirements` are ERRORs; Purpose <50 chars WARNING;
  requirement text >500 chars INFO; `--strict` makes WARNINGs fail.
- Exit code 1 if any item fails. Bulk results sorted by id (changes and specs interleaved).

**Fixture result: 7 passed, 1 failed.** `add-due-dates` (proposal-only) fails with "Change must have at least one delta". This is
inherent: any change before its specs exist fails validation. `add-workspace-invites` is not validated at all (no proposal.md).
All 4 main specs and the other 3 changes pass.

## 9. JSON output shapes (trimmed)

`openspec list --json` (only `changes`; `{"changes": []}` when empty):
```json
{"changes": [
  {"name": "harden-session-security", "completedTasks": 4, "totalTasks": 10,
   "lastModified": "2026-10-13T17:30:00.000Z", "status": "in-progress"},
  {"name": "add-workspace-invites", "completedTasks": 0, "totalTasks": 0, "lastModified": "...", "status": "no-tasks"},
  {"name": "add-csv-export-filters", "completedTasks": 8, "totalTasks": 8, "lastModified": "...", "status": "complete"}]}
```

`openspec list --specs --json`: **BUG in 1.4.1: `--json` is ignored, prints text**:
```
Specs:
  data-export         requirements 3
  ...
```
Use the deprecated `openspec spec list --json` instead (prints a deprecation warning on stderr):
```json
[{"id": "data-export", "title": "data-export", "requirementCount": 3}, ...]
```
(`title` is always the id, never the H1.) `openspec change list --json` (deprecated) is the change equivalent:
`[{"id","title","deltaCount","taskStatus":{"total","completed"}}]`, proposal-having changes only, sorted by id.

`openspec show <change> --json` (always prints `Warning: Ignoring flags not applicable to change: scenarios` on stderr):
```json
{"id": "harden-session-security", "title": "harden-session-security", "deltaCount": 4,
 "deltas": [
  {"spec": "user-auth", "operation": "ADDED",
   "description": "Add requirement: Users SHALL be able to list ...",
   "requirement": {"text": "Users SHALL ...", "scenarios": [{"rawText": "- **WHEN** ...\n- **THEN** ..."}]},
   "requirements": [ /* same object again */ ]},
  {"spec": "user-auth", "operation": "MODIFIED", "description": "Modify requirement: ...", "requirement": {...}, "requirements": [...]},
  {"spec": "user-auth", "operation": "REMOVED", "description": "Remove requirement: **Reason**: ...",
   "requirement": {"text": "**Reason**: ...", "scenarios": []}, "requirements": [...]},
  {"spec": "user-auth", "operation": "RENAMED", "description": "Rename requirement from \"Password login\" to \"Password authentication\"",
   "rename": {"from": "Password login", "to": "Password authentication"}}]}
```
**Lossy:** requirement names (the `### Requirement:` text) and scenario names are NOT in the JSON; `text` is the first body line
and scenarios are raw bodies. If no `specs/` deltas exist it falls back to parsing `- **cap**: desc` bullets in What Changes
(`add-due-dates` => `deltaCount: 0`). Errors (no Why / What Changes) abort with exit 1.

`openspec show <spec> --json --type spec`:
```json
{"id": "user-auth", "title": "user-auth", "overview": "<Purpose text>", "requirementCount": 4,
 "requirements": [{"text": "The system SHALL authenticate ...", "scenarios": [{"rawText": "- **WHEN** ...\n- **THEN** ..."}]}],
 "metadata": {"version": "1.0.0", "format": "openspec"}}
```
Same lossiness (no requirement/scenario names). Flags: `--requirements` (drop scenarios), `--no-scenarios`, `-r <1-based index>`.

`openspec status --change <name> --json`:
```json
{"changeName": "add-csv-export-filters", "schemaName": "spec-driven",
 "planningHome": {"kind": "repo", "root": "/abs/fixture", "changesDir": "/abs/fixture/openspec/changes", "defaultSchema": "spec-driven"},
 "changeRoot": "/abs/.../changes/add-csv-export-filters",
 "artifactPaths": {"specs": {"outputPath": "specs/**/*.md", "resolvedOutputPath": "/abs/.../specs/**/*.md",
                             "existingOutputPaths": ["/abs/.../specs/data-export/spec.md"]}, "proposal": {...}, ...},
 "isComplete": false, "applyRequires": ["tasks"],
 "nextSteps": ["Run openspec instructions design --change \"add-csv-export-filters\" --json before writing that artifact."],
 "actionContext": {"mode": "repo-local", "sourceOfTruth": "repo", "planningArtifacts": [...], "linkedContext": [],
                   "allowedEditRoots": ["/abs/fixture"], "requiresAffectedAreaSelection": false, "constraints": [...]},
 "artifacts": [
  {"id": "proposal", "outputPath": "proposal.md", "status": "done"},
  {"id": "design", "outputPath": "design.md", "status": "ready"},
  {"id": "specs", "outputPath": "specs/**/*.md", "status": "done"},
  {"id": "tasks", "outputPath": "tasks.md", "status": "done"}]}
```
Blocked entries add `"missingDeps": ["design", "specs"]`. Optional keys: `initiative`, `affectedAreas` (workspace mode).
`status` without `--change` and no changes: `{"changes": [], "message": "No active changes."}`; with changes it errors.

`openspec validate --all --json`:
```json
{"items": [
   {"id": "add-due-dates", "type": "change", "valid": false, "durationMs": 4,
    "issues": [{"level": "ERROR", "path": "file", "message": "Change must have at least one delta. ..."}]},
   {"id": "data-export", "type": "spec", "valid": true, "issues": [], "durationMs": 2}],
 "summary": {"totals": {"items": 8, "passed": 7, "failed": 1},
             "byType": {"change": {"items": 4, "passed": 3, "failed": 1}, "spec": {"items": 4, "passed": 4, "failed": 0}}},
 "version": "1.0"}
```
Issue `level`: `ERROR|WARNING|INFO`; `path` is `file`, a JSON-ish path (`requirements[0].scenarios`) or `<cap>/spec.md`; spec
structure issues may add `line`.

Also captured: `instructions apply --change X --json` (`progress {total,complete,remaining}`, `tasks[]`, `state`, `contextFiles`).

## 10. Timings (Apple M5 Pro, Node 22.22, fixture repo, median of 10)

| command | `OPENSPEC_TELEMETRY=0` | default (telemetry on) |
|---|---|---|
| `node -e ''` baseline | 19 ms | |
| `openspec --version` | 105 ms | |
| `list --json` | 110 ms | ~330 ms |
| `spec list --json` | 112 ms | |
| `show <change> --json` | 109 ms | |
| `show <spec> --json --type spec` | 109 ms | |
| `status --change X --json` | 121 ms | ~350 ms |
| `validate --all --json` | 115 ms | |
| `instructions apply --change X --json` | 121 ms | |

- Cost is essentially all Node startup + module loading (~100 ms); parsing is <10 ms. A scaled copy with 45 changes / 44 specs
  took the same ~110-120 ms per command.
- **Telemetry is on by default** (PostHog, flushed at exit, adds ~220 ms and a network call per invocation). If you shell out,
  set `OPENSPEC_TELEMETRY=0` (or `DO_NOT_TRACK=1`) in the child env.
- Full refresh of the fixture (12 processes: list, spec list, validate --all, 5x status, 4x show): ~1.6 s sequential, ~250 ms
  with all 12 in parallel.

Verdict: shelling out per file change is viable only if debounced (>= 300 ms), parallelised, scoped to the affected change, and
telemetry is disabled. Reading the files directly in Go is cheap, gives you data the JSON drops (requirement and scenario names),
and has no cwd/discovery quirks. Use the CLI only for rare things like validate.

## 11. Other 1.4.1 things a viewer should know

- **Global config**: `~/.config/openspec/config.json` (XDG_CONFIG_HOME; holds profile, delivery, workflows, telemetry id). Global
  data dir `~/.local/share/openspec/` (XDG_DATA_HOME): user schemas, `workspaces/registry.yaml`, `context-stores/`.
- **Workspaces** (`openspec workspace setup|link|...`): a workspace root is a dir containing `.openspec-workspace/view.yaml`
  (`version: 1, name, links: {name: path|null}, ...`) plus `changes/` directly at the root (**no `openspec/` dir**), optionally a
  `<name>.code-workspace` (written by `workspace open`). Default schema there is `workspace-planning`; area specs go under `specs/<area>/<cap>/spec.md`.
  `status` chooses the nearest planning home: a workspace wins if it is at least as deep as the nearest repo `openspec/`.
  `.openspec.yaml` in workspace changes may carry `goal` and `affected_areas`.
- **Context stores** (`openspec context-store setup|register|...`): store dirs (default `~/.local/share/openspec/context-stores/<id>/`)
  contain `.openspec-store/store.yaml`; registry at `context-stores/registry.yaml`.
- **Initiatives** (`openspec initiative create|show|list`): live inside a store at `<store>/initiatives/<id>/` with
  `initiative.yaml` (`version: 1, id, title, summary, status: exploring|active|complete|archived, created: YYYY-MM-DD, owners[], metadata{}`)
  and `requirements.md, design.md, decisions.md, questions.md, tasks.md`. A repo change links via `.openspec.yaml`
  `initiative: {store, id}` (`openspec set change <name> --initiative <id> --store <id>`); `status` then includes `initiative`.
- **config.yaml**: `schema` (string), `context` (string, <= 50 KB, injected into AI instructions), `rules` (`{artifactId: string[]}`).
  Parsed leniently field-by-field; `.yml` also accepted.
- **Schemas dir**: project-local `openspec/schemas/<name>/schema.yaml` (`openspec schema init|fork`) can define different
  artifact ids/paths, so a viewer that wants to be schema-aware should read `schema.yaml` `artifacts[].{id,generates,requires}`
  and resolve per-change schema as in section 3 rather than hard-coding the 4 artifacts.
- **`openspec view`**: dashboard; same discovery as `list` (any non-archive dir), buckets changes as draft (0 tasks), active (partial), completed (all tasks done).
- Archived changes are invisible to `list`, `show`, `status`, `validate` (no CLI command lists them); read
  `changes/archive/*/` directly. The archive date prefix is the only archive timestamp on disk.
- Line endings: all parsers normalise `\r\n?` to `\n` (except the task counter, whose regex is CRLF-safe anyway).
