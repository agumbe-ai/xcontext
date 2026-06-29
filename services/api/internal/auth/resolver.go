package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

type Resolver struct {
	Store      store.Store
	JWTSecret  []byte
	DevEnabled bool
	DevScope   models.Scope
}

func (a Resolver) Resolve(r *http.Request) (models.Scope, error) {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer xctx_") {
		return a.apiKey(strings.TrimPrefix(header, "Bearer "))
	}
	if strings.HasPrefix(header, "Bearer ") && len(a.JWTSecret) > 0 {
		return a.jwt(strings.TrimPrefix(header, "Bearer "))
	}
	if a.DevEnabled {
		return a.DevScope, nil
	}
	return models.Scope{}, errors.New("valid bearer authentication is required")
}
func (a Resolver) apiKey(raw string) (models.Scope, error) {
	sum := sha256.Sum256([]byte(raw))
	key, e := a.Store.GetAPIKeyByHash(hex.EncodeToString(sum[:]))
	if e != nil {
		return models.Scope{}, errors.New("invalid API key")
	}
	now := time.Now().UTC()
	_ = a.Store.TouchAPIKey(key.ID, now)
	return models.Scope{TenantID: key.TenantID, WorkspaceID: key.WorkspaceID, UserID: "api-key:" + key.ID, TrustedInterceptor: has(key.Scopes, "intercept"), Scopes: key.Scopes}, nil
}
func (a Resolver) jwt(raw string) (models.Scope, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return models.Scope{}, errors.New("invalid JWT")
	}
	signed := parts[0] + "." + parts[1]
	sig, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil {
		return models.Scope{}, errors.New("invalid JWT")
	}
	mac := hmac.New(sha256.New, a.JWTSecret)
	_, _ = mac.Write([]byte(signed))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return models.Scope{}, errors.New("invalid JWT")
	}
	payload, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return models.Scope{}, errors.New("invalid JWT")
	}
	var c struct {
		TenantID       string `json:"tenant_id"`
		TenantIDAlt    string `json:"tenantId"`
		WorkspaceID    string `json:"workspace_id"`
		WorkspaceIDAlt string `json:"workspaceId"`
		UserID         string `json:"id"`
		Subject        string `json:"sub"`
		Expires        int64  `json:"exp"`
	}
	if json.Unmarshal(payload, &c) != nil {
		return models.Scope{}, errors.New("invalid JWT")
	}
	tenant := c.TenantID
	if tenant == "" {
		tenant = c.TenantIDAlt
	}
	workspace := c.WorkspaceID
	if workspace == "" {
		workspace = c.WorkspaceIDAlt
	}
	// Current Agumbe user sessions are tenant-scoped and may not carry an app/workspace claim.
	// Use an explicit, deterministic tenant namespace rather than a shared or fake workspace.
	if workspace == "" && tenant != "" {
		workspace = "tenant:" + tenant
	}
	user := c.UserID
	if user == "" {
		user = c.Subject
	}
	if tenant == "" || user == "" {
		return models.Scope{}, errors.New("JWT missing tenant, workspace, or subject")
	}
	if c.Expires > 0 && time.Now().Unix() >= c.Expires {
		return models.Scope{}, errors.New("JWT expired")
	}
	return models.Scope{TenantID: tenant, WorkspaceID: workspace, UserID: user, Scopes: []string{"admin"}}, nil
}
func has(scopes []string, want string) bool {
	for _, v := range scopes {
		if v == want || v == "admin" {
			return true
		}
	}
	return false
}
func Permits(scope models.Scope, want string) bool {
	if len(scope.Scopes) == 0 {
		return false
	}
	return has(scope.Scopes, want)
}
