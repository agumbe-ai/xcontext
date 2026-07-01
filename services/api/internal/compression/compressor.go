package compression

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
)

type Result struct {
	Summary  string
	Signals  []models.Signal
	Warnings []string
	Version  string
}

type Compressor interface {
	Compress(text string, maxTokens int) Result
}

func For(contentType string) Compressor {
	switch contentType {
	case "json":
		return JSON{}
	case "stack_trace":
		return StackTrace{}
	case "log", "test_output":
		return Log{}
	default:
		return Text{}
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var interesting = regexp.MustCompile(`(?i)error|fail(?:ed|ure)?|exception|warn|denied|timeout|fatal|panic`)

func clean(s string) string { return ansi.ReplaceAllString(strings.ReplaceAll(s, "\r\n", "\n"), "") }

func limit(s string, maxTokens int) string {
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	max := maxTokens * 4
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n[summary truncated]"
}

type Log struct{}

func (Log) Compress(text string, max int) Result {
	lines := strings.Split(clean(text), "\n")
	counts := map[string]int{}
	firstLine := map[string]int{}
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			counts[l]++
			if _, ok := firstLine[l]; !ok {
				firstLine[l] = i + 1
			}
		}
	}
	type pair struct {
		line  string
		count int
	}
	var hits []pair
	for l, n := range counts {
		if interesting.MatchString(l) {
			hits = append(hits, pair{l, n})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].count == hits[j].count {
			return firstLine[hits[i].line] < firstLine[hits[j].line]
		}
		return hits[i].count > hits[j].count
	})
	if len(hits) > 12 {
		hits = hits[:12]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Log summary:\n- %d lines processed; %d unique lines.\n", len(lines), len(counts))
	var signals []models.Signal
	for _, h := range hits {
		suffix := ""
		if h.count > 1 {
			suffix = fmt.Sprintf(" (repeated %d times)", h.count)
		}
		fmt.Fprintf(&b, "- %s%s\n", h.line, suffix)
		signals = append(signals, models.Signal{Type: "diagnostic", Text: h.line, SourceRange: models.SourceRange{StartLine: firstLine[h.line], EndLine: firstLine[h.line]}})
	}
	if len(hits) == 0 {
		b.WriteString("- No error, warning, failure, timeout, or fatal lines detected.\n")
	}
	return Result{Summary: limit(b.String(), max), Signals: signals, Version: "log-v1"}
}

type StackTrace struct{}

func (StackTrace) Compress(text string, max int) Result {
	lines := strings.Split(clean(text), "\n")
	var kept []string
	var signals []models.Signal
	seen := map[string]int{}
	repeated := 0
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if len(kept) < 12 && (i == 0 || strings.HasPrefix(t, "at ") || strings.Contains(t, ".go:") || strings.Contains(t, ".py:")) {
			if seen[t] > 0 {
				repeated++
				seen[t]++
				continue
			}
			seen[t] = 1
			kept = append(kept, t)
			signals = append(signals, models.Signal{Type: "stack_frame", Text: t, SourceRange: models.SourceRange{StartLine: i + 1, EndLine: i + 1}})
		}
	}
	summary := fmt.Sprintf("Stack trace summary (%d lines, %d repeated frames collapsed):\n- %s", len(lines), repeated, strings.Join(kept, "\n- "))
	return Result{Summary: limit(summary, max), Signals: signals, Version: "stacktrace-v2"}
}

type JSON struct{}

func (JSON) Compress(text string, max int) Result {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		r := Text{}.Compress(text, max)
		r.Warnings = append(r.Warnings, "invalid JSON; used text compressor")
		return r
	}
	var summarize func(any, int) any
	summarize = func(v any, depth int) any {
		if depth > 3 {
			return "[nested content omitted]"
		}
		switch x := v.(type) {
		case []any:
			sample := x
			if len(sample) > 3 {
				sample = sample[:3]
			}
			out := make([]any, len(sample))
			for i := range sample {
				out[i] = summarize(sample[i], depth+1)
			}
			return map[string]any{"length": len(x), "sample": out}
		case map[string]any:
			out := map[string]any{}
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if depth < 2 || regexp.MustCompile(`(?i)error|message|status|code|id|name|type|reason|details`).MatchString(k) {
					out[k] = summarize(x[k], depth+1)
				}
			}
			return out
		default:
			return x
		}
	}
	b, _ := json.MarshalIndent(summarize(value, 0), "", "  ")
	return Result{Summary: limit("JSON structural summary:\n"+string(b), max), Version: "json-v1"}
}

type Text struct{}

func (Text) Compress(text string, max int) Result {
	lines := strings.Split(clean(text), "\n")
	selected := map[int]bool{}
	for i := 0; i < len(lines) && i < 5; i++ {
		selected[i] = true
	}
	for i := len(lines) - 5; i < len(lines); i++ {
		if i >= 0 {
			selected[i] = true
		}
	}
	for i, l := range lines {
		if interesting.MatchString(l) {
			selected[i] = true
		}
	}
	idx := make([]int, 0, len(selected))
	for i := range selected {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var out []string
	var signals []models.Signal
	for _, i := range idx {
		t := strings.TrimSpace(lines[i])
		if t != "" {
			out = append(out, t)
			if interesting.MatchString(t) {
				signals = append(signals, models.Signal{Type: "diagnostic", Text: t, SourceRange: models.SourceRange{StartLine: i + 1, EndLine: i + 1}})
			}
		}
	}
	return Result{Summary: limit(fmt.Sprintf("Text summary (%d lines):\n%s", len(lines), strings.Join(out, "\n")), max), Signals: signals, Version: "text-v1"}
}
