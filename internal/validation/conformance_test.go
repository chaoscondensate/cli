package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/document"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestAllPinnedInvalidCasesAreRejected(t *testing.T) {
	data, err := fs.ReadFile(contractschema.Conformance(), "invalid-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name       string `json:"name"`
		Base       string `json:"base"`
		Operations []struct {
			Op    string `json:"op"`
			Path  string `json:"path"`
			Value any    `json:"value"`
		} `json:"operations"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 13 {
		t.Fatalf("got %d invalid cases, want 13", len(cases))
	}

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			baseName := "individual-ledger.json"
			if strings.HasSuffix(testCase.Base, "team-ledger.yaml") {
				baseName = "team-ledger.yaml"
			}
			base := loadValidLedgerDocument(t, baseName).Root.Any()
			for _, operation := range testCase.Operations {
				if err := applyFixtureOperation(base, operation.Op, operation.Path, normalizeFixtureNumber(operation.Value)); err != nil {
					t.Fatal(err)
				}
			}
			schemaIssues, semanticIssues := validateGenericDocument(t, base)
			if len(schemaIssues) == 0 && len(semanticIssues) == 0 {
				t.Fatal("invalid upstream fixture was accepted")
			}
			if testCase.Name == "unsupported-opentimestamps-timestamp" && len(schemaIssues) == 0 {
				t.Fatalf("unsupported timestamp protocol fixture must be rejected by the pinned schema, got semantic=%#v", semanticIssues)
			}
			if testCase.Name == "rfc3161-timestamp-after-known-outcome" && !hasSemanticCode(semanticIssues, "semantic.timestamp_chronology") {
				t.Fatalf("late timestamp fixture must be rejected by chronology semantics, got schema=%#v semantic=%#v", schemaIssues, semanticIssues)
			}
		})
	}
}

func loadValidLedgerDocument(t *testing.T, name string) *document.Document {
	t.Helper()
	data, err := fs.ReadFile(contractschema.Conformance(), name)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(name, ".json") {
		parsed, err := document.ParseJSON(bytes.NewReader(data), document.DefaultLimits)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	parsed, err := document.ParseYAML(bytes.NewReader(data), document.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func validateGenericDocument(t *testing.T, value any) ([]SchemaIssue, []SemanticIssue) {
	t.Helper()
	structural, err := DefaultStructuralValidator()
	if err != nil {
		t.Fatal(err)
	}
	schemaIssues, err := structural.Validate(value)
	if err != nil {
		t.Fatal(err)
	}
	if len(schemaIssues) > 0 {
		return schemaIssues, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := document.ParseJSON(bytes.NewReader(encoded), document.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	model, err := DecodeLedger(parsed)
	if err != nil {
		t.Fatal(err)
	}
	semanticIssues, err := ValidateSemantics(model, nil)
	if err != nil {
		t.Fatal(err)
	}
	return nil, semanticIssues
}

func applyFixtureOperation(root any, operation, pointer string, value any) error {
	if operation != "replace" && operation != "add" {
		return fmt.Errorf("unsupported pinned fixture operation %q", operation)
	}
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	for index, token := range tokens {
		tokens[index] = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
	}
	current := root
	for _, token := range tokens[:len(tokens)-1] {
		switch container := current.(type) {
		case map[string]any:
			current = container[token]
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(container) {
				return fmt.Errorf("invalid fixture pointer %q", pointer)
			}
			current = container[index]
		default:
			return fmt.Errorf("invalid fixture pointer %q", pointer)
		}
	}
	last := tokens[len(tokens)-1]
	switch container := current.(type) {
	case map[string]any:
		container[last] = value
	case []any:
		index, err := strconv.Atoi(last)
		if err != nil || index < 0 || index >= len(container) {
			return fmt.Errorf("invalid fixture pointer %q", pointer)
		}
		container[index] = value
	default:
		return fmt.Errorf("invalid fixture pointer %q", pointer)
	}
	return nil
}

func normalizeFixtureNumber(value any) any {
	switch value := value.(type) {
	case json.Number:
		integer, err := value.Int64()
		if err != nil {
			return value.String()
		}
		return integer
	case []any:
		for index := range value {
			value[index] = normalizeFixtureNumber(value[index])
		}
		return value
	case map[string]any:
		for key := range value {
			value[key] = normalizeFixtureNumber(value[key])
		}
		return value
	default:
		return value
	}
}
