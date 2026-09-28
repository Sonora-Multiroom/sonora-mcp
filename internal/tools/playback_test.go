package tools

import (
	"net/http"
	"testing"
)

func TestPlaybackOmitsUnsetOptions(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/play", http.StatusOK, playbackJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT",
	})
	expectRequest(t, hub, "POST", "/api/v2/play", "",
		`{"uri":"http://radio.example/stream","targetId":"kitchen","targetType":"SINGLE_OUTPUT"}`)
	expectStructured(t, res, playbackJSON)
	expectAnnotations(t, s, "playback", StateChanging)
}

func TestPlaybackSendsGivenOptions(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/play", http.StatusOK, playbackJSON)
	s := newTestSession(t, hub.URL, nil)

	res := callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "downstairs", "targetType": "OUTPUT_GROUP",
		"displayName": "Morning radio", "volume": 0,
	})
	expectRequest(t, hub, "POST", "/api/v2/play", "",
		`{"uri":"http://radio.example/stream","targetId":"downstairs","targetType":"OUTPUT_GROUP","displayName":"Morning radio","volume":0}`)
	expectOK(t, res)
}

func TestPlaybackRejectsOutOfRangeVolume(t *testing.T) {
	hub := newFakeHub(t)
	s := newTestSession(t, hub.URL, nil)

	expectInvalidInput(t, callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT", "volume": 101,
	}))
	hub.assertNoRequests(t)
}

func TestPlaybackSourceUnreachable(t *testing.T) {
	hub := newFakeHub(t)
	hub.handle("POST", "/api/v2/play", http.StatusBadGateway,
		`{"type":"about:blank","title":"Bad Gateway","status":502,"detail":"could not open http://radio.example/stream"}`)
	s := newTestSession(t, hub.URL, nil)

	expectErrorPrefix(t, callTool(t, s, "playback", map[string]any{
		"uri": "http://radio.example/stream", "targetId": "kitchen", "targetType": "SINGLE_OUTPUT",
	}), "SourceUnreachable:")
}
