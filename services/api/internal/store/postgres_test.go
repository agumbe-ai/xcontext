package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/service"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

func TestPostgresProductFlow(t *testing.T) {
	url := os.Getenv("XCONTEXT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("XCONTEXT_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	st, err := store.OpenPostgres(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err = st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = st.ResetForTest(ctx); err != nil {
		t.Fatal(err)
	}

	svc := service.New(st, service.Config{StoreRawMode: "redacted"})
	scope := models.Scope{TenantID: "tenant-a", WorkspaceID: "workspace-a", UserID: "user-a", TrustedInterceptor: true}
	created, err := svc.Ingest(scope, service.IngestRequest{ContentType: "test_output", Source: "go test", Text: strings.Repeat("PASS package\n", 50) + "FATAL provider timeout", DeliveryVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	if created.Savings.DeliveredTokensSaved == 0 {
		t.Fatal("delivered savings were not persisted")
	}
	if got := svc.Usage(scope); got.TotalObjects != 1 {
		t.Fatalf("usage: %+v", got)
	}
	if got := svc.Search(scope, "", "provider timeout"); len(got) != 1 {
		t.Fatalf("search: %+v", got)
	}
	if got := svc.Objects(models.Scope{TenantID: "tenant-b", WorkspaceID: "workspace-a"}); len(got) != 0 {
		t.Fatal("cross-tenant objects leaked")
	}
	key, err := svc.CreateAPIKey(scope, "ci", "test", []string{"ingest"})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(key.Key))
	if _, err = st.GetAPIKeyByHash(hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
}
