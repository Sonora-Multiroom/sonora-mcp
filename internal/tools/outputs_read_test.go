package tools

import (
	"net/http"
	"testing"
)

// newOutputsHub serves output A for the default list and A plus the disabled
// B for includeDisabled=true; filtering is the hub's job.
func newOutputsHub(t *testing.T) *fakeHub {
	t.Helper()
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/outputs?includeDisabled=false", http.StatusOK, "["+outputJSON+"]")
	hub.handle("GET", "/api/v2/outputs?includeDisabled=true", http.StatusOK, "["+outputJSON+","+disabledOutputJSON+"]")
	return hub
}

func TestListOutputs(t *testing.T) {
	hub := newOutputsHub(t)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listOutputs", map[string]any{})
	expectRequest(t, hub, "GET", "/api/v2/outputs", "includeDisabled=false", "")
	expectStructured(t, res, `{"outputs":[`+outputJSON+`]}`)
	expectAnnotations(t, s, "listOutputs", ReadOnly)
}

func TestListOutputsIncludeDisabled(t *testing.T) {
	hub := newOutputsHub(t)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listOutputs", map[string]any{"includeDisabled": true})
	expectRequest(t, hub, "GET", "/api/v2/outputs", "includeDisabled=true", "")
	expectStructured(t, res, `{"outputs":[`+outputJSON+","+disabledOutputJSON+`]}`)
}

func TestGetOutput(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/outputs/kitchen", http.StatusOK, outputJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getOutput", map[string]any{"outputId": "kitchen"})
	expectRequest(t, hub, "GET", "/api/v2/outputs/kitchen", "", "")
	expectStructured(t, res, outputJSON)
	expectAnnotations(t, s, "getOutput", ReadOnly)
}

func TestGetOutputRejectsEmptyID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "getOutput", map[string]any{"outputId": ""}))
	hub.assertNoRequests(t)
}
