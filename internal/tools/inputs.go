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

type createInputIn struct {
	InputID     string `json:"inputId" jsonschema:"ID for the new input: 1-255 letters, digits, hyphens or underscores"`
	DisplayName string `json:"displayName" jsonschema:"Human-readable name for the input"`
	URI         string `json:"uri" jsonschema:"Stream or file URI the input plays, e.g. an http(s) URL"`
	Enabled     *bool  `json:"enabled,omitempty" jsonschema:"Whether the input starts enabled; the hub's default applies when omitted"`
	AutoRemove  *bool  `json:"autoRemove,omitempty" jsonschema:"Whether the hub removes the input once its routes end; the hub's default applies when omitted"`
}

// inputDeleted confirms a deleted input; the hub answers 204 with no body.
type inputDeleted struct {
	Deleted bool   `json:"deleted" jsonschema:"Always true"`
	InputID string `json:"inputId" jsonschema:"ID of the deleted input"`
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
			"Returns {\"inputs\": [...]} with each input's ID, name, URI, enabled and auto-remove flags, source (STATIC or EPHEMERAL), creation time, whether it can be paused and its default join mode (null if none).",
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
			"Returns the input's ID, name, URI, enabled and auto-remove flags, source, creation time, whether it can be paused and its default join mode (null if none); fails with NotFound if there is no such input.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/inputs/{inputId}",
	}, inputSchema[inputIDIn](withMinLength("inputId", 1)),
		func(ctx context.Context, in inputIDIn) (*hub.Input, error) {
			return hub.GetInput(ctx, client, hubURL, in.InputID)
		})

	add(s, toolSpec{
		Name: "setInputEnabled",
		Description: "Enable or disable one audio input; a disabled input is hidden from listInputs by default and cannot be played. Idempotent. " +
			"Returns the updated input (ID, name, URI, enabled and auto-remove flags, source, creation time, pauseable, default join mode); fails with NotFound if there is no such input.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/inputs/{inputId}/enabled",
	}, inputSchema[inputEnabledIn](withMinLength("inputId", 1)),
		func(ctx context.Context, in inputEnabledIn) (*hub.Input, error) {
			return hub.SetInputEnabled(ctx, client, hubURL, in.InputID, in.Enabled)
		})

	add(s, toolSpec{
		Name: "createInput",
		Description: "Register a new ephemeral audio input (a stream or file URI) on the hub so it can be routed to outputs. Changes state. " +
			"Returns the new input (ID, name, URI, enabled and auto-remove flags, source, creation time, pauseable, default join mode); fails with Conflict if the ID is taken and Validation if the request is invalid.",
		Kind: StateChanging, Method: "POST", Path: "/api/v2/inputs",
	}, inputSchema[createInputIn](
		withMinLength("inputId", 1), withPattern("inputId", `^[a-zA-Z0-9\-_]{1,255}$`),
		withMinLength("displayName", 1), withMinLength("uri", 1)),
		func(ctx context.Context, in createInputIn) (*hub.Input, error) {
			return hub.CreateInput(ctx, client, hubURL, hub.CreateInputRequest{
				InputID: in.InputID, DisplayName: in.DisplayName, URI: in.URI,
				Enabled: in.Enabled, AutoRemove: in.AutoRemove,
			})
		})

	add(s, toolSpec{
		Name: "deleteInput",
		Description: "Delete an ephemeral audio input. Destructive. Static inputs from the hub's configuration cannot be deleted (fails with Validation). " +
			"Returns {\"deleted\": true, \"inputId\": \"<id>\"}; fails with NotFound if there is no such input.",
		Kind: Destructive, Method: "DELETE", Path: "/api/v2/inputs/{inputId}",
	}, inputSchema[inputIDIn](withMinLength("inputId", 1)),
		func(ctx context.Context, in inputIDIn) (*inputDeleted, error) {
			if err := hub.DeleteInput(ctx, client, hubURL, in.InputID); err != nil {
				return nil, err
			}
			return &inputDeleted{Deleted: true, InputID: in.InputID}, nil
		})

	return 5
}
