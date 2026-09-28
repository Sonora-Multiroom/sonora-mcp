// Package config parses and validates the sonora-mcp command line.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
)

// DefaultPort is the port used when --port is not given.
const DefaultPort = 3001

// ErrHelp is returned by Parse when -h or --help was given.
var ErrHelp = flag.ErrHelp

// ErrUsage is wrapped by every error Parse returns for a missing or invalid
// flag; the caller prints the usage text with it.
var ErrUsage = errors.New("usage error")

// Config is the validated server configuration.
type Config struct {
	// HubURL is the base URL of the Multiroom Audio Hub.
	HubURL string
	// Port is the TCP port the server listens on.
	Port int
	// Host is the address to listen on: an IP literal or "localhost". Empty
	// means all addresses.
	Host string
}

// ListenAddr returns the host:port address to listen on.
func (c Config) ListenAddr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Usage returns the command-line usage text.
func Usage() string {
	return fmt.Sprintf(`Usage:
  sonora-mcp --multiroom-url <url> [--port <port>] [--host <address>]
  sonora-mcp -h | --help

Options:
  --multiroom-url <url>  Base URL of the Multiroom Audio Hub, http or https (required)
  --port <port>          Port to listen on, 1-65535 (default %d)
  --host <address>       Address to listen on: an IP such as 127.0.0.1 or ::1,
                         or localhost (default: all addresses)
  -h, --help             Show this help
`, DefaultPort)
}

// Parse parses and validates args (without the program name).
func Parse(args []string) (Config, error) {
	fs := flag.NewFlagSet("sonora-mcp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	var cfg Config
	fs.StringVar(&cfg.HubURL, "multiroom-url", "", "")
	fs.IntVar(&cfg.Port, "port", DefaultPort, "")
	fs.StringVar(&cfg.Host, "host", "", "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, ErrHelp
		}
		return Config{}, usageErr("%v", err)
	}
	if fs.NArg() > 0 {
		return Config{}, usageErr("unexpected argument %q", fs.Arg(0))
	}
	if err := validateHubURL(cfg.HubURL); err != nil {
		return Config{}, err
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, usageErr("--port must be between 1 and 65535, got %d", cfg.Port)
	}
	if cfg.Host != "" && cfg.Host != "localhost" && net.ParseIP(cfg.Host) == nil {
		return Config{}, usageErr("--host must be an IP address or localhost, got %q", cfg.Host)
	}
	return cfg, nil
}

func validateHubURL(raw string) error {
	if raw == "" {
		return usageErr("--multiroom-url is required")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return usageErr("--multiroom-url must be an http or https URL with a host, got %q", raw)
	}
	return nil
}

// usageError is a flag error; it matches ErrUsage with errors.Is.
type usageError string

func (e usageError) Error() string { return string(e) }

func (e usageError) Is(target error) bool { return target == ErrUsage }

func usageErr(format string, args ...any) error {
	return usageError(fmt.Sprintf(format, args...))
}
