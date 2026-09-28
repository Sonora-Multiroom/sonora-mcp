package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Sonora-Multiroom/sonora-mcp/internal/config"
	"github.com/Sonora-Multiroom/sonora-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runWith(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRunHelp(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		code, stdout, stderr := runWith(t, flag)
		if code != 0 {
			t.Errorf("%s: exit %d, want 0", flag, code)
		}
		if !strings.Contains(stdout, "--multiroom-url") {
			t.Errorf("%s: usage not on stdout: %q", flag, stdout)
		}
		if stderr != "" {
			t.Errorf("%s: unexpected stderr %q", flag, stderr)
		}
	}
}

func TestRunMissingURL(t *testing.T) {
	code, stdout, stderr := runWith(t)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "--multiroom-url is required") || !strings.Contains(stderr, "Usage") {
		t.Errorf("want message and usage on stderr, got %q", stderr)
	}
	if stdout != "" {
		t.Errorf("unexpected stdout %q", stdout)
	}
}

func TestRunInvalidPort(t *testing.T) {
	code, _, stderr := runWith(t, "--multiroom-url", "http://hub", "--port", "70000")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "--port must be between 1 and 65535") || !strings.Contains(stderr, "Usage") {
		t.Errorf("want message and usage on stderr, got %q", stderr)
	}
}

func TestRunPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	code, _, stderr := runWith(t, "--multiroom-url", "http://hub", "--port", port)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if stderr == "" || strings.Contains(stderr, "Usage") {
		t.Errorf("want an error without usage on stderr, got %q", stderr)
	}
}

func TestRunInvalidHost(t *testing.T) {
	code, _, stderr := runWith(t, "--multiroom-url", "http://hub", "--host", "not a host!")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr, "--host") || !strings.Contains(stderr, "Usage") {
		t.Errorf("want message and usage on stderr, got %q", stderr)
	}
}

// setVersion sets version.Version for the test.
func setVersion(t *testing.T, v string) {
	t.Helper()
	old := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = old })
}

func TestNewAppUsesOneVersionAndBoundedClient(t *testing.T) {
	setVersion(t, "9.9.9-test")
	a := newApp(config.Config{HubURL: "http://127.0.0.1:1", Port: 3001}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if a.client.Timeout <= 0 || a.client.Timeout > 5*time.Second {
		t.Errorf("hub client timeout = %v, want 0 < t <= 5s", a.client.Timeout)
	}
	if a.tools != 24 {
		t.Errorf("tools registered = %d, want 24", a.tools)
	}

	ts := httptest.NewServer(a.server.Handler)
	defer ts.Close()

	mc := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := mc.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()
	if got := cs.InitializeResult().ServerInfo.Version; got != "9.9.9-test" {
		t.Errorf("MCP server version = %q, want 9.9.9-test", got)
	}
	if got := healthVersion(t, ts.URL); got != "9.9.9-test" {
		t.Errorf("/health version = %q, want 9.9.9-test", got)
	}
}

func TestRunStopsCleanlyOnSignal(t *testing.T) {
	setVersion(t, "9.9.9-test")
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var stdout, stderr bytes.Buffer
	exit := make(chan int, 1)
	go func() {
		exit <- run(ctx, []string{"--multiroom-url", "http://127.0.0.1:1", "--host", "127.0.0.1", "--port", port}, &stdout, &stderr)
	}()

	base := "http://127.0.0.1:" + port
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server did not serve /health: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := healthVersion(t, base); got != "9.9.9-test" {
		t.Errorf("/health version = %q, want 9.9.9-test", got)
	}

	cancel() // stands in for SIGTERM / Ctrl+C
	select {
	case code := <-exit:
		if code != 0 {
			t.Errorf("exit %d after stop signal, want 0; stderr:\n%s", code, stderr.String())
		}
	case <-time.After(6 * time.Second):
		t.Fatal("run did not return within 6s of the stop signal")
	}
	if !strings.Contains(stderr.String(), "version=9.9.9-test") {
		t.Errorf("startup log lacks the version:\n%s", stderr.String())
	}
}

// healthVersion returns the version reported by GET base/health.
func healthVersion(t *testing.T, base string) string {
	t.Helper()
	resp, err := http.Get(base + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("health body: %v", err)
	}
	return body.Version
}
