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
	assertDigest(t, "LICENSE", License(), "7084b3fb14e3a306691af23e58ab0ccfa336b202853740f5e1ea0ebab39cacf2")

	fixtures := map[string]string{
		"empty-ledger.json":               "d7718493a5dcdb4d6af8ce398e103788adc9c43739916eb82475af0ab0617426",
		"forecast-seal-v1.json":           "59f3996b22e135d5c2d1a6977c2e5dfa025d3f2ececd226b6fb4096ddc7272f5",
		"individual-ledger.json":          "77cb761c714e36a31347e1d8630c99a0c73cf2ae3425438faef5a060e436828c",
		"invalid-cases.json":              "d0e4b8036bf9119bb402687b753740ea8dcbd2d8dbef49b7673b7745add2cfee",
		"question-without-forecasts.yaml": "7b53e9db6cdd669562b94136ed5a7a882d27d8b43bfc752de6e37894a9aeee5a",
		"team-ledger.yaml":                "ed00a6bfc47711727bc9d40c1c40fb0e5a0efa08a94cf6d1b8912ecfeca65386",
	}
	for name, expected := range fixtures {
		data, err := fs.ReadFile(Conformance(), name)
		if err != nil {
			t.Fatalf("read embedded fixture %s: %v", name, err)
		}
		assertDigest(t, name, data, expected)
	}

	if ReleaseArchiveSHA256 != "3b6b9f274a67d2714edaa308f9aad51b218dbf24ed95de1a1340292ad1df1f2a" {
		t.Fatalf("unexpected release archive digest %q", ReleaseArchiveSHA256)
	}
	if Version != "1.3.0" || Commit != "32218f682b3a650f41153e98817473bf429973a7" || AnnotatedTagObject != "d3d1f06a7f27501b1419eaf78fc4a48e51de9ee3" {
		t.Fatalf("unexpected upstream identity %q %q %q", Version, Commit, AnnotatedTagObject)
	}
	if ReleaseChecksumsSHA256 != "6042508976246ddc62974ad3054dca9885525024d4bb543572b75b23c60ac284" {
		t.Fatalf("unexpected release checksums digest %q", ReleaseChecksumsSHA256)
	}
}

func assertDigest(t *testing.T, name string, data []byte, expected string) {
	t.Helper()
	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual != expected {
		t.Fatalf("%s SHA-256 = %s, want %s", name, actual, expected)
	}
}
