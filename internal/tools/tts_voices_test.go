package tools

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

const voicesPath = "/api/tts/providers/piper-local/voices"

func TestListTtsVoices(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", voicesPath, http.StatusOK, voiceListJSON)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "listTtsVoices", map[string]any{"providerName": "piper-local"})

	expectStructured(t, res, voiceListResultJSON)
	expectRequest(t, h, "GET", voicesPath, "", "")
}

func TestListTtsVoicesFilters(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]any
		query string
	}{
		{"language", map[string]any{"language": "uk-UA"}, "language=uk-UA"},
		{"engine", map[string]any{"engine": "piper"}, "engine=piper"},
		{"both", map[string]any{"language": "uk-UA", "engine": "piper"}, "engine=piper&language=uk-UA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			h.handle("GET", voicesPath, http.StatusOK, voiceListJSON)
			s := newTestSession(t, h.URL, nil)
			args := map[string]any{"providerName": "piper-local"}
			for k, v := range tt.extra {
				args[k] = v
			}
			expectOK(t, callTool(t, s, "listTtsVoices", args))
			expectRequest(t, h, "GET", voicesPath, tt.query, "")
		})
	}
}

func TestListTtsVoicesUnknownProvider(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", "/api/tts/providers/nope/voices", http.StatusBadRequest, `{"error":"PROVIDER_NOT_FOUND","message":"Provider 'nope' was not found"}`)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "listTtsVoices", map[string]any{"providerName": "nope"})

	expectErrorPrefix(t, res, "NotFound:")
	if !strings.Contains(errorText(res), "nope") {
		t.Errorf("error %q does not name the provider", errorText(res))
	}
}

func TestListTtsVoicesEmptyIsSuccess(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", voicesPath, http.StatusOK, `{"providerName":"piper-local","voices":[]}`)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "listTtsVoices", map[string]any{"providerName": "piper-local"})

	expectStructured(t, res, `{"providerName":"piper-local","voices":[]}`)
}

func TestListTtsVoicesMetadata(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	expectAnnotations(t, s, "listTtsVoices", ReadOnly)
}

func TestListTtsVoicesRejectsEmptyProvider(t *testing.T) {
	h := newFakeHub(t)
	s := newTestSession(t, h.URL, nil)
	expectInvalidInput(t, callTool(t, s, "listTtsVoices", map[string]any{"providerName": ""}))
	h.assertNoRequests(t)
}

func TestListTtsVoicesTTSUnavailable(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", "/api/v2/extensions", http.StatusOK,
		inventoryJSON(true, `{"id":"tts","status":"REJECTED","rejectionReason":"needs API 0.2"}`))
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "listTtsVoices", map[string]any{"providerName": "piper-local"})

	expectErrorPrefix(t, res, "TtsUnavailable:")
	if !strings.Contains(errorText(res), "failed to load: needs API 0.2") {
		t.Errorf("error %q lacks the reason", errorText(res))
	}
}

func TestListTtsVoicesUsesOrdinaryBound(t *testing.T) {
	h := newFakeHub(t)
	h.handleDelayed("GET", voicesPath, http.StatusOK, voiceListJSON, 300*time.Millisecond)
	s := newTestSessionClients(t, h.URL, &http.Client{Timeout: 100 * time.Millisecond}, &http.Client{Timeout: 2 * time.Second})

	res := callTool(t, s, "listTtsVoices", map[string]any{"providerName": "piper-local"})

	expectErrorPrefix(t, res, "Timeout:")
}

func TestListTtsVoicesDescribesUnsupportedProviders(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	desc := toolByName(t, s, "listTtsVoices").Description
	if !strings.Contains(desc, "does not support voice listing") {
		t.Errorf("description does not warn that some providers cannot list voices: %s", desc)
	}
}
