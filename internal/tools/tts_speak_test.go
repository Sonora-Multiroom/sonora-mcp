package tools

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

const speakPath = "/api/tts/speak"

func speakArgs(extra map[string]any) map[string]any {
	args := map[string]any{"text": "Dinner is ready", "targetName": "kitchen", "targetType": "SINGLE_OUTPUT"}
	for k, v := range extra {
		args[k] = v
	}
	return args
}

func TestSpeakSendsRequest(t *testing.T) {
	h := newFakeHub(t)
	h.handle("POST", speakPath, http.StatusAccepted, speakAcceptedJSON)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "speak", speakArgs(nil))

	expectStructured(t, res, speakAcceptedJSON)
	expectRequest(t, h, "POST", speakPath, "", `{"text":"Dinner is ready","targetName":"kitchen","targetType":"SINGLE_OUTPUT"}`)
}

func TestSpeakOmitsUnsetOptionalFields(t *testing.T) {
	h := newFakeHub(t)
	h.handle("POST", speakPath, http.StatusAccepted, speakAcceptedJSON)
	s := newTestSession(t, h.URL, nil)

	expectOK(t, callTool(t, s, "speak", speakArgs(nil)))

	body := h.onlyRequest(t).Body
	for _, field := range []string{"providerName", "voice", "language", "playbackMode", "routeVolume", "duckingLevelDb"} {
		if strings.Contains(body, field) {
			t.Errorf("body %s contains %q, want it omitted", body, field)
		}
	}
}

func TestSpeakPassesProviderVoiceAndLanguage(t *testing.T) {
	h := newFakeHub(t)
	h.handle("POST", speakPath, http.StatusAccepted, speakAcceptedJSON)
	s := newTestSession(t, h.URL, nil)

	expectOK(t, callTool(t, s, "speak", speakArgs(map[string]any{
		"providerName": "piper-local", "voice": "uk_UA-lada-x_low", "language": "uk-UA",
	})))

	expectRequest(t, h, "POST", speakPath, "", `{"text":"Dinner is ready","targetName":"kitchen","targetType":"SINGLE_OUTPUT","providerName":"piper-local","voice":"uk_UA-lada-x_low","language":"uk-UA"}`)
}

func TestSpeakRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"empty text", speakArgs(map[string]any{"text": ""})},
		{"missing text", map[string]any{"targetName": "kitchen", "targetType": "SINGLE_OUTPUT"}},
		{"missing target", map[string]any{"text": "hi", "targetType": "SINGLE_OUTPUT"}},
		{"empty target", speakArgs(map[string]any{"targetName": ""})},
		{"bad target type", speakArgs(map[string]any{"targetType": "ROOM"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			s := newTestSession(t, h.URL, nil)
			expectInvalidInput(t, callTool(t, s, "speak", tt.args))
			h.assertNoRequests(t)
		})
	}
}

func TestSpeakUnknownTarget(t *testing.T) {
	h := newFakeHub(t)
	h.handle("POST", speakPath, http.StatusBadRequest, `{"error":"TARGET_NOT_FOUND","message":"Target 'kitchn' was not found"}`)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "speak", speakArgs(map[string]any{"targetName": "kitchn"}))

	expectErrorPrefix(t, res, "NotFound:")
	if !strings.Contains(errorText(res), "kitchn") {
		t.Errorf("error %q does not name the target", errorText(res))
	}
}

func TestSpeakTTSUnavailable(t *testing.T) {
	t.Run("diagnosed", func(t *testing.T) {
		h := newFakeHub(t)
		h.handle("GET", "/api/v2/extensions", http.StatusOK, ttsDisabledInventoryJSON)
		s := newTestSession(t, h.URL, nil)

		res := callTool(t, s, "speak", speakArgs(nil))

		const want = "TtsUnavailable: text-to-speech is not available on this hub: the TTS extension is installed but disabled in the hub's configuration"
		if got := errorText(res); got != want {
			t.Errorf("error = %q, want %q", got, want)
		}
	})
	t.Run("not a hub", func(t *testing.T) {
		h := newFakeHub(t)
		s := newTestSession(t, h.URL, nil)
		expectErrorPrefix(t, callTool(t, s, "speak", speakArgs(nil)), "Network:")
	})
	t.Run("diagnosis slower than the bound", func(t *testing.T) {
		h := newFakeHub(t)
		h.handleDelayed("GET", "/api/v2/extensions", http.StatusOK, ttsDisabledInventoryJSON, time.Second)
		s := newTestSessionClients(t, h.URL, &http.Client{Timeout: 100 * time.Millisecond}, &http.Client{Timeout: 2 * time.Second})

		start := time.Now()
		res := callTool(t, s, "speak", speakArgs(nil))

		const want = "TtsUnavailable: text-to-speech is not available on this hub"
		if got := errorText(res); got != want {
			t.Errorf("error = %q, want %q", got, want)
		}
		if d := time.Since(start); d >= time.Second {
			t.Errorf("call took %v, want it bounded", d)
		}
	})
}

func TestSpeakMetadata(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	expectAnnotations(t, s, "speak", StateChanging)
	desc := toolByName(t, s, "speak").Description
	for _, want := range []string{"queued", "targetId", "billed"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description does not mention %q: %s", want, desc)
		}
	}
}

func TestSpeakUsesItsOwnClient(t *testing.T) {
	fast := &http.Client{Timeout: 100 * time.Millisecond}
	slow := &http.Client{Timeout: 2 * time.Second}

	t.Run("longer bound", func(t *testing.T) {
		h := newFakeHub(t)
		h.handleDelayed("POST", speakPath, http.StatusAccepted, speakAcceptedJSON, 300*time.Millisecond)
		s := newTestSessionClients(t, h.URL, fast, slow)
		expectStructured(t, callTool(t, s, "speak", speakArgs(nil)), speakAcceptedJSON)
	})
	t.Run("swapped clients time out", func(t *testing.T) {
		h := newFakeHub(t)
		h.handleDelayed("POST", speakPath, http.StatusAccepted, speakAcceptedJSON, 300*time.Millisecond)
		s := newTestSessionClients(t, h.URL, slow, fast)
		expectErrorPrefix(t, callTool(t, s, "speak", speakArgs(nil)), "Timeout:")
	})
}

func TestSpeakPlaybackFields(t *testing.T) {
	h := newFakeHub(t)
	h.handle("POST", speakPath, http.StatusAccepted, speakAcceptedJSON)
	s := newTestSession(t, h.URL, nil)

	expectOK(t, callTool(t, s, "speak", speakArgs(map[string]any{
		"playbackMode": "duck-others", "routeVolume": 40, "duckingLevelDb": -20,
	})))

	expectRequest(t, h, "POST", speakPath, "", `{"text":"Dinner is ready","targetName":"kitchen","targetType":"SINGLE_OUTPUT","playbackMode":"duck-others","routeVolume":40,"duckingLevelDb":-20}`)
}

func TestSpeakRejectsOutOfRangeLevels(t *testing.T) {
	for _, extra := range []map[string]any{
		{"routeVolume": 101},
		{"routeVolume": -1},
		{"duckingLevelDb": -61},
		{"duckingLevelDb": 0.5},
	} {
		h := newFakeHub(t)
		s := newTestSession(t, h.URL, nil)
		expectInvalidInput(t, callTool(t, s, "speak", speakArgs(extra)))
		h.assertNoRequests(t)
	}
}

func TestSpeakHubRejectsPlaybackMode(t *testing.T) {
	h := newFakeHub(t)
	h.handle("POST", speakPath, http.StatusBadRequest, `{"error":"INVALID_REQUEST","message":"duckingLevelDb is not allowed with playbackMode mix"}`)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "speak", speakArgs(map[string]any{"playbackMode": "mix", "duckingLevelDb": -10}))

	expectErrorPrefix(t, res, "Validation:")
	if !strings.Contains(errorText(res), "not allowed with playbackMode mix") {
		t.Errorf("error %q lacks the hub's explanation", errorText(res))
	}
}

func TestSpeakDescribesPlayback(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	desc := toolByName(t, s, "speak").Description
	for _, want := range []string{"duck-others", "mix", "changes no output's volume"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description does not mention %q: %s", want, desc)
		}
	}
}

func TestSpeakDescribesTimeout(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	desc := toolByName(t, s, "speak").Description
	for _, want := range []string{"Timeout", "may still"} {
		if !strings.Contains(desc, want) {
			t.Errorf("description does not mention %q: %s", want, desc)
		}
	}
}
