package tools

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
)

func TestTTSError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		inventory  string
		want       hub.TTSDiagnosis
		reason     string
		loadingOff bool
	}{
		{"disabled", 200, ttsDisabledInventoryJSON, hub.DiagnosisDisabled, "", false},
		{"not installed", 200, inventoryJSON(true, ""), hub.DiagnosisNotInstalled, "", false},
		{"not installed, loading off", 200, inventoryJSON(false, ""), hub.DiagnosisNotInstalled, "", true},
		{"rejected", 200, inventoryJSON(true, `{"id":"tts","status":"REJECTED","rejectionReason":"needs API 0.2"}`), hub.DiagnosisRejected, "needs API 0.2", false},
		{"inert", 200, inventoryJSON(true, `{"id":"tts","status":"INERT","rejectionReason":null}`), hub.DiagnosisInert, "", false},
		{"active", 200, inventoryJSON(true, `{"id":"tts","status":"ACTIVE","rejectionReason":null}`), hub.DiagnosisVersionMismatch, "", false},
		{"inventory 404", 404, `{}`, hub.DiagnosisHubAddress, "", false},
		{"inventory 500", 500, `{}`, hub.DiagnosisUnknown, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			h.handle("GET", "/api/v2/extensions", tt.status, tt.inventory)
			err := ttsError(context.Background(), hub.NewClient(), h.URL, &hub.TTSNotOfferedError{})
			var ue *hub.TTSUnavailableError
			if !errors.As(err, &ue) {
				t.Fatalf("ttsError = %v, want *hub.TTSUnavailableError", err)
			}
			if ue.Diagnosis != tt.want || ue.Reason != tt.reason || ue.LoadingDisabled != tt.loadingOff {
				t.Errorf("diagnosis = %+v, want %v reason %q loadingDisabled %v", ue, tt.want, tt.reason, tt.loadingOff)
			}
			expectRequest(t, h, http.MethodGet, "/api/v2/extensions", "", "")
		})
	}
}

func TestTTSErrorWrappedNotOffered(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", "/api/v2/extensions", 200, ttsDisabledInventoryJSON)
	err := ttsError(context.Background(), hub.NewClient(), h.URL, errors.Join(errors.New("x"), &hub.TTSNotOfferedError{}))
	var ue *hub.TTSUnavailableError
	if !errors.As(err, &ue) {
		t.Fatalf("ttsError = %v, want a diagnosis", err)
	}
}

func TestTTSErrorPassesOtherErrorsThrough(t *testing.T) {
	h := newFakeHub(t)
	in := &hub.TTSError{StatusCode: 400, Code: "INVALID_REQUEST", Message: "no"}
	if got := ttsError(context.Background(), hub.NewClient(), h.URL, in); got != error(in) {
		t.Errorf("ttsError = %v, want the input error unchanged", got)
	}
	h.assertNoRequests(t)
}
