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
		"empty-ledger.json":               "80d0542b2429d50531c7cd43969799311688630840d88195156a8a52505ab710",
		"forecast-seal-v1.json":           "59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5",
		"individual-ledger.json":          "54a22c6d154d864ed0dee85ee2f2ba7e985354e722690127c3606c8bfb582fd4",
		"invalid-cases.json":              "5999288b3eb0bfd4fc99bd8a3bdc2f6520d245bac41845976e8d09aa922c758f",
		"question-without-forecasts.yaml": "065d8acde34c2918bccc289a06f267ec0db79ee4c379f603458e8cbb7f79f7dd",
		"team-ledger.yaml":                "b4d739d8f0730eeea4c147f44cd622f497e5be59a682b85dc44ffb6c6c28a5b9",
	}
	for name, expected := range fixtures {
		data, err := fs.ReadFile(Conformance(), name)
		if err != nil {
			t.Fatalf("read embedded fixture %s: %v", name, err)
		}
		assertDigest(t, name, data, expected)
	}

	if ReleaseArchiveSHA256 != "5081c740cef4c0063a77a7e4aa51e142d355a30c09d41be9d4acfd8f7356ef8e" {
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
