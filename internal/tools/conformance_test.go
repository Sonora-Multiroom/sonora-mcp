package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Sonora-Multiroom/sonora-cli/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// openAPI is the subset of an OpenAPI document the conformance test reads.
type openAPI struct {
	Paths      map[string]map[string]operation `json:"paths"`
	Components struct {
		Schemas map[string]map[string]any `json:"schemas"`
	} `json:"components"`
}

type operation struct {
	Parameters []struct {
		Name     string         `json:"name"`
		In       string         `json:"in"`
		Required bool           `json:"required"`
		Schema   map[string]any `json:"schema"`
	} `json:"parameters"`
	RequestBody *struct {
		Content map[string]struct {
			Schema map[string]any `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema map[string]any `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

// specField is one input the hub operation accepts, normalized for comparison.
type specField struct {
	source   string // "path", "query" or "body"
	required bool
	schema   map[string]any
}

// listEnvelopes are the wrapper properties of list results.
var listEnvelopes = []string{"inputs", "outputs", "groups", "routes"}

// constraintKeys are the schema keywords compared between tool and spec.
var constraintKeys = []string{"enum", "minimum", "maximum", "minLength", "pattern"}

func loadSpec(t *testing.T) *openAPI {
	t.Helper()
	var spec openAPI
	if err := json.Unmarshal(api.Spec, &spec); err != nil {
		t.Fatalf("parse api.Spec: %v", err)
	}
	return &spec
}

// resolve follows a local "#/components/schemas/X" reference.
func (s *openAPI) resolve(schema map[string]any) map[string]any {
	for {
		ref, ok := schema["$ref"].(string)
		if !ok {
			return schema
		}
		schema = s.Components.Schemas[strings.TrimPrefix(ref, "#/components/schemas/")]
	}
}

// fields returns every input the operation accepts, keyed by name.
func (s *openAPI) fields(op operation) map[string]specField {
	out := map[string]specField{}
	for _, p := range op.Parameters {
		schema := s.resolve(p.Schema)
		if p.In == "path" {
			// An empty path segment changes the route, so a path parameter
			// means a non-empty string (research R4).
			schema = cloneMap(schema)
			if _, ok := schema["minLength"]; !ok {
				schema["minLength"] = 1.0
			}
		}
		out[p.Name] = specField{source: p.In, required: p.Required || p.In == "path", schema: schema}
	}
	if op.RequestBody != nil {
		body := s.resolve(op.RequestBody.Content["application/json"].Schema)
		required, _ := body["required"].([]any)
		props, _ := body["properties"].(map[string]any)
		for name, raw := range props {
			schema := s.resolve(raw.(map[string]any))
			out[name] = specField{
				source:   "body",
				required: slices.Contains(required, any(name)) && !nullable(schema),
				schema:   schema,
			}
		}
	}
	return out
}

// successSchema returns the JSON schema of the operation's success response,
// or nil when it has no JSON body (204).
func (s *openAPI) successSchema(op operation) map[string]any {
	for _, code := range []string{"200", "201", "202"} {
		if r, ok := op.Responses[code]; ok {
			if c, ok := r.Content["application/json"]; ok {
				schema := s.resolve(c.Schema)
				if schema["type"] == "array" {
					schema = s.resolve(schema["items"].(map[string]any))
				}
				return schema
			}
		}
	}
	return nil
}

func nullable(schema map[string]any) bool {
	types, ok := schema["type"].([]any)
	return ok && slices.Contains(types, any("null"))
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// listedTools returns the tools of a fully registered server, as an agent
// sees them.
func listedTools(t *testing.T) map[string]*mcp.Tool {
	t.Helper()
	s := newTestSession(t, "http://127.0.0.1:1", nil)
	res, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	out := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		out[tool.Name] = tool
	}
	return out
}

// schemaMap returns a listed schema as generic JSON.
func schemaMap(t *testing.T, schema any) map[string]any {
	t.Helper()
	m, _ := asJSONValue(t, schema).(map[string]any)
	return m
}

// TestConformance checks every registered tool against the hub spec that
// sonora-cli publishes (Principle I, FR-018, research R4).
func TestConformance(t *testing.T) {
	spec := loadSpec(t)
	tools := listedTools(t)

	for _, ts := range registeredSpecs() {
		tool, ok := tools[ts.Name]
		if !ok {
			continue // registered only by another test
		}
		t.Run(ts.Name, func(t *testing.T) {
			op, ok := spec.Paths[ts.Path][strings.ToLower(ts.Method)]
			if !ok {
				t.Fatalf("%s: operation %s %s not in the hub spec", ts.Name, ts.Method, ts.Path)
			}
			checkInputs(t, ts.Name, schemaMap(t, tool.InputSchema), spec.fields(op))
			checkOutputs(t, ts.Name, schemaMap(t, tool.OutputSchema), spec.successSchema(op))
		})
	}
}

func checkInputs(t *testing.T, tool string, input map[string]any, fields map[string]specField) {
	t.Helper()
	props, _ := input["properties"].(map[string]any)
	required, _ := input["required"].([]any)

	for name, raw := range props {
		prop := raw.(map[string]any)
		f, ok := fields[name]
		if !ok {
			t.Errorf("%s: input %q is not a path, query or body parameter of the operation", tool, name)
			continue
		}
		toolRequired := slices.Contains(required, any(name))
		if toolRequired != f.required {
			t.Errorf("%s: %s: required: tool %v, spec %v", tool, name, toolRequired, f.required)
		}
		for _, key := range constraintKeys {
			tv, sv := prop[key], f.schema[key]
			if fmt.Sprint(tv) != fmt.Sprint(sv) {
				t.Errorf("%s: %s: %s: tool %v, spec %v", tool, name, key, show(tv), show(sv))
			}
		}
	}
	for name, f := range fields {
		if _, ok := props[name]; !ok && f.required {
			t.Errorf("%s: required %s parameter %q missing from the tool input", tool, f.source, name)
		}
	}
}

func checkOutputs(t *testing.T, tool string, output, response map[string]any) {
	t.Helper()
	if response == nil {
		return // 204: the tool returns a deletion confirmation
	}
	props, _ := output["properties"].(map[string]any)
	if len(props) == 1 {
		for _, env := range listEnvelopes {
			if list, ok := props[env].(map[string]any); ok {
				items, _ := list["items"].(map[string]any)
				props, _ = items["properties"].(map[string]any)
			}
		}
	}
	specProps, _ := response["properties"].(map[string]any)
	for name := range specProps {
		if _, ok := props[name]; !ok {
			t.Errorf("%s: response field %q missing from the tool output schema", tool, name)
		}
	}
}

func show(v any) string {
	if v == nil {
		return "(none)"
	}
	return fmt.Sprint(v)
}

// TestToolDescriptions checks that every tool and every input field is
// described for agents (FR-003).
func TestToolDescriptions(t *testing.T) {
	for name, tool := range listedTools(t) {
		if strings.TrimSpace(tool.Description) == "" {
			t.Errorf("%s: empty tool description", name)
		}
		props, _ := schemaMap(t, tool.InputSchema)["properties"].(map[string]any)
		for field, raw := range props {
			if d, _ := raw.(map[string]any)["description"].(string); strings.TrimSpace(d) == "" {
				t.Errorf("%s: input %q has no description", name, field)
			}
		}
	}
}

// TestRegisterCount checks that Register reports the number of tools it
// actually registered (used by the startup log).
func TestRegisterCount(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "sonora-mcp", Version: "test"}, nil)
	n := Register(server, nil, "http://127.0.0.1:1")
	res, err := connect(t, server).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if n != len(res.Tools) {
		t.Errorf("Register returned %d, ListTools has %d tools", n, len(res.Tools))
	}
}
