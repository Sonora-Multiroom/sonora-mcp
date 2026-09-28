package tools

import (
	"net/http"
	"testing"
)

func TestSetMasterMute(t *testing.T) {
	for _, muted := range []bool{true, false} {
		want := mustJSON(t, map[string]bool{"muted": muted})
		hub := newFakeHub(t)
		hub.handle("PUT", "/api/v2/master-mute", http.StatusOK, want)
		s := newTestSession(t, hub.URL, nil)

		res := callTool(t, s, "setMasterMute", map[string]any{"muted": muted})
		expectRequest(t, hub, "PUT", "/api/v2/master-mute", "", want)
		expectStructured(t, res, want)
	}
	expectAnnotations(t, newTestSession(t, "http://unused", nil), "setMasterMute", Idempotent)
}
