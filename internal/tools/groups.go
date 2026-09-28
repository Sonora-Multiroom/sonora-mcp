package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listGroupsIn struct {
	IncludeDisabled bool `json:"includeDisabled,omitempty" jsonschema:"Include disabled groups in the response (default false)"`
}

type groupIDIn struct {
	GroupID string `json:"groupId" jsonschema:"Unique group identifier, as returned by listGroups"`
}

// groupList wraps a list of groups so the structured result is an object.
type groupList struct {
	Groups []hub.Group `json:"groups" jsonschema:"The groups the hub returned"`
}

func registerGroups(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "listGroups",
		Description: "List the hub's output groups (sets of outputs controlled together). " +
			"Only enabled groups are returned unless includeDisabled is true. Read-only. " +
			"Returns {\"groups\": [...]} with each group's ID, name, member output IDs, mute state and enabled flag.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/groups",
	}, inputSchema[listGroupsIn](),
		func(ctx context.Context, in listGroupsIn) (*groupList, error) {
			groups, err := hub.ListGroups(ctx, client, hubURL, in.IncludeDisabled)
			if err != nil {
				return nil, err
			}
			return &groupList{Groups: groups}, nil
		})

	add(s, toolSpec{
		Name: "getGroup",
		Description: "Get one output group by ID. Read-only. " +
			"Returns the group's ID, name, member output IDs, mute state and enabled flag; fails with NotFound if there is no such group.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/groups/{groupId}",
	}, inputSchema[groupIDIn](withMinLength("groupId", 1)),
		func(ctx context.Context, in groupIDIn) (*hub.Group, error) {
			return hub.GetGroup(ctx, client, hubURL, in.GroupID)
		})

	return 2
}
