package tools

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorNotFoundNamesResource(t *testing.T) {
	hub := newFakeHub(t) // unregistered routes answer 404
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getOutput", map[string]any{"outputId": "garage"})
	expectErrorPrefix(t, res, "NotFound:")
	if !strings.Contains(errorText(res), "garage") {
		t.Errorf("error %q does not name the output", errorText(res))
	}
}

func TestErrorValidationKeepsHubDetail(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("PUT", "/api/v2/outputs/garage/volume", http.StatusBadRequest,
		`{"type":"about:blank","title":"Bad Request","status":400,"detail":"output is disabled"}`)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "setOutputVolume", map[string]any{"outputId": "garage", "volume": 20})
	expectErrorPrefix(t, res, "Validation:")
	if !strings.Contains(errorText(res), "output is disabled") {
		t.Errorf("error %q lacks the hub's detail", errorText(res))
	}
}

func TestErrorConflictNamesReason(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/routes", http.StatusConflict,
		`{"type":"urn:multiroom:error:route-admission","title":"Route not admitted","status":409,"detail":"Output 'bathroom' is disabled","reason":"OUTPUT_DISABLED","outputId":"bathroom"}`)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "createRoute", map[string]any{"inputId": "radio-1", "targetId": "bathroom", "targetType": "SINGLE_OUTPUT"})
	if got, want := errorText(res), "Conflict: Output 'bathroom' is disabled (OUTPUT_DISABLED)"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestErrorConflictOnPlaybackAndTransfer(t *testing.T) {
	const refusal = `{"type":"urn:multiroom:error:route-admission","title":"Route not admitted","status":409,"detail":"Group 'downstairs' is disabled","reason":"GROUP_DISABLED"}`
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/play", http.StatusConflict, refusal)
	hub.handle("POST", "/api/v2/routes/r-1/transfer", http.StatusConflict, refusal)
	s := newTestSession(t, hub.URL, nil)

	const want = "Conflict: Group 'downstairs' is disabled (GROUP_DISABLED)"
	if got := errorText(callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "downstairs", "targetType": "OUTPUT_GROUP",
	})); got != want {
		t.Errorf("playback error = %q, want %q", got, want)
	}
	if got := errorText(callTool(t, s, "transferRoute", map[string]any{
		"routeId": "r-1", "targetId": "downstairs", "targetType": "OUTPUT_GROUP",
	})); got != want {
		t.Errorf("transferRoute error = %q, want %q", got, want)
	}
}

func TestErrorConflictDuplicateInput(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/inputs", http.StatusConflict,
		`{"type":"urn:multiroom:error:conflict","title":"Conflict","status":409,"detail":"Input ID already exists"}`)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "createInput", map[string]any{
		"inputId": "radio-1", "displayName": "Radio", "uri": "http://radio.example/stream",
	})
	if got, want := errorText(res), "Conflict: Input ID already exists"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

func TestErrorServiceUnavailable(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/play", http.StatusServiceUnavailable,
		`{"type":"about:blank","title":"Service Unavailable","status":503,"detail":"audio engine restarting"}`)
	s := newTestSession(t, hub.URL, nil)

	expectErrorPrefix(t, callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT",
	}), "ServiceUnavailable:")
}

func TestErrorMalformedResponse(t *testing.T) {
	tests := []struct {
		name, tool, method, path, body string
		args                           map[string]any
	}{
		{"not json", "listOutputs", "GET", "/api/v2/outputs", `not json`, map[string]any{}},
		{"array for object", "getOutput", "GET", "/api/v2/outputs/kitchen", `[]`, map[string]any{"outputId": "kitchen"}},
		{"wrong field type", "getOutput", "GET", "/api/v2/outputs/kitchen", `{"volume":"loud"}`, map[string]any{"outputId": "kitchen"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := newFakeHub(t)
			hub.handle(tt.method, tt.path, http.StatusOK, tt.body)
			s := newTestSession(t, hub.URL, nil)

			res := callTool(t, s, tt.tool, tt.args)
			expectErrorPrefix(t, res, "MalformedResponse:")
			if res.StructuredContent != nil {
				t.Errorf("error result carries partial data: %v", res.StructuredContent)
			}
		})
	}
}

func TestErrorNetwork(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	url := closed.URL
	closed.Close()
	s := newTestSession(t, url, nil)

	expectErrorPrefix(t, callTool(t, s, "listOutputs", map[string]any{}), "Network:")
}

// TestErrorNotFoundNamesMissingTarget covers a 404 that can mean more than
// one missing resource: the error names the one the hub reports missing.
func TestErrorNotFoundNamesMissingTarget(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/routes/r-1/transfer", http.StatusNotFound,
		`{"type":"urn:multiroom:error:not-found","title":"Resource Not Found","detail":"Output not found: garage","status":404}`)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "transferRoute", map[string]any{"routeId": "r-1", "targetId": "garage", "targetType": "SINGLE_OUTPUT"})
	if got := errorText(res); got != "NotFound: output not found: garage" {
		t.Errorf("error = %q, want %q", got, "NotFound: output not found: garage")
	}
}
