package publication

import (
	"bytes"
	"testing"
)

func TestManifestCanonicalRoundTripAndClosedRoles(t *testing.T) {
	manifest := Manifest{Profile: ManifestProfile, LedgerSchema: SchemaPin{Version: "1.0.0", Commit: "commit", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, LedgerPath: "ledger/example.yaml", Entries: []Entry{{Role: RoleLedger, Path: "ledger/example.yaml", Size: 10, Digest: Digest{Algorithm: "sha-256", Value: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}}}
	encoded, err := Encode(manifest)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(encoded)
	if err != nil || decoded.LedgerPath != manifest.LedgerPath {
		t.Fatalf("Decode = %#v, %v", decoded, err)
	}
	if _, err := Decode(bytes.Replace(encoded, []byte(RoleLedger), []byte("secret"), 1)); err == nil {
		t.Fatal("unknown manifest role accepted")
	}
}

func FuzzManifestDecode(f *testing.F) {
	manifest := Manifest{Profile: ManifestProfile, LedgerSchema: SchemaPin{Version: "1.0.0", Commit: "commit", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, LedgerPath: "ledger/example.json", Entries: []Entry{{Role: RoleLedger, Path: "ledger/example.json", Size: 1, Digest: Digest{Algorithm: "sha-256", Value: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}}}}
	seed, _ := Encode(manifest)
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxManifestBytes {
			return
		}
		_, _ = Decode(data)
	})
}
