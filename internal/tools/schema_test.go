package tools

import (
	"slices"
	"testing"
)

type schemaTestInput struct {
	OutputID   string `json:"outputId" jsonschema:"ID of the output"`
	Volume     int    `json:"volume" jsonschema:"Volume level"`
	TargetType string `json:"targetType" jsonschema:"Kind of target"`
	InputID    string `json:"inputId" jsonschema:"ID of the input"`
	Note       *bool  `json:"note,omitempty" jsonschema:"Optional flag"`
}

func TestInputSchemaAppliesConstraints(t *testing.T) {
	s := inputSchema[schemaTestInput](
		withRange("volume", 0, 100),
		withEnum("targetType", "SINGLE_OUTPUT", "OUTPUT_GROUP"),
		withMinLength("outputId", 1),
		withPattern("inputId", `^[a-z]+$`),
	)

	if s.Type != "object" {
		t.Fatalf("type = %q, want object", s.Type)
	}
	vol := s.Properties["volume"]
	if vol.Minimum == nil || *vol.Minimum != 0 || vol.Maximum == nil || *vol.Maximum != 100 {
		t.Errorf("volume range = %v..%v, want 0..100", vol.Minimum, vol.Maximum)
	}
	if got := s.Properties["targetType"].Enum; !slices.Equal(got, []any{"SINGLE_OUTPUT", "OUTPUT_GROUP"}) {
		t.Errorf("targetType enum = %v", got)
	}
	if ml := s.Properties["outputId"].MinLength; ml == nil || *ml != 1 {
		t.Errorf("outputId minLength = %v, want 1", ml)
	}
	if got := s.Properties["inputId"].Pattern; got != `^[a-z]+$` {
		t.Errorf("inputId pattern = %q", got)
	}
	if got := s.Properties["outputId"].Description; got != "ID of the output" {
		t.Errorf("outputId description = %q, want the jsonschema tag", got)
	}
	if !slices.Contains(s.Required, "volume") || slices.Contains(s.Required, "note") {
		t.Errorf("required = %v, want volume but not note", s.Required)
	}
}

func TestInputSchemaUnknownPropertyPanics(t *testing.T) {
	for name, opt := range map[string]schemaOption{
		"range":     withRange("nope", 0, 1),
		"enum":      withEnum("nope", "A"),
		"minLength": withMinLength("nope", 1),
		"pattern":   withPattern("nope", "x"),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("want panic for unknown property")
				}
			}()
			inputSchema[schemaTestInput](opt)
		})
	}
}
