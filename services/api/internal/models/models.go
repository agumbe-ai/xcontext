package models

import "time"

type Scope struct {
	TenantID           string   `json:"tenantId"`
	WorkspaceID        string   `json:"workspaceId"`
	UserID             string   `json:"userId,omitempty"`
	TrustedInterceptor bool     `json:"-"`
	Scopes             []string `json:"-"`
}

type Session struct {
	ID string `json:"id"`
	Scope
	Name               string     `json:"name"`
	Source             string     `json:"source"`
	Agent              string     `json:"agent"`
	Repo               string     `json:"repo"`
	Branch             string     `json:"branch"`
	Provider           string     `json:"provider"`
	Status             string     `json:"status"`
	TokenOriginal      int        `json:"tokenOriginal"`
	TokenReturned      int        `json:"tokenReturned"`
	TokenSaved         int        `json:"tokenSaved"`
	ReductionPercent   float64    `json:"reductionPercent"`
	EstimatedCostSaved float64    `json:"estimatedCostSaved"`
	RedactionCount     int        `json:"redactionCount"`
	ObjectCount        int        `json:"objectCount"`
	RetrievedCount     int        `json:"retrievedCount"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
}

type SourceRange struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

type Signal struct {
	Type        string      `json:"type"`
	Text        string      `json:"text"`
	SourceRange SourceRange `json:"sourceRange"`
}

type Object struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionId"`
	ContextRef  string `json:"contextRef"`
	ContentType string `json:"contentType"`
	Source      string `json:"source"`
	Scope
	Summary              string     `json:"summary"`
	RawContent           string     `json:"-"`
	RawURI               string     `json:"-"`
	ContentHash          string     `json:"contentHash"`
	CompressionVersion   string     `json:"compressionVersion"`
	Signals              []Signal   `json:"signals,omitempty"`
	Warnings             []string   `json:"warnings,omitempty"`
	TokenOriginal        int        `json:"tokenOriginal"`
	TokenReturned        int        `json:"tokenReturned"`
	TokenSaved           int        `json:"tokenSaved"`
	PotentialTokensSaved int        `json:"potentialTokensSaved"`
	DeliveredTokensSaved int        `json:"deliveredTokensSaved"`
	ReductionPercent     float64    `json:"reductionPercent"`
	EstimatedCostSaved   float64    `json:"estimatedCostSaved"`
	RedactionCount       int        `json:"redactionCount"`
	RetrievedCount       int        `json:"retrievedCount"`
	CreatedAt            time.Time  `json:"createdAt"`
	ExpiresAt            *time.Time `json:"expiresAt,omitempty"`
	RawAvailable         bool       `json:"rawAvailable"`
}

type Redaction struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenantId"`
	WorkspaceID string    `json:"workspaceId"`
	SessionID   string    `json:"sessionId"`
	ObjectID    string    `json:"objectId"`
	Type        string    `json:"type"`
	Source      string    `json:"source"`
	ContentType string    `json:"contentType"`
	Count       int       `json:"count"`
	CreatedAt   time.Time `json:"createdAt"`
}

type UsageEvent struct {
	ID                   string         `json:"id"`
	TenantID             string         `json:"tenantId"`
	WorkspaceID          string         `json:"workspaceId"`
	SessionID            string         `json:"sessionId"`
	ObjectID             string         `json:"objectId"`
	EventType            string         `json:"eventType"`
	TokenOriginal        int            `json:"tokenOriginal"`
	TokenReturned        int            `json:"tokenReturned"`
	TokenSaved           int            `json:"tokenSaved"`
	DeliveredTokensSaved int            `json:"deliveredTokensSaved"`
	Metadata             map[string]any `json:"metadata,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
}

type Savings struct {
	TokenOriginal        int     `json:"tokenOriginal"`
	TokenReturned        int     `json:"tokenReturned"`
	TokenSaved           int     `json:"tokenSaved"`
	PotentialTokensSaved int     `json:"potentialTokensSaved"`
	DeliveredTokensSaved int     `json:"deliveredTokensSaved"`
	ReductionPercent     float64 `json:"reductionPercent"`
	EstimatedCostSaved   float64 `json:"estimatedCostSaved"`
}

type UsageSummary struct {
	TotalSessions        int     `json:"totalSessions"`
	ActiveSessions       int     `json:"activeSessions"`
	TotalObjects         int     `json:"totalObjects"`
	OriginalTokens       int     `json:"originalTokens"`
	ReturnedTokens       int     `json:"returnedTokens"`
	SavedTokens          int     `json:"savedTokens"`
	PotentialTokensSaved int     `json:"potentialTokensSaved"`
	DeliveredTokensSaved int     `json:"deliveredTokensSaved"`
	ReductionPercent     float64 `json:"reductionPercent"`
	EstimatedCostSaved   float64 `json:"estimatedCostSaved"`
	RedactionCount       int     `json:"redactionCount"`
	RetrievalCount       int     `json:"retrievalCount"`
}

type APIKey struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenantId"`
	WorkspaceID string     `json:"workspaceId"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	Prefix      string     `json:"prefix"`
	Scopes      []string   `json:"scopes"`
	CreatedBy   string     `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	Status      string     `json:"status"`
}
