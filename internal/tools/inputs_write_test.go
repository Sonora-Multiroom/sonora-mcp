package tools

import (
	"net/http"
	"testing"
)

const createdInputJSON = `{"inputId":"podcast-1","displayName":"Podcast","uri":"http://podcast.example/ep1.mp3","enabled":true,"autoRemove":true,"source":"EPHEMERAL","createdAt":"2026-09-28T10:00:00Z","pauseable":true}`

func TestCreateInputOmitsUnsetFlags(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/inputs", http.StatusCreated, createdInputJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "createInput", map[string]any{
		"inputId": "podcast-1", "displayName": "Podcast", "uri": "http://podcast.example/ep1.mp3",
	})
	// No enabled/autoRemove keys: the hub applies its own defaults.
	expectRequest(t, hub, "POST", "/api/v2/inputs", "",
		`{"inputId":"podcast-1","displayName":"Podcast","uri":"http://podcast.example/ep1.mp3"}`)
	expectStructured(t, res, createdInputJSON)
	expectAnnotations(t, s, "createInput", StateChanging)
}

func TestCreateInputSendsGivenFlags(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/inputs", http.StatusCreated, createdInputJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "createInput", map[string]any{
		"inputId": "podcast-1", "displayName": "Podcast", "uri": "http://podcast.example/ep1.mp3",
		"enabled": false, "autoRemove": true,
	})
	expectRequest(t, hub, "POST", "/api/v2/inputs", "",
		`{"inputId":"podcast-1","displayName":"Podcast","uri":"http://podcast.example/ep1.mp3","enabled":false,"autoRemove":true}`)
	expectOK(t, res)
}

func TestCreateInputRejectsBadID(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "createInput", map[string]any{
		"inputId": "bad id!", "displayName": "Podcast", "uri": "http://podcast.example/ep1.mp3",
	}))
	hub.assertNoRequests(t)
}

func TestDeleteInput(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("DELETE", "/api/v2/inputs/podcast-1", http.StatusNoContent, "")
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "deleteInput", map[string]any{"inputId": "podcast-1"})
	expectRequest(t, hub, "DELETE", "/api/v2/inputs/podcast-1", "", "")
	expectStructured(t, res, `{"deleted":true,"inputId":"podcast-1"}`)
	expectAnnotations(t, s, "deleteInput", Destructive)
}

func TestDeleteInputStatic(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("DELETE", "/api/v2/inputs/radio-1", http.StatusBadRequest,
		`{"type":"about:blank","title":"Bad Request","status":400,"detail":"static inputs cannot be deleted"}`)
	s := newTestSession(t, hub.URL, nil)

	expectErrorPrefix(t, callTool(t, s, "deleteInput", map[string]any{"inputId": "radio-1"}), "Validation:")
}
