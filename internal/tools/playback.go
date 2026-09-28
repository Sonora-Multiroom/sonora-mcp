package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// noInput is the input of tools that take no arguments.
type noInput struct{}

func registerPlayback(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "getMasterMute",
		Description: "Get the hub's master mute state, which silences every output at once regardless of their own mute settings. Read-only. " +
			"Returns {\"muted\": true|false}.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/master-mute",
	}, inputSchema[noInput](),
		func(ctx context.Context, _ noInput) (*hub.MasterMute, error) {
			return hub.GetMasterMute(ctx, client, hubURL)
		})

	return 1
}
