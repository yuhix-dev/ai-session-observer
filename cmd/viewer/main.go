package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yuhix-dev/ai-session-observer/internal/viewer"
)

func main() {
	opts := viewer.Options{}
	flag.StringVar(&opts.Source, "source", "all", "source to watch: all, claude, codex")
	flag.Func("mode", "render mode: summary, details, raw", func(value string) error {
		switch viewer.Mode(value) {
		case viewer.ModeSummary, viewer.ModeDetails, viewer.ModeRaw:
			opts.Mode = viewer.Mode(value)
			return nil
		default:
			return fmt.Errorf("unsupported mode: %s", value)
		}
	})
	flag.BoolVar(&opts.Once, "once", false, "render a single snapshot and exit")
	flag.StringVar(&opts.PanePID, "pane-pid", "", "watch files attached to this pane PID")
	flag.StringVar(&opts.PaneTarget, "pane-target", "", "resolve pane PID from this tmux pane target")
	flag.IntVar(&opts.Lines, "lines", 12, "number of recent normalized events per stream")
	flag.DurationVar(&opts.Refresh, "refresh", 500*time.Millisecond, "refresh interval in follow mode")
	flag.Parse()

	if opts.Mode == "" {
		opts.Mode = viewer.ModeSummary
	}

	if err := viewer.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
