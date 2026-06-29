package compression

import (
	"strings"
	"testing"
)

func TestLogCollapsesRepetitionAndKeepsError(t *testing.T) {
	r := Log{}.Compress(strings.Repeat("noise\n", 100)+strings.Repeat("ERROR boom\n", 20), 1000)
	if !strings.Contains(r.Summary, "ERROR boom") || !strings.Contains(r.Summary, "repeated 20 times") {
		t.Fatal(r.Summary)
	}
	if len(r.Signals) != 1 || r.Signals[0].SourceRange.StartLine != 101 {
		t.Fatalf("signals: %+v", r.Signals)
	}
}
func TestJSONSummarizesArray(t *testing.T) {
	r := JSON{}.Compress(`{"items":[{"id":1},{"id":2},{"id":3},{"id":4}],"status":"ok"}`, 1000)
	if !strings.Contains(r.Summary, `"length": 4`) {
		t.Fatal(r.Summary)
	}
}
