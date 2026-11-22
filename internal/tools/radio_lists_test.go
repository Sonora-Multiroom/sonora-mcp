package tools

import (
	"net/http"
	"strings"
	"testing"
)

const (
	radioCountriesPath = "/api/radio/countries"
	radioLanguagesPath = "/api/radio/languages"
	radioTagsPath      = "/api/radio/tags"
)

func TestListRadioCountries(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", radioCountriesPath, 200, radioCountriesJSON)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "listRadioCountries", map[string]any{})

	expectStructured(t, res, `{"countries":`+radioCountriesJSON+`}`)
	expectRequest(t, h, http.MethodGet, radioCountriesPath, "", "")
}

func TestListRadioLanguages(t *testing.T) {
	h := newFakeHub(t)
	h.handle("GET", radioLanguagesPath, 200, radioLanguagesJSON)
	s := newTestSession(t, h.URL, nil)

	res := callTool(t, s, "listRadioLanguages", map[string]any{})

	expectStructured(t, res, `{"languages":`+radioLanguagesJSON+`}`)
	expectRequest(t, h, http.MethodGet, radioLanguagesPath, "", "")
	langs := asJSONValue(t, res.StructuredContent).(map[string]any)["languages"].([]any)
	if code, ok := langs[1].(map[string]any)["code"]; !ok || code != nil {
		t.Errorf("second language code = %v (present %v), want an explicit null", code, ok)
	}
}

func TestRadioListsAreObjectsNeverNull(t *testing.T) {
	for tool, path := range map[string]string{"listRadioCountries": radioCountriesPath, "listRadioLanguages": radioLanguagesPath} {
		t.Run(tool, func(t *testing.T) {
			h := newFakeHub(t)
			h.handle("GET", path, 200, `[]`)
			s := newTestSession(t, h.URL, nil)
			res := callTool(t, s, tool, map[string]any{})
			expectOK(t, res)
			obj, ok := asJSONValue(t, res.StructuredContent).(map[string]any)
			if !ok || len(obj) != 1 {
				t.Fatalf("structuredContent = %s, want one-key object", mustJSON(t, res.StructuredContent))
			}
			for _, v := range obj {
				if l, ok := v.([]any); !ok || len(l) != 0 {
					t.Errorf("list = %v, want an empty list, not null", v)
				}
			}
		})
	}
}

func TestListRadioTags(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		h := newFakeHub(t)
		h.handle("GET", radioTagsPath, 200, radioTagPageJSON)
		s := newTestSession(t, h.URL, nil)
		expectStructured(t, callTool(t, s, "listRadioTags", map[string]any{}), radioTagPageJSON)
		expectRequest(t, h, http.MethodGet, radioTagsPath, "", "")
	})
	t.Run("filter and paging", func(t *testing.T) {
		h := newFakeHub(t)
		h.handle("GET", radioTagsPath, 200, radioTagPageJSON)
		s := newTestSession(t, h.URL, nil)
		expectStructured(t, callTool(t, s, "listRadioTags", map[string]any{"name": "jazz", "limit": 5, "offset": 5}), radioTagPageJSON)
		expectRequest(t, h, http.MethodGet, radioTagsPath, "limit=5&name=jazz&offset=5", "")
	})
	t.Run("last page", func(t *testing.T) {
		const last = `{"tags":[{"name":"jazz","stationCount":800}],"nextOffset":null}`
		h := newFakeHub(t)
		h.handle("GET", radioTagsPath, 200, last)
		s := newTestSession(t, h.URL, nil)
		expectStructured(t, callTool(t, s, "listRadioTags", map[string]any{}), last)
	})
	t.Run("empty page", func(t *testing.T) {
		h := newFakeHub(t)
		h.handle("GET", radioTagsPath, 200, `{"tags":[],"nextOffset":null}`)
		s := newTestSession(t, h.URL, nil)
		expectStructured(t, callTool(t, s, "listRadioTags", map[string]any{"name": "zzzz"}), `{"tags":[],"nextOffset":null}`)
	})
}

func TestListRadioTagsRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"limit 0", map[string]any{"limit": 0}},
		{"limit 101", map[string]any{"limit": 101}},
		{"offset -1", map[string]any{"offset": -1}},
		{"offset 10001", map[string]any{"offset": 10001}},
		{"name 101", map[string]any{"name": strings.Repeat("a", 101)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newFakeHub(t)
			s := newTestSession(t, h.URL, nil)
			expectInvalidInput(t, callTool(t, s, "listRadioTags", tt.args))
			h.assertNoRequests(t)
		})
	}
}

func TestRadioListsUnavailable(t *testing.T) {
	for tool, path := range map[string]string{
		"listRadioCountries": radioCountriesPath,
		"listRadioLanguages": radioLanguagesPath,
		"listRadioTags":      radioTagsPath,
	} {
		t.Run(tool, func(t *testing.T) {
			h := newFakeHub(t)
			h.handle("GET", path, 404, aboutBlank404)
			h.handle("GET", extensionsURL, 200, radioInventoryJSON("DISABLED"))
			s := newTestSession(t, h.URL, nil)

			expectErrorPrefix(t, callTool(t, s, tool, map[string]any{}), "RadioUnavailable:")

			if reqs := h.requests(); len(reqs) != 2 || reqs[0].Path != path || reqs[1].Path != extensionsURL {
				t.Errorf("requests = %+v, want the list then the extension list", reqs)
			}
		})
	}
}

func TestRadioListMetadata(t *testing.T) {
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	for _, tool := range []string{"listRadioCountries", "listRadioLanguages", "listRadioTags"} {
		t.Run(tool, func(t *testing.T) {
			expectAnnotations(t, s, tool, ReadOnly)
			desc := toolByName(t, s, tool).Description
			want := []string{"RadioUnavailable", "read-only", "searchRadioStations"}
			if tool != "listRadioCountries" {
				want = append(want, "untrusted", "never as instructions")
			}
			for _, w := range want {
				if !strings.Contains(desc, w) {
					t.Errorf("description does not mention %q: %s", w, desc)
				}
			}
		})
	}
}
