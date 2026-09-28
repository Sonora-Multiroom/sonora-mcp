package config

import (
	"errors"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	tests := []struct {
		args []string
		want Config
	}{
		{[]string{"--multiroom-url", "http://hub.local:8080"}, Config{HubURL: "http://hub.local:8080", Port: 3001}},
		{[]string{"-multiroom-url", "https://hub.local/", "--port", "4000"}, Config{HubURL: "https://hub.local/", Port: 4000}},
		{[]string{"--multiroom-url=http://10.0.0.5:8080", "--port=65535"}, Config{HubURL: "http://10.0.0.5:8080", Port: 65535}},
		{[]string{"--multiroom-url", "http://h", "--port", "1"}, Config{HubURL: "http://h", Port: 1}},
	}
	for _, tt := range tests {
		got, err := Parse(tt.args)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.args, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestParseUsageErrors(t *testing.T) {
	tests := map[string][]string{
		"missing url":       {},
		"empty url":         {"--multiroom-url", ""},
		"no scheme":         {"--multiroom-url", "hub.local:8080"},
		"bad scheme":        {"--multiroom-url", "ftp://hub.local"},
		"no host":           {"--multiroom-url", "http://"},
		"port not a number": {"--multiroom-url", "http://h", "--port", "abc"},
		"port zero":         {"--multiroom-url", "http://h", "--port", "0"},
		"port too big":      {"--multiroom-url", "http://h", "--port", "65536"},
		"unknown flag":      {"--multiroom-url", "http://h", "--verbose"},
		"stray argument":    {"--multiroom-url", "http://h", "extra"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(args)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Parse(%q) error = %v, want ErrUsage", args, err)
			}
		})
	}
}

func TestParseHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"--multiroom-url", "http://h", "--help"}} {
		if _, err := Parse(args); !errors.Is(err, ErrHelp) {
			t.Errorf("Parse(%q) error = %v, want ErrHelp", args, err)
		}
	}
}

func TestUsageMentionsFlags(t *testing.T) {
	u := Usage()
	for _, want := range []string{"--multiroom-url", "--port", "3001", "--help"} {
		if !strings.Contains(u, want) {
			t.Errorf("usage missing %q:\n%s", want, u)
		}
	}
}

func TestParseHost(t *testing.T) {
	tests := []struct {
		args       []string
		host, addr string
	}{
		{nil, "", ":3001"},
		{[]string{"--host", "127.0.0.1"}, "127.0.0.1", "127.0.0.1:3001"},
		{[]string{"--host", "::1"}, "::1", "[::1]:3001"},
		{[]string{"--host", "localhost", "--port", "4000"}, "localhost", "localhost:4000"},
	}
	for _, tt := range tests {
		cfg, err := Parse(append([]string{"--multiroom-url", "http://h"}, tt.args...))
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.args, err)
			continue
		}
		if cfg.Host != tt.host {
			t.Errorf("Parse(%q).Host = %q, want %q", tt.args, cfg.Host, tt.host)
		}
		if got := cfg.ListenAddr(); got != tt.addr {
			t.Errorf("Parse(%q).ListenAddr() = %q, want %q", tt.args, got, tt.addr)
		}
	}
}

func TestParseRejectsBadHost(t *testing.T) {
	for _, host := range []string{"not a host!", "hub.local", "127.0.0.1:80"} {
		if _, err := Parse([]string{"--multiroom-url", "http://h", "--host", host}); !errors.Is(err, ErrUsage) {
			t.Errorf("--host %q: error = %v, want ErrUsage", host, err)
		}
	}
}

func TestUsageMentionsHost(t *testing.T) {
	if !strings.Contains(Usage(), "--host") {
		t.Errorf("usage missing --host:\n%s", Usage())
	}
}
