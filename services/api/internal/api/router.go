package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/auth"
	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/service"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

type ScopeResolver interface {
	Resolve(*http.Request) (models.Scope, error)
}
type DevScopeResolver struct {
	Enabled bool
	Scope   models.Scope
}

func (d DevScopeResolver) Resolve(r *http.Request) (models.Scope, error) {
	if !d.Enabled {
		return models.Scope{}, errors.New("production auth is not configured")
	}
	return d.Scope, nil
}

type Router struct {
	svc  *service.Service
	auth ScopeResolver
	base string
	log  *slog.Logger
}

func New(svc *service.Service, auth ScopeResolver, base string, log *slog.Logger) http.Handler {
	r := &Router{svc: svc, auth: auth, base: strings.TrimSuffix(base, "/"), log: log}
	return r.middleware(http.HandlerFunc(r.serve))
}
func (a *Router) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		start := time.Now()
		next.ServeHTTP(w, r)
		a.log.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	write(w, status, map[string]any{"error": map[string]string{"message": msg}})
}
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		fail(w, 400, "invalid request body")
		return false
	}
	return true
}
func (a *Router) scope(w http.ResponseWriter, r *http.Request) (models.Scope, bool) {
	s, e := a.auth.Resolve(r)
	if e != nil {
		fail(w, 401, e.Error())
		return s, false
	}
	return s, true
}
func permit(w http.ResponseWriter, s models.Scope, permission string) bool {
	if !auth.Permits(s, permission) {
		fail(w, 403, "insufficient scope")
		return false
	}
	return true
}
func (a *Router) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		write(w, 200, map[string]string{"status": "ok"})
		return
	}
	if !strings.HasPrefix(r.URL.Path, a.base) {
		fail(w, 404, "not found")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, a.base)
	sc, ok := a.scope(w, r)
	if !ok {
		return
	}
	switch {
	case r.Method == "POST" && path == "/objects":
		if !permit(w, sc, "ingest") {
			return
		}
		var req service.IngestRequest
		if !decode(w, r, &req) {
			return
		}
		v, e := a.svc.Ingest(sc, req)
		if e != nil {
			fail(w, 400, e.Error())
			return
		}
		write(w, 201, v)
	case r.Method == "GET" && path == "/objects":
		if !permit(w, sc, "read") {
			return
		}
		write(w, 200, map[string]any{"items": a.svc.Objects(sc)})
	case r.Method == "GET" && strings.HasPrefix(path, "/objects/"):
		if !permit(w, sc, "read") {
			return
		}
		v, e := a.svc.Object(sc, strings.TrimPrefix(path, "/objects/"))
		if e != nil {
			fail(w, 404, "not found")
			return
		}
		write(w, 200, v)
	case r.Method == "GET" && path == "/sessions":
		if !permit(w, sc, "read") {
			return
		}
		write(w, 200, map[string]any{"items": a.svc.Sessions(sc)})
	case r.Method == "GET" && strings.HasPrefix(path, "/sessions/"):
		if !permit(w, sc, "read") {
			return
		}
		v, o, red, e := a.svc.Session(sc, strings.TrimPrefix(path, "/sessions/"))
		if e != nil {
			fail(w, 404, "not found")
			return
		}
		write(w, 200, map[string]any{"session": v, "objects": o, "redactions": red})
	case r.Method == "GET" && path == "/usage/summary":
		if !permit(w, sc, "read") {
			return
		}
		write(w, 200, a.svc.Usage(sc))
	case r.Method == "GET" && path == "/redactions":
		if !permit(w, sc, "read") {
			return
		}
		write(w, 200, map[string]any{"items": a.svc.Redactions(sc)})
	case r.Method == "POST" && path == "/retrieve":
		if !permit(w, sc, "retrieve") {
			return
		}
		var req struct {
			ContextRef string `json:"contextRef"`
		}
		if !decode(w, r, &req) {
			return
		}
		content, count, e := a.svc.Retrieve(sc, req.ContextRef)
		if e != nil {
			if errors.Is(e, store.ErrNotFound) {
				fail(w, 404, "not found")
			} else {
				fail(w, 400, e.Error())
			}
			return
		}
		write(w, 200, map[string]any{"contextRef": req.ContextRef, "content": content, "retrievedCount": count})
	case r.Method == "POST" && path == "/search":
		if !permit(w, sc, "read") {
			return
		}
		var req struct {
			SessionID string `json:"sessionId"`
			Query     string `json:"query"`
		}
		if !decode(w, r, &req) {
			return
		}
		if strings.TrimSpace(req.Query) == "" {
			fail(w, 400, "query is required")
			return
		}
		write(w, 200, map[string]any{"items": a.svc.Search(sc, req.SessionID, req.Query)})
	case r.Method == "GET" && path == "/api-keys":
		if !permit(w, sc, "admin") {
			return
		}
		write(w, 200, map[string]any{"items": a.svc.APIKeys(sc)})
	case r.Method == "POST" && path == "/api-keys":
		if !permit(w, sc, "admin") {
			return
		}
		var req struct {
			Name        string   `json:"name"`
			Environment string   `json:"environment"`
			Scopes      []string `json:"scopes"`
		}
		if !decode(w, r, &req) {
			return
		}
		v, e := a.svc.CreateAPIKey(sc, req.Name, req.Environment, req.Scopes)
		if e != nil {
			fail(w, 400, e.Error())
			return
		}
		write(w, 201, v)
	case r.Method == "DELETE" && strings.HasPrefix(path, "/api-keys/"):
		if !permit(w, sc, "admin") {
			return
		}
		if e := a.svc.RevokeAPIKey(sc, strings.TrimPrefix(path, "/api-keys/")); e != nil {
			fail(w, 404, "not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		fail(w, 404, "not found")
	}
}
