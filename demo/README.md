# Demo

From the repo root, record with `vhs demo/demo.tape` (needs vhs 0.12+, ttyd, ffmpeg). It builds lazyopenspec, runs `scripts/swarm-sim.sh` against `/tmp/lazyopenspec-demo`, writes the raw recording to `/tmp/lazyopenspec-demo-raw.gif` and the still to `docs/media/demo.png`. The simulator is random, so re-record if a take is dull. Then shrink it for the README (12 fps, 64 colors, last frame held 2s):

```sh
ffmpeg -y -i /tmp/lazyopenspec-demo-raw.gif -filter_complex "fps=12,tpad=stop_mode=clone:stop_duration=2,split[a][b];[a]palettegen=max_colors=64:stats_mode=diff[p];[b][p]paletteuse=dither=none:diff_mode=rectangle" docs/media/demo.gif
```
