# README demo: three simulated agents (scripts/swarm-sim.sh) work through
# a copy of testdata/fixture while lazyopenspec follows them. Run from the
# repo root with `vhs demo/demo.tape`; demo/README.md has the shrink step.
# The raw recording goes to /tmp, the still to docs/media/demo.png.
Output "/tmp/lazyopenspec-demo-raw.gif"

# 150x40 cells in Menlo 14 with 012's padding and pace.
Set Shell "bash"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1420
Set Height 780
Set Padding 20
Set LineHeight 1.15
Set TypingSpeed 55ms
Set CursorBlink false
Set Theme "Catppuccin Mocha"

# Build to a temp dir, start the swarm with its output discarded, and kill
# it (and its agents) when this shell exits, even if the recording fails.
# VHS's terminal does truecolor but doesn't say so; without COLORTERM the
# theme's hex colors are rounded to the 256-color palette.
Hide
Type `export COLORTERM=truecolor; bin=$(mktemp -d) && go build -o "$bin/lazyopenspec" . && export PATH="$bin:$PATH"` Enter
Wait@120s
Type `scripts/swarm-sim.sh /tmp/lazyopenspec-demo >/dev/null 2>&1 & sim=$!; trap 'kill $sim 2>/dev/null; rm -rf "$bin" /tmp/lazyopenspec-demo' EXIT HUP TERM` Enter
Sleep 2s
Type "clear" Enter
Sleep 300ms
Show

Type "lazyopenspec /tmp/lazyopenspec-demo --follow"
Sleep 500ms
Enter
Wait+Screen@10s /Activity/

# Follow mode jumps to whatever the agents touch.
Sleep 7s

# The git diff of the followed file, then back to the rendered view.
Type "d"
Sleep 4s
Type "d"
Sleep 1.5s

# Follow off to read a spec and its pending delta, then back on.
Type "f"
Sleep 800ms
Type "2"
Sleep 1.5s
Type "]"
Sleep 3s
Type "f"
Sleep 3s
Screenshot "docs/media/demo.png"
Sleep 4s

Type "q"
Sleep 300ms
Hide
Type `kill $sim; wait; rm -rf "$bin" /tmp/lazyopenspec-demo; trap - EXIT HUP TERM` Enter
Sleep 500ms
