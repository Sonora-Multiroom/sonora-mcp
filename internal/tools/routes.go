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

// routeList wraps a list of routes so the structured result is an object.
type routeList struct {
	Routes []hub.Route `json:"routes" jsonschema:"The routes the hub returned"`
}

func registerRoutes(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "listRoutes",
		Description: "List audio routes (an input playing to an output or group). " +
			"Optional filters by status, input ID and target ID are combined with AND. Read-only. " +
			"Returns {\"routes\": [...]} with each route's ID, input, target and target type, status, creation and start times, and whether it can be transferred or paused and is paused.",
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
			"Returns the route's ID, input, target and target type, status, creation and start times (startedAt is null until playback starts), and whether it can be transferred or paused and is paused; fails with NotFound if there is no such route.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/routes/{routeId}",
	}, inputSchema[routeIDIn](withMinLength("routeId", 1)),
		func(ctx context.Context, in routeIDIn) (*hub.Route, error) {
			return hub.GetRoute(ctx, client, hubURL, in.RouteID)
		})

	return 2
}
