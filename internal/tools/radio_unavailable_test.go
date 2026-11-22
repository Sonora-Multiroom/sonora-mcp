package tools

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
)

func TestRadioError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		inventory  string
		want       hub.ExtensionDiagnosis
		reason     string
		loadingOff bool
	}{
		{"disabled", 200, inventoryJSON(true, `{"id":"radio","status":"DISABLED","rejectionReason":null}`), hub.DiagnosisDisabled, "", false},
		{"not installed", 200, inventoryJSON(true, ""), hub.DiagnosisNotInstalled, "", false},
		{"not installed, loading off", 200, inventoryJSON(false, ""), hub.DiagnosisNotInstalled, "", true},
		{"rejected", 200, inventoryJSON(true, `{"id":"radio","status":"REJECTED","rejectionReason":"needs API 0.2"}`), hub.DiagnosisRejected, "needs API 0.2", false},
		{"inert", 200, inventoryJSON(true, `{"id":"radio","status":"INERT","rejectionReason":null}`), hub.DiagnosisInert, "", false},
		{"active", 200, inventoryJSON(true, `{"id":"radio","status":"ACTIVE","rejectionReason":null}`), hub.DiagnosisVersionMismatch, "", false},
		{"inventory 404", 404, `{}`, hub.DiagnosisHubAddress, "", false},
		{"inventory 500", 500, `{}`, hub.DiagnosisUnknown, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			h.handle("GET", "/api/v2/extensions", tt.status, tt.inventory)
			err := radioError(context.Background(), hub.NewClient(), h.URL, &hub.RadioNotOfferedError{})
			var ue *hub.RadioUnavailableError
			if !errors.As(err, &ue) {
				t.Fatalf("radioError = %v, want *hub.RadioUnavailableError", err)
			}
			if ue.Diagnosis != tt.want || ue.Reason != tt.reason || ue.LoadingDisabled != tt.loadingOff {
				t.Errorf("diagnosis = %+v, want %v reason %q loadingDisabled %v", ue, tt.want, tt.reason, tt.loadingOff)
			}
			expectRequest(t, h, http.MethodGet, "/api/v2/extensions", "", "")
		})
	}
}

func TestRadioErrorWrappedNotOffered(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", "/api/v2/extensions", 200, inventoryJSON(true, `{"id":"radio","status":"DISABLED","rejectionReason":null}`))
	err := radioError(context.Background(), hub.NewClient(), h.URL, errors.Join(errors.New("x"), &hub.RadioNotOfferedError{}))
	var ue *hub.RadioUnavailableError
	if !errors.As(err, &ue) {
		t.Fatalf("radioError = %v, want a diagnosis", err)
	}
}

func TestRadioErrorPassesOtherErrorsThrough(t *testing.T) {
	h := newFakeHub(t)
	in := &hub.APIError{StatusCode: 400}
	if got := radioError(context.Background(), hub.NewClient(), h.URL, in); got != error(in) {
		t.Errorf("radioError = %v, want the input error unchanged", got)
	}
	h.assertNoRequests(t)
}

const (
	aboutBlank404 = `{"type":"about:blank","title":"Not Found","status":404}`
	extensionsURL = "/api/v2/extensions"
)

// radioSearchHub is a fake hub whose radio search is not offered (404) and
// whose extension inventory is inventory with the given status.
func radioSearchHub(t *testing.T, inventoryStatus int, inventory string) *fakeHub {
	t.Helper()
	h := newFakeHub(t)
	h.handle("GET", radioStationsPath, 404, aboutBlank404)
	h.handle("GET", extensionsURL, inventoryStatus, inventory)
	return h
}

func TestSearchRadioStationsUnavailable(t *testing.T) {
	tests := []struct {
		name      string
		inventory string
		contains  []string
	}{
		{"not installed", inventoryJSON(true, ""), []string{"not installed"}},
		{"not installed, loading off", inventoryJSON(false, ""), []string{"not installed", "switched off"}},
		{"disabled", radioInventoryJSON("DISABLED"), []string{"disabled"}},
		{"rejected", inventoryJSON(true, `{"id":"radio","status":"REJECTED","rejectionReason":"needs API 0.2"}`), []string{"failed to load: needs API 0.2"}},
		{"inert", radioInventoryJSON("INERT"), []string{"inactive"}},
		{"too old", radioInventoryJSON("ACTIVE"), []string{"0.2.0 or later"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := radioSearchHub(t, 200, tt.inventory)
			s := newTestSession(t, h.URL, nil)

			res := callTool(t, s, "searchRadioStations", map[string]any{"name": "kiss"})

			expectErrorPrefix(t, res, "RadioUnavailable: ")
			got := errorText(res)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("error = %q, want it to contain %q", got, want)
				}
			}
			if strings.Contains(got, "Kiss FM") {
				t.Errorf("error = %q, want no station text", got)
			}
			reqs := h.requests()
			if len(reqs) != 2 || reqs[0].Path != radioStationsPath || reqs[1].Path != extensionsURL {
				t.Errorf("requests = %+v, want the search then the extension list", reqs)
			}
		})
	}
}

func TestSearchRadioStationsInventoryUnreadable(t *testing.T) {
	t.Run("500 gives the generic message", func(t *testing.T) {
		h := radioSearchHub(t, 500, `{}`)
		s := newTestSession(t, h.URL, nil)
		const want = "RadioUnavailable: radio is not available on this hub"
		if got := errorText(callTool(t, s, "searchRadioStations", map[string]any{})); got != want {
			t.Errorf("error = %q, want %q", got, want)
		}
	})
	t.Run("404 is a wrong hub address", func(t *testing.T) {
		h := radioSearchHub(t, 404, `{}`)
		s := newTestSession(t, h.URL, nil)
		expectErrorPrefix(t, callTool(t, s, "searchRadioStations", map[string]any{}), "Network:")
	})
}

func TestSearchRadioStationsReadsInventoryOnlyAfterNotOffered(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		header map[string]string
		prefix string
	}{
		{"success", 200, radioStationPageJSON, nil, ""},
		{"invalid parameter", 400, radioProblemJSON("invalid-parameter", 400, "countryCode must be two letters", "countryCode"), nil, "Validation:"},
		{"catalogue unavailable", 503, radioProblemJSON("catalogue-unavailable", 503, "down", ""), map[string]string{"Retry-After": "30"}, "ServiceUnavailable:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			h.handleHeaders("GET", radioStationsPath, tt.status, tt.body, tt.header)
			h.handle("GET", extensionsURL, 200, radioInventoryJSON("DISABLED"))
			s := newTestSession(t, h.URL, nil)

			res := callTool(t, s, "searchRadioStations", map[string]any{})

			if tt.prefix == "" {
				expectOK(t, res)
			} else {
				expectErrorPrefix(t, res, tt.prefix)
			}
			if reqs := h.requests(); len(reqs) != 1 || reqs[0].Path != radioStationsPath {
				t.Errorf("requests = %+v, want only the search", reqs)
			}
		})
	}
}

func TestSearchRadioStationsProblemMapping(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		header   map[string]string
		prefix   string
		contains []string
	}{
		{"invalid parameter", 400, radioProblemJSON("invalid-parameter", 400, "countryCode must be two letters", "countryCode"), nil,
			"Validation:", []string{"countryCode must be two letters", "countryCode"}},
		{"catalogue unavailable with delay", 503, radioProblemJSON("catalogue-unavailable", 503, "down", ""), map[string]string{"Retry-After": "30"},
			"ServiceUnavailable:", []string{"retry in 30 s"}},
		{"catalogue unavailable without delay", 503, radioProblemJSON("catalogue-unavailable", 503, "down", ""), nil,
			"ServiceUnavailable:", []string{"retry later"}},
		{"catalogue bad answer", 502, radioProblemJSON("catalogue-bad-answer", 502, "not JSON", ""), nil,
			"SourceUnreachable:", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			h.handleHeaders("GET", radioStationsPath, tt.status, tt.body, tt.header)
			s := newTestSession(t, h.URL, nil)

			res := callTool(t, s, "searchRadioStations", map[string]any{})

			expectErrorPrefix(t, res, tt.prefix)
			for _, want := range tt.contains {
				if !strings.Contains(errorText(res), want) {
					t.Errorf("error = %q, want it to contain %q", errorText(res), want)
				}
			}
		})
	}
}

func TestSearchRadioStationsUsesTheRadioClient(t *testing.T) {
	short := &http.Client{Timeout: 200 * time.Millisecond}
	long := &http.Client{Timeout: 2 * time.Second}

	t.Run("longer bound succeeds", func(t *testing.T) {
		h := newFakeHub(t)
		h.handleDelayed("GET", radioStationsPath, 200, radioStationPageJSON, 500*time.Millisecond)
		s := newTestSessionRadio(t, h.URL, short, long)
		expectStructured(t, callTool(t, s, "searchRadioStations", map[string]any{}), radioStationPageJSON)
	})
	t.Run("shorter bound times out", func(t *testing.T) {
		h := newFakeHub(t)
		h.handleDelayed("GET", radioStationsPath, 200, radioStationPageJSON, 500*time.Millisecond)
		s := newTestSessionRadio(t, h.URL, long, short)
		expectErrorPrefix(t, callTool(t, s, "searchRadioStations", map[string]any{}), "Timeout:")
	})
}

func TestUnavailableRadioLeavesOtherToolsAlone(t *testing.T) {
	h := radioSearchHub(t, 500, `{}`)
	h.handle("GET", "/api/v2/outputs", 200, "["+outputJSON+"]")
	s := newTestSession(t, h.URL, nil)

	expectErrorPrefix(t, callTool(t, s, "searchRadioStations", map[string]any{}), "RadioUnavailable:")
	expectOK(t, callTool(t, s, "listOutputs", map[string]any{}))
}
