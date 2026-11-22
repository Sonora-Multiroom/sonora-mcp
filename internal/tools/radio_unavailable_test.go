package tools

import (
	"context"
	"errors"
	"net/http"
	"testing"

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
