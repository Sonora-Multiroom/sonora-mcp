package tools

import (
	"net/http"
	"testing"
)

// The hub package always sends includeDisabled; false is the spec default,
// so the default call means the same as omitting it.

func TestListInputs(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/inputs", http.StatusOK, "["+inputJSON+"]")
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listInputs", map[string]any{})
	expectRequest(t, hub, "GET", "/api/v2/inputs", "includeDisabled=false", "")
	expectStructured(t, res, `{"inputs":[`+inputJSON+`]}`)
	expectAnnotations(t, s, "listInputs", ReadOnly)
}

func TestListInputsIncludeDisabled(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/inputs", http.StatusOK, "["+inputJSON+"]")
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listInputs", map[string]any{"includeDisabled": true})
	expectRequest(t, hub, "GET", "/api/v2/inputs", "includeDisabled=true", "")
	expectStructured(t, res, `{"inputs":[`+inputJSON+`]}`)
}

func TestListInputsTrailingSlashHubURL(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/inputs", http.StatusOK, "[]")
	s := newTestSession(t, hub.URL+"/", nil)

	res := callTool(t, s, "listInputs", nil)
	expectRequest(t, hub, "GET", "/api/v2/inputs", "includeDisabled=false", "")
	expectStructured(t, res, `{"inputs":[]}`)
}

func TestGetInput(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/inputs/radio-1", http.StatusOK, inputJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getInput", map[string]any{"inputId": "radio-1"})
	expectRequest(t, hub, "GET", "/api/v2/inputs/radio-1", "", "")
	expectStructured(t, res, inputJSON)
	expectAnnotations(t, s, "getInput", ReadOnly)
}

func TestGetInputEscapesID(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/inputs/a%20b%2Fc%23d", http.StatusOK, inputJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getInput", map[string]any{"inputId": "a b/c#d"})
	expectRequest(t, hub, "GET", "/api/v2/inputs/a%20b%2Fc%23d", "", "")
	expectOK(t, res)
}

func TestGetInputRejectsEmptyID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "getInput", map[string]any{"inputId": ""}))
	expectInvalidInput(t, callTool(t, s, "getInput", map[string]any{}))
	hub.assertNoRequests(t)
}
