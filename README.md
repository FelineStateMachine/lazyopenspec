# lazyopenspec

A lazygit-style terminal viewer for [OpenSpec](https://github.com/Fission-AI/OpenSpec) repos, built on
[Bubble Tea v2](https://github.com/charmbracelet/bubbletea). Leave it open in a split while agents work and watch
changes, task progress, and spec deltas update as files land on disk, with a running feed of what was touched.

![Following a swarm of simulated agents: tasks checked off, new changes drafted, a change archived](docs/media/demo.gif)

## Install

```sh
go install github.com/FelineStateMachine/lazyopenspec@latest   # needs Go 1.27+
```

Or download a binary for macOS, Linux, or Windows from
[Releases](https://github.com/FelineStateMachine/lazyopenspec/releases).

## Run

```sh
lazyopenspec                    # finds openspec/ in the current directory or any parent
lazyopenspec ~/code/repo        # or point it at a repo
lazyopenspec --follow           # jump to whatever the agents touch next
lazyopenspec --portrait always  # stack panels above the main pane: auto (default), always, never
lazyopenspec --interval 1s      # poll less often (default 400ms)
```

It only reads files. `git` is optional; without it you lose the uncommitted markers and the diff view.

## What it shows

- **Changes**: every active change under `openspec/changes/`. `PDST` lights up as the proposal, design, spec
  deltas, and tasks appear. The progress bar counts top-level checkboxes the way `openspec list` does, so the
  numbers match the CLI.
- **Specs**: the main specs with requirement counts. `Δ2` means two active changes have a delta against that spec;
  open the spec and press `]` to read them next to it.
- **Archive**: archived changes, newest first.
- **Activity**: new changes, added, edited, or removed files, task progress (`tasks 4/10 → 5/10`), requirement
  count changes, and archives, newest first. `enter` jumps to the item.
- **Main pane**: the selected file rendered as markdown, one tab per artifact. `d` switches to `git diff HEAD` for
  that file, so you see what an agent changed since its last commit.

A bright `●` means edited in the last 15 seconds and a dim one the last 2 minutes. `M` and `+` mark uncommitted and
untracked files.

## Follow mode

`f` (or `--follow`) moves the selection to whatever changed most recently and opens the file that changed, so the
main pane always shows what the agents are doing right now. Scroll position is kept per file, so an edit to the
file you're reading doesn't send you back to the top.

## Vertical stack (portrait) mode

This works like lazygit's `portraitMode`. In a tall, narrow pane (84 columns or fewer and 46 rows or more,
lazygit's defaults), the panels stack on top and take a third of the height, and the main pane gets the rest at
full width. `--portrait always` or `never` overrides that, and `v` flips the layout while it runs.

When the panels are short on space, the focused one expands and the others shrink to their selected row, or to
just their title if space is very tight. Tab between panels to open each one in turn.

## Keys

| Key | Action |
|---|---|
| `tab` / `shift+tab`, `1`-`4`, `0` | switch panel (`0` is the main pane) |
| `j` `k` / arrows, `g` `G` | move the selection, or scroll when the main pane is focused |
| `enter` / `l`, `esc` / `h` | focus the main pane, back to the list |
| `J` `K`, `ctrl+d` `ctrl+u` | scroll the main pane from anywhere |
| `[` `]` | previous / next file tab |
| `d` | git diff against HEAD |
| `f` | follow mode |
| `s` | sort changes by name or by most recent edit |
| `z` | zoom the main pane |
| `v` | toggle the vertical stack layout |
| `e` | open the file in `$VISUAL` or `$EDITOR` |
| `c` | clear the activity feed |
| `r` | reload |
| `?` | help |
| `q` | quit |

The mouse wheel scrolls the pane under the pointer, and clicking selects a row.

## How it works

- **Polling, not fsnotify.** It stats every file under `openspec/` every `--interval` and re-parses only when
  something changed. On macOS, fsnotify holds a file descriptor open per watched file, and agents save files by
  writing a temp file and renaming it, which polling handles without special cases.
- **Reads the files directly.** It doesn't call the `openspec` CLI, which takes about 100 ms per call and can't read
  archived changes. The parsers follow the CLI's rules; [docs/openspec-format.md](docs/openspec-format.md) has the
  format notes they're based on.
- **Stays out of git's way.** Git runs with `GIT_OPTIONAL_LOCKS=0`, so the viewer never takes `.git/index.lock`
  while agents are committing.

## Try it without agents

`scripts/swarm-sim.sh` copies the test fixture to a temp repo and runs three simulated agents that draft changes,
check off tasks, add requirements, and archive finished work:

```sh
scripts/swarm-sim.sh /tmp/lazyopenspec-demo   # ctrl+c to stop
lazyopenspec /tmp/lazyopenspec-demo --follow  # in another pane
```

## Development

```sh
make check   # gofmt, go vet, staticcheck, tests with -race
make build   # bin/lazyopenspec
make demo    # re-record docs/media/demo.gif with vhs
```

## License

[MIT](LICENSE)
