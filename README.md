# lazyopenspec

A lazygit-style terminal viewer for [OpenSpec](https://github.com/Fission-AI/OpenSpec) repos. Leave it open in a
split while agents work. It shows each change's artifacts, task progress, and spec deltas updating live, with an
activity feed of what was touched.

```
 lazyopenspec  │  ~/code/tally  │   main +3 uncommitted  │  5 active, 1 ready to archive        FOLLOW  │  ● last edit just now
╭─[1] Changes─────────────────────────────╮╭─Change harden-session-security───────────────────────────────────────╮
│   add-csv-export-filters  PDST ▰▰▰▰▰ 8/8││ harden-session-security                         spec-driven  edited 2s ago │
│▌● harden-session-security M PDST ▰▰▱▱ 5/10││ ✓ proposal  ✓ design  ✓ specs  ✓ tasks  ▰▰▰▰▰▱▱▱▱▱▱▱ 5/10  Δ +1 ~1 -1 →1 │
╭─[4] Activity────────────────────────────╮│ proposal.md │ design.md │ ● tasks.md M │ Δ user-auth                       │
│ 14:05:25 ▸ harden-session…  tasks 4/10 → 5/10 ...
```

## Install / run

```sh
go install .                 # or: go build -o lazyopenspec .
lazyopenspec                 # finds openspec/ in the current dir or any parent
lazyopenspec ~/code/repo     # or point it at a repo
lazyopenspec --follow        # start in follow mode
lazyopenspec --interval 1s   # poll less often (default 400ms)
```

Requires Go 1.27+. `git` is optional: without it, the uncommitted markers and diff view are unavailable.

## What it shows

- **Changes**: every active change under `openspec/changes/`. `PDST` lights up for proposal, design, spec deltas,
  and tasks. The bar and count use the same top-level checkbox rule as `openspec list`, so the numbers match the CLI.
- **Specs**: main specs with requirement counts. `Δn` means n active changes have a delta against that spec. Open
  the spec and use `]` to read the pending deltas next to it.
- **Archive**: archived changes, newest first.
- **Activity**: a feed of new changes, added, edited, or removed artifacts, task progress (`tasks 4/10 → 5/10`),
  requirement count changes, and archives. `enter` jumps to the item.
- **Main pane**: the selected file rendered as markdown, with a tab per artifact. `d` switches to `git diff HEAD`
  for that file (untracked files show in full), so you see what an agent changed since the last commit.

Freshness: `●` bright = edited in the last 15s, dim = last 2 minutes. `M` / `+` = uncommitted / untracked in git.

## Keys

| key | action |
|---|---|
| `tab` / `shift+tab`, `1`-`4`, `0` | switch panel (`0` = main pane) |
| `j` `k` / arrows, `g` `G` | move selection, or scroll when the main pane is focused |
| `enter` / `l`, `esc` / `h` | focus main pane / back to list |
| `J` `K`, `ctrl+d` `ctrl+u` | scroll the main pane from anywhere |
| `[` `]` | previous / next file tab |
| `d` | toggle git diff vs HEAD |
| `f` | follow mode: jump to the item and file the agents touch next |
| `s` | sort changes by name / most recently edited |
| `z` | zoom the main pane |
| `e` | open the current file in `$VISUAL` / `$EDITOR` |
| `c` | clear the activity feed |
| `r` | force reload |
| `?` | help |
| `q` | quit |

Mouse: wheel scrolls the pane under the pointer, and clicking selects a row.

## How it watches

It does a stat-only walk of `openspec/` every `--interval` and re-parses only when something changed. It doesn't
use fsnotify because on macOS kqueue holds an open file descriptor per watched file. Polling also picks up
atomic write-then-rename saves, which agents and editors use all the time. Scroll position is kept per file, so a
live edit to the file you're reading doesn't send you back to the top.

It reads the files directly and doesn't shell out to the `openspec` CLI. The CLI takes about 100ms per call, mostly
Node startup, and can't read archived changes. See `docs/openspec-format.md` for the format notes it's based on.

## Try it with a fake swarm

```sh
scripts/swarm-sim.sh /tmp/lazyopenspec-demo   # copies testdata/fixture and starts simulated agents
go run . /tmp/lazyopenspec-demo --follow      # in another pane
```
