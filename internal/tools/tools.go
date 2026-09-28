// Package tools defines the sonora-mcp MCP tools. Each tool wraps exactly one
// hub operation through sonora-cli's hub package.
package tools

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register adds every sonora-mcp tool to s. Tools call the hub at hubURL with
// client and the request context; nothing is cached between calls. It
// returns the number of tools registered.
func Register(s *mcp.Server, client *http.Client, hubURL string) int {
	registrars := []func(*mcp.Server, *http.Client, string) int{}
	n := 0
	for _, r := range registrars {
		n += r(s, client, hubURL)
	}
	return n
}
