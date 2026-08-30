package publication

import (
	"bytes"
	"slices"
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

func TestManifestRejectsUnsafePathsRolesAndCollisions(t *testing.T) {
	base := Manifest{
		Profile:      ManifestProfile,
		LedgerSchema: SchemaPin{Version: "1.0.0", Commit: "commit", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		LedgerPath:   "ledger/example.yaml",
		Entries: []Entry{{
			Role: RoleLedger, Path: "ledger/example.yaml", Size: 10,
			Digest: Digest{Algorithm: "sha-256", Value: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		}},
	}
	tests := map[string]func(*Manifest){
		"traversal":     func(manifest *Manifest) { manifest.Entries[0].Path = "ledger/../example.yaml" },
		"wrong ledger":  func(manifest *Manifest) { manifest.LedgerPath = "ledger/other.yaml" },
		"wrong role":    func(manifest *Manifest) { manifest.Entries[0].Role = RoleTarget },
		"unknown role":  func(manifest *Manifest) { manifest.Entries[0].Role = "secret" },
		"wrong digest":  func(manifest *Manifest) { manifest.Entries[0].Digest.Value = "ABC" },
		"negative size": func(manifest *Manifest) { manifest.Entries[0].Size = -1 },
		"duplicate":     func(manifest *Manifest) { manifest.Entries = append(manifest.Entries, manifest.Entries[0]) },
		"case collision": func(manifest *Manifest) {
			manifest.Entries = append(manifest.Entries, Entry{Role: RoleTarget, Path: "Ledger/Example.yaml", Digest: manifest.Entries[0].Digest})
			slices.SortFunc(manifest.Entries, func(a, b Entry) int {
				if a.Path < b.Path {
					return -1
				}
				if a.Path > b.Path {
					return 1
				}
				return 0
			})
		},
		"request path": func(manifest *Manifest) {
			manifest.Entries = append(manifest.Entries, Entry{Role: RoleRequest, Path: "proofs/targets/request.tsq", Digest: manifest.Entries[0].Digest})
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			manifest := base
			manifest.Entries = append([]Entry(nil), base.Entries...)
			mutate(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("unsafe manifest succeeded")
			}
		})
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
