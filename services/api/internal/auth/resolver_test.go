package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/service"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

func TestAPIKeyScopesAndTrustedInterceptor(t *testing.T) {
	st := store.NewMemory()
	svc := service.New(st, service.Config{})
	created, e := svc.CreateAPIKey(models.Scope{TenantID: "t", WorkspaceID: "w", UserID: "u"}, "mcp", "test", []string{"ingest", "intercept"})
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+created.Key)
	scope, e := (Resolver{Store: st}).Resolve(r)
	if e != nil {
		t.Fatal(e)
	}
	if !scope.TrustedInterceptor || !Permits(scope, "ingest") || Permits(scope, "admin") {
		t.Fatalf("scope: %+v", scope)
	}
}
func TestJWTSignatureAndClaims(t *testing.T) {
	secret := []byte("secret")
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"sub":"u","tenant_id":"t","workspace_id":"w","exp":%d}`, time.Now().Add(time.Hour).Unix())))
	signed := header + "." + payload
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signed))
	token := signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	scope, e := (Resolver{Store: store.NewMemory(), JWTSecret: secret}).Resolve(r)
	if e != nil {
		t.Fatal(e)
	}
	if scope.TenantID != "t" || !Permits(scope, "admin") {
		t.Fatalf("scope: %+v", scope)
	}
}
