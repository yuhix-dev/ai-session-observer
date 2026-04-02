package viewer

import (
	"encoding/json"
	"strings"
	"time"
)

func parseEvents(source SourceKind, rawLines []string) []Event {
	events := make([]Event, 0, len(rawLines))
	for _, raw := range rawLines {
		event, ok := parseEvent(source, raw)
		if !ok {
			continue
		}
		events = append(events, event)
	}
	return events
}

func parseEvent(source SourceKind, raw string) (Event, bool) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return Event{}, false
	}
	switch source {
	case SourceClaude:
		return parseClaudeEvent(obj, raw), true
	case SourceCodex:
		return parseCodexEvent(obj, raw), true
	default:
		return Event{}, false
	}
}

func parseCodexEvent(obj map[string]any, raw string) Event {
	entryType := mapString(obj, "type")
	payload, _ := obj["payload"].(map[string]any)
	event := Event{
		Source:       SourceCodex,
		Timestamp:    mapString(obj, "timestamp"),
		EventKind:    EventUnknown,
		Raw:          raw,
		DiscoveredAt: time.Now(),
	}

	switch {
	case entryType == "event_msg" && mapString(payload, "type") == "task_started":
		event.EventKind = EventStatus
		event.Summary = "turn started"
	case entryType == "event_msg" && mapString(payload, "type") == "task_complete":
		event.EventKind = EventStatus
		event.Summary = headline(mapString(payload, "last_agent_message"), 110)
	case entryType == "event_msg" && mapString(payload, "type") == "agent_message":
		event.EventKind = EventThinking
		event.Summary = headline(mapString(payload, "message"), 110)
	case entryType == "event_msg" && mapString(payload, "type") == "user_message":
		event.EventKind = EventUser
		event.Summary = headline(mapString(payload, "message"), 110)
	case entryType == "event_msg" && mapString(payload, "type") == "turn_aborted":
		event.EventKind = EventStatus
		event.Summary = "turn aborted"
	case entryType == "event_msg" && mapString(payload, "type") == "mcp_tool_call_start":
		event.EventKind = EventToolCall
		event.Summary = codexToolSummary(payload, 110)
	case entryType == "event_msg" && mapString(payload, "type") == "mcp_tool_call_end":
		event.EventKind = EventToolOutput
		event.Summary = codexToolOutputSummary(payload, 110)
	case entryType == "response_item" && mapString(payload, "type") == "function_call":
		event.EventKind = EventToolCall
		name := mapString(payload, "name")
		if name == "spawn_agent" {
			event.EventKind = EventSubagent
		}
		event.Summary = strings.TrimSpace(name)
	case entryType == "response_item" && mapString(payload, "type") == "function_call_output":
		event.EventKind = EventToolOutput
		event.Summary = codexToolOutputSummary(payload, 110)
	case entryType == "response_item" && mapString(payload, "type") == "message":
		event.EventKind = EventResponse
		event.Summary = headline(listText(payload["content"]), 110)
	default:
		event.EventKind = EventUnknown
		event.Summary = headline(raw, 110)
	}

	event.Details = mustJSON(map[string]any{
		"timestamp": event.Timestamp,
		"type":      entryType,
		"payload":   payload,
	})
	return event
}

func codexToolSummary(payload map[string]any, limit int) string {
	for _, key := range []string{"name", "tool_name", "tool", "call_id"} {
		if value := mapString(payload, key); value != "" {
			return headline(value, limit)
		}
	}
	if value := summaryValue(payload["arguments"], limit); value != "" {
		return value
	}
	return "tool call"
}

func codexToolOutputSummary(payload map[string]any, limit int) string {
	for _, key := range []string{"name", "tool_name", "tool"} {
		if value := mapString(payload, key); value != "" {
			if output := summaryValue(payload["output"], limit); output != "" {
				return headline(value+": "+output, limit)
			}
			return headline(value, limit)
		}
	}
	if output := summaryValue(payload["output"], limit); output != "" {
		return output
	}
	if output := summaryValue(payload["content"], limit); output != "" {
		return output
	}
	if callID := mapString(payload, "call_id"); callID != "" {
		return headline("call "+callID, limit)
	}
	return "tool output"
}

func parseClaudeEvent(obj map[string]any, raw string) Event {
	entryType := mapString(obj, "type")
	event := Event{
		Source:       SourceClaude,
		Timestamp:    mapString(obj, "timestamp"),
		SessionID:    clip(mapString(obj, "sessionId"), 8),
		EventKind:    EventUnknown,
		Raw:          raw,
		DiscoveredAt: time.Now(),
	}

	switch entryType {
	case "assistant":
		event.EventKind = EventResponse
		content, _ := nestedMap(obj, "message")["content"].([]any)
		if len(content) > 0 {
			if first, ok := content[0].(map[string]any); ok && mapString(first, "type") == "tool_use" {
				event.EventKind = EventToolCall
				event.Summary = mapString(first, "name")
			}
		}
		if event.Summary == "" {
			event.Summary = headline(listText(nestedMap(obj, "message")["content"]), 96)
		}
	case "user":
		content := nestedMap(obj, "message")["content"]
		event.EventKind = EventUser
		if firstType(content) == "tool_result" {
			event.EventKind = EventToolOutput
			event.Summary = "tool result"
			break
		}
		event.Summary = headline(listText(content), 96)
	case "progress":
		data, _ := obj["data"].(map[string]any)
		progressType := mapString(data, "type")
		switch progressType {
		case "query_update":
			event.EventKind = EventThinking
			event.Summary = clip(clean(mapString(data, "query")), 96)
		case "search_results_received":
			event.EventKind = EventThinking
			event.Summary = mapString(data, "resultCount") + " results"
		case "agent_progress":
			event.EventKind = EventSubagent
			event.Summary = clip(clean(mapString(data, "prompt")), 96)
		default:
			event.EventKind = EventStatus
			event.Summary = progressType
		}
	case "system":
		event.EventKind = EventSystem
		event.Summary = mapString(obj, "subtype")
	case "agent-setting":
		event.EventKind = EventStatus
		event.Summary = mapString(obj, "agentSetting")
	default:
		event.EventKind = EventUnknown
		event.Summary = headline(raw, 96)
	}

	event.Details = mustJSON(map[string]any{
		"timestamp":    event.Timestamp,
		"sessionId":    mapString(obj, "sessionId"),
		"type":         entryType,
		"subtype":      mapString(obj, "subtype"),
		"agentSetting": mapString(obj, "agentSetting"),
		"message":      obj["message"],
		"data":         obj["data"],
	})
	return event
}

func firstType(content any) string {
	items, ok := content.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	return mapString(first, "type")
}
