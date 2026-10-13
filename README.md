# lazyopenspec

A lazygit-style terminal viewer for [OpenSpec](https://github.com/Fission-AI/OpenSpec) repos. Leave it open in a
split while agents work. It shows each change's artifacts, task progress, and spec deltas updating live, with an
activity feed of what was touched.

```
 lazyopenspec  │  /private/tmp/los-shot  │   main +1 uncommitted  │  5 active, 1 rea… FOLLOW  │  ● last edit just now
╭─[1] Changes──────────────────────────────╮╭─Change harden-session-security─────────────────────────────────────────╮
│   add-csv-export-filters PDST ▰▰▰▰▰   8/8││ harden-session-security                   spec-driven  edited just now │
│   add-due-dates          PDST ▱▱▱▱▱      ││ A September penetration test flagged our 30-day idle sessions (90 days…│
│   add-task-search        PDST ▱▱▱▱▱      ││ ✓ proposal  ✓ design  ✓ specs  ✓ tasks  ▰▰▰▰▰▰▱▱▱▱▱▱ 5/10  Δ +1 ~1 -1 …│
│   add-workspace-invites  PDST ▱▱▱▱▱      ││ proposal.md │ design.md │ ● tasks.md M │ Δ user-auth                   │
│▌● harden-session-secu… M PDST ▰▰▱▱▱  5/10││────────────────────────────────────────────────────────────────────────│
│                                          ││                                                                        │
│                                          ││  ## 1. Session storage                                                 │
│                                          ││                                                                        │
│                                          ││  [✓] 1.1 Add nullable  absolute_expires_at  to  sessions               │
│                                          ││  (migration 0042)                                                      │
│                                          ││  [✓] 1.2 Backfill  absolute_expires_at = now() + interval '7           │
│                                          ││  days'  in batches of 5,000                                            │
│                                          ││                                                                        │
╰─5 of 5───────────────────────────────────╯│  ## 2. Expiry and revocation                                           │
╭─[2] Specs────────────────────────────────╮│                                                                        │
│   data-export                  Δ1   3 req││  [✓] 2.1 Enforce 12h idle timeout in session middleware                │
│   notifications                     2 req││  [✓] 2.2 Enforce 7-day absolute lifetime (treat NULL as                │
│   task-management              Δ1   4 req││   created_at + 7d )                                                    │
│   user-auth                    Δ1   4 req││    [✓] Unit test: session used 1h ago but created 8 days ago is        │
╰─1 of 4───────────────────────────────────╯│    rejected                                                            │
╭─[3] Archive──────────────────────────────╮│    [ ] Unit test: NULL  absolute_expires_at  falls back to             │
│ 09-23   add-email-notif… PDST ▰▰▰▰▰   7/7││     created_at                                                         │
│ 08-19   add-user-auth    PDST ▰▰▰▰▰ 10/10││  [ ] 2.3 Add  GET /account/sessions  and  DELETE                       │
╰─1 of 2───────────────────────────────────╯│  /account/sessions/:id                                                 │
╭─[4] Activity─────────────────────────────╮│  [ ] 2.4 Build Active sessions page with "This device" label           │
│ 14:26:02 ▸ harden-sessio… tasks 4/10 → 5…││                                                                        │
╰─1 of 1───────────────────────────────────╯╰─  0%───────────────────────────────────────────────────────────────────╯
 j/k move  enter read  [ ] tab  d diff  e edit  s sort  f follow  z zoom  ? help  q quit
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
Git is run with `GIT_OPTIONAL_LOCKS=0`, so the viewer never holds `.git/index.lock` while agents are committing.

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
