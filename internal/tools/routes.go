package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// routeStatuses are the route states the hub reports.
var routeStatuses = []string{"STARTING", "ACTIVE", "STOPPING", "STOPPED", "FAILED"}

type listRoutesIn struct {
	Status   string `json:"status,omitempty" jsonschema:"Only routes in this status: STARTING, ACTIVE, STOPPING, STOPPED or FAILED"`
	InputID  string `json:"inputId,omitempty" jsonschema:"Only routes playing this input ID"`
	TargetID string `json:"targetId,omitempty" jsonschema:"Only routes playing to this output or group ID"`
}

type routeIDIn struct {
	RouteID string `json:"routeId" jsonschema:"Unique route identifier, as returned by listRoutes"`
}

// targetTypes are the kinds of target a route or playback can play to.
var targetTypes = []string{"SINGLE_OUTPUT", "OUTPUT_GROUP"}

type createRouteIn struct {
	InputID     string `json:"inputId" jsonschema:"ID of the input to play, as returned by listInputs"`
	TargetID    string `json:"targetId" jsonschema:"ID of the output or group to play to"`
	TargetType  string `json:"targetType" jsonschema:"SINGLE_OUTPUT if targetId is an output, OUTPUT_GROUP if it is a group"`
	RouteVolume *int   `json:"routeVolume,omitempty" jsonschema:"Volume of this playback only, an integer from 0 to 100; the room's volume is not changed and its volume control does not reach this playback, while mute, master mute and each output's gain still apply; use it for announcements and alerts; when omitted the input's default route volume applies, else the room's volume"`
}

type transferRouteIn struct {
	RouteID    string `json:"routeId" jsonschema:"Unique route identifier, as returned by listRoutes"`
	TargetID   string `json:"targetId" jsonschema:"ID of the output or group to move the route to"`
	TargetType string `json:"targetType" jsonschema:"SINGLE_OUTPUT if targetId is an output, OUTPUT_GROUP if it is a group"`
}

type routePauseIn struct {
	RouteID string `json:"routeId" jsonschema:"Unique route identifier, as returned by listRoutes"`
	Paused  bool   `json:"paused" jsonschema:"true to pause playback, false to resume it"`
}

type routeVolumeIn struct {
	RouteID string `json:"routeId" jsonschema:"Unique route identifier, as returned by listRoutes"`
	Volume  int    `json:"volume" jsonschema:"The route's own volume, an integer from 0 to 100"`
}

// routeDeleted confirms a deleted route; the hub answers 204 with no body.
type routeDeleted struct {
	Deleted bool   `json:"deleted" jsonschema:"Always true"`
	RouteID string `json:"routeId" jsonschema:"ID of the deleted route"`
}

// routeList wraps a list of routes so the structured result is an object.
type routeList struct {
	Routes []hub.Route `json:"routes" jsonschema:"The routes the hub returned"`
}

func registerRoutes(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "listRoutes",
		Description: "List audio routes (an input playing to an output or group). " +
			"Optional filters by status, input ID and target ID are combined with AND. Read-only. " +
			"Returns {\"routes\": [...]} with each route's ID, input, target and target type, status, creation and start times, whether it can be transferred or paused and is paused, join mode (REPLACE, MIX or DUCK_OTHERS) and the outputs it plays on right now.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/routes",
	}, inputSchema[listRoutesIn](withEnum("status", routeStatuses...)),
		func(ctx context.Context, in listRoutesIn) (*routeList, error) {
			routes, err := hub.ListRoutes(ctx, client, hubURL, in.Status, in.InputID, in.TargetID)
			if err != nil {
				return nil, err
			}
			return &routeList{Routes: routes}, nil
		})

	add(s, toolSpec{
		Name: "getRoute",
		Description: "Get one audio route by ID. Read-only. " +
			"Returns the route's ID, input, target and target type, status, creation and start times (startedAt is null until playback starts), whether it can be transferred or paused and is paused, join mode (REPLACE, MIX or DUCK_OTHERS) and the outputs it plays on right now (a group route may play on fewer outputs than its group has); fails with NotFound if there is no such route.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/routes/{routeId}",
	}, inputSchema[routeIDIn](withMinLength("routeId", 1)),
		func(ctx context.Context, in routeIDIn) (*hub.Route, error) {
			return hub.GetRoute(ctx, client, hubURL, in.RouteID)
		})

	add(s, toolSpec{
		Name: "createRoute",
		Description: "Start playing an input to an output or group by creating a route. Changes state: calling it again creates another route. " +
			"Returns the new route (ID, input, target and target type, status, creation and start times, whether it can be transferred or paused and is paused, join mode, current outputs); " +
			"fails with NotFound if the input or target does not exist, Conflict if the hub refuses the route in its current state (the reason in parentheses says why: a disabled input, output or group, the route limit, or the input already playing there; change that state and retry), and RouteFailed if the hub cannot start playback.",
		Kind: StateChanging, Method: "POST", Path: "/api/v2/routes",
	}, inputSchema[createRouteIn](withMinLength("inputId", 1), withMinLength("targetId", 1), withEnum("targetType", targetTypes...),
		withRange("routeVolume", 0, 100)),
		func(ctx context.Context, in createRouteIn) (*hub.Route, error) {
			return hub.CreateRoute(ctx, client, hubURL, hub.CreateRouteRequest{
				InputID: in.InputID, TargetID: in.TargetID, TargetType: in.TargetType,
				RouteVolume: in.RouteVolume,
			})
		})

	add(s, toolSpec{
		Name: "transferRoute",
		Description: "Move a playing route to a different output or group without restarting the input. Changes state. " +
			"Returns the updated route, which keeps its ID; fails with NotFound if the route or target does not exist, Conflict if the hub refuses the new target in its current state (for example it is disabled; the reason is in parentheses), and Validation if the route cannot be transferred. A Conflict with reason ROUTE_CHANGING_OUTPUTS means the route is still adding an output; it clears within moments, and the same request can then be retried.",
		Kind: StateChanging, Method: "POST", Path: "/api/v2/routes/{routeId}/transfer",
	}, inputSchema[transferRouteIn](withMinLength("routeId", 1), withMinLength("targetId", 1), withEnum("targetType", targetTypes...)),
		func(ctx context.Context, in transferRouteIn) (*hub.Route, error) {
			return hub.TransferRoute(ctx, client, hubURL, in.RouteID, hub.TransferRequest{
				TargetID: in.TargetID, TargetType: in.TargetType,
			})
		})

	add(s, toolSpec{
		Name: "setRoutePause",
		Description: "Pause or resume a route. Idempotent. Only routes whose pauseable flag is true can be paused. " +
			"Returns the updated route; fails with NotFound if there is no such route and Validation if it cannot be paused.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/routes/{routeId}/pause",
	}, inputSchema[routePauseIn](withMinLength("routeId", 1)),
		func(ctx context.Context, in routePauseIn) (*hub.Route, error) {
			return hub.SetPauseState(ctx, client, hubURL, in.RouteID, in.Paused)
		})

	add(s, toolSpec{
		Name: "setRouteVolume",
		Description: "Set a playing route's own volume, for example to make an announcement louder or quieter while it plays. Idempotent. " +
			"Changes only this route: no output's or group's volume changes, and other routes on the same outputs are unaffected. It also works on a route started without a route volume, which then stops following the room's volume. " +
			"A route volume can be changed but never cleared. Returns the updated route; fails with NotFound if there is no such route.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/routes/{routeId}/volume",
	}, inputSchema[routeVolumeIn](withMinLength("routeId", 1), withRange("volume", 0, 100)),
		func(ctx context.Context, in routeVolumeIn) (*hub.Route, error) {
			return hub.SetRouteVolume(ctx, client, hubURL, in.RouteID, in.Volume)
		})

	add(s, toolSpec{
		Name: "deleteRoute",
		Description: "Stop playback by deleting a route. Destructive: the route is gone and must be created again to resume. " +
			"Returns {\"deleted\": true, \"routeId\": \"<id>\"}; fails with NotFound if there is no such route.",
		Kind: Destructive, Method: "DELETE", Path: "/api/v2/routes/{routeId}",
	}, inputSchema[routeIDIn](withMinLength("routeId", 1)),
		func(ctx context.Context, in routeIDIn) (*routeDeleted, error) {
			if err := hub.DeleteRoute(ctx, client, hubURL, in.RouteID); err != nil {
				return nil, err
			}
			return &routeDeleted{Deleted: true, RouteID: in.RouteID}, nil
		})

	return 7
}
