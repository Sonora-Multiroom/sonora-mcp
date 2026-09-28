package tools

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// outcomeInvalidInput is the logged outcome of a call the SDK rejected
// because its arguments did not match the input schema.
const outcomeInvalidInput = "InvalidInput"

// outcomeCanceled is the logged outcome of a call the client canceled
// before it finished. The hub call fails with context.Canceled, which would
// otherwise be logged as a Network error.
const outcomeCanceled = "Canceled"

// LogToolCalls returns receiving middleware that writes one log record per
// tools/call with the tool name, its arguments as JSON, the outcome ("ok",
// the error category, InvalidInput, or Canceled) and the duration (FR-012). Other
// methods are not logged.
func LogToolCalls(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			start := time.Now()
			res, err := next(ctx, method, req)

			name, args := "", "{}"
			if r, ok := req.(*mcp.CallToolRequest); ok && r.Params != nil {
				name = r.Params.Name
				if len(r.Params.Arguments) > 0 {
					args = string(r.Params.Arguments)
				}
			}
			logger.Info("tool call",
				"tool", name,
				"args", args,
				"outcome", outcome(ctx, res, err),
				"duration", time.Since(start))
			return res, err
		}
	}
}

func outcome(ctx context.Context, res mcp.Result, err error) string {
	if errors.Is(ctx.Err(), context.Canceled) {
		return outcomeCanceled
	}
	if err != nil {
		return "ProtocolError"
	}
	r, ok := res.(*mcp.CallToolResult)
	if !ok || !r.IsError {
		return "ok"
	}
	if c := errorCategory(r.GetError()); c != "" {
		return c
	}
	return outcomeInvalidInput
}
