package tools

import (
	"context"
	"testing"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type echoInput struct {
	Name   string `json:"name" jsonschema:"Name to echo"`
	Volume int    `json:"volume" jsonschema:"A volume"`
	Panic  bool   `json:"panic,omitempty" jsonschema:"Panic instead of answering"`
	Fail   bool   `json:"fail,omitempty" jsonschema:"Return a hub error"`
}

type echoOutput struct {
	Echo string `json:"echo"`
}

func newEchoSession(t *testing.T, kind Kind, name string) (*mcp.ClientSession, *int) {
	t.Helper()
	calls := new(int)
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	add(server,
		toolSpec{Name: name, Description: "Echoes its name.", Kind: kind, Method: "GET", Path: "/echo"},
		inputSchema[echoInput](withRange("volume", 0, 100), withMinLength("name", 1)),
		func(_ context.Context, in echoInput) (*echoOutput, error) {
			*calls++
			if in.Panic {
				panic("boom")
			}
			if in.Fail {
				return nil, &hub.StatusError{StatusCode: 500}
			}
			return &echoOutput{Echo: in.Name}, nil
		})
	return connect(t, server), calls
}

func TestAddListsTool(t *testing.T) {
	s, _ := newEchoSession(t, ReadOnly, "echoTool")
	tool := toolByName(t, s, "echoTool")
	if tool.Description != "Echoes its name." {
		t.Errorf("description = %q", tool.Description)
	}
	in := asJSONValue(t, tool.InputSchema).(map[string]any)
	vol := in["properties"].(map[string]any)["volume"].(map[string]any)
	if vol["minimum"] != 0.0 || vol["maximum"] != 100.0 {
		t.Errorf("volume schema = %v, want 0..100", vol)
	}
	out := asJSONValue(t, tool.OutputSchema).(map[string]any)
	if _, ok := out["properties"].(map[string]any)["echo"]; !ok {
		t.Errorf("output schema = %v, want property echo", out)
	}

	found := false
	for _, spec := range registeredSpecs() {
		if spec.Name == "echoTool" {
			found = spec.Method == "GET" && spec.Path == "/echo" && spec.Kind == ReadOnly
		}
	}
	if !found {
		t.Errorf("registry has no matching toolSpec for echoTool: %+v", registeredSpecs())
	}
}

func TestAddAnnotationsPerKind(t *testing.T) {
	f, tr := false, true
	tests := []struct {
		kind Kind
		want mcp.ToolAnnotations
	}{
		{ReadOnly, mcp.ToolAnnotations{ReadOnlyHint: true}},
		{Idempotent, mcp.ToolAnnotations{DestructiveHint: &f, IdempotentHint: true}},
		{StateChanging, mcp.ToolAnnotations{DestructiveHint: &f}},
		{Destructive, mcp.ToolAnnotations{DestructiveHint: &tr}},
	}
	for _, tt := range tests {
		s, _ := newEchoSession(t, tt.kind, "echoKind")
		got := toolByName(t, s, "echoKind").Annotations
		if !jsonEqual(asJSONValue(t, got), asJSONValue(t, tt.want)) {
			t.Errorf("kind %d annotations = %s, want %s", tt.kind, mustJSON(t, got), mustJSON(t, tt.want))
		}
	}
}

func TestAddSuccessIsStructured(t *testing.T) {
	s, _ := newEchoSession(t, ReadOnly, "echoTool")
	res := callTool(t, s, "echoTool", map[string]any{"name": "hi", "volume": 3})
	expectStructured(t, res, `{"echo":"hi"}`)
}

func TestAddInvalidArgumentsSkipHandler(t *testing.T) {
	s, calls := newEchoSession(t, ReadOnly, "echoTool")
	for _, args := range []map[string]any{
		{"name": "hi", "volume": 150},
		{"name": "", "volume": 1},
		{"volume": 1},
	} {
		res := callTool(t, s, "echoTool", args)
		expectInvalidInput(t, res)
	}
	if *calls != 0 {
		t.Errorf("handler ran %d times for invalid input", *calls)
	}
}

func TestAddHubErrorIsCategorized(t *testing.T) {
	s, _ := newEchoSession(t, ReadOnly, "echoTool")
	res := callTool(t, s, "echoTool", map[string]any{"name": "hi", "volume": 1, "fail": true})
	expectErrorPrefix(t, res, CategoryHubError+": ")
}

func TestAddRecoversPanics(t *testing.T) {
	s, _ := newEchoSession(t, ReadOnly, "echoTool")
	res := callTool(t, s, "echoTool", map[string]any{"name": "hi", "volume": 1, "panic": true})
	if got := errorText(res); got != "Internal: unexpected server error" {
		t.Fatalf("panic result = %q", got)
	}
	res = callTool(t, s, "echoTool", map[string]any{"name": "again", "volume": 1})
	expectStructured(t, res, `{"echo":"again"}`)
}

func TestToolErrorNotProtocolError(t *testing.T) {
	// A categorized tool error must travel as a CallToolResult with isError,
	// never as a JSON-RPC error.
	s, _ := newEchoSession(t, ReadOnly, "echoTool")
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: "echoTool", Arguments: map[string]any{"name": "x", "volume": 1, "fail": true}})
	if err != nil {
		t.Fatalf("hub error surfaced as protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatal("want isError result")
	}
}
