package service

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestCommitRootMetadataFileUpdatePreservesUnrelatedYAMLBytes(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "team-ledger.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.yaml")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	input := RootMetadataPatchInput{
		Title: Optional[string]{Set: true, Value: "Updated team ledger"},
		Forecaster: Optional[ForecasterMetadataPatchInput]{Set: true, Value: ForecasterMetadataPatchInput{
			Name: Optional[string]{Set: true, Value: "Current Team Name"},
		}},
	}
	result, err := CommitRootMetadataFileUpdate(context.Background(), path, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || len(result.ChangedPointers) != 2 || result.BeforeSHA256 == result.AfterSHA256 {
		t.Fatalf("result = %#v", result)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeQuestions := raw[bytes.Index(raw, []byte("questions:\n")):]
	afterQuestions := after[bytes.Index(after, []byte("questions:\n")):]
	if !bytes.Equal(beforeQuestions, afterQuestions) {
		t.Fatal("question bytes changed during YAML metadata update")
	}
	if !strings.HasPrefix(string(after), "# yaml-language-server:") || !strings.Contains(string(after), "title: Updated team ledger") || !strings.Contains(string(after), "name: Current Team Name") {
		t.Fatalf("YAML metadata update missing or presentation changed:\n%s", after)
	}
	loaded, err := LoadAndValidateLedger(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Model.Forecaster.ID != ledger.Slug("example-research-team") || loaded.Model.Publication != nil && loaded.Model.Publication.History == "" {
		t.Fatalf("immutable metadata changed: %#v", loaded.Model)
	}
}

func TestPlanRootMetadataFileUpdateDoesNotWrite(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := PlanRootMetadataFileUpdate(context.Background(), path, RootMetadataPatchInput{DefaultTimezone: Optional[string]{Set: true, Value: "UTC"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Fatal("planned change reported unchanged")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, raw) {
		t.Fatal("plan changed ledger bytes")
	}
}
