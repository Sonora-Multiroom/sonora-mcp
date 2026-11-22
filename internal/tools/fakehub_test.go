package tools

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Spec-shaped hub responses with every field the hub types carry.
const (
	outputJSON = `{"outputId":"kitchen","displayName":"Kitchen","volume":40,"muted":false,"available":true,"enabled":true}`

	disabledOutputJSON = `{"outputId":"garage","displayName":"Garage","volume":0,"muted":false,"available":true,"enabled":false}`

	outputVolumeJSON = `{"outputId":"kitchen","volume":35,"updatedAt":"2026-09-28T10:05:00Z"}`

	groupVolumeJSON = `{"groupId":"downstairs","volume":35,"updatedAt":"2026-09-28T10:05:00Z"}`

	groupJSON = `{"groupId":"downstairs","displayName":"Downstairs","outputIds":["kitchen","living-room"],"muted":false,"enabled":true,"system":false}`

	inputJSON = `{"inputId":"radio-1","displayName":"Radio","uri":"http://radio.example/stream","enabled":true,"autoRemove":false,"source":"STATIC","createdAt":"2026-09-01T10:00:00Z","pauseable":false,"defaultJoinMode":null,"defaultRouteVolume":null,"defaultDuckingLevelDb":null,"origin":"CONFIGURATION","playable":true,"unplayableReason":null}`

	routeJSON = `{"routeId":"r-1","inputId":"radio-1","targetId":"kitchen","targetType":"SINGLE_OUTPUT","status":"ACTIVE","createdAt":"2026-09-28T10:00:00Z","startedAt":null,"transferable":true,"pauseable":true,"paused":false,"joinMode":"REPLACE","outputs":["kitchen"],"routeVolume":null,"duckingLevelDb":null}`

	masterMuteJSON = `{"muted":false}`

	playbackJSON = `{"inputId":"play-1","displayName":"Radio","route":` + routeJSON + `,"message":"Playback started"}`
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
	header map[string]string
}

// fakeHub is an httptest server standing in for the Multiroom Audio Hub. It
// serves canned responses per method and path and records every request.
type fakeHub struct {
	*httptest.Server

	mu       sync.Mutex
	routes   map[string]fakeRoute
	received []recordedRequest

	// cancelled receives the path of every delayed request whose context
	// was cancelled before the delay ended.
	cancelled chan string
}

func newFakeHub(t *testing.T) *fakeHub {
	t.Helper()
	h := &fakeHub{routes: map[string]fakeRoute{}, cancelled: make(chan string, 16)}
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

// handleHeaders is handle with extra response headers, such as Retry-After.
func (h *fakeHub) handleHeaders(method, path string, status int, body string, header map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.routes[method+" "+path] = fakeRoute{status: status, body: body, header: header}
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
			select {
			case h.cancelled <- path:
			default:
			}
			return
		}
	}
	if route.status == http.StatusNoContent {
		w.WriteHeader(route.status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	for k, v := range route.header {
		w.Header().Set(k, v)
	}
	w.WriteHeader(route.status)
	_, _ = io.WriteString(w, route.body)
}

// ttsDisabledInventoryJSON is an extension inventory whose TTS extension is
// disabled.
const ttsDisabledInventoryJSON = `{"loadingEnabled":true,"extensions":[{"id":"tts","status":"DISABLED","rejectionReason":null}]}`

// inventoryJSON builds an extension inventory with loadingEnabled and the
// given extension entries (a comma-separated JSON list, possibly empty).
func inventoryJSON(loadingEnabled bool, extensions string) string {
	flag := "false"
	if loadingEnabled {
		flag = "true"
	}
	return `{"loadingEnabled":` + flag + `,"extensions":[` + extensions + `]}`
}

// speakAcceptedJSON is the hub's 202 for an announcement.
const speakAcceptedJSON = `{"announcementId":"6f1c2d3e-0000-4000-8000-000000000001","cacheHit":false,"queueDepth":1}`

// voiceListJSON is the hub's voice list for piper-local; the second voice has
// no gender.
const voiceListJSON = `{"providerName":"piper-local","voices":[` +
	`{"shortName":"uk_UA-lada-x_low","fullName":"Lada (Ukrainian)","engine":"piper","language":"uk-UA","gender":"female"},` +
	`{"shortName":"en_US-amy-low","fullName":"Amy (English)","engine":"piper","language":"en-US"}]}`

// voiceListResultJSON is voiceListJSON as the tool returns it: an absent field
// comes back as null.
const voiceListResultJSON = `{"providerName":"piper-local","voices":[` +
	`{"shortName":"uk_UA-lada-x_low","fullName":"Lada (Ukrainian)","engine":"piper","language":"uk-UA","gender":"female"},` +
	`{"shortName":"en_US-amy-low","fullName":"Amy (English)","engine":"piper","language":"en-US","gender":null}]}`

// cacheStatsJSON is the hub's TTS cache statistics.
const cacheStatsJSON = `{"totalEntries":42,"totalSizeBytes":15728640,"maxSizeBytes":524288000,"entriesByProvider":{"piper-local":7}}`

// Radio fixtures. A plain 404 from the fake hub ({"type":"about:blank",...})
// is what a hub without the radio extension answers on a radio path.

// radioStationJSON is a station with every field set.
const radioStationJSON = `{"uuid":"9617a958-0601-11e8-ae97-52543be04c81","uri":"radiobrowser://9617a958-0601-11e8-ae97-52543be04c81",` +
	`"name":"Kiss FM","favicon":"https://kissfm.example/icon.png","homepage":"https://kissfm.example",` +
	`"countryCode":"UA","languages":["ukrainian"],"tags":["pop","dance"],"codec":"MP3","bitrate":128,` +
	`"hls":false,"lastCheckOk":true,"votes":1200,"clickCount":35}`

// radioStationSparseJSON is a station whose optional fields are unknown.
const radioStationSparseJSON = `{"uuid":"00000000-0000-4000-8000-000000000002","uri":"radiobrowser://00000000-0000-4000-8000-000000000002",` +
	`"name":null,"favicon":null,"homepage":null,"countryCode":null,"languages":[],"tags":[],"codec":null,"bitrate":null,` +
	`"hls":false,"lastCheckOk":false,"votes":0,"clickCount":0}`

// radioStationPageJSON is a page of two stations with a next page at offset 2.
const radioStationPageJSON = `{"stations":[` + radioStationJSON + `,` + radioStationSparseJSON + `],"nextOffset":2}`

// radioLastPageJSON is a last page: one station and no next offset.
const radioLastPageJSON = `{"stations":[` + radioStationJSON + `],"nextOffset":null}`

// radioEmptyPageJSON is a search with no match.
const radioEmptyPageJSON = `{"stations":[],"nextOffset":null}`

// radioCountriesJSON is the hub's bare array of countries, largest first.
const radioCountriesJSON = `[{"code":"US","name":"United States","stationCount":5000},{"code":"UA","name":"Ukraine","stationCount":300}]`

// radioLanguagesJSON is the hub's bare array of languages; the second has no code.
const radioLanguagesJSON = `[{"name":"english","code":"en","stationCount":9000},{"name":"pidgin","code":null,"stationCount":12}]`

// radioTagPageJSON is a page of two tags with a next page at offset 5.
const radioTagPageJSON = `{"tags":[{"name":"jazz","stationCount":800},{"name":"smooth jazz","stationCount":90}],"nextOffset":5}`

// radioProblemJSON builds a radio extension problem body of the given type
// suffix (for example "invalid-parameter"), status and detail; parameter is
// omitted when empty.
func radioProblemJSON(typeSuffix string, status int, detail, parameter string) string {
	b, _ := json.Marshal(detail)
	s := `{"type":"urn:multiroom:radio:error:` + typeSuffix + `","title":"` + typeSuffix + `","status":` + strconv.Itoa(status) + `,"detail":` + string(b)
	if parameter != "" {
		p, _ := json.Marshal(parameter)
		s += `,"parameter":` + string(p)
	}
	return s + `}`
}

// radioInventoryJSON is an extension inventory with loading on and one radio
// extension in the given status (for example "DISABLED").
func radioInventoryJSON(status string) string {
	return inventoryJSON(true, `{"id":"radio","status":"`+status+`","rejectionReason":null}`)
}
