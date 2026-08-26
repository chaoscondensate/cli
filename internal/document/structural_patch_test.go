package document

import (
	"strings"
	"testing"
)

func TestApplyPatchPreservesUnchangedJSONBytesAcrossStructuralEdits(t *testing.T) {
	input := "{\r\n  \"title\": \"Old\",\r\n  \"platforms\": {\"old\": {\"kind\":\"informal\"}},\r\n  \"questions\": [{\"id\":\"q-one\"}],\r\n  \"untouched\" : { \"odd spacing\" : true }\r\n}\r\n"
	doc, err := ParseJSON(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyPatch(doc, []PatchOperation{
		{Kind: PatchReplace, Pointer: "/title", Value: "New"},
		{Kind: PatchAdd, Pointer: "/platforms/new", Value: map[string]any{"kind": "internal"}},
		{Kind: PatchRemove, Pointer: "/platforms/old"},
		{Kind: PatchAdd, Pointer: "/questions/-", Value: map[string]any{"id": "q-two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := string(got)
	if !strings.Contains(output, `"untouched" : { "odd spacing" : true }`) || !strings.Contains(output, "\r\n") {
		t.Fatalf("untouched JSON presentation changed:\n%s", output)
	}
	if strings.Contains(output, `"old"`) || !strings.Contains(output, `"new"`) || !strings.Contains(output, `"q-two"`) {
		t.Fatalf("structural JSON edits missing:\n%s", output)
	}
}

func TestApplyPatchPreservesYAMLCommentsOrderStyleAndCRLF(t *testing.T) {
	input := "# ledger\r\ntitle: 'Old' # title\r\nplatforms:\r\n  old: {kind: informal}\r\nquestions:\r\n  - id: q-one\r\nuntouched: |\r\n  keep this\r\n# tail\r\n"
	doc, err := ParseYAML(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyPatch(doc, []PatchOperation{
		{Kind: PatchReplace, Pointer: "/title", Value: "New's"},
		{Kind: PatchAdd, Pointer: "/platforms/new", Value: map[string]any{"kind": "internal"}},
		{Kind: PatchRemove, Pointer: "/platforms/old"},
		{Kind: PatchAdd, Pointer: "/questions/-", Value: map[string]any{"id": "q-two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := string(got)
	for _, unchanged := range []string{"# ledger\r\n", " # title\r\n", "untouched: |\r\n  keep this\r\n", "# tail\r\n"} {
		if !strings.Contains(output, unchanged) {
			t.Fatalf("untouched YAML bytes %q changed:\n%s", unchanged, output)
		}
	}
	if !strings.Contains(output, "title: 'New''s'") || strings.Contains(output, "  old:") || !strings.Contains(output, "  new:") || !strings.Contains(output, "q-two") {
		t.Fatalf("structural YAML edits missing:\n%s", output)
	}
}

func TestApplyPatchCanAddRemoveOptionalFieldsAndReplaceSubtrees(t *testing.T) {
	input := "root:\n  keep: yes\n  remove: old\n  replace: {old: value}\n"
	doc, err := ParseYAML(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ApplyPatch(doc, []PatchOperation{
		{Kind: PatchRemove, Pointer: "/root/remove"},
		{Kind: PatchAdd, Pointer: "/root/added", Value: "new"},
		{Kind: PatchReplace, Pointer: "/root/replace", Value: map[string]any{"new": "value"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := string(got)
	if !strings.Contains(output, "  keep: yes\n") || strings.Contains(output, "remove:") || !strings.Contains(output, "added: new") || !strings.Contains(output, `"new":"value"`) {
		t.Fatalf("unexpected patched YAML:\n%s", output)
	}
}
