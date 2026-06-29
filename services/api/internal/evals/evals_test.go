package evals

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agumbe-ai/xcontext/services/api/internal/compression"
	"github.com/agumbe-ai/xcontext/services/api/internal/redaction"
)

type fixture struct {
	ContentType, Text       string
	MustContain             []string
	MinimumReductionPercent float64
	ExpectedRedactions      int
}

func TestCompressionQualityCorpus(t *testing.T) {
	files, e := filepath.Glob("testdata/*.json")
	if e != nil {
		t.Fatal(e)
	}
	if len(files) < 4 {
		t.Fatalf("eval corpus unexpectedly small: %d", len(files))
	}
	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			b, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			var f fixture
			if e = json.Unmarshal(b, &f); e != nil {
				t.Fatal(e)
			}
			safe, findings := redaction.New().Redact(f.Text)
			count := 0
			for _, v := range findings {
				count += v.Count
			}
			if count != f.ExpectedRedactions {
				t.Fatalf("redactions: got %d want %d", count, f.ExpectedRedactions)
			}
			result := compression.For(f.ContentType).Compress(safe, 1200)
			for _, want := range f.MustContain {
				if !strings.Contains(result.Summary, want) {
					t.Errorf("lost required signal %q in:\n%s", want, result.Summary)
				}
			}
			original := math.Ceil(float64(len([]rune(f.Text))) / 4)
			returned := math.Ceil(float64(len([]rune(result.Summary))) / 4)
			reduction := 0.0
			if original > 0 {
				reduction = math.Max(0, original-returned) * 100 / original
			}
			if reduction < f.MinimumReductionPercent {
				t.Errorf("reduction %.1f%% below %.1f%%", reduction, f.MinimumReductionPercent)
			}
		})
	}
}
