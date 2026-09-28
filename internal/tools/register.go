package tools

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"slices"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Kind describes a tool's effect on the audio system; it decides the tool's
// annotations.
type Kind int

const (
	// ReadOnly tools only read hub state.
	ReadOnly Kind = iota
	// Idempotent tools set state; repeating the call changes nothing more.
	Idempotent
	// StateChanging tools create or move things; repeating the call repeats
	// the effect.
	StateChanging
	// Destructive tools remove things.
	Destructive
)

// annotations returns the MCP tool annotations for k.
func (k Kind) annotations() *mcp.ToolAnnotations {
	no, yes := false, true
	switch k {
	case ReadOnly:
		return &mcp.ToolAnnotations{ReadOnlyHint: true}
	case Idempotent:
		return &mcp.ToolAnnotations{DestructiveHint: &no, IdempotentHint: true}
	case StateChanging:
		return &mcp.ToolAnnotations{DestructiveHint: &no}
	default:
		return &mcp.ToolAnnotations{DestructiveHint: &yes}
	}
}

// toolSpec describes a tool and the hub operation it wraps.
type toolSpec struct {
	Name, Description string
	Kind              Kind
	Method, Path      string // hub operation as in the spec, e.g. PUT /api/v2/outputs/{outputId}/volume
}

// registry holds the spec of every tool registered through add, for the
// conformance test. It describes tool definitions only, never hub state.
var registry struct {
	sync.Mutex
	specs []toolSpec
}

func record(spec toolSpec) {
	registry.Lock()
	defer registry.Unlock()
	i := slices.IndexFunc(registry.specs, func(s toolSpec) bool { return s.Name == spec.Name })
	if i >= 0 {
		registry.specs[i] = spec
		return
	}
	registry.specs = append(registry.specs, spec)
}

// registeredSpecs returns a copy of the registry.
func registeredSpecs() []toolSpec {
	registry.Lock()
	defer registry.Unlock()
	return slices.Clone(registry.specs)
}

// errInternal is returned when a handler panics.
var errInternal = &categoryError{category: CategoryInternal, message: "unexpected server error"}

// add registers a tool on s with the explicit input schema in; the output
// schema is inferred from Out. The SDK validates arguments against in before
// h runs and builds the structured result from h's output. A hub error
// returned by h becomes a categorized error result, and a panic in h becomes
// "Internal: unexpected server error" without affecting other calls.
func add[In, Out any](s *mcp.Server, spec toolSpec, in *jsonschema.Schema, h func(context.Context, In) (Out, error)) {
	record(spec)
	tool := &mcp.Tool{
		Name:        spec.Name,
		Description: spec.Description,
		InputSchema: in,
		Annotations: spec.Kind.annotations(),
	}
	mcp.AddTool(s, tool, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (_ *mcp.CallToolResult, out Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("tool panicked", "tool", spec.Name, "panic", r, "stack", string(debug.Stack()))
				var zero Out
				out, err = zero, errInternal
			}
		}()
		out, err = h(ctx, input)
		if err != nil {
			var zero Out
			return nil, zero, toolError(err)
		}
		return nil, out, nil
	})
}

// errorCategory returns the category of a tool error, or "" if err is not one.
func errorCategory(err error) string {
	var ce *categoryError
	if errors.As(err, &ce) {
		return ce.category
	}
	return ""
}
