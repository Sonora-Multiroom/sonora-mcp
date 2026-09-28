package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listOutputsIn struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty" jsonschema:"Include disabled outputs in the response (default false)"`
}

type outputIDIn struct {
	OutputID string `json:"outputId" jsonschema:"Unique output identifier, as returned by listOutputs"`
}

type outputVolumeIn struct {
	OutputID string `json:"outputId" jsonschema:"Unique output identifier, as returned by listOutputs"`
	Volume   int    `json:"volume" jsonschema:"New volume level, an integer from 0 (silent) to 100 (maximum)"`
}

type outputMuteIn struct {
	OutputID string `json:"outputId" jsonschema:"Unique output identifier, as returned by listOutputs"`
	Muted    bool   `json:"muted" jsonschema:"true to mute the output, false to unmute it"`
}

type outputEnabledIn struct {
	OutputID string `json:"outputId" jsonschema:"Unique output identifier, as returned by listOutputs"`
	Enabled  bool   `json:"enabled" jsonschema:"true to enable the output, false to disable it"`
}

// outputList wraps a list of outputs so the structured result is an object.
type outputList struct {
	Outputs []hub.Output `json:"outputs" jsonschema:"The outputs the hub returned"`
}

func registerOutputs(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "listOutputs",
		Description: "List the hub's audio outputs (speakers or zones). " +
			"Only enabled outputs are returned unless includeDisabled is true. Read-only. " +
			"Returns {\"outputs\": [...]} with each output's ID, name, volume (0-100), mute state, availability and enabled flag.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/outputs",
	}, inputSchema[listOutputsIn](),
		func(ctx context.Context, in listOutputsIn) (*outputList, error) {
			outputs, err := hub.ListOutputs(ctx, client, hubURL, in.IncludeDisabled)
			if err != nil {
				return nil, err
			}
			return &outputList{Outputs: outputs}, nil
		})

	add(s, toolSpec{
		Name: "getOutput",
		Description: "Get one audio output by ID. Read-only. " +
			"Returns the output's ID, name, volume (0-100), mute state, availability and enabled flag; fails with NotFound if there is no such output.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/outputs/{outputId}",
	}, inputSchema[outputIDIn](withMinLength("outputId", 1)),
		func(ctx context.Context, in outputIDIn) (*hub.Output, error) {
			return hub.GetOutput(ctx, client, hubURL, in.OutputID)
		})

	add(s, toolSpec{
		Name: "setOutputVolume",
		Description: "Set the volume of one audio output. Idempotent: setting the same volume again changes nothing. " +
			"Returns the output's ID, new volume (0-100) and the time it was updated; fails with NotFound if there is no such output.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/outputs/{outputId}/volume",
	}, inputSchema[outputVolumeIn](withMinLength("outputId", 1), withRange("volume", 0, 100)),
		func(ctx context.Context, in outputVolumeIn) (*hub.OutputVolume, error) {
			return hub.SetOutputVolume(ctx, client, hubURL, in.OutputID, in.Volume)
		})

	add(s, toolSpec{
		Name: "setOutputMute",
		Description: "Mute or unmute one audio output. Idempotent. " +
			"Returns the output's ID, new mute state and the time it was updated; fails with NotFound if there is no such output.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/outputs/{outputId}/mute",
	}, inputSchema[outputMuteIn](withMinLength("outputId", 1)),
		func(ctx context.Context, in outputMuteIn) (*hub.OutputMute, error) {
			return hub.SetOutputMuted(ctx, client, hubURL, in.OutputID, in.Muted)
		})

	add(s, toolSpec{
		Name: "setOutputEnabled",
		Description: "Enable or disable one audio output; a disabled output is hidden from listOutputs by default and cannot play. Idempotent. " +
			"Returns the updated output (ID, name, volume, mute state, availability, enabled flag); fails with NotFound if there is no such output.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/outputs/{outputId}/enabled",
	}, inputSchema[outputEnabledIn](withMinLength("outputId", 1)),
		func(ctx context.Context, in outputEnabledIn) (*hub.Output, error) {
			return hub.SetOutputEnabled(ctx, client, hubURL, in.OutputID, in.Enabled)
		})

	return 5
}
