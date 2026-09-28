// Package server serves the sonora-mcp HTTP endpoints.
package server

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler returns the HTTP handler for the server: MCP Streamable HTTP at
// /mcp, stateless, sharing s across requests; 404 for anything else. client
// and hubURL are the hub connection the tools use.
func Handler(s *mcp.Server, client *http.Client, hubURL string) http.Handler {
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{Stateless: true, PropagateRequestCancellation: true},
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/", http.NotFoundHandler())
	return mux
}
