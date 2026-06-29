package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/agumbe-ai/xcontext/pkg/client"
)

type Server struct {
	client    *client.Client
	in        io.Reader
	out       io.Writer
	maxOutput int
}

func New(c *client.Client, in io.Reader, out io.Writer) *Server {
	return &Server{client: c, in: in, out: out, maxOutput: 10 << 20}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Run(ctx context.Context) error {
	scan := bufio.NewScanner(s.in)
	scan.Buffer(make([]byte, 64*1024), 12<<20)
	enc := json.NewEncoder(s.out)
	for scan.Scan() {
		var r request
		if json.Unmarshal(scan.Bytes(), &r) != nil {
			continue
		}
		if r.ID == nil {
			continue
		}
		res := s.handle(ctx, r)
		if e := enc.Encode(res); e != nil {
			return e
		}
	}
	return scan.Err()
}
func ok(id any, v any) response { return response{JSONRPC: "2.0", ID: id, Result: v} }
func fail(id any, e error) response {
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: -32000, Message: e.Error()}}
}
func (s *Server) handle(ctx context.Context, r request) response {
	switch r.Method {
	case "initialize":
		return ok(r.ID, map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "xcontext", "version": "0.1.0"}})
	case "ping":
		return ok(r.ID, map[string]any{})
	case "tools/list":
		return ok(r.ID, map[string]any{"tools": tools})
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if e := json.Unmarshal(r.Params, &p); e != nil {
			return fail(r.ID, e)
		}
		v, e := s.call(ctx, p.Name, p.Arguments)
		if e != nil {
			return fail(r.ID, e)
		}
		return ok(r.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": v}}})
	default:
		return response{JSONRPC: "2.0", ID: r.ID, Error: &rpcError{Code: -32601, Message: "method not found"}}
	}
}

var tools = []map[string]any{
	{"name": "xcontext_execute", "description": "Run a local executable without a shell, keep full output out of model context, and return a compressed receipt.", "inputSchema": map[string]any{"type": "object", "required": []string{"argv"}, "properties": map[string]any{"argv": map[string]any{"type": "array", "items": map[string]string{"type": "string"}, "minItems": 1}, "contentType": map[string]string{"type": "string"}, "source": map[string]string{"type": "string"}, "sessionId": map[string]string{"type": "string"}, "timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": 600}}}},
	{"name": "xcontext_ingest", "description": "Redact, compress, preserve, and index a context artifact.", "inputSchema": map[string]any{"type": "object", "required": []string{"text", "contentType"}, "properties": map[string]any{"text": map[string]string{"type": "string"}, "contentType": map[string]string{"type": "string"}, "source": map[string]string{"type": "string"}, "sessionId": map[string]string{"type": "string"}}}},
	{"name": "xcontext_retrieve", "description": "Retrieve preserved content by context reference.", "inputSchema": map[string]any{"type": "object", "required": []string{"contextRef"}, "properties": map[string]any{"contextRef": map[string]string{"type": "string"}}}},
	{"name": "xcontext_search", "description": "Search preserved context within the authenticated workspace.", "inputSchema": map[string]any{"type": "object", "required": []string{"query"}, "properties": map[string]any{"query": map[string]string{"type": "string"}, "sessionId": map[string]string{"type": "string"}}}},
	{"name": "xcontext_stats", "description": "Return context savings and protection statistics.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
}

func (s *Server) call(ctx context.Context, name string, args json.RawMessage) (string, error) {
	switch name {
	case "xcontext_execute":
		var a struct {
			Argv                           []string `json:"argv"`
			ContentType, Source, SessionID string
			TimeoutSeconds                 int `json:"timeoutSeconds"`
		}
		if e := json.Unmarshal(args, &a); e != nil {
			return "", e
		}
		if len(a.Argv) == 0 || strings.TrimSpace(a.Argv[0]) == "" {
			return "", fmt.Errorf("argv is required")
		}
		if a.TimeoutSeconds == 0 {
			a.TimeoutSeconds = 120
		}
		runCtx, cancel := context.WithTimeout(ctx, time.Duration(a.TimeoutSeconds)*time.Second)
		defer cancel()
		cmd := exec.CommandContext(runCtx, a.Argv[0], a.Argv[1:]...)
		out, e := cmd.CombinedOutput()
		if len(out) > s.maxOutput {
			out = out[:s.maxOutput]
		}
		exit := 0
		if e != nil {
			if ee, ok := e.(*exec.ExitError); ok {
				exit = ee.ExitCode()
			} else {
				return "", e
			}
		}
		source := a.Source
		if source == "" {
			source = strings.Join(a.Argv, " ")
		}
		kind := a.ContentType
		if kind == "" {
			kind = "log"
		}
		receipt, e := s.client.Ingest(ctx, client.IngestRequest{SessionID: a.SessionID, Source: source, ContentType: kind, Text: string(out), DeliveryVerified: true})
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s\n\nContext ref: %s\nExit code: %d\nPotential tokens saved: %d\nDelivered tokens saved: %d\nEstimated cost avoided: %.4f", receipt.Summary, receipt.ContextRef, exit, receipt.Savings.PotentialTokensSaved, receipt.Savings.DeliveredTokensSaved, receipt.Savings.EstimatedCostSaved), nil
	case "xcontext_ingest":
		var a struct{ Text, ContentType, Source, SessionID string }
		if e := json.Unmarshal(args, &a); e != nil {
			return "", e
		}
		v, e := s.client.Ingest(ctx, client.IngestRequest{SessionID: a.SessionID, Source: a.Source, ContentType: a.ContentType, Text: a.Text})
		if e != nil {
			return "", e
		}
		return fmt.Sprintf("%s\n\nContext ref: %s\nPotential tokens saved: %d", v.Summary, v.ContextRef, v.Savings.PotentialTokensSaved), nil
	case "xcontext_retrieve":
		var a struct {
			ContextRef string `json:"contextRef"`
		}
		if e := json.Unmarshal(args, &a); e != nil {
			return "", e
		}
		return s.client.Retrieve(ctx, a.ContextRef)
	case "xcontext_search":
		var a struct {
			Query     string `json:"query"`
			SessionID string `json:"sessionId"`
		}
		if e := json.Unmarshal(args, &a); e != nil {
			return "", e
		}
		v, e := s.client.Search(ctx, a.SessionID, a.Query)
		if e != nil {
			return "", e
		}
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b), nil
	case "xcontext_stats":
		v, e := s.client.Usage(ctx)
		if e != nil {
			return "", e
		}
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b), nil
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}
