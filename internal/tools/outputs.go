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

	return 2
}
