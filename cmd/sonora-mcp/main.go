// Command sonora-mcp is an MCP server that lets AI agents control a
// Multiroom Audio Hub.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tiger-seo/sonora-mcp/internal/config"
	"github.com/tiger-seo/sonora-mcp/internal/server"
	"github.com/tiger-seo/sonora-mcp/internal/tools"
	"github.com/tiger-seo/sonora-mcp/internal/version"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// run starts the server with args and returns the process exit status: 0
// after --help, 2 for a flag error, 1 if the server cannot listen.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cfg, err := config.Parse(args)
	if errors.Is(err, config.ErrHelp) {
		fmt.Fprint(stdout, config.Usage())
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "sonora-mcp: %v\n\n%s", err, config.Usage())
		return 2
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	slog.SetDefault(logger)

	client := hub.NewClient()
	s := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: version.Version},
		&mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}}})
	count := tools.Register(s, client, cfg.HubURL)
	s.AddReceivingMiddleware(tools.LogToolCalls(logger))

	addr := net.JoinHostPort("", strconv.Itoa(cfg.Port))
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		fmt.Fprintf(stderr, "sonora-mcp: cannot listen on %s: %v\n", addr, err)
		return 1
	}
	logger.Info("sonora-mcp started",
		"version", version.Version,
		"hub", cfg.HubURL,
		"listen", ln.Addr().String(),
		"tools", count)

	srv := &http.Server{Handler: server.Handler(s, client, cfg.HubURL)}
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "sonora-mcp: %v\n", err)
		return 1
	}
	return 0
}
