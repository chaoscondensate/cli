package canonical

import (
	"errors"
	"testing"
)

func TestMarshalUsesUTF16OrderAndMinimalStrings(t *testing.T) {
	value := map[string]any{
		"\U0001f600": "emoji",
		"\ufffd":     "replacement",
		"text":       "<&\n",
		"integer":    int64(42),
	}
	got, err := Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"integer":42,"text":"<&\n","😀":"emoji","�":"replacement"}`
	if string(got) != want {
		t.Fatalf("canonical bytes:\n got %s\nwant %s", got, want)
	}
}

func TestMarshalRejectsFloatsAndUnsafeIntegers(t *testing.T) {
	for _, value := range []any{1.0, int64(9_007_199_254_740_992), uint64(9_007_199_254_740_992)} {
		_, err := Marshal(value)
		if !errors.Is(err, ErrUnsupportedValue) {
			t.Fatalf("value %#v: got %v, want ErrUnsupportedValue", value, err)
		}
	}
}
