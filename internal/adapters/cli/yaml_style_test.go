package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestApplicationWrittenYAMLUsesBlockStyleForPopulatedCollections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.yaml")
	commands := [][]string{
		{"forecast-ledger", "init", "--file", path, "--ledger-id", "yaml-style", "--timezone", "Europe/London", "--forecaster-id", "owner", "--forecaster-name", "Owner", "--initial-platform", "internal,Internal,internal,https://example.com"},
		{"forecast-ledger", "question", "add", "--file", path, "--question", "q-style", "--type", "binary", "--title", "Will it happen?", "--resolution-criteria", "Use the named public result.", "--expected-resolution-at", "10 Aug 2030", "--tag", "review", "--initial-forecast", "f-style-001", "--initial-value-kind", "binary", "--initial-probability-bp", "5100", "--initial-key-factor", "First factor", "--initial-key-factor", "Second factor"},
		{"forecast-ledger", "forecast", "add", "--file", path, "--question", "q-style", "--forecast", "f-style-002", "--value-kind", "binary", "--probability-bp", "5200", "--key-factor", "Third factor", "--supersedes-forecast", "f-style-001"},
		{"forecast-ledger", "question", "add", "--file", path, "--question", "q-empty", "--type", "binary", "--title", "Backlog question", "--resolution-criteria", "Use the named public result.", "--expected-resolution-at", "11 Aug 2030"},
	}
	for _, arguments := range commands {
		code, _, stderr := runCLI(arguments...)
		if code != 0 {
			t.Fatalf("%s failed with %d: %s", strings.Join(arguments[1:3], " "), code, stderr)
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	assertNoPopulatedFlowCollections(t, &document, "$")
	if !strings.Contains(string(raw), "forecasts:\n") || !strings.Contains(string(raw), "forecasts: []") {
		t.Fatalf("expected expanded populated collections:\n%s", raw)
	}
}

func assertNoPopulatedFlowCollections(t *testing.T, node *yaml.Node, path string) {
	t.Helper()
	if (node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode) && len(node.Content) > 0 && node.Style&yaml.FlowStyle != 0 {
		t.Errorf("populated collection %s uses YAML flow style", path)
	}
	for index, child := range node.Content {
		assertNoPopulatedFlowCollections(t, child, path+"/"+child.Value)
		_ = index
	}
}
