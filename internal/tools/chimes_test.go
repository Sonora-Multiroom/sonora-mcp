package tools

import (
	"net/http"
	"testing"
)

const chimesJSON = `{"chimes":[{"id":"bell","durationMillis":1200},{"id":"ding","durationMillis":400}]}`

func TestListChimes(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/chimes", http.StatusOK, chimesJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "listChimes", map[string]any{})
	expectRequest(t, hub, "GET", "/api/v2/chimes", "", "")
	expectStructured(t, res, chimesJSON)
	expectAnnotations(t, s, "listChimes", ReadOnly)
}

func TestListChimesEmpty(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/chimes", http.StatusOK, `{"chimes":[]}`)
	s := newTestSession(t, hub.URL, nil)

	expectStructured(t, callTool(t, s, "listChimes", map[string]any{}), `{"chimes":[]}`)
}

func TestListChimesOldHub(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/chimes", http.StatusNotFound,
		`{"type":"about:blank","title":"Not Found","status":404}`)
	s := newTestSession(t, hub.URL, nil)

	expectErrorPrefix(t, callTool(t, s, "listChimes", map[string]any{}), "HubError:")
}

func TestPlaybackSendsChime(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/play", http.StatusOK, playbackJSON)
	s := newTestSession(t, hub.URL, nil)

	expectOK(t, callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT", "chime": "none",
	}))
	expectRequest(t, hub, "POST", "/api/v2/play", "",
		`{"uri":"http://radio.example/stream","targetId":"kitchen","targetType":"SINGLE_OUTPUT","chime":"none"}`)
}

func TestCreateRouteSendsChime(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/routes", http.StatusCreated, routeJSON)
	s := newTestSession(t, hub.URL, nil)

	expectOK(t, callTool(t, s, "createRoute", map[string]any{
		"inputId": "radio-1", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT", "chime": "bell",
	}))
	expectRequest(t, hub, "POST", "/api/v2/routes", "",
		`{"inputId":"radio-1","targetId":"kitchen","targetType":"SINGLE_OUTPUT","chime":"bell"}`)
}

func TestCreateInputSendsDefaultChime(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/inputs", http.StatusCreated, createdInputJSON)
	s := newTestSession(t, hub.URL, nil)

	expectOK(t, callTool(t, s, "createInput", map[string]any{
		"inputId": "podcast-1", "displayName": "Podcast", "uri": "http://podcast.example/ep1.mp3", "defaultChime": "ding",
	}))
	expectRequest(t, hub, "POST", "/api/v2/inputs", "",
		`{"inputId":"podcast-1","displayName":"Podcast","uri":"http://podcast.example/ep1.mp3","defaultChime":"ding"}`)
}

func TestChimeRejectsEmpty(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT", "chime": "",
	}))
	expectInvalidInput(t, callTool(t, s, "createRoute", map[string]any{
		"inputId": "radio-1", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT", "chime": "",
	}))
	expectInvalidInput(t, callTool(t, s, "createInput", map[string]any{
		"inputId": "p", "displayName": "P", "uri": "http://x.example/a", "defaultChime": "",
	}))
	hub.assertNoRequests(t)
}

// The chime settings pass through the read tools; "none" is a value.
func TestReadToolsShowChimes(t *testing.T) {
	const (
		output = `{"outputId":"kitchen","displayName":"Kitchen","volume":40,"muted":false,"available":true,"enabled":true,"chime":"bell"}`
		group  = `{"groupId":"downstairs","displayName":"Downstairs","outputIds":["kitchen","living-room"],"muted":false,"enabled":true,"system":false,"chime":"none"}`
		input  = `{"inputId":"radio-1","displayName":"Radio","uri":"http://radio.example/stream","enabled":true,"autoRemove":false,"source":"STATIC","createdAt":"2026-09-01T10:00:00Z","pauseable":false,"defaultJoinMode":null,"defaultRouteVolume":null,"defaultDuckingLevelDb":null,"defaultChime":"ding","origin":"CONFIGURATION","playable":true,"unplayableReason":null}`
	)
	hub := newFakeHub(t)
	hub.handle("GET", "/api/v2/outputs/kitchen", http.StatusOK, output)
	hub.handle("GET", "/api/v2/groups", http.StatusOK, "["+group+"]")
	hub.handle("GET", "/api/v2/inputs/radio-1", http.StatusOK, input)
	s := newTestSession(t, hub.URL, nil)

	expectStructured(t, callTool(t, s, "getOutput", map[string]any{"outputId": "kitchen"}), output)
	expectStructured(t, callTool(t, s, "listGroups", map[string]any{}), `{"groups":[`+group+`]}`)
	expectStructured(t, callTool(t, s, "getInput", map[string]any{"inputId": "radio-1"}), input)
}
