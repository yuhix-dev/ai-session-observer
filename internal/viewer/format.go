package viewer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var tokenPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._-]+`), `${1}[REDACTED]`},
	{regexp.MustCompile(`\b(?:sk|rk|pk)_[A-Za-z0-9_-]+\b`), `[REDACTED]`},
	{regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|password|credential)\b\s*[:=]\s*["']?[^,"'\s}]+`), `${1}:[REDACTED]`},
}

func redact(text string) string {
	out := text
	for _, pattern := range tokenPatterns {
		out = pattern.re.ReplaceAllString(out, pattern.repl)
	}
	return out
}

func clean(text string) string {
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}

func clip(text string, limit int) string {
	if limit <= 3 || len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
}

func headline(text string, limit int) string {
	text = clean(text)
	for _, mark := range []string{"。", ".", "!", "?"} {
		if idx := strings.Index(text, mark); idx >= 0 {
			text = text[:idx+len(mark)]
			break
		}
	}
	return clip(text, limit)
}

func stringValue(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case int, int32, int64, float32, float64, bool:
		return fmt.Sprint(value)
	default:
		return ""
	}
}

func listText(content any) string {
	items, ok := content.([]any)
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if obj["type"] == "text" || obj["type"] == "output_text" {
			if text := stringValue(obj["text"]); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return clean(strings.Join(parts, " "))
}

func summaryValue(value any, limit int) string {
	switch typed := value.(type) {
	case string:
		return headline(typed, limit)
	case []any:
		if text := listText(typed); text != "" {
			return headline(text, limit)
		}
	case map[string]any:
		if text := mapString(typed, "text"); text != "" {
			return headline(text, limit)
		}
		if text := mapString(typed, "message"); text != "" {
			return headline(text, limit)
		}
	}

	if text := stringValue(value); text != "" {
		return headline(text, limit)
	}

	return headline(mustJSON(value), limit)
}

func formatEvent(mode Mode, event Event) string {
	switch mode {
	case ModeRaw:
		return redact(event.Raw)
	case ModeDetails:
		if event.Details != "" {
			return redact(event.Details)
		}
		return redact(event.Raw)
	default:
		ts := compactTimestamp(event.Timestamp)
		parts := []string{ts, string(event.EventKind), event.Summary}
		if event.SessionID != "" {
			parts = append(parts, "#"+event.SessionID)
		}
		return redact(strings.Join(filterEmpty(parts), "\t"))
	}
}

func compactTimestamp(ts string) string {
	if ts == "" {
		return "--:--:--"
	}
	if idx := strings.Index(ts, "T"); idx >= 0 {
		ts = ts[idx+1:]
	}
	if idx := strings.Index(ts, "."); idx >= 0 {
		ts = ts[:idx]
	}
	ts = strings.TrimSuffix(ts, "Z")
	return ts
}

func filterEmpty(items []string) []string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if item != "" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func mustJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func mapString(obj map[string]any, key string) string {
	return stringValue(obj[key])
}

func nestedMap(obj map[string]any, keys ...string) map[string]any {
	current := obj
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return map[string]any{}
		}
		current = next
	}
	return current
}
