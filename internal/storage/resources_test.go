package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestResourceRecoveryUsesOwnershipAndRollbackClass(t *testing.T) {
	directory := t.TempDir()
	packageRoot := filepath.Join(directory, "package")
	if err := os.Mkdir(packageRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, "target.json")
	receipt := filepath.Join(directory, "receipt.ots")
	key := filepath.Join(directory, "key.json")
	ledger := filepath.Join(directory, "ledger.json")
	packageFile := filepath.Join(packageRoot, "manifest.json")
	unowned := filepath.Join(directory, "existing.txt")
	files := map[string][]byte{
		target: []byte("target"), receipt: []byte("receipt"), key: []byte("key"),
		ledger: []byte("ledger"), packageFile: []byte("manifest"), unowned: []byte("existing"),
	}
	for path, data := range files {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	journal := filepath.Join(directory, ".resources.json")
	plan, err := NewResourcePlan(journal, "timestamp.stamp", []ResourceEntry{
		{Kind: ResourceTarget, Type: ResourceFile, Path: target, Owned: true, Rollback: ResourceRollbackRemoveOwned, State: ResourcePlanned},
		{Kind: ResourceReceipt, Type: ResourceFile, Path: receipt, Owned: true, Rollback: ResourceRollbackRemoveOwned, State: ResourcePlanned},
		{Kind: ResourceKey, Type: ResourceFile, Path: key, Owned: true, Rollback: ResourceRollbackRetainSecret, State: ResourcePlanned},
		{Kind: ResourceLedger, Type: ResourceFile, Path: ledger, Owned: false, Rollback: ResourceRollbackNone, State: ResourcePlanned},
		{Kind: ResourcePackage, Type: ResourceFile, Path: packageFile, Owned: true, Rollback: ResourceRollbackRemoveOwned, State: ResourcePlanned},
		{Kind: ResourcePackage, Type: ResourceDirectory, Path: packageRoot, Owned: true, Rollback: ResourceRollbackRemoveOwned, State: ResourcePlanned},
		{Kind: ResourceTarget, Type: ResourceFile, Path: unowned, Owned: false, Rollback: ResourceRollbackNone, State: ResourcePlanned},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Begin(); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string][]byte{target: files[target], receipt: files[receipt], packageFile: files[packageFile]} {
		if err := plan.MarkCreated(path, ResourceDigest(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := plan.MarkCreated(packageRoot, ""); err != nil {
		t.Fatal(err)
	}
	if err := plan.MarkCreated(key, ResourceDigest(files[key])); err != nil {
		t.Fatal(err)
	}
	if err := plan.MarkReplaced(ledger, ResourceDigest(files[ledger])); err != nil {
		t.Fatal(err)
	}
	report, err := RecoverResourcePlan(context.Background(), journal)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Removed) != 4 || len(report.Retained) != 2 {
		t.Fatalf("recovery report = %#v", report)
	}
	for _, retained := range []string{key, ledger, unowned} {
		if _, err := os.Stat(retained); err != nil {
			t.Fatalf("retained resource %s: %v", retained, err)
		}
	}
	for _, removed := range []string{target, receipt, packageFile, packageRoot} {
		if _, err := os.Stat(removed); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("owned rollback resource %s remains: %v", removed, err)
		}
	}
}

func TestResourceRecoveryRetainsChangedOwnedFileAndJournal(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "target.json")
	if err := os.WriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	journal := filepath.Join(directory, ".resources.json")
	plan, err := NewResourcePlan(journal, "target.build", []ResourceEntry{{
		Kind: ResourceTarget, Type: ResourceFile, Path: path, Owned: true,
		Rollback: ResourceRollbackRemoveOwned, State: ResourcePlanned,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Begin(); err != nil {
		t.Fatal(err)
	}
	if err := plan.MarkCreated(path, ResourceDigest([]byte("first"))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = RecoverResourcePlan(context.Background(), journal)
	if app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("changed recovery error = %v", err)
	}
	if data, readErr := os.ReadFile(path); readErr != nil || string(data) != "changed" {
		t.Fatalf("changed resource was removed: data=%q err=%v", data, readErr)
	}
	if _, statErr := os.Stat(journal); statErr != nil {
		t.Fatalf("journal was removed after conflict: %v", statErr)
	}
}

func TestResourceCrashRecoveryAndRetryNeverChangesUnownedFiles(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	targetPath := filepath.Join(directory, "target.json")
	receiptPath := filepath.Join(directory, "receipt.ots")
	keyPath := filepath.Join(directory, "key.json")
	journalPath := filepath.Join(directory, ".stamp-resources.json")
	ledgerBytes := []byte("original ledger\n")
	targetBytes := []byte("deterministic target")
	receiptBytes := []byte("pending receipt")
	keyBytes := []byte("durable key")
	for path, content := range map[string][]byte{
		ledgerPath: ledgerBytes, targetPath: targetBytes, receiptPath: receiptBytes, keyPath: keyBytes,
	} {
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := NewResourcePlan(journalPath, "timestamp.stamp", []ResourceEntry{
		{Kind: ResourceLedger, Type: ResourceFile, Path: ledgerPath, Owned: false, Rollback: ResourceRollbackNone, State: ResourcePlanned},
		{Kind: ResourceTarget, Type: ResourceFile, Path: targetPath, Owned: false, Rollback: ResourceRollbackNone, State: ResourcePlanned},
		{Kind: ResourceReceipt, Type: ResourceFile, Path: receiptPath, Owned: true, Rollback: ResourceRollbackRemoveOwned, State: ResourcePlanned},
		{Kind: ResourceKey, Type: ResourceFile, Path: keyPath, Owned: true, Rollback: ResourceRollbackRetainSecret, State: ResourcePlanned},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Begin(); err != nil {
		t.Fatal(err)
	}
	if err := plan.MarkCreated(receiptPath, ResourceDigest(receiptBytes)); err != nil {
		t.Fatal(err)
	}
	if err := plan.MarkCreated(keyPath, ResourceDigest(keyBytes)); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverResourcePlan(context.Background(), journalPath); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{ledgerPath: ledgerBytes, targetPath: targetBytes, keyPath: keyBytes} {
		got, readErr := os.ReadFile(path)
		if readErr != nil || string(got) != string(want) {
			t.Fatalf("retained %s = %q, want %q, err=%v", path, got, want, readErr)
		}
	}
	if _, err := os.Stat(receiptPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created receipt survived rollback: %v", err)
	}

	targetResult, err := EnsureDeterministicFile(targetPath, targetBytes, 0o600, 1024)
	if err != nil || targetResult.State != DeterministicUnchanged {
		t.Fatalf("retry target result=%#v err=%v", targetResult, err)
	}
	receiptResult, err := EnsureDeterministicFile(receiptPath, receiptBytes, 0o600, 1024)
	if err != nil || receiptResult.State != DeterministicCreated {
		t.Fatalf("retry receipt result=%#v err=%v", receiptResult, err)
	}
	if got, err := os.ReadFile(ledgerPath); err != nil || string(got) != string(ledgerBytes) {
		t.Fatalf("retry changed original ledger: data=%q err=%v", got, err)
	}
}
