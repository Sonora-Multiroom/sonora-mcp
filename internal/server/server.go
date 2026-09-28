// Package server serves the sonora-mcp HTTP endpoints.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// drainTimeout bounds how long Run waits for in-flight requests on shutdown.
const drainTimeout = 6 * time.Second

// Handler returns the HTTP handler for the server: MCP Streamable HTTP at
// /mcp, stateless, sharing s across requests; the health check at /health;
// 404 for anything else. client and hubURL are the hub connection the tools
// and the health check use.
func Handler(s *mcp.Server, client *http.Client, hubURL string) http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{Stateless: true, PropagateRequestCancellation: true},
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("GET /health", healthHandler(client, hubURL))
	mux.Handle("/", http.NotFoundHandler())
	return mux
}

// Run serves srv until ctx is done, then shuts it down: new connections are
// refused and in-flight requests get up to 6 seconds to finish. It returns
// nil after a clean shutdown, or the error that stopped the server, such as
// a failure to listen.
func Run(ctx context.Context, srv *http.Server) error {
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down, draining in-flight requests")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
