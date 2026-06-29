package store

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	CommitIngest(models.Session, bool, models.Object, []models.Redaction, models.UsageEvent) error
	CreateSession(models.Session) error
	UpdateSession(models.Session) error
	GetSession(models.Scope, string) (models.Session, error)
	ListSessions(models.Scope) []models.Session
	CreateObject(models.Object) error
	UpdateObject(models.Object) error
	GetObject(models.Scope, string) (models.Object, error)
	GetObjectByRef(models.Scope, string) (models.Object, error)
	ListObjects(models.Scope, string) []models.Object
	AddRedactions([]models.Redaction) error
	ListRedactions(models.Scope, string) []models.Redaction
	AddEvent(models.UsageEvent) error
	ListEvents(models.Scope) []models.UsageEvent
	Search(models.Scope, string, string) []models.Object
	CreateAPIKey(models.APIKey) error
	GetAPIKeyByHash(string) (models.APIKey, error)
	ListAPIKeys(models.Scope) []models.APIKey
	RevokeAPIKey(models.Scope, string) error
	TouchAPIKey(string, time.Time) error
}

type Memory struct {
	mu         sync.RWMutex
	sessions   map[string]models.Session
	objects    map[string]models.Object
	redactions []models.Redaction
	events     []models.UsageEvent
	apiKeys    map[string]models.APIKey
}

func NewMemory() *Memory {
	return &Memory{sessions: map[string]models.Session{}, objects: map[string]models.Object{}, apiKeys: map[string]models.APIKey{}}
}
func (m *Memory) CommitIngest(session models.Session, isNew bool, obj models.Object, reds []models.Redaction, event models.UsageEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !isNew {
		current, ok := m.sessions[session.ID]
		if !ok || current.TenantID != session.TenantID || current.WorkspaceID != session.WorkspaceID {
			return ErrNotFound
		}
	}
	m.sessions[session.ID] = session
	m.objects[obj.ID] = obj
	m.redactions = append(m.redactions, reds...)
	m.events = append(m.events, event)
	return nil
}
func scoped(scope models.Scope, tenant, workspace string) bool {
	return scope.TenantID == tenant && (scope.WorkspaceID == "" || scope.WorkspaceID == workspace)
}
func (m *Memory) CreateSession(v models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[v.ID] = v
	return nil
}
func (m *Memory) UpdateSession(v models.Session) error { return m.CreateSession(v) }
func (m *Memory) GetSession(s models.Scope, id string) (models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.sessions[id]
	if !ok || !scoped(s, v.TenantID, v.WorkspaceID) {
		return v, ErrNotFound
	}
	return v, nil
}
func (m *Memory) ListSessions(s models.Scope) []models.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.Session
	for _, v := range m.sessions {
		if scoped(s, v.TenantID, v.WorkspaceID) {
			out = append(out, v)
		}
	}
	return out
}
func (m *Memory) CreateObject(v models.Object) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[v.ID] = v
	return nil
}
func (m *Memory) UpdateObject(v models.Object) error { return m.CreateObject(v) }
func (m *Memory) GetObject(s models.Scope, id string) (models.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.objects[id]
	if !ok || !scoped(s, v.TenantID, v.WorkspaceID) {
		return v, ErrNotFound
	}
	return v, nil
}
func (m *Memory) GetObjectByRef(s models.Scope, ref string) (models.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.objects {
		if v.ContextRef == ref && scoped(s, v.TenantID, v.WorkspaceID) {
			return v, nil
		}
	}
	return models.Object{}, ErrNotFound
}
func (m *Memory) ListObjects(s models.Scope, session string) []models.Object {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.Object
	for _, v := range m.objects {
		if scoped(s, v.TenantID, v.WorkspaceID) && (session == "" || v.SessionID == session) {
			out = append(out, v)
		}
	}
	return out
}
func (m *Memory) AddRedactions(v []models.Redaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.redactions = append(m.redactions, v...)
	return nil
}
func (m *Memory) ListRedactions(s models.Scope, session string) []models.Redaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.Redaction
	for _, v := range m.redactions {
		if scoped(s, v.TenantID, v.WorkspaceID) && (session == "" || v.SessionID == session) {
			out = append(out, v)
		}
	}
	return out
}
func (m *Memory) AddEvent(v models.UsageEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, v)
	return nil
}
func (m *Memory) ListEvents(s models.Scope) []models.UsageEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.UsageEvent
	for _, v := range m.events {
		if scoped(s, v.TenantID, v.WorkspaceID) {
			out = append(out, v)
		}
	}
	return out
}
func (m *Memory) Search(s models.Scope, session, q string) []models.Object {
	q = strings.ToLower(q)
	var out []models.Object
	for _, v := range m.ListObjects(s, session) {
		if strings.Contains(strings.ToLower(v.Summary+"\n"+v.RawContent), q) {
			v.RawContent = ""
			out = append(out, v)
		}
	}
	return out
}

func (m *Memory) CreateAPIKey(v models.APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiKeys[v.ID] = v
	return nil
}
func (m *Memory) GetAPIKeyByHash(hash string) (models.APIKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.apiKeys {
		if v.KeyHash == hash && v.Status == "active" {
			return v, nil
		}
	}
	return models.APIKey{}, ErrNotFound
}
func (m *Memory) ListAPIKeys(s models.Scope) []models.APIKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []models.APIKey
	for _, v := range m.apiKeys {
		if scoped(s, v.TenantID, v.WorkspaceID) {
			v.KeyHash = ""
			out = append(out, v)
		}
	}
	return out
}
func (m *Memory) RevokeAPIKey(s models.Scope, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.apiKeys[id]
	if !ok || !scoped(s, v.TenantID, v.WorkspaceID) {
		return ErrNotFound
	}
	v.Status = "revoked"
	m.apiKeys[id] = v
	return nil
}
func (m *Memory) TouchAPIKey(id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.apiKeys[id]
	if !ok {
		return ErrNotFound
	}
	v.LastUsedAt = &at
	m.apiKeys[id] = v
	return nil
}
