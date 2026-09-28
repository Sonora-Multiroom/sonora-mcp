package tools

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// syncBuffer is a bytes.Buffer safe for concurrent writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(s.b.String()), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

var errNotFoundForTest = &hub.NotFoundError{Resource: "output", ID: "a"}

func TestLogToolCalls(t *testing.T) {
	var buf syncBuffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	add(server,
		toolSpec{Name: "echoLog", Description: "Echo.", Kind: ReadOnly, Method: "GET", Path: "/echo"},
		inputSchema[echoInput](withRange("volume", 0, 100)),
		func(_ context.Context, in echoInput) (*echoOutput, error) {
			if in.Fail {
				return nil, errNotFoundForTest
			}
			return &echoOutput{Echo: in.Name}, nil
		})
	server.AddReceivingMiddleware(LogToolCalls(logger))
	s := connect(t, server)

	if _, err := s.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if n := len(buf.lines()); n != 0 {
		t.Fatalf("tools/list logged %d lines, want 0: %v", n, buf.lines())
	}

	cases := []struct {
		args    map[string]any
		outcome string
	}{
		{map[string]any{"name": "a", "volume": 1}, "outcome=ok"},
		{map[string]any{"name": "a", "volume": 1, "fail": true}, "outcome=NotFound"},
		{map[string]any{"name": "a", "volume": 500}, "outcome=InvalidInput"},
	}
	for i, c := range cases {
		callTool(t, s, "echoLog", c.args)
		lines := buf.lines()
		if len(lines) != i+1 {
			t.Fatalf("after call %d: %d log lines, want %d: %v", i+1, len(lines), i+1, lines)
		}
		line := lines[i]
		for _, want := range []string{"tool=echoLog", "args=", `\"name\":\"a\"`, c.outcome, "duration="} {
			if !strings.Contains(line, want) {
				t.Errorf("log line %q missing %q", line, want)
			}
		}
	}
}
