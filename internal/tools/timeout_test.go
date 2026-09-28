package tools

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestTimeoutDoesNotBlockOtherCalls(t *testing.T) {
	hub := newFakeHub(t)
	hub.handleDelayed("GET", "/api/v2/outputs", http.StatusOK, "["+outputJSON+"]", 2*time.Second)
	hub.handle("GET", "/api/v2/master-mute", http.StatusOK, masterMuteJSON)
	s := newTestSession(t, hub.URL, &http.Client{Timeout: 200 * time.Millisecond})

	type result struct {
		res *mcp.CallToolResult
		err error
		d   time.Duration
	}
	slow := make(chan result, 1)
	go func() {
		start := time.Now()
		res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: "listOutputs", Arguments: map[string]any{}})
		slow <- result{res, err, time.Since(start)}
	}()

	expectStructured(t, callTool(t, s, "getMasterMute", map[string]any{}), masterMuteJSON)

	r := <-slow
	if r.err != nil {
		t.Fatalf("listOutputs: protocol error: %v", r.err)
	}
	if r.d >= time.Second {
		t.Errorf("listOutputs took %v, want under 1s", r.d)
	}
	expectErrorPrefix(t, r.res, "Timeout:")
}
