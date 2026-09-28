package tools

import (
	"net/http"
	"testing"
)

func TestCreateRoute(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/routes", http.StatusCreated, routeJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "createRoute", map[string]any{"inputId": "radio-1", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT"})
	expectRequest(t, hub, "POST", "/api/v2/routes", "", `{"inputId":"radio-1","targetId":"kitchen","targetType":"SINGLE_OUTPUT"}`)
	expectStructured(t, res, routeJSON)
	expectAnnotations(t, s, "createRoute", StateChanging)
}

func TestCreateRouteRejectsUnknownTargetType(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "createRoute", map[string]any{"inputId": "radio-1", "targetId": "kitchen", "targetType": "SPEAKER"}))
	hub.assertNoRequests(t)
}

func TestTransferRoute(t *testing.T) {
	const transferred = `{"routeId":"r-1","inputId":"radio-1","targetId":"downstairs","targetType":"OUTPUT_GROUP","status":"ACTIVE","createdAt":"2026-09-28T10:00:00Z","startedAt":"2026-09-28T10:00:01Z","transferable":true,"pauseable":true,"paused":false}`
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/routes/r-1/transfer", http.StatusOK, transferred)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "transferRoute", map[string]any{"routeId": "r-1", "targetId": "downstairs", "targetType": "OUTPUT_GROUP"})
	expectRequest(t, hub, "POST", "/api/v2/routes/r-1/transfer", "", `{"targetId":"downstairs","targetType":"OUTPUT_GROUP"}`)
	expectStructured(t, res, transferred)
	expectAnnotations(t, s, "transferRoute", StateChanging)
}

func TestTransferRouteRejectsUnknownTargetType(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "transferRoute", map[string]any{"routeId": "r-1", "targetId": "kitchen", "targetType": "SPEAKER"}))
	hub.assertNoRequests(t)
}

func TestSetRoutePause(t *testing.T) {
	for _, paused := range []bool{true, false} {
		want := mustJSON(t, map[string]any{
			"routeId": "r-1", "inputId": "radio-1", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT",
			"status": "ACTIVE", "createdAt": "2026-09-28T10:00:00Z", "startedAt": "2026-09-28T10:00:01Z",
			"transferable": true, "pauseable": true, "paused": paused,
		})
		hub := newFakeHub(t)
		hub.handle("PUT", "/api/v2/routes/r-1/pause", http.StatusOK, want)
		s := newTestSession(t, hub.URL, nil)

		res := callTool(t, s, "setRoutePause", map[string]any{"routeId": "r-1", "paused": paused})
		expectRequest(t, hub, "PUT", "/api/v2/routes/r-1/pause", "", mustJSON(t, map[string]bool{"paused": paused}))
		expectStructured(t, res, want)
	}
	expectAnnotations(t, newTestSession(t, "http://unused", nil), "setRoutePause", Idempotent)
}

func TestDeleteRoute(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("DELETE", "/api/v2/routes/r-1", http.StatusNoContent, "")
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "deleteRoute", map[string]any{"routeId": "r-1"})
	expectRequest(t, hub, "DELETE", "/api/v2/routes/r-1", "", "")
	expectStructured(t, res, `{"deleted":true,"routeId":"r-1"}`)
	expectAnnotations(t, s, "deleteRoute", Destructive)
}

func TestDeleteRouteRejectsEmptyID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "deleteRoute", map[string]any{"routeId": ""}))
	hub.assertNoRequests(t)
}
