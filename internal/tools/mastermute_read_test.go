package tools

import (
	"net/http"
	"testing"
)

func TestGetMasterMute(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/master-mute", http.StatusOK, masterMuteJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getMasterMute", map[string]any{})
	expectRequest(t, hub, "GET", "/api/v2/master-mute", "", "")
	expectStructured(t, res, `{"muted":false}`)
	expectAnnotations(t, s, "getMasterMute", ReadOnly)
}
