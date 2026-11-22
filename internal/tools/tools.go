// Package tools defines the sonora-mcp MCP tools. Each tool wraps exactly one
// hub operation through sonora-cli's hub package.
package tools

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register adds every sonora-mcp tool to s. Tools call the hub at hubURL with
// client and the request context; nothing is cached between calls. speakClient
// serves speak, whose bound is longer than the other tools'. radioClient
// serves the radio tools, whose bound is longer than the standard one. It
// returns the number of tools registered.
func Register(s *mcp.Server, client, speakClient, radioClient *http.Client, hubURL string) int {
	registrars := []func(*mcp.Server, *http.Client, string) int{
		registerInputs,
		registerOutputs,
		registerGroups,
		registerRoutes,
		registerPlayback,
		registerChimes,
	}
	n := 0
	for _, r := range registrars {
		n += r(s, client, hubURL)
	}
	n += registerTTS(s, client, speakClient, hubURL)
	n += registerRadio(s, client, radioClient, hubURL)
	return n
}
