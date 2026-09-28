package tools

import (
	"net/http"
	"testing"
)

func TestListRoutesFilters(t *testing.T) {
	tests := []struct {
		name  string
		args  map[string]any
		query string
	}{
		{"none", map[string]any{}, ""},
		{"status only", map[string]any{"status": "ACTIVE"}, "status=ACTIVE"},
		{"all three", map[string]any{"status": "FAILED", "inputId": "radio-1", "targetId": "kitchen"}, "inputId=radio-1&status=FAILED&targetId=kitchen"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := newFakeHub(t)
			hub.handle("GET", "/api/v2/routes", http.StatusOK, "["+routeJSON+"]")
			s := newTestSession(t, hub.URL, nil)

			res := callTool(t, s, "listRoutes", tt.args)
			expectRequest(t, hub, "GET", "/api/v2/routes", tt.query, "")
			expectStructured(t, res, `{"routes":[`+routeJSON+`]}`)
		})
	}
}

func TestListRoutesRejectsUnknownStatus(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "listRoutes", map[string]any{"status": "PLAYING"}))
	hub.assertNoRequests(t)
	expectAnnotations(t, s, "listRoutes", ReadOnly)
}

func TestGetRoute(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/routes/r-1", http.StatusOK, routeJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getRoute", map[string]any{"routeId": "r-1"})
	expectRequest(t, hub, "GET", "/api/v2/routes/r-1", "", "")
	expectStructured(t, res, routeJSON) // every Route field, including startedAt: null
	expectAnnotations(t, s, "getRoute", ReadOnly)
}

func TestGetRouteRejectsEmptyID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "getRoute", map[string]any{"routeId": ""}))
	hub.assertNoRequests(t)
}
