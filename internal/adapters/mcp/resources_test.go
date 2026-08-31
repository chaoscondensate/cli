package mcp

import (
	"strings"
	"testing"
)

func TestResourceMarshalingRedactsSecretAndRawByteCanaries(t *testing.T) {
	encoded, err := marshalResourceData(map[string]any{
		"ledger_id": "public-ledger",
		"nested":    map[string]any{"access_token": "CANARY-TOKEN"},
		"bytes":     []byte("CANARY-BYTES"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encoded, "CANARY") || !strings.Contains(encoded, `"access_token":"[redacted]"`) || !strings.Contains(encoded, `"bytes":"[redacted bytes]"`) {
		t.Fatalf("unsafe resource JSON: %s", encoded)
	}
}
