// Command lazyopenspec is a live terminal viewer for OpenSpec specs and
// changes. Leave it open next to your agents and watch the specs move.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FelineStateMachine/lazyopenspec/internal/spec"
	"github.com/FelineStateMachine/lazyopenspec/internal/ui"
)

// version is set by release builds (-ldflags "-X main.version=v1.2.3").
var version = ""

// versionString reports the release version, or the module version that
// `go install ...@vX` records, or "dev".
func versionString() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func main() {
	interval := flag.Duration("interval", 400*time.Millisecond, "how often to poll openspec/ for edits")
	follow := flag.Bool("follow", false, "start in follow mode (jump to whatever changes)")
	portrait := flag.String("portrait", "auto", "stack panels above the main pane like lazygit's portraitMode: auto (when <=84 cols and >=46 rows), always, never")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: lazyopenspec [flags] [path]\n\n")
		fmt.Fprintf(os.Stderr, "Watches the openspec/ directory of the repo containing path (default: .).\n\n")
		flag.PrintDefaults()
	}
	// Accept flags before or after the path: `lazyopenspec ~/repo --follow`.
	var positional []string
	for args := os.Args[1:]; ; {
		_ = flag.CommandLine.Parse(args) // ExitOnError handles failures
		args = flag.Args()
		if len(args) == 0 {
			break
		}
		positional, args = append(positional, args[0]), args[1:]
	}
	if *showVersion {
		fmt.Println("lazyopenspec", versionString())
		return
	}

	portraitMode, ok := ui.ParsePortraitMode(*portrait)
	if !ok {
		fmt.Fprintf(os.Stderr, "lazyopenspec: --portrait must be auto, always, or never (got %q)\n", *portrait)
		os.Exit(2)
	}

	start := "."
	if len(positional) > 0 {
		start = positional[0]
	}
	root, err := spec.FindRoot(start)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lazyopenspec:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(ui.Options{Root: root, Interval: *interval, Follow: *follow, Portrait: portraitMode}))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyopenspec:", err)
		os.Exit(1)
	}
}
