package service

import "testing"

func TestOperationRegistryIsCompleteAndUnique(t *testing.T) {
	definitions := OperationDefinitions()
	if len(definitions) != 31 {
		t.Fatalf("operation definitions = %d, want 31", len(definitions))
	}
	names := make(map[OperationName]struct{}, len(definitions))
	tools := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if _, duplicate := names[definition.Name]; duplicate {
			t.Fatalf("duplicate operation %q", definition.Name)
		}
		names[definition.Name] = struct{}{}
		if definition.MCPTool == "" {
			t.Fatalf("operation %q has no MCP tool", definition.Name)
		}
		if _, duplicate := tools[definition.MCPTool]; duplicate {
			t.Fatalf("duplicate MCP tool %q", definition.MCPTool)
		}
		tools[definition.MCPTool] = struct{}{}
		if definition.InputSchema != "" {
			if _, err := InputSchema(definition.InputSchema); err != nil {
				t.Fatalf("operation %q input: %v", definition.Name, err)
			}
		}
	}
}
