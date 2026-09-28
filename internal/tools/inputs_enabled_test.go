package tools

import (
	"net/http"
	"testing"
)

func TestSetInputEnabled(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		hub := newFakeHub(t)
		hub.handle("PUT", "/api/v2/inputs/radio-1/enabled", http.StatusOK, inputJSON)
		s := newTestSession(t, hub.URL, nil)

		res := callTool(t, s, "setInputEnabled", map[string]any{"inputId": "radio-1", "enabled": enabled})
		expectRequest(t, hub, "PUT", "/api/v2/inputs/radio-1/enabled", "", mustJSON(t, map[string]bool{"enabled": enabled}))
		expectStructured(t, res, inputJSON)
	}
	expectAnnotations(t, newTestSession(t, "http://unused", nil), "setInputEnabled", Idempotent)
}

func TestSetInputEnabledRejectsEmptyID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "setInputEnabled", map[string]any{"inputId": "", "enabled": true}))
	hub.assertNoRequests(t)
}
