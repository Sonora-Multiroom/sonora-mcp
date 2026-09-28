package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// noInput is the input of tools that take no arguments.
type noInput struct{}

type masterMuteIn struct {
	Muted bool `json:"muted" jsonschema:"true to silence every output at once, false to lift the master mute"`
}

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

	add(s, toolSpec{
		Name: "setMasterMute",
		Description: "Turn the hub's master mute on or off. When on, every output is silent regardless of its own mute setting; turning it off restores each output's own state. Idempotent. " +
			"Returns {\"muted\": true|false}.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/master-mute",
	}, inputSchema[masterMuteIn](),
		func(ctx context.Context, in masterMuteIn) (*hub.MasterMute, error) {
			return hub.SetMasterMute(ctx, client, hubURL, in.Muted)
		})

	return 2
}
