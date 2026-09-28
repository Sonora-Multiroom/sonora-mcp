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

type groupVolumeIn struct {
	GroupID string `json:"groupId" jsonschema:"Unique group identifier, as returned by listGroups"`
	Volume  int    `json:"volume" jsonschema:"New volume level for every output in the group, an integer from 0 (silent) to 100 (maximum)"`
}

type groupMuteIn struct {
	GroupID string `json:"groupId" jsonschema:"Unique group identifier, as returned by listGroups"`
	Muted   bool   `json:"muted" jsonschema:"true to mute the group, false to unmute it"`
}

type groupEnabledIn struct {
	GroupID string `json:"groupId" jsonschema:"Unique group identifier, as returned by listGroups"`
	Enabled bool   `json:"enabled" jsonschema:"true to enable the group, false to disable it"`
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

	add(s, toolSpec{
		Name: "setGroupVolume",
		Description: "Set the volume of every output in an output group. Idempotent: setting the same volume again changes nothing. " +
			"Returns the group's ID, new volume (0-100) and the time it was updated; fails with NotFound if there is no such group.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/groups/{groupId}/volume",
	}, inputSchema[groupVolumeIn](withMinLength("groupId", 1), withRange("volume", 0, 100)),
		func(ctx context.Context, in groupVolumeIn) (*hub.GroupVolume, error) {
			return hub.SetGroupVolume(ctx, client, hubURL, in.GroupID, in.Volume)
		})

	add(s, toolSpec{
		Name: "setGroupMute",
		Description: "Mute or unmute an output group. Idempotent. " +
			"Returns the group's ID, new mute state and the time it was updated; fails with NotFound if there is no such group.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/groups/{groupId}/mute",
	}, inputSchema[groupMuteIn](withMinLength("groupId", 1)),
		func(ctx context.Context, in groupMuteIn) (*hub.GroupMute, error) {
			return hub.SetGroupMuted(ctx, client, hubURL, in.GroupID, in.Muted)
		})

	add(s, toolSpec{
		Name: "setGroupEnabled",
		Description: "Enable or disable an output group; a disabled group is hidden from listGroups by default and cannot play. Idempotent. " +
			"Returns the updated group (ID, name, member output IDs, mute state, enabled flag); fails with NotFound if there is no such group.",
		Kind: Idempotent, Method: "PUT", Path: "/api/v2/groups/{groupId}/enabled",
	}, inputSchema[groupEnabledIn](withMinLength("groupId", 1)),
		func(ctx context.Context, in groupEnabledIn) (*hub.Group, error) {
			return hub.SetGroupEnabled(ctx, client, hubURL, in.GroupID, in.Enabled)
		})

	return 5
}
