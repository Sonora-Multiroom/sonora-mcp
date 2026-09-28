// Package version holds the build version of sonora-mcp.
package version

// Version is the build version reported in the MCP server info, /health and
// the startup log. It defaults to "dev" and is set at build time with
//
//	-ldflags "-X github.com/Sonora-Multiroom/sonora-mcp/internal/version.Version=<v>"
var Version = "dev"
