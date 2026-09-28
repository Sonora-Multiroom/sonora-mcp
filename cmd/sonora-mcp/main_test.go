package main

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
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
