package document

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

var fuzzLimits = Limits{
	MaxBytes:         4 << 10,
	MaxDepth:         16,
	MaxNodes:         256,
	MaxScalarBytes:   256,
	MaxAliases:       8,
	MaxExpandedNodes: 512,
}

func FuzzParseJSON(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"a":[1,true,null,"text"]}`),
		[]byte(`{"a":1,"\u0061":2}`),
		[]byte(`[[[[0]]]]`),
		{'"', 0xff, '"'},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		document, err := ParseJSON(bytes.NewReader(input), fuzzLimits)
		if err != nil {
			return
		}
		assertBoundedTree(t, document.Root, fuzzLimits.MaxNodes)
	})
}

func FuzzParseYAML(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("a: [1, true, null, text]\n"),
		[]byte("source: &a [1, 2]\ncopy: *a\n"),
		[]byte("cycle: &a [*a]\n"),
		[]byte("value: !!float .nan\n"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		document, err := ParseYAML(bytes.NewReader(input), fuzzLimits)
		if err != nil {
			return
		}
		assertBoundedTree(t, document.Root, fuzzLimits.MaxExpandedNodes)
	})
}

func FuzzApplyYAMLStructuralReplacement(f *testing.F) {
	for _, seed := range []struct {
		source      string
		replacement string
	}{
		{source: "target:\n  old: value\nuntouched: {keep: flow}\n", replacement: `{"first":"one","second":{"nested":true}}`},
		{source: "target: {old: value}\nuntouched: 'quoted'\n", replacement: `{"first":"one","second":"two"}`},
		{source: "target:\r\n  - old\r\nuntouched: [keep, flow]\r\n", replacement: `[{"id":"one"},{"id":"two"}]`},
	} {
		f.Add([]byte(seed.source), []byte(seed.replacement))
	}
	f.Fuzz(func(t *testing.T, source, replacement []byte) {
		doc, err := ParseYAML(bytes.NewReader(source), fuzzLimits)
		if err != nil {
			return
		}
		original := bytes.Clone(doc.Raw)
		beforeUntouched, beforeErr := lookupValue(doc.Root, "/untouched")
		if beforeErr != nil {
			return
		}
		valueDoc, err := ParseJSON(bytes.NewReader(replacement), fuzzLimits)
		if err != nil {
			return
		}
		result, err := ApplyPatch(doc, []PatchOperation{{Kind: PatchReplace, Pointer: "/target", Value: Ordered(valueDoc.Root)}})
		if !bytes.Equal(doc.Raw, original) {
			t.Fatal("ApplyPatch mutated the source document on return")
		}
		if err != nil {
			return
		}
		parsed, err := ParseYAML(bytes.NewReader(result), fuzzLimits)
		if err != nil {
			t.Fatalf("successful replacement did not reparse: %v", err)
		}
		afterUntouched, err := lookupValue(parsed.Root, "/untouched")
		if err != nil || !reflect.DeepEqual(beforeUntouched.Any(), afterUntouched.Any()) {
			t.Fatalf("untouched value changed: before=%#v after=%#v err=%v", beforeUntouched.Any(), afterUntouched.Any(), err)
		}
		if strings.Contains(string(source), "\r\n") && strings.Contains(strings.ReplaceAll(string(result), "\r\n", ""), "\n") {
			t.Fatal("successful replacement introduced bare LF into CRLF source")
		}
	})
}

func assertBoundedTree(t *testing.T, root *Value, max int) {
	t.Helper()
	count := 0
	stack := []*Value{root}
	for len(stack) > 0 {
		value := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		count++
		if count > max {
			t.Fatalf("returned tree exceeds limit: %d > %d", count, max)
		}
		switch value.Kind {
		case ValueArray:
			stack = append(stack, value.Array...)
		case ValueObject:
			for _, member := range value.Object {
				stack = append(stack, member.Value)
			}
		case ValueNull, ValueBool, ValueInt, ValueString:
		default:
			t.Fatalf("unknown value kind %q", value.Kind)
		}
	}
}
