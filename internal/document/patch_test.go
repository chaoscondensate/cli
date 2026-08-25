package document

import (
	"errors"
	"strings"
	"testing"
)

func TestReplaceScalarsPreservesUntouchedJSONBytes(t *testing.T) {
	input := "{\n  \"known\" : \"old\",\n  \"unknown\": { \"presentation\" : [1,2,3] },\n  \"count\": 1\n}\n"
	doc, err := ParseJSON(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplaceScalars(doc, []ScalarEdit{
		{Pointer: "/known", Value: "new"},
		{Pointer: "/count", Value: int64(2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"known\" : \"new\",\n  \"unknown\": { \"presentation\" : [1,2,3] },\n  \"count\": 2\n}\n"
	if string(got) != want {
		t.Fatalf("patched JSON:\n%s\nwant:\n%s", got, want)
	}
}

func TestReplaceScalarsPreservesUntouchedYAMLPresentation(t *testing.T) {
	input := "# heading\r\ntitle: 'Old title' # keep\r\ncount: 1\r\ndescription: |\r\n  old line\r\nnext: [keep, style]\r\n"
	doc, err := ParseYAML(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReplaceScalars(doc, []ScalarEdit{
		{Pointer: "/title", Value: "New's title"},
		{Pointer: "/count", Value: int64(2)},
		{Pointer: "/description", Value: "first\nsecond\n"},
	})
	if err != nil {
		t.Fatal(err)
	}
	output := string(got)
	for _, untouched := range []string{"# heading\r\n", " # keep\r\n", "next: [keep, style]\r\n"} {
		if !strings.Contains(output, untouched) {
			t.Fatalf("untouched presentation %q was lost:\n%s", untouched, output)
		}
	}
	if !strings.Contains(output, "title: 'New''s title' # keep") {
		t.Fatalf("single-quoted style was not retained:\n%s", output)
	}
	if !strings.Contains(output, "description: |\r\n  first\r\n  second\r\n") {
		t.Fatalf("literal block style/newlines were not retained:\n%s", output)
	}
	parsed, err := ParseYAML(strings.NewReader(output), DefaultLimits)
	if err != nil {
		t.Fatalf("patched YAML does not parse: %v", err)
	}
	value, err := lookupValue(parsed.Root, "/description")
	if err != nil || value.String != "first\nsecond\n" {
		t.Fatalf("replacement semantic value = %q, err=%v", value.String, err)
	}
}

func TestReplaceScalarsRejectsContainersAliasesAndMissingPointers(t *testing.T) {
	doc, err := ParseYAML(strings.NewReader("source: &a value\ncopy: *a\nitems: [one]\n"), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	for _, edit := range []ScalarEdit{
		{Pointer: "/copy", Value: "changed"},
		{Pointer: "/items", Value: "changed"},
		{Pointer: "/missing", Value: "changed"},
	} {
		_, err := ReplaceScalars(doc, []ScalarEdit{edit})
		if err == nil {
			t.Fatalf("unsafe edit %#v accepted", edit)
		}
		if edit.Pointer != "/missing" && !errors.Is(err, ErrUnsupportedPatch) {
			t.Fatalf("edit %#v: got %v, want ErrUnsupportedPatch", edit, err)
		}
	}
}
