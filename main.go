// Command lazyopenspec is a live terminal viewer for OpenSpec specs and
// changes. Leave it open next to your agents and watch the specs move.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FelineStateMachine/lazyopenspec/internal/spec"
	"github.com/FelineStateMachine/lazyopenspec/internal/ui"
)

var version = "dev"

func main() {
	interval := flag.Duration("interval", 400*time.Millisecond, "how often to poll openspec/ for edits")
	follow := flag.Bool("follow", false, "start in follow mode (jump to whatever changes)")
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
		fmt.Println("lazyopenspec", version)
		return
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

	p := tea.NewProgram(ui.New(ui.Options{Root: root, Interval: *interval, Follow: *follow}))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyopenspec:", err)
		os.Exit(1)
	}
}
