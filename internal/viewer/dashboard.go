package viewer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Run(opts Options) error {
	if opts.Lines <= 0 {
		opts.Lines = 12
	}
	if opts.Refresh <= 0 {
		opts.Refresh = 500 * time.Millisecond
	}

	if opts.Once {
		snapshots := collectSnapshots(opts)
		renderDashboard(opts, snapshots)
		return nil
	}

	ticker := time.NewTicker(opts.Refresh)
	defer ticker.Stop()

	for {
		snapshots := collectSnapshots(opts)
		renderDashboard(opts, snapshots)
		<-ticker.C
	}
}

func collectSnapshots(opts Options) []Snapshot {
	kinds := selectedSources(opts.Source)
	snapshots := make([]Snapshot, 0, len(kinds))
	for _, source := range kinds {
		files := filesForSource(source, opts)
		if len(files) == 0 {
			snapshots = append(snapshots, Snapshot{
				Source:        source,
				Label:         "no readable source",
				LastRefreshed: time.Now(),
			})
			continue
		}
		for _, path := range files {
			snapshots = append(snapshots, snapshotForFile(source, path, opts))
		}
	}
	return snapshots
}

func snapshotForFile(source SourceKind, path string, opts Options) Snapshot {
	lines, err := readTailLines(path, opts.Lines*3)
	if err != nil {
		return Snapshot{
			Source:        source,
			Path:          path,
			Label:         err.Error(),
			LastRefreshed: time.Now(),
		}
	}
	events := parseEvents(source, lines)
	formatted := make([]string, 0, len(events))
	for _, event := range events {
		formatted = append(formatted, formatEvent(opts.Mode, event))
	}
	if len(formatted) > opts.Lines {
		formatted = formatted[len(formatted)-opts.Lines:]
	}
	return Snapshot{
		Source:        source,
		Path:          path,
		Label:         labelFor(source, path),
		Lines:         formatted,
		LastRefreshed: time.Now(),
	}
}

func filesForSource(source SourceKind, opts Options) []string {
	switch source {
	case SourceClaude:
		return discoverClaudeFiles(opts)
	case SourceCodex:
		return discoverCodexFiles(opts)
	default:
		return nil
	}
}

func selectedSources(value string) []SourceKind {
	switch value {
	case "", "all":
		return []SourceKind{SourceClaude, SourceCodex}
	case "claude":
		return []SourceKind{SourceClaude}
	case "codex":
		return []SourceKind{SourceCodex}
	default:
		return []SourceKind{SourceClaude, SourceCodex}
	}
}

func renderDashboard(opts Options, snapshots []Snapshot) {
	clearScreen()
	fmt.Printf("ai-session-observer  source=%s  mode=%s  updated=%s\n", defaultSource(opts.Source), opts.Mode, time.Now().Format("15:04:05"))
	fmt.Println(strings.Repeat("=", 88))

	for idx, snapshot := range snapshots {
		if idx > 0 {
			fmt.Println()
		}
		fmt.Printf("[%s] %s\n", strings.ToUpper(string(snapshot.Source)), snapshot.Label)
		if snapshot.Path != "" {
			fmt.Printf("file: %s\n", snapshot.Path)
		}
		if len(snapshot.Lines) == 0 {
			fmt.Println("(no normalized events yet)")
			continue
		}
		for _, line := range snapshot.Lines {
			fmt.Println(line)
		}
	}
}

func clearScreen() {
	fmt.Fprint(os.Stdout, "\033[H\033[2J")
}

func defaultSource(source string) string {
	if source == "" {
		return "all"
	}
	return source
}

func labelFor(source SourceKind, path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	switch source {
	case SourceClaude:
		if strings.Contains(path, "/subagents/agent-") {
			return "subagent " + strings.TrimPrefix(base, "agent-")
		}
		return "session " + base
	case SourceCodex:
		return "rollout " + base
	default:
		return base
	}
}
