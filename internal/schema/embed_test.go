package schema

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"testing"
)

func TestPinnedContractDigests(t *testing.T) {
	t.Parallel()

	assertDigest(t, "forecast-ledger.schema.json", Contract(), SchemaSHA256)

	fixtures := map[string]string{
		"forecast-seal-v1.json":  "59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5",
		"individual-ledger.json": "b05d3ad403ba85d962e1f8d1e6219b789ff763b81928f120b90926603b67dd68",
		"invalid-cases.json":     "a7e0275d216f8f81285bfcb2c37e4095395fa2f7bc95b3ebdce51f55d7c29d59",
		"team-ledger.yaml":       "fc42a7b70c5cef89e6524cf45f8c7be07bedaf1c1368eed761739d835200e4c1",
	}
	for name, expected := range fixtures {
		data, err := fs.ReadFile(Conformance(), name)
		if err != nil {
			t.Fatalf("read embedded fixture %s: %v", name, err)
		}
		assertDigest(t, name, data, expected)
	}

	if ReleaseArchiveSHA256 != "a3d6afcf8a3cd9b9e9a650ebac684cbe2f155a81db309797d77694b5f4b9bbda" {
		t.Fatalf("unexpected release archive digest %q", ReleaseArchiveSHA256)
	}
}

func assertDigest(t *testing.T, name string, data []byte, expected string) {
	t.Helper()
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual != expected {
		t.Fatalf("%s SHA-256 = %s, want %s", name, actual, expected)
	}
}
