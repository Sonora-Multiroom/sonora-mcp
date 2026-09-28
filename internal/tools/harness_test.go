package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestSession builds an MCP server with every tool registered against
// hubURL, connects an in-memory MCP client to it and returns the client
// session.
func newTestSession(t *testing.T, hubURL string, client *http.Client) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: "test"}, nil)
	Register(server, client, hubURL)
	return connect(t, server)
}

// connect connects an in-memory MCP client to server.
func connect(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// callTool calls the named tool and fails the test on a protocol error.
func callTool(t *testing.T, s *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): protocol error: %v", name, err)
	}
	return res
}

// toolByName returns the tool as listed by ListTools.
func toolByName(t *testing.T, s *mcp.ClientSession, name string) *mcp.Tool {
	t.Helper()
	res, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not listed", name)
	return nil
}

// errorText returns the text of an error result, or "" if res is not an error.
func errorText(res *mcp.CallToolResult) string {
	if !res.IsError {
		return ""
	}
	return resultText(res)
}

// resultText concatenates the text content blocks of res.
func resultText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// expectOK fails the test if res is an error result.
func expectOK(t *testing.T, res *mcp.CallToolResult) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %s", resultText(res))
	}
}

// expectInvalidInput fails the test unless res is the SDK's input-validation
// error.
func expectInvalidInput(t *testing.T, res *mcp.CallToolResult) {
	t.Helper()
	if !strings.HasPrefix(errorText(res), `validating "arguments"`) {
		t.Fatalf("want input validation error, got isError=%v text %q", res.IsError, resultText(res))
	}
}

// expectErrorPrefix fails the test unless res is an error whose text starts
// with prefix.
func expectErrorPrefix(t *testing.T, res *mcp.CallToolResult, prefix string) {
	t.Helper()
	if got := errorText(res); !strings.HasPrefix(got, prefix) {
		t.Fatalf("want error starting %q, got isError=%v text %q", prefix, res.IsError, resultText(res))
	}
}

// expectStructured fails the test unless both the structured content and the
// JSON text copy of res equal wantJSON (compared as JSON values).
func expectStructured(t *testing.T, res *mcp.CallToolResult, wantJSON string) {
	t.Helper()
	expectOK(t, res)
	var want any
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatalf("bad wantJSON: %v", err)
	}
	gotStructured := asJSONValue(t, res.StructuredContent)
	if !jsonEqual(gotStructured, want) {
		t.Errorf("structuredContent = %s\nwant %s", mustJSON(t, gotStructured), wantJSON)
	}
	var gotText any
	if err := json.Unmarshal([]byte(resultText(res)), &gotText); err != nil {
		t.Fatalf("text content is not JSON: %v: %q", err, resultText(res))
	}
	if !jsonEqual(gotText, want) {
		t.Errorf("text content = %s\nwant %s", resultText(res), wantJSON)
	}
}

// expectAnnotations fails the test unless the listed tool carries the
// annotations for kind.
func expectAnnotations(t *testing.T, s *mcp.ClientSession, name string, kind Kind) {
	t.Helper()
	tool := toolByName(t, s, name)
	if !jsonEqual(asJSONValue(t, tool.Annotations), asJSONValue(t, kind.annotations())) {
		t.Errorf("%s annotations = %s, want %s", name, mustJSON(t, tool.Annotations), mustJSON(t, kind.annotations()))
	}
}

// expectRequest fails the test unless the hub received exactly one request
// with the given method, escaped path, raw query and (if wantBody is not
// empty) a JSON body equal to wantBody.
func expectRequest(t *testing.T, hub *fakeHub, method, path, rawQuery, wantBody string) {
	t.Helper()
	req := hub.onlyRequest(t)
	if req.Method != method || req.Path != path || req.RawQuery != rawQuery {
		t.Errorf("hub request = %s %s?%s, want %s %s?%s", req.Method, req.Path, req.RawQuery, method, path, rawQuery)
	}
	if wantBody == "" {
		if req.Body != "" {
			t.Errorf("hub request body = %q, want none", req.Body)
		}
		return
	}
	var got, want any
	if err := json.Unmarshal([]byte(req.Body), &got); err != nil {
		t.Fatalf("hub request body is not JSON: %v: %q", err, req.Body)
	}
	if err := json.Unmarshal([]byte(wantBody), &want); err != nil {
		t.Fatalf("bad wantBody: %v", err)
	}
	if !jsonEqual(got, want) {
		t.Errorf("hub request body = %s, want %s", req.Body, wantBody)
	}
}

// asJSONValue round-trips v through JSON into generic values.
func asJSONValue(t *testing.T, v any) any {
	t.Helper()
	var out any
	if err := json.Unmarshal([]byte(mustJSON(t, v)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
