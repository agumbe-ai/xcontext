package service

import (
	"strings"
	"testing"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

func TestIngestRedactsCompressesAndAggregates(t *testing.T) {
	s := New(store.NewMemory(), Config{StoreRawMode: "redacted", CostPer1K: .01})
	scope := models.Scope{TenantID: "t1", WorkspaceID: "w1", UserID: "u1", TrustedInterceptor: true}
	text := strings.Repeat("PASS test one\n", 100) + "FATAL ProviderRouteNotFoundError\nAuthorization: Bearer abcdefghijklmnopqrstuvwxyz"
	got, err := s.Ingest(scope, IngestRequest{ContentType: "test_output", Source: "npm test", Text: text, DeliveryVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Savings.TokenSaved <= 0 || got.Savings.DeliveredTokensSaved <= 0 {
		t.Fatalf("unexpected savings: %+v", got.Savings)
	}
	if got.Redactions.Count != 1 {
		t.Fatalf("redactions: %+v", got.Redactions)
	}
	content, _, err := s.Retrieve(scope, got.ContextRef)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "abcdefghijklmnopqrstuvwxyz") || !strings.Contains(content, "[REDACTED:bearer_token]") {
		t.Fatalf("unsafe raw content: %s", content)
	}
	usage := s.Usage(scope)
	if usage.TotalObjects != 1 || usage.DeliveredTokensSaved == 0 || usage.RetrievalCount != 1 {
		t.Fatalf("usage: %+v", usage)
	}
}
func TestPotentialSavingsAreNotClaimedAsDelivered(t *testing.T) {
	s := New(store.NewMemory(), Config{})
	got, err := s.Ingest(models.Scope{TenantID: "t", WorkspaceID: "w"}, IngestRequest{ContentType: "text", Text: strings.Repeat("hello world\n", 100)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Savings.PotentialTokensSaved == 0 || got.Savings.DeliveredTokensSaved != 0 || got.Savings.EstimatedCostSaved != 0 {
		t.Fatalf("savings: %+v", got.Savings)
	}
}
func TestUntrustedCallerCannotAttestDelivery(t *testing.T) {
	s := New(store.NewMemory(), Config{})
	got, err := s.Ingest(models.Scope{TenantID: "t", WorkspaceID: "w"}, IngestRequest{ContentType: "text", Text: strings.Repeat("hello world\n", 100), DeliveryVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Savings.DeliveredTokensSaved != 0 || got.Savings.EstimatedCostSaved != 0 {
		t.Fatalf("untrusted delivery was accepted: %+v", got.Savings)
	}
}
func TestTenantIsolation(t *testing.T) {
	s := New(store.NewMemory(), Config{})
	got, _ := s.Ingest(models.Scope{TenantID: "one", WorkspaceID: "w"}, IngestRequest{ContentType: "text", Text: "secret context"})
	if _, _, err := s.Retrieve(models.Scope{TenantID: "two", WorkspaceID: "w"}, got.ContextRef); err == nil {
		t.Fatal("cross-tenant retrieval succeeded")
	}
}

func TestAPIKeyIsReturnedOnceAndCanBeRevoked(t *testing.T) {
	s := New(store.NewMemory(), Config{})
	scope := models.Scope{TenantID: "t", WorkspaceID: "w", UserID: "u"}
	created, e := s.CreateAPIKey(scope, "ci", "live", []string{"ingest"})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(created.Key, "xctx_live_") {
		t.Fatal(created.Key)
	}
	listed := s.APIKeys(scope)
	if len(listed) != 1 || listed[0].KeyHash != "" {
		t.Fatalf("listed: %+v", listed)
	}
	if e = s.RevokeAPIKey(scope, created.ID); e != nil {
		t.Fatal(e)
	}
}
