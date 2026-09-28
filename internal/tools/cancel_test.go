package tools

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCancelReachesHub(t *testing.T) {
	hub := newFakeHub(t)
	hub.handleDelayed("GET", "/api/v2/master-mute", http.StatusOK, masterMuteJSON, 5*time.Second)
	s := newTestSession(t, hub.URL, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = s.CallTool(ctx, &mcp.CallToolParams{Name: "getMasterMute", Arguments: map[string]any{}})
	}()

	waitForRequest(t, hub)
	cancel()

	select {
	case path := <-hub.cancelled:
		if path != "/api/v2/master-mute" {
			t.Errorf("cancelled request path = %q", path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hub request was not cancelled")
	}
	<-done
}

// waitForRequest waits until the hub has received a request.
func waitForRequest(t *testing.T, hub *fakeHub) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for len(hub.requests()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("hub received no request")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
