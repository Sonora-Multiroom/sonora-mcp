package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listInputsIn struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty" jsonschema:"Include disabled inputs in the response (default false)"`
}

type inputIDIn struct {
	InputID string `json:"inputId" jsonschema:"Unique input identifier, as returned by listInputs"`
}

type inputEnabledIn struct {
	InputID string `json:"inputId" jsonschema:"Unique input identifier, as returned by listInputs"`
	Enabled bool   `json:"enabled" jsonschema:"true to enable the input, false to disable it"`
}

// inputList wraps a list of inputs so the structured result is an object.
type inputList struct {
	Inputs []hub.Input `json:"inputs" jsonschema:"The inputs the hub returned"`
}

func registerInputs(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "listInputs",
		Description: "List the hub's audio inputs (sources such as radio streams or line-ins). " +
			"Only enabled inputs are returned unless includeDisabled is true. Read-only. " +
			"Returns {\"inputs\": [...]} with each input's ID, name, URI, enabled and auto-remove flags, source (STATIC or EPHEMERAL), creation time and whether it can be paused.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/inputs",
	}, inputSchema[listInputsIn](),
		func(ctx context.Context, in listInputsIn) (*inputList, error) {
			inputs, err := hub.ListInputs(ctx, client, hubURL, in.IncludeDisabled)
			if err != nil {
				return nil, err
			}
			return &inputList{Inputs: inputs}, nil
		})

	add(s, toolSpec{
		Name: "getInput",
		Description: "Get one audio input by ID. Read-only. " +
			"Returns the input's ID, name, URI, enabled and auto-remove flags, source, creation time and whether it can be paused; fails with NotFound if there is no such input.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/inputs/{inputId}",
	}, inputSchema[inputIDIn](withMinLength("inputId", 1)),
		func(ctx context.Context, in inputIDIn) (*hub.Input, error) {
			return hub.GetInput(ctx, client, hubURL, in.InputID)
		})

	add(s, toolSpec{
		Name: "setInputEnabled",
		Description: "Enable or disable one audio input; a disabled input is hidden from listInputs by default and cannot be played. Idempotent. " +
			"Returns the updated input (ID, name, URI, enabled and auto-remove flags, source, creation time, pauseable); fails with NotFound if there is no such input.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/inputs/{inputId}/enabled",
	}, inputSchema[inputEnabledIn](withMinLength("inputId", 1)),
		func(ctx context.Context, in inputEnabledIn) (*hub.Input, error) {
			return hub.SetInputEnabled(ctx, client, hubURL, in.InputID, in.Enabled)
		})

	return 3
}
