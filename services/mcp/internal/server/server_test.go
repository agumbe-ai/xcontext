package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/agumbe-ai/xcontext/pkg/client"
)

func TestExecuteUsesArgvAndAttestsDelivery(t *testing.T) {
	var received client.IngestRequest
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/objects" {
			t.Fatalf("path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"contextRef":"ctx://t/s/o","summary":"command summary","savings":{"potentialTokensSaved":10,"deliveredTokensSaved":10,"estimatedCostSaved":0.01}}`))}, nil
	})
	c := client.New("http://xcontext.test", "xctx_test_key")
	c.HTTP = &http.Client{Transport: transport}
	s := New(c, strings.NewReader(""), &strings.Builder{})
	args, _ := json.Marshal(map[string]any{"argv": []string{"printf", "literal; echo not-a-shell"}, "contentType": "log"})
	out, err := s.call(context.Background(), "xcontext_execute", args)
	if err != nil {
		t.Fatal(err)
	}
	if !received.DeliveryVerified {
		t.Fatal("execution was not attested")
	}
	if received.Source != "command:printf" {
		t.Fatalf("unsafe command provenance: %q", received.Source)
	}
	if received.Text != "literal; echo not-a-shell" {
		t.Fatalf("shell syntax was interpreted: %q", received.Text)
	}
	if !strings.Contains(out, "Delivered tokens saved: 10") {
		t.Fatal(out)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestInitializeAndToolList(t *testing.T) {
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n"
	var out strings.Builder
	s := New(client.New("http://invalid", "key"), strings.NewReader(input), &out)
	if err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "xcontext_execute") || !strings.Contains(out.String(), "protocolVersion") {
		t.Fatal(out.String())
	}
}
