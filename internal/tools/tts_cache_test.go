package tools

import (
	"net/http"
	"strings"
	"testing"
)

const (
	cacheStatsPath = "/api/tts/cache/stats"
	cachePath      = "/api/tts/cache"
)

func TestGetTtsCacheStats(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", cacheStatsPath, http.StatusOK, cacheStatsJSON)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "getTtsCacheStats", map[string]any{})

	expectStructured(t, res, cacheStatsJSON)
	expectRequest(t, h, "GET", cacheStatsPath, "", "")
}

func TestClearTtsCacheAll(t *testing.T) {
	h := newFakeHub(t)
	h.handle("DELETE", cachePath, http.StatusNoContent, "")
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "clearTtsCache", map[string]any{})

	expectStructured(t, res, `{"cleared":true}`)
	expectRequest(t, h, "DELETE", cachePath, "", "")
}

func TestClearTtsCacheOneProvider(t *testing.T) {
	h := newFakeHub(t)
	h.handle("DELETE", cachePath, http.StatusNoContent, "")
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "clearTtsCache", map[string]any{"providerName": "piper-local"})

	expectStructured(t, res, `{"cleared":true,"providerName":"piper-local"}`)
	expectRequest(t, h, "DELETE", cachePath, "providerName=piper-local", "")
}

func TestClearTtsCacheUnknownProvider(t *testing.T) {
	h := newFakeHub(t)
	h.handle("DELETE", cachePath, http.StatusBadRequest, `{"error":"PROVIDER_NOT_FOUND","message":"Provider 'nope' was not found"}`)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "clearTtsCache", map[string]any{"providerName": "nope"})

	expectErrorPrefix(t, res, "NotFound:")
	if !strings.Contains(errorText(res), "nope") {
		t.Errorf("error %q does not name the provider", errorText(res))
	}
}

func TestTtsCacheMetadata(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	expectAnnotations(t, s, "getTtsCacheStats", ReadOnly)
	expectAnnotations(t, s, "clearTtsCache", DestructiveIdempotent)
	desc := toolByName(t, s, "clearTtsCache").Description
	for _, want := range []string{"synthesised again", "charges again"} {
		if !strings.Contains(desc, want) {
			t.Errorf("clearTtsCache description does not mention %q: %s", want, desc)
		}
	}
}

func TestGetTtsCacheStatsTTSUnavailable(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", "/api/v2/extensions", http.StatusOK,
		inventoryJSON(true, `{"id":"tts","status":"INERT","rejectionReason":null}`))
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "getTtsCacheStats", map[string]any{})

	expectErrorPrefix(t, res, "TtsUnavailable:")
	if !strings.Contains(errorText(res), "loaded but inactive") {
		t.Errorf("error %q lacks the reason", errorText(res))
	}
}
