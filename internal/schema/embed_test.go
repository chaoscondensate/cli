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
		"empty-ledger.json":               "e31ffdf26a63742871686c4cbf6a62ed26dd7289e33d644a315dedba1ddc4a1f",
		"forecast-seal-v1.json":           "59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5",
		"individual-ledger.json":          "1ce53fef071e017bf629c93f4b6304316321933d1b2b47ce95e89c4dde8a35ed",
		"invalid-cases.json":              "a7e0275d216f8f81285bfcb2c37e4095395fa2f7bc95b3ebdce51f55d7c29d59",
		"question-without-forecasts.yaml": "f4786c542a11d1bd411f03e6b3246fa3972e9ba1f90ab680262679d82b1d53a5",
		"team-ledger.yaml":                "68179c3b38ea7a54f0c7d3562c56e1890a975d10d57d7e2dc3c5d880cb04b6db",
	}
	for name, expected := range fixtures {
		data, err := fs.ReadFile(Conformance(), name)
		if err != nil {
			t.Fatalf("read embedded fixture %s: %v", name, err)
		}
		assertDigest(t, name, data, expected)
	}

	if ReleaseArchiveSHA256 != "edb2e307a7ce55984d17306556f0538f49a3a2a9fa66c9bfec973c90f0cb88dd" {
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
