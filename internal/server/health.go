package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/tiger-seo/sonora-mcp/internal/version"
)

// hubCheckTimeout bounds the hub request made by each health check.
const hubCheckTimeout = 2 * time.Second

// health is the /health response body.
type health struct {
	Status  string `json:"status"`
	Server  string `json:"server"`
	Version string `json:"version"`
	Hub     string `json:"hub"`
}

// healthHandler answers 200 while the server runs and reports whether the
// hub at hubURL answered a read-only request within 2 seconds. Nothing is
// cached: every check asks the hub.
func healthHandler(client *http.Client, hubURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), hubCheckTimeout)
		defer cancel()
		hubState := "reachable"
		if _, err := hub.GetMasterMute(ctx, client, hubURL); err != nil {
			hubState = "unreachable"
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(health{
			Status:  "ok",
			Server:  "sonora-mcp",
			Version: version.Version,
			Hub:     hubState,
		})
	})
}
