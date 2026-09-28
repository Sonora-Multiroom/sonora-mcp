package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tiger-seo/sonora-mcp/internal/tools"
)

func newTestServer(t *testing.T, hubURL string) *httptest.Server {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: "test"}, nil)
	client := &http.Client{}
	tools.Register(s, client, hubURL)
	ts := httptest.NewServer(Handler(s, client, hubURL))
	t.Cleanup(ts.Close)
	return ts
}

func TestMCPEndToEnd(t *testing.T) {
	ts := newTestServer(t, "http://127.0.0.1:1")

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()

	if got := cs.InitializeResult().ServerInfo.Name; got != "sonora-mcp" {
		t.Errorf("server name = %q", got)
	}
	if _, err := cs.ListTools(context.Background(), nil); err != nil {
		t.Fatalf("ListTools: %v", err)
	}
}

func TestMCPIsStateless(t *testing.T) {
	ts := newTestServer(t, "http://127.0.0.1:1")

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"raw","version":"1"}}}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d", resp.StatusCode)
	}
	if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
		t.Errorf("Mcp-Session-Id = %q, want none (stateless)", id)
	}
}

func TestOtherRoutes(t *testing.T) {
	ts := newTestServer(t, "http://127.0.0.1:1")
	for _, tt := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/mcp", http.StatusMethodNotAllowed},
		{http.MethodGet, "/nope", http.StatusNotFound},
		{http.MethodGet, "/", http.StatusNotFound},
	} {
		req, _ := http.NewRequestWithContext(context.Background(), tt.method, ts.URL+tt.path, nil)
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tt.want {
			t.Errorf("%s %s = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.want)
		}
	}
}
