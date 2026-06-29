package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL, APIKey string
	HTTP            *http.Client
}

func New(base, key string) *Client {
	return &Client{BaseURL: strings.TrimSuffix(base, "/"), APIKey: key, HTTP: &http.Client{Timeout: 60 * time.Second}}
}
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return e
		}
		reader = bytes.NewReader(b)
	}
	r, e := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if e != nil {
		return e
	}
	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	resp, e := c.HTTP.Do(r)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("xcontext API %s: %s", resp.Status, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

type IngestRequest struct {
	SessionID        string `json:"sessionId,omitempty"`
	SessionName      string `json:"sessionName,omitempty"`
	Source           string `json:"source"`
	ContentType      string `json:"contentType"`
	Agent            string `json:"agent,omitempty"`
	Repo             string `json:"repo,omitempty"`
	Branch           string `json:"branch,omitempty"`
	Provider         string `json:"provider,omitempty"`
	Text             string `json:"text"`
	DeliveryVerified bool   `json:"deliveryVerified"`
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
type IngestResponse struct {
	SessionID  string  `json:"sessionId"`
	ObjectID   string  `json:"objectId"`
	ContextRef string  `json:"contextRef"`
	Summary    string  `json:"summary"`
	Savings    Savings `json:"savings"`
	ConsoleURL string  `json:"consoleUrl"`
}

func (c *Client) Ingest(ctx context.Context, r IngestRequest) (IngestResponse, error) {
	var out IngestResponse
	e := c.do(ctx, "POST", "/objects", r, &out)
	return out, e
}
func (c *Client) Retrieve(ctx context.Context, ref string) (string, error) {
	var out struct {
		Content string `json:"content"`
	}
	e := c.do(ctx, "POST", "/retrieve", map[string]string{"contextRef": ref}, &out)
	return out.Content, e
}
func (c *Client) Search(ctx context.Context, session, q string) ([]map[string]any, error) {
	var out struct {
		Items []map[string]any `json:"items"`
	}
	e := c.do(ctx, "POST", "/search", map[string]string{"sessionId": session, "query": q}, &out)
	return out.Items, e
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

func (c *Client) Usage(ctx context.Context) (UsageSummary, error) {
	var out UsageSummary
	e := c.do(ctx, "GET", "/usage/summary", nil, &out)
	return out, e
}
