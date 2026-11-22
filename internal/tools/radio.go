package tools

import (
	"context"
	"errors"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// radioError replaces a "not offered" answer from the hub's radio endpoints
// with the shared client's diagnosis of why radio is unavailable; it costs one
// extension-inventory call and only follows a failure. Any other error is
// returned unchanged.
func radioError(ctx context.Context, client *http.Client, hubURL string, err error) error {
	var notOffered *hub.RadioNotOfferedError
	if errors.As(err, &notOffered) {
		return hub.DiagnoseRadioUnavailable(ctx, client, hubURL)
	}
	return err
}

// registerRadio registers the radio catalogue tools. They call the hub with
// radioClient, whose bound is longer than the standard one; client reads the
// extension inventory after a failure. It returns the number of tools
// registered.
func registerRadio(s *mcp.Server, client, radioClient *http.Client, hubURL string) int {
	return 0
}
