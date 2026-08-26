package service

import (
	"context"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
)

func TestDecodeOperationInputRejectsUnsafeOrAmbiguousDocuments(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		schema InputSchemaName
	}{
		{name: "duplicate key", schema: InputSchemaPlatformCreate, input: `{"name":"A","name":"B","kind":"informal"}`},
		{name: "unknown field", schema: InputSchemaPlatformCreate, input: `{"name":"A","kind":"informal","extra":true}`},
		{name: "multiple yaml documents", schema: InputSchemaPlatformCreate, input: "name: A\nkind: informal\n---\nname: B\nkind: informal\n"},
		{name: "float instead of exact value", schema: InputSchemaForecastCreate, input: `{"forecasted_at":"2026-01-01T00:00:00Z","value":{"kind":"numeric","point":1.25}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var destination map[string]any
			err := DecodeOperationInput(context.Background(), "-", strings.NewReader(test.input), test.schema, &destination)
			if err == nil || app.ErrorCodeOf(err) != app.CodeInvalidData {
				t.Fatalf("error = %#v, want invalid data", err)
			}
		})
	}
}

func TestDecodeOperationInputEnforcesDepthAndByteLimits(t *testing.T) {
	base := document.Limits{MaxBytes: 200, MaxDepth: 8, MaxNodes: 20, MaxScalarBytes: 100, MaxAliases: 0, MaxExpandedNodes: 20}
	tests := []struct {
		name   string
		input  string
		limits document.Limits
	}{
		{name: "bytes", input: `{"name":"this input is deliberately larger than the configured byte limit","kind":"informal"}`, limits: func() document.Limits { value := base; value.MaxBytes = 40; return value }()},
		{name: "depth", input: `{"name":"A","kind":"informal","account":{"username":"nested"}}`, limits: func() document.Limits { value := base; value.MaxDepth = 1; return value }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var destination PlatformCreateInput
			err := decodeOperationInput(context.Background(), "-", strings.NewReader(test.input), InputSchemaPlatformCreate, &destination, test.limits)
			if err == nil || app.ErrorCodeOf(err) != app.CodeInvalidData {
				t.Fatalf("error = %#v, want bounded invalid data", err)
			}
		})
	}
}

func TestDecodeOperationInputAcceptsTypedJSONAndYAML(t *testing.T) {
	for _, input := range []string{
		`{"name":"Metaculus","kind":"scoring_platform","url":"https://www.metaculus.com"}`,
		"name: Metaculus\nkind: scoring_platform\nurl: https://www.metaculus.com\n",
	} {
		var destination PlatformCreateInput
		if err := DecodeOperationInput(context.Background(), "-", strings.NewReader(input), InputSchemaPlatformCreate, &destination); err != nil {
			t.Fatal(err)
		}
		if destination.Name != "Metaculus" || destination.Kind != "scoring_platform" {
			t.Fatalf("decoded input = %#v", destination)
		}
	}
}
