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
		{"other api error", &hub.APIError{StatusCode: 409, Detail: "input already exists"}, CategoryHubError, []string{"409", "input already exists"}},
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

func TestToolErrorKeepsCause(t *testing.T) {
	cause := &hub.StatusError{StatusCode: 500}
	var se *hub.StatusError
	if !errors.As(toolError(cause), &se) {
		t.Fatal("toolError must wrap the hub error")
	}
}
