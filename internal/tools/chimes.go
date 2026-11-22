package tools

import (
	"context"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// chimeList wraps a list of chimes so the structured result is an object.
type chimeList struct {
	Chimes []hub.Chime `json:"chimes" jsonschema:"The chimes the hub has loaded"`
}

func registerChimes(s *mcp.Server, client *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "listChimes",
		Description: "List the short sounds the hub can play before a route, to pass as chime to playback or createRoute or as defaultChime to createInput. Read-only. " +
			"Returns {\"chimes\": [...]} with each chime's id and durationMillis; the list depends on the hub's files and can be empty. " +
			"The reserved id none (no chime) is never listed.",
		Kind: ReadOnly, Method: "GET", Path: "/api/v2/chimes",
	}, inputSchema[noInput](),
		func(ctx context.Context, _ noInput) (*chimeList, error) {
			chimes, err := hub.ListChimes(ctx, client, hubURL)
			if err != nil {
				return nil, err
			}
			return &chimeList{Chimes: chimes}, nil
		})

	return 1
}
