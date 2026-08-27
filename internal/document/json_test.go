package document

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSourceRefOmitsUnknownSpanAndKeepsKnownOneBasedSpan(t *testing.T) {
	unknown, err := json.Marshal(SourceRef{Pointer: "/value"})
	if err != nil || strings.Contains(string(unknown), "start") || strings.Contains(string(unknown), `"line":0`) {
		t.Fatalf("unknown source = %s, %v", unknown, err)
	}
	known, err := json.Marshal(SourceRef{Pointer: "/value", Start: Position{Offset: 4, Line: 2, Column: 3}})
	if err != nil || !strings.Contains(string(known), `"line":2`) || !strings.Contains(string(known), `"column":3`) {
		t.Fatalf("known source = %s, %v", known, err)
	}
}

func TestParseJSONBuildsTypedTreeAndLocations(t *testing.T) {
	input := "{\r\n  \"name\": \"прогноз\",\r\n  \"values\": [1, true, null]\r\n}\r\n"
	doc, err := ParseJSON(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Root.Kind != ValueObject || len(doc.Root.Object) != 2 {
		t.Fatalf("unexpected root: %#v", doc.Root)
	}
	if got := doc.Root.Object[1].Value.Array[0].Int; got != 1 {
		t.Fatalf("integer = %d, want 1", got)
	}
	if got := doc.Locations["/values/1"][0].Start.Line; got != 3 {
		t.Fatalf("line = %d, want 3", got)
	}
	if doc.Newlines.CRLF != 4 || doc.Newlines.LF != 0 || !doc.Newlines.FinalNewline {
		t.Fatalf("unexpected newline info: %#v", doc.Newlines)
	}
	if string(doc.Raw) != input {
		t.Fatal("raw source was not retained exactly")
	}
}

func TestParseJSONRejectsDecodedDuplicateKey(t *testing.T) {
	_, err := ParseJSON(strings.NewReader(`{"a":1,"\u0061":2}`), DefaultLimits)
	parseErr := requireParseCode(t, err, "document.duplicate_key")
	if parseErr.Diagnostic.Location.Pointer != "/a" || len(parseErr.Diagnostic.Related) != 1 {
		t.Fatalf("unexpected duplicate diagnostic: %#v", parseErr.Diagnostic)
	}
	if parseErr.Diagnostic.Related[0].Location.Start.Offset >= parseErr.Diagnostic.Location.Start.Offset {
		t.Fatal("first duplicate location must precede the second")
	}
}

func TestParseJSONRejectsFloatsAndUnsafeIntegers(t *testing.T) {
	for _, input := range []string{`1.0`, `1e0`, `-2E+3`} {
		_, err := ParseJSON(strings.NewReader(input), DefaultLimits)
		requireParseCode(t, err, "document.float_not_allowed")
	}
	for _, input := range []string{`9007199254740992`, `-9007199254740992`, `999999999999999999999999999`} {
		_, err := ParseJSON(strings.NewReader(input), DefaultLimits)
		requireParseCode(t, err, "document.unsafe_integer")
	}
	for _, input := range []string{`9007199254740991`, `-9007199254740991`, `-0`} {
		if _, err := ParseJSON(strings.NewReader(input), DefaultLimits); err != nil {
			t.Fatalf("safe integer %s rejected: %v", input, err)
		}
	}
}

func TestParseJSONRejectsInvalidUnicodeAndMultipleRoots(t *testing.T) {
	_, err := ParseJSON(strings.NewReader(`"\uD800"`), DefaultLimits)
	requireParseCode(t, err, "document.syntax")

	_, err = ParseJSON(strings.NewReader("{} {}"), DefaultLimits)
	requireParseCode(t, err, "document.syntax")

	_, err = ParseJSON(strings.NewReader(string([]byte{'"', 0xff, '"'})), DefaultLimits)
	requireParseCode(t, err, "document.invalid_utf8")
}

func TestParseJSONLimitsAtBoundary(t *testing.T) {
	limits := DefaultLimits
	limits.MaxBytes = 2
	if _, err := ParseJSON(strings.NewReader(`[]`), limits); err != nil {
		t.Fatal(err)
	}
	_, err := ParseJSON(strings.NewReader(`[ ]`), limits)
	requireParseCode(t, err, "document.too_large")

	limits = DefaultLimits
	limits.MaxDepth = 2
	if _, err := ParseJSON(strings.NewReader(`[0]`), limits); err != nil {
		t.Fatal(err)
	}
	_, err = ParseJSON(strings.NewReader(`[[0]]`), limits)
	requireParseCode(t, err, "document.too_deep")

	limits = DefaultLimits
	limits.MaxNodes = 3 // object, key, value
	if _, err := ParseJSON(strings.NewReader(`{"a":1}`), limits); err != nil {
		t.Fatal(err)
	}
	limits.MaxNodes = 2
	_, err = ParseJSON(strings.NewReader(`{"a":1}`), limits)
	requireParseCode(t, err, "document.too_many_nodes")
}

func TestJSONPointerEscaping(t *testing.T) {
	doc, err := ParseJSON(strings.NewReader(`{"a/b":{"~x":1}}`), DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Locations["/a~1b/~0x"]; !ok {
		t.Fatalf("escaped pointer missing: %#v", doc.Locations)
	}
}

func requireParseCode(t *testing.T, err error, code string) *ParseError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error", code)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error %T is not ParseError: %v", err, err)
	}
	if parseErr.Diagnostic.Code != code {
		t.Fatalf("code = %q, want %q (%v)", parseErr.Diagnostic.Code, code, err)
	}
	return parseErr
}
