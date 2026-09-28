package tools

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Spec-shaped hub responses with every field the hub types carry.
const (
	outputJSON = `{"outputId":"kitchen","displayName":"Kitchen","volume":40,"muted":false,"available":true,"enabled":true}`

	disabledOutputJSON = `{"outputId":"garage","displayName":"Garage","volume":0,"muted":false,"available":true,"enabled":false}`

	groupJSON = `{"groupId":"downstairs","displayName":"Downstairs","outputIds":["kitchen","living-room"],"muted":false,"enabled":true}`

	inputJSON = `{"inputId":"radio-1","displayName":"Radio","uri":"http://radio.example/stream","enabled":true,"autoRemove":false,"source":"STATIC","createdAt":"2026-09-01T10:00:00Z","pauseable":false}`

	routeJSON = `{"routeId":"r-1","inputId":"radio-1","targetId":"kitchen","targetType":"SINGLE_OUTPUT","status":"ACTIVE","createdAt":"2026-09-28T10:00:00Z","startedAt":null,"transferable":true,"pauseable":true,"paused":false}`

	masterMuteJSON = `{"muted":false}`

	playbackJSON = `{"inputId":"play-1","route":` + routeJSON + `,"message":"Playback started"}`
)

// recordedRequest is one request the fake hub received.
type recordedRequest struct {
	Method   string
	Path     string // escaped path, as sent on the wire
	RawQuery string
	Body     string
}

type fakeRoute struct {
	status int
	body   string
	delay  time.Duration
}

// fakeHub is an httptest server standing in for the Multiroom Audio Hub. It
// serves canned responses per method and path and records every request.
type fakeHub struct {
	*httptest.Server

	mu       sync.Mutex
	routes   map[string]fakeRoute
	received []recordedRequest
}

func newFakeHub(t *testing.T) *fakeHub {
	t.Helper()
	h := &fakeHub{routes: map[string]fakeRoute{}}
	h.Server = httptest.NewServer(http.HandlerFunc(h.serve))
	t.Cleanup(h.Close)
	return h
}

// handle registers a canned response for method and escaped path. A path
// with "?query" matches only requests with exactly that raw query, and takes
// precedence over the same path without one.
func (h *fakeHub) handle(method, path string, status int, body string) {
	h.handleDelayed(method, path, status, body, 0)
}

// handleDelayed is handle with a delay before the response, to simulate a
// slow hub. The delay ends early if the request is cancelled.
func (h *fakeHub) handleDelayed(method, path string, status int, body string, delay time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.routes[method+" "+path] = fakeRoute{status: status, body: body, delay: delay}
}

// requests returns a copy of every request received so far.
func (h *fakeHub) requests() []recordedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]recordedRequest(nil), h.received...)
}

// onlyRequest fails the test unless exactly one request was received, and
// returns it.
func (h *fakeHub) onlyRequest(t *testing.T) recordedRequest {
	t.Helper()
	reqs := h.requests()
	if len(reqs) != 1 {
		t.Fatalf("hub received %d requests, want 1: %+v", len(reqs), reqs)
	}
	return reqs[0]
}

// assertNoRequests fails the test if the hub received any request.
func (h *fakeHub) assertNoRequests(t *testing.T) {
	t.Helper()
	if reqs := h.requests(); len(reqs) != 0 {
		t.Fatalf("hub received %d requests, want none: %+v", len(reqs), reqs)
	}
}

func (h *fakeHub) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	path := r.URL.EscapedPath()

	h.mu.Lock()
	h.received = append(h.received, recordedRequest{
		Method:   r.Method,
		Path:     path,
		RawQuery: r.URL.RawQuery,
		Body:     string(body),
	})
	route, ok := h.routes[r.Method+" "+path+"?"+r.URL.RawQuery]
	if !ok {
		route, ok = h.routes[r.Method+" "+path]
	}
	h.mu.Unlock()

	if !ok {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"type":"about:blank","title":"Not Found","status":404,"detail":"no such resource"}`)
		return
	}
	if route.delay > 0 {
		select {
		case <-time.After(route.delay):
		case <-r.Context().Done():
			return
		}
	}
	if route.status == http.StatusNoContent {
		w.WriteHeader(route.status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(route.status)
	_, _ = io.WriteString(w, route.body)
}
