package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestOptionalTracksOmittedNullAndValue(t *testing.T) {
	var patch RootMetadataPatchInput
	if err := json.Unmarshal([]byte(`{"title":null,"description":"kept"}`), &patch); err != nil {
		t.Fatal(err)
	}
	if !patch.Title.Set || !patch.Title.Null {
		t.Fatalf("title state = %#v", patch.Title)
	}
	if !patch.Description.Set || patch.Description.Null || patch.Description.Value != "kept" {
		t.Fatalf("description state = %#v", patch.Description)
	}
	if patch.DefaultTimezone.Set {
		t.Fatalf("omitted field marked as set: %#v", patch.DefaultTimezone)
	}
}

func TestAllOperationInputSchemasCompileAndAreClosed(t *testing.T) {
	for _, name := range InputSchemaNames() {
		t.Run(string(name), func(t *testing.T) {
			schemaBytes, err := InputSchema(name)
			if err != nil {
				t.Fatal(err)
			}
			document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
			if err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource("schema.json", document); err != nil {
				t.Fatal(err)
			}
			if _, err := compiler.Compile("schema.json"); err != nil {
				t.Fatalf("schema does not compile: %v", err)
			}
			if !strings.Contains(string(schemaBytes), `"additionalProperties": false`) {
				t.Fatal("schema has no closed object")
			}
		})
	}
}

func TestInitAndQuestionAddHaveDeliberatelyDifferentTypeInputs(t *testing.T) {
	initSchema := compileInputSchema(t, InputSchemaInit)
	questionSchema := compileInputSchema(t, InputSchemaQuestionAdd)

	initDocument := minimalQuestionDocument(true)
	if err := initSchema.Validate(initDocument); err != nil {
		t.Fatalf("init input rejected its question type: %v", err)
	}
	questionDocument := minimalQuestionDocument(false)
	if err := questionSchema.Validate(questionDocument["question"]); err != nil {
		t.Fatalf("question-add input rejected scalar-normalized shape: %v", err)
	}
	questionDocument["question"].(map[string]any)["type"] = "binary"
	if err := questionSchema.Validate(questionDocument["question"]); err == nil {
		t.Fatal("question-add schema accepted duplicate input type")
	}
}

func compileInputSchema(t *testing.T, name InputSchemaName) *jsonschema.Schema {
	t.Helper()
	schemaBytes, err := InputSchema(name)
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", document); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func minimalQuestionDocument(includeType bool) map[string]any {
	question := map[string]any{
		"id": "q-one", "title": "Will it happen?", "resolution_criteria": "Resolve from the named source.",
		"forecast_window":        map[string]any{"closes_at": "2027-01-01T00:00:00Z"},
		"expected_resolution_at": "2027-01-02T00:00:00Z",
		"initial_forecast": map[string]any{
			"id": "f-one", "visibility": "public", "forecasted_at": "2026-01-01T00:00:00Z",
			"value": map[string]any{"kind": "binary", "probability_bp": 5000},
		},
	}
	if includeType {
		question["type"] = "binary"
		return map[string]any{"question": question}
	}
	delete(question, "id")
	return map[string]any{"question": question}
}
