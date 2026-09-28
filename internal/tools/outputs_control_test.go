package tools

import (
	"net/http"
	"testing"
)

func TestSetOutputVolume(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("PUT", "/api/v2/outputs/kitchen/volume", http.StatusOK, outputVolumeJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "setOutputVolume", map[string]any{"outputId": "kitchen", "volume": 35})
	expectRequest(t, hub, "PUT", "/api/v2/outputs/kitchen/volume", "", `{"volume":35}`)
	expectStructured(t, res, outputVolumeJSON)
	expectAnnotations(t, s, "setOutputVolume", Idempotent)
}

func TestSetOutputVolumeRejectsOutOfRange(t *testing.T) {
	for _, volume := range []any{150, -1, 35.5} {
		hub := newFakeHub(t)
		s := newTestSession(t, hub.URL, nil)

		expectInvalidInput(t, callTool(t, s, "setOutputVolume", map[string]any{"outputId": "kitchen", "volume": volume}))
		hub.assertNoRequests(t)
	}
}

func TestSetOutputMute(t *testing.T) {
	for _, muted := range []bool{true, false} {
		want := mustJSON(t, map[string]any{"outputId": "kitchen", "muted": muted, "updatedAt": "2026-09-28T10:05:00Z"})
		hub := newFakeHub(t)
		hub.handle("PUT", "/api/v2/outputs/kitchen/mute", http.StatusOK, want)
		s := newTestSession(t, hub.URL, nil)

		res := callTool(t, s, "setOutputMute", map[string]any{"outputId": "kitchen", "muted": muted})
		expectRequest(t, hub, "PUT", "/api/v2/outputs/kitchen/mute", "", mustJSON(t, map[string]bool{"muted": muted}))
		expectStructured(t, res, want)
	}
	expectAnnotations(t, newTestSession(t, "http://unused", nil), "setOutputMute", Idempotent)
}

func TestSetOutputEnabled(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("PUT", "/api/v2/outputs/garage/enabled", http.StatusOK, disabledOutputJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "setOutputEnabled", map[string]any{"outputId": "garage", "enabled": false})
	expectRequest(t, hub, "PUT", "/api/v2/outputs/garage/enabled", "", `{"enabled":false}`)
	expectStructured(t, res, disabledOutputJSON)
	expectAnnotations(t, s, "setOutputEnabled", Idempotent)
}
