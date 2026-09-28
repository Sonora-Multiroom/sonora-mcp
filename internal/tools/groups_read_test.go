package tools

import (
	"net/http"
	"testing"
)

func TestListGroups(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/groups", http.StatusOK, "["+groupJSON+"]")
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listGroups", map[string]any{})
	expectRequest(t, hub, "GET", "/api/v2/groups", "includeDisabled=false", "")
	expectStructured(t, res, `{"groups":[`+groupJSON+`]}`)
	expectAnnotations(t, s, "listGroups", ReadOnly)
}

func TestListGroupsIncludeDisabled(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/groups", http.StatusOK, "["+groupJSON+"]")
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listGroups", map[string]any{"includeDisabled": true})
	expectRequest(t, hub, "GET", "/api/v2/groups", "includeDisabled=true", "")
	expectStructured(t, res, `{"groups":[`+groupJSON+`]}`)
}

func TestGetGroup(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/groups/downstairs", http.StatusOK, groupJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "getGroup", map[string]any{"groupId": "downstairs"})
	expectRequest(t, hub, "GET", "/api/v2/groups/downstairs", "", "")
	expectStructured(t, res, groupJSON)
	expectAnnotations(t, s, "getGroup", ReadOnly)
}

func TestGetGroupRejectsEmptyID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "getGroup", map[string]any{"groupId": ""}))
	hub.assertNoRequests(t)
}
