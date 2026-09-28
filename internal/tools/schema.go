package tools

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// schemaOption adds a constraint to one property of an inferred input schema.
type schemaOption func(*jsonschema.Schema)

// inputSchema infers an object schema from the struct T (json tags name the
// properties, jsonschema tags describe them) and applies opts. The jsonschema
// tag only carries the description, so constraints are set here (research R3).
// It panics on a programming error, such as a constraint on a property T does
// not have, so a bad tool definition fails at startup.
func inputSchema[T any](opts ...schemaOption) *jsonschema.Schema {
	s, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("inputSchema: %v", err))
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// property returns the named property of s, or panics.
func property(s *jsonschema.Schema, name string) *jsonschema.Schema {
	p, ok := s.Properties[name]
	if !ok {
		panic(fmt.Sprintf("inputSchema: no property %q", name))
	}
	return p
}

// withRange sets minimum and maximum on a numeric property.
func withRange(name string, lo, hi float64) schemaOption {
	return func(s *jsonschema.Schema) {
		p := property(s, name)
		p.Minimum, p.Maximum = &lo, &hi
	}
}

// withEnum restricts a string property to values.
func withEnum(name string, values ...string) schemaOption {
	return func(s *jsonschema.Schema) {
		p := property(s, name)
		p.Enum = make([]any, len(values))
		for i, v := range values {
			p.Enum[i] = v
		}
	}
}

// withMinLength sets minLength on a string property.
func withMinLength(name string, n int) schemaOption {
	return func(s *jsonschema.Schema) {
		property(s, name).MinLength = &n
	}
}

// withPattern sets a regular-expression pattern on a string property.
func withPattern(name, pattern string) schemaOption {
	return func(s *jsonschema.Schema) {
		property(s, name).Pattern = pattern
	}
}
