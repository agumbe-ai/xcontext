package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/compression"
	"github.com/agumbe-ai/xcontext/services/api/internal/contextref"
	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/redaction"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

type Config struct {
	CostPer1K                       float64
	StoreRawMode                    string
	MaxInputBytes, MaxSummaryTokens int
	ConsoleURL                      string
}
type Service struct {
	store    store.Store
	redactor redaction.Redactor
	cfg      Config
	now      func() time.Time
}

func New(st store.Store, cfg Config) *Service {
	if cfg.CostPer1K == 0 {
		cfg.CostPer1K = .01
	}
	if cfg.StoreRawMode == "" {
		cfg.StoreRawMode = "redacted"
	}
	if cfg.MaxInputBytes == 0 {
		cfg.MaxInputBytes = 10 << 20
	}
	if cfg.MaxSummaryTokens == 0 {
		cfg.MaxSummaryTokens = 1200
	}
	return &Service{store: st, redactor: redaction.New(), cfg: cfg, now: time.Now}
}

type IngestRequest struct {
	SessionID        string `json:"sessionId"`
	SessionName      string `json:"sessionName"`
	Source           string `json:"source"`
	ContentType      string `json:"contentType"`
	Agent            string `json:"agent"`
	Repo             string `json:"repo"`
	Branch           string `json:"branch"`
	Provider         string `json:"provider"`
	Text             string `json:"text"`
	DeliveryVerified bool   `json:"deliveryVerified"`
}
type IngestResponse struct {
	SessionID  string         `json:"sessionId"`
	ObjectID   string         `json:"objectId"`
	ContextRef string         `json:"contextRef"`
	Summary    string         `json:"summary"`
	Savings    models.Savings `json:"savings"`
	Redactions struct {
		Count int      `json:"count"`
		Types []string `json:"types"`
	} `json:"redactions"`
	Signals            []models.Signal `json:"signals"`
	CompressionVersion string          `json:"compressionVersion"`
	ContentHash        string          `json:"contentHash"`
	ConsoleURL         string          `json:"consoleUrl"`
}

type CreatedAPIKey struct {
	models.APIKey
	Key string `json:"key"`
}

func (s *Service) CreateAPIKey(scope models.Scope, name, environment string, scopes []string) (CreatedAPIKey, error) {
	if strings.TrimSpace(name) == "" {
		return CreatedAPIKey{}, fmt.Errorf("name is required")
	}
	if environment != "test" && environment != "live" {
		return CreatedAPIKey{}, fmt.Errorf("environment must be live or test")
	}
	allowed := map[string]bool{"ingest": true, "read": true, "retrieve": true, "admin": true, "intercept": true}
	for _, v := range scopes {
		if !allowed[v] {
			return CreatedAPIKey{}, fmt.Errorf("invalid scope %q", v)
		}
	}
	if len(scopes) == 0 {
		scopes = []string{"ingest", "read"}
	}
	secret := make([]byte, 24)
	if _, e := rand.Read(secret); e != nil {
		return CreatedAPIKey{}, e
	}
	raw := "xctx_" + environment + "_" + hex.EncodeToString(secret)
	sum := sha256.Sum256([]byte(raw))
	now := s.now().UTC()
	v := models.APIKey{ID: id("ctxk_"), TenantID: scope.TenantID, WorkspaceID: scope.WorkspaceID, Name: name, KeyHash: hex.EncodeToString(sum[:]), Prefix: raw[:min(len(raw), 18)] + "...", Scopes: scopes, CreatedBy: scope.UserID, CreatedAt: now, Status: "active"}
	if e := s.store.CreateAPIKey(v); e != nil {
		return CreatedAPIKey{}, e
	}
	return CreatedAPIKey{APIKey: v, Key: raw}, nil
}
func (s *Service) APIKeys(scope models.Scope) []models.APIKey { return s.store.ListAPIKeys(scope) }
func (s *Service) RevokeAPIKey(scope models.Scope, id string) error {
	return s.store.RevokeAPIKey(scope, id)
}

func id(prefix string) string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
func tokens(s string) int { return int(math.Ceil(float64(len([]rune(s))) / 4)) }
func savings(original, returned int, rate float64, verified bool) models.Savings {
	saved := max(0, original-returned)
	delivered := 0
	if verified {
		delivered = saved
	}
	pct := 0.0
	if original > 0 {
		pct = float64(saved) * 100 / float64(original)
	}
	return models.Savings{TokenOriginal: original, TokenReturned: returned, TokenSaved: saved, PotentialTokensSaved: saved, DeliveredTokensSaved: delivered, ReductionPercent: pct, EstimatedCostSaved: float64(delivered) / 1000 * rate}
}

func (s *Service) Ingest(scope models.Scope, r IngestRequest) (IngestResponse, error) {
	if scope.TenantID == "" || scope.WorkspaceID == "" {
		return IngestResponse{}, fmt.Errorf("tenant and workspace are required")
	}
	if r.Text == "" {
		return IngestResponse{}, fmt.Errorf("text is required")
	}
	if len(r.Text) > s.cfg.MaxInputBytes {
		return IngestResponse{}, fmt.Errorf("input exceeds %d bytes", s.cfg.MaxInputBytes)
	}
	now := s.now().UTC()
	sessionID := r.SessionID
	var session models.Session
	var err error
	isNewSession := sessionID == ""
	if sessionID != "" {
		session, err = s.store.GetSession(scope, sessionID)
		if err != nil {
			return IngestResponse{}, err
		}
	} else {
		sessionID = id("ctxs_")
		name := r.SessionName
		if name == "" {
			name = r.Source
		}
		if name == "" {
			name = "xcontext session"
		}
		session = models.Session{ID: sessionID, Scope: scope, Name: name, Source: r.Source, Agent: r.Agent, Repo: r.Repo, Branch: r.Branch, Provider: r.Provider, Status: "active", CreatedAt: now, UpdatedAt: now}
	}
	redacted, findings := s.redactor.Redact(r.Text)
	result := compression.For(r.ContentType).Compress(redacted, s.cfg.MaxSummaryTokens)
	deliveryVerified := scope.TrustedInterceptor && r.DeliveryVerified
	sum := savings(tokens(r.Text), tokens(result.Summary), s.cfg.CostPer1K, deliveryVerified)
	objectID := id("ctxo_")
	ref := contextref.Encode(scope.TenantID, sessionID, objectID)
	hash := sha256.Sum256([]byte(r.Text))
	raw := redacted
	if s.cfg.StoreRawMode == "original" {
		raw = r.Text
	} else if s.cfg.StoreRawMode == "none" {
		raw = ""
	}
	obj := models.Object{ID: objectID, SessionID: sessionID, ContextRef: ref, ContentType: r.ContentType, Source: r.Source, Scope: scope, Summary: result.Summary, RawContent: raw, ContentHash: hex.EncodeToString(hash[:]), CompressionVersion: result.Version, Signals: result.Signals, Warnings: result.Warnings, TokenOriginal: sum.TokenOriginal, TokenReturned: sum.TokenReturned, TokenSaved: sum.TokenSaved, PotentialTokensSaved: sum.PotentialTokensSaved, DeliveredTokensSaved: sum.DeliveredTokensSaved, ReductionPercent: sum.ReductionPercent, EstimatedCostSaved: sum.EstimatedCostSaved, RedactionCount: 0, CreatedAt: now, RawAvailable: raw != ""}
	var reds []models.Redaction
	var types []string
	for _, f := range findings {
		obj.RedactionCount += f.Count
		types = append(types, f.Type)
		reds = append(reds, models.Redaction{ID: id("ctxr_"), TenantID: scope.TenantID, WorkspaceID: scope.WorkspaceID, SessionID: sessionID, ObjectID: objectID, Type: f.Type, Source: r.Source, ContentType: r.ContentType, Count: f.Count, CreatedAt: now})
	}
	session.TokenOriginal += sum.TokenOriginal
	session.TokenReturned += sum.TokenReturned
	session.TokenSaved += sum.TokenSaved
	session.EstimatedCostSaved += sum.EstimatedCostSaved
	session.RedactionCount += obj.RedactionCount
	session.ObjectCount++
	session.UpdatedAt = now
	if session.TokenOriginal > 0 {
		session.ReductionPercent = float64(session.TokenSaved) * 100 / float64(session.TokenOriginal)
	}
	event := models.UsageEvent{ID: id("ctxu_"), TenantID: scope.TenantID, WorkspaceID: scope.WorkspaceID, SessionID: sessionID, ObjectID: objectID, EventType: "object_ingested", TokenOriginal: sum.TokenOriginal, TokenReturned: sum.TokenReturned, TokenSaved: sum.TokenSaved, DeliveredTokensSaved: sum.DeliveredTokensSaved, Metadata: map[string]any{"compressionVersion": result.Version, "deliveryVerified": deliveryVerified}, CreatedAt: now}
	if err = s.store.CommitIngest(session, isNewSession, obj, reds, event); err != nil {
		return IngestResponse{}, err
	}
	resp := IngestResponse{SessionID: sessionID, ObjectID: objectID, ContextRef: ref, Summary: result.Summary, Savings: sum, Signals: result.Signals, CompressionVersion: result.Version, ContentHash: obj.ContentHash, ConsoleURL: strings.TrimSuffix(s.cfg.ConsoleURL, "/") + "/sessions/" + sessionID}
	resp.Redactions.Count = obj.RedactionCount
	resp.Redactions.Types = types
	return resp, nil
}

func (s *Service) Sessions(sc models.Scope) []models.Session { return s.store.ListSessions(sc) }
func (s *Service) Session(sc models.Scope, id string) (models.Session, []models.Object, []models.Redaction, error) {
	v, e := s.store.GetSession(sc, id)
	if e != nil {
		return v, nil, nil, e
	}
	return v, s.store.ListObjects(sc, id), s.store.ListRedactions(sc, id), nil
}
func (s *Service) Objects(sc models.Scope) []models.Object {
	out := s.store.ListObjects(sc, "")
	for i := range out {
		out[i].RawContent = ""
	}
	return out
}
func (s *Service) Object(sc models.Scope, id string) (models.Object, error) {
	v, e := s.store.GetObject(sc, id)
	v.RawContent = ""
	return v, e
}
func (s *Service) Redactions(sc models.Scope) []models.Redaction {
	return s.store.ListRedactions(sc, "")
}
func (s *Service) Search(sc models.Scope, session, q string) []models.Object {
	return s.store.Search(sc, session, q)
}
func (s *Service) Retrieve(sc models.Scope, ref string) (string, int, error) {
	tenant, _, _, e := contextref.Parse(ref)
	if e != nil {
		return "", 0, e
	}
	if tenant != sc.TenantID {
		return "", 0, store.ErrNotFound
	}
	v, e := s.store.GetObjectByRef(sc, ref)
	if e != nil {
		return "", 0, e
	}
	if !v.RawAvailable {
		return "", 0, store.ErrNotFound
	}
	v.RetrievedCount++
	_ = s.store.UpdateObject(v)
	session, e := s.store.GetSession(sc, v.SessionID)
	if e == nil {
		session.RetrievedCount++
		session.UpdatedAt = s.now().UTC()
		_ = s.store.UpdateSession(session)
	}
	_ = s.store.AddEvent(models.UsageEvent{ID: id("ctxu_"), TenantID: sc.TenantID, WorkspaceID: sc.WorkspaceID, SessionID: v.SessionID, ObjectID: v.ID, EventType: "object_retrieved", CreatedAt: s.now().UTC()})
	return v.RawContent, v.RetrievedCount, nil
}
func (s *Service) Usage(sc models.Scope) models.UsageSummary {
	sessions := s.store.ListSessions(sc)
	objects := s.store.ListObjects(sc, "")
	var u models.UsageSummary
	u.TotalSessions = len(sessions)
	u.TotalObjects = len(objects)
	for _, v := range sessions {
		if v.Status == "active" {
			u.ActiveSessions++
		}
	}
	for _, v := range objects {
		u.OriginalTokens += v.TokenOriginal
		u.ReturnedTokens += v.TokenReturned
		u.SavedTokens += v.TokenSaved
		u.PotentialTokensSaved += v.PotentialTokensSaved
		u.DeliveredTokensSaved += v.DeliveredTokensSaved
		u.EstimatedCostSaved += v.EstimatedCostSaved
		u.RedactionCount += v.RedactionCount
		u.RetrievalCount += v.RetrievedCount
	}
	if u.OriginalTokens > 0 {
		u.ReductionPercent = float64(u.SavedTokens) * 100 / float64(u.OriginalTokens)
	}
	return u
}
