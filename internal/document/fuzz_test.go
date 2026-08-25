package document

import (
	"bytes"
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
