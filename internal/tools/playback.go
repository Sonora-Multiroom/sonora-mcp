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

type playbackIn struct {
	URI         string  `json:"uri" jsonschema:"Stream or file URI to play, e.g. an http(s) URL"`
	TargetID    string  `json:"targetId" jsonschema:"ID of the output or group to play to"`
	TargetType  string  `json:"targetType" jsonschema:"SINGLE_OUTPUT if targetId is an output, OUTPUT_GROUP if it is a group"`
	DisplayName *string `json:"displayName,omitempty" jsonschema:"Name for the temporary input the hub creates; the hub chooses one when omitted"`
	Volume      *int    `json:"volume,omitempty" jsonschema:"Volume to set on the target before playing, an integer from 0 to 100; unchanged when omitted"`
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

	add(s, toolSpec{
		Name: "playback",
		Description: "Play a URI on an output or group in one step: the hub creates a temporary input and a route to the target. Changes state: calling it again starts another playback. " +
			"Returns {\"inputId\", \"route\", \"message\"} with the created input's ID and the new route; " +
			"fails with NotFound if the target does not exist and SourceUnreachable if the hub cannot open the URI.",
		Kind: StateChanging, Method: "POST", Path: "/api/v2/play",
	}, inputSchema[playbackIn](withMinLength("uri", 1), withMinLength("targetId", 1),
		withEnum("targetType", targetTypes...), withRange("volume", 0, 100)),
		func(ctx context.Context, in playbackIn) (*hub.PlaybackResponse, error) {
			return hub.Playback(ctx, client, hubURL, hub.PlaybackRequest{
				URI: in.URI, TargetID: in.TargetID, TargetType: in.TargetType,
				DisplayName: in.DisplayName, Volume: in.Volume,
			})
		})

	return 3
}
