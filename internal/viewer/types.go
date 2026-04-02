package viewer

import "time"

type SourceKind string

const (
	SourceClaude SourceKind = "claude"
	SourceCodex  SourceKind = "codex"
)

type Mode string

const (
	ModeSummary Mode = "summary"
	ModeDetails Mode = "details"
	ModeRaw     Mode = "raw"
)

type EventKind string

const (
	EventThinking   EventKind = "thinking"
	EventToolCall   EventKind = "tool_call"
	EventToolOutput EventKind = "tool_output"
	EventSubagent   EventKind = "subagent"
	EventResponse   EventKind = "response"
	EventStatus     EventKind = "status"
	EventUser       EventKind = "user"
	EventSystem     EventKind = "system"
	EventUnknown    EventKind = "unknown"
)

type Event struct {
	Source       SourceKind
	EventKind    EventKind
	Timestamp    string
	SessionID    string
	Summary      string
	Details      string
	Raw          string
	DiscoveredAt time.Time
}

type Snapshot struct {
	Source        SourceKind
	Path          string
	Label         string
	Lines         []string
	DiscoveredAt  time.Time
	LastRefreshed time.Time
}

type Options struct {
	Source     string
	Mode       Mode
	Once       bool
	PanePID    string
	PaneTarget string
	Lines      int
	Refresh    time.Duration
}
