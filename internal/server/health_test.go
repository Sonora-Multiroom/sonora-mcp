package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tiger-seo/sonora-mcp/internal/version"
)

// getHealth serves GET /health from a server whose tools use hubURL and
// returns the response and its decoded body.
func getHealth(t *testing.T, hubURL string) (*http.Response, map[string]any) {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: "test"}, nil)
	ts := httptest.NewServer(Handler(s, &http.Client{}, hubURL))
	t.Cleanup(ts.Close)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL+"/health", nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("health body is not JSON: %v", err)
	}
	return resp, body
}

func expectHealth(t *testing.T, resp *http.Response, body map[string]any, hub string) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	want := map[string]any{"status": "ok", "server": "sonora-mcp", "version": version.Version, "hub": hub}
	if len(body) != len(want) {
		t.Errorf("body = %v, want %v", body, want)
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body[%q] = %v, want %v", k, body[k], v)
		}
	}
}

func TestHealthHubReachable(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/master-mute" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"muted":false}`))
	}))
	t.Cleanup(hub.Close)

	resp, body := getHealth(t, hub.URL)
	expectHealth(t, resp, body, "reachable")
}

func TestHealthHubDown(t *testing.T) {
	hub := httptest.NewServer(http.NotFoundHandler())
	url := hub.URL
	hub.Close()

	resp, body := getHealth(t, url)
	expectHealth(t, resp, body, "unreachable")
}

func TestHealthHubSlow(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(hub.Close)

	start := time.Now()
	resp, body := getHealth(t, hub.URL)
	// The 2 s bound is the handler's; 0.5 s absorbs scheduling jitter.
	if d := time.Since(start); d > 2500*time.Millisecond {
		t.Errorf("health took %v, want <= 2.5s", d)
	}
	expectHealth(t, resp, body, "unreachable")
}
