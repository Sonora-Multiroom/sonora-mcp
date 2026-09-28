package tools

import (
	"net/http"
	"testing"
)

const disabledGroupJSON = `{"groupId":"downstairs","displayName":"Downstairs","outputIds":["kitchen","living-room"],"muted":false,"enabled":false}`

func TestSetGroupVolume(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("PUT", "/api/v2/groups/downstairs/volume", http.StatusOK, groupVolumeJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "setGroupVolume", map[string]any{"groupId": "downstairs", "volume": 35})
	expectRequest(t, hub, "PUT", "/api/v2/groups/downstairs/volume", "", `{"volume":35}`)
	expectStructured(t, res, groupVolumeJSON)
	expectAnnotations(t, s, "setGroupVolume", Idempotent)
}

func TestSetGroupVolumeRejectsOutOfRange(t *testing.T) {
	for _, volume := range []any{150, -1, 35.5} {
		hub := newFakeHub(t)
		s := newTestSession(t, hub.URL, nil)

		expectInvalidInput(t, callTool(t, s, "setGroupVolume", map[string]any{"groupId": "downstairs", "volume": volume}))
		hub.assertNoRequests(t)
	}
}

func TestSetGroupMute(t *testing.T) {
	for _, muted := range []bool{true, false} {
		want := mustJSON(t, map[string]any{"groupId": "downstairs", "muted": muted, "updatedAt": "2026-09-28T10:05:00Z"})
		hub := newFakeHub(t)
		hub.handle("PUT", "/api/v2/groups/downstairs/mute", http.StatusOK, want)
		s := newTestSession(t, hub.URL, nil)

		res := callTool(t, s, "setGroupMute", map[string]any{"groupId": "downstairs", "muted": muted})
		expectRequest(t, hub, "PUT", "/api/v2/groups/downstairs/mute", "", mustJSON(t, map[string]bool{"muted": muted}))
		expectStructured(t, res, want)
	}
	expectAnnotations(t, newTestSession(t, "http://unused", nil), "setGroupMute", Idempotent)
}

func TestSetGroupEnabled(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("PUT", "/api/v2/groups/downstairs/enabled", http.StatusOK, disabledGroupJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "setGroupEnabled", map[string]any{"groupId": "downstairs", "enabled": false})
	expectRequest(t, hub, "PUT", "/api/v2/groups/downstairs/enabled", "", `{"enabled":false}`)
	expectStructured(t, res, disabledGroupJSON)
	expectAnnotations(t, s, "setGroupEnabled", Idempotent)
}
