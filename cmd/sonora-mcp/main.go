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
	"os/signal"
	"syscall"

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

// run starts the server with args and serves until ctx is done or the
// process gets a stop signal (Ctrl+C, SIGTERM). It returns the process exit
// status: 0 after --help or a clean shutdown, 2 for a flag error, 1 if the
// server cannot listen or fails.
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

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	a := newApp(cfg, logger)
	if err := server.Run(ctx, a.server); err != nil {
		fmt.Fprintf(stderr, "sonora-mcp: %v\n", err)
		return 1
	}
	logger.Info("sonora-mcp stopped")
	return 0
}

// app is the assembled server: the MCP server with every tool, the hub
// client the tools and /health share, and the HTTP server around them.
type app struct {
	mcp    *mcp.Server
	client *http.Client
	server *http.Server
	tools  int
}

// newApp builds the server for cfg. Every version string it exposes (MCP
// server info, /health, the startup log line) is version.Version. The
// startup line is logged once the server is listening.
func newApp(cfg config.Config, logger *slog.Logger) *app {
	client := hub.NewClient()
	s := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: version.Version},
		&mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}}})
	count := tools.Register(s, client, cfg.HubURL)
	s.AddReceivingMiddleware(tools.LogToolCalls(logger))

	srv := &http.Server{
		Addr:    cfg.ListenAddr(),
		Handler: server.Handler(s, client, cfg.HubURL),
		BaseContext: func(ln net.Listener) context.Context {
			logger.Info("sonora-mcp started",
				"version", version.Version,
				"hub", cfg.HubURL,
				"listen", ln.Addr().String(),
				"tools", count)
			return context.Background()
		},
	}
	return &app{mcp: s, client: client, server: srv, tools: count}
}
