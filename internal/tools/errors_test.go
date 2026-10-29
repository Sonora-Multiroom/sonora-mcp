package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
)

// timeoutErr is a net.Error that reports a timeout.
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestToolError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		category string
		contains []string
	}{
		{"decode", &hub.DecodeError{Err: errors.New("unexpected EOF")}, CategoryMalformedResponse, []string{"unexpected EOF"}},
		{"wrapped decode", fmt.Errorf("x: %w", &hub.DecodeError{Err: errors.New("bad")}), CategoryMalformedResponse, nil},
		{"deadline", context.DeadlineExceeded, CategoryTimeout, nil},
		{"client timeout", &url.Error{Op: "Get", URL: "http://hub", Err: timeoutErr{}}, CategoryTimeout, nil},
		{"not found", &hub.NotFoundError{Resource: "output", ID: "garage"}, CategoryNotFound, []string{"garage"}},
		{"target not found", &hub.NotFoundError{Resource: "target", ID: "attic"}, CategoryNotFound, []string{"attic"}},
		{"validation", &hub.APIError{StatusCode: 400, Title: "Bad Request", Detail: "output is disabled"}, CategoryValidation, []string{"output is disabled"}},
		{"route failed", &hub.APIError{StatusCode: 422, Detail: "no capacity"}, CategoryRouteFailed, []string{"no capacity"}},
		{"source unreachable", &hub.APIError{StatusCode: 502, Detail: "stream 404"}, CategorySourceUnreachable, []string{"stream 404"}},
		{"service unavailable", &hub.APIError{StatusCode: 503, Detail: "restarting"}, CategoryServiceUnavailable, []string{"restarting"}},
		{"other api error", &hub.APIError{StatusCode: 418, Detail: "short and stout"}, CategoryHubError, []string{"418", "short and stout"}},
		{"status", &hub.StatusError{StatusCode: 500}, CategoryHubError, []string{"500"}},
		{"network", &url.Error{Op: "Get", URL: "http://hub", Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}, CategoryNetwork, nil},
		{"unknown", errors.New("boom"), CategoryNetwork, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolError(tt.err).Error()
			if !strings.HasPrefix(got, tt.category+": ") {
				t.Fatalf("toolError = %q, want prefix %q", got, tt.category+": ")
			}
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("toolError = %q, want it to contain %q", got, s)
				}
			}
		})
	}
}

func TestToolErrorConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"reason", &hub.APIError{StatusCode: 409, Title: "Route not admitted", Detail: "Output 'bathroom' is disabled", Reason: "OUTPUT_DISABLED", OutputID: "bathroom"},
			"Conflict: Output 'bathroom' is disabled (OUTPUT_DISABLED)"},
		{"reason without output", &hub.APIError{StatusCode: 409, Detail: "Group 'downstairs' is disabled", Reason: "GROUP_DISABLED"},
			"Conflict: Group 'downstairs' is disabled (GROUP_DISABLED)"},
		{"unknown reason", &hub.APIError{StatusCode: 409, Detail: "Output 'bathroom' is busy", Reason: "SPEAKER_ON_FIRE"},
			"Conflict: Output 'bathroom' is busy (SPEAKER_ON_FIRE)"},
		{"no reason", &hub.APIError{StatusCode: 409, Title: "Conflict", Detail: "Input ID already exists"},
			"Conflict: Input ID already exists"},
		{"title only", &hub.APIError{StatusCode: 409, Title: "Route not admitted"},
			"Conflict: Route not admitted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toolError(tt.err).Error(); got != tt.want {
				t.Errorf("toolError = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToolErrorConflictWithoutBody(t *testing.T) {
	got := toolError(&hub.StatusError{StatusCode: 409}).Error()
	if !strings.HasPrefix(got, CategoryConflict+": ") || strings.Contains(got, "(") {
		t.Errorf("toolError = %q, want a generic Conflict message without a reason", got)
	}
}

func TestToolErrorKeepsCause(t *testing.T) {
	cause := &hub.StatusError{StatusCode: 500}
	var se *hub.StatusError
	if !errors.As(toolError(cause), &se) {
		t.Fatal("toolError must wrap the hub error")
	}
}

func TestToolErrorTTS(t *testing.T) {
	const head = "text-to-speech is not available on this hub"
	tests := []struct {
		name     string
		err      error
		category string
		want     string // exact text after the category, or "" to skip
		contains []string
	}{
		{"not installed", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisNotInstalled}, CategoryTtsUnavailable, "", []string{head, "not installed"}},
		{"not installed, loading off", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisNotInstalled, LoadingDisabled: true}, CategoryTtsUnavailable, "", []string{"not installed", "switched off"}},
		{"disabled", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisDisabled}, CategoryTtsUnavailable, "", []string{"disabled"}},
		{"rejected", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisRejected, Reason: "bad jar"}, CategoryTtsUnavailable, "", []string{"failed to load: bad jar"}},
		{"inert", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisInert}, CategoryTtsUnavailable, "", []string{"loaded but inactive"}},
		{"version mismatch", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisVersionMismatch}, CategoryTtsUnavailable, "", []string{"does not match"}},
		{"unknown after timeout", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisUnknown, Cause: context.DeadlineExceeded}, CategoryTtsUnavailable, head, nil},
		{"unknown after decode error", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisUnknown, Cause: &hub.DecodeError{Err: errors.New("eof")}}, CategoryTtsUnavailable, head, nil},
		{"hub address", &hub.TTSUnavailableError{Diagnosis: hub.DiagnosisHubAddress, BaseURL: "http://nope.lan"}, CategoryNetwork, "http://nope.lan is not serving the Multiroom Audio Hub API: the hub URL is wrong, or the hub's control API (REST) extension is not installed or not loaded", nil},
		{"bare not offered", &hub.TTSNotOfferedError{}, CategoryTtsUnavailable, head, nil},
		{"wrapped not offered", fmt.Errorf("speak: %w", &hub.TTSNotOfferedError{}), CategoryTtsUnavailable, head, nil},
		{"target not found", &hub.TTSError{StatusCode: 400, Code: "TARGET_NOT_FOUND", Message: "no target kitchn"}, CategoryNotFound, "", []string{"kitchn"}},
		{"provider not found", &hub.TTSError{StatusCode: 400, Code: "PROVIDER_NOT_FOUND", Message: "no provider foo"}, CategoryNotFound, "", []string{"foo"}},
		{"invalid request", &hub.TTSError{StatusCode: 400, Code: "INVALID_REQUEST", Message: "mix has no ducking"}, CategoryValidation, "", []string{"mix has no ducking"}},
		{"provider timeout", &hub.TTSError{StatusCode: 503, Code: "PROVIDER_TIMEOUT", Message: "slow cloud"}, CategoryServiceUnavailable, "", []string{"slow cloud"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolError(tt.err).Error()
			if !strings.HasPrefix(got, tt.category+": ") {
				t.Fatalf("toolError = %q, want prefix %q", got, tt.category+": ")
			}
			if tt.want != "" && got != tt.category+": "+tt.want {
				t.Errorf("toolError = %q, want %q", got, tt.category+": "+tt.want)
			}
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("toolError = %q, want it to contain %q", got, s)
				}
			}
		})
	}
}
