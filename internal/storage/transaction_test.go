package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
)

func TestUpdateLedgerValidatesTwiceAndPreservesPresentation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.yaml")
	original := "# keep\ntitle: 'Before'\ncount: 1\n"
	if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}
	validations := 0
	err := UpdateLedger(context.Background(), path, TransactionOptions{
		Validate: func(parsed *document.Document) error {
			validations++
			if parsed.Root.Kind != document.ValueObject {
				return errors.New("root is not an object")
			}
			return nil
		},
		Mutate: func(parsed *document.Document) ([]byte, error) {
			return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/title", Value: "After"}})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if validations != 2 {
		t.Fatalf("validations = %d, want 2", validations)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# keep\ntitle: 'After'\ncount: 1\n" {
		t.Fatalf("unexpected ledger bytes:\n%s", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
	if _, err := os.Stat(JournalPath(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("journal remains after success: %v", err)
	}
	assertNoTransactionTemps(t, filepath.Dir(path))
}

func TestUpdateLedgerRollsBackOnPreAndPostValidationFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	original := []byte("{\"value\":1}\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	mutated := false
	err := UpdateLedger(context.Background(), path, TransactionOptions{
		Validate: func(*document.Document) error { return errors.New("invalid before") },
		Mutate: func(*document.Document) ([]byte, error) {
			mutated = true
			return nil, nil
		},
	})
	if app.ErrorCodeOf(err) != app.CodeInvalidData || mutated {
		t.Fatalf("pre-validation err=%v mutated=%v", err, mutated)
	}

	validations := 0
	err = UpdateLedger(context.Background(), path, TransactionOptions{
		Validate: func(*document.Document) error {
			validations++
			if validations == 2 {
				return errors.New("invalid after")
			}
			return nil
		},
		Mutate: func(parsed *document.Document) ([]byte, error) {
			return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/value", Value: int64(2)}})
		},
	})
	if app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("post-validation err=%v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != string(original) {
		t.Fatalf("ledger changed after failed validation: %s", data)
	}
	assertNoTransactionTemps(t, filepath.Dir(path))
}

func TestRecoveryCompletesCrashAfterJournalSync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte("{\"value\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	crash := errors.New("simulated crash")
	err := UpdateLedger(context.Background(), path, TransactionOptions{
		Mutate: func(parsed *document.Document) ([]byte, error) {
			return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/value", Value: int64(2)}})
		},
		Fault: func(stage TransactionStage) error {
			if stage == StageJournalSynced {
				return crash
			}
			return nil
		},
	})
	if app.ErrorCodeOf(err) != app.CodeIO {
		t.Fatalf("injected crash err=%v", err)
	}
	beforeRecovery, _ := os.ReadFile(path)
	if string(beforeRecovery) != "{\"value\":1}\n" {
		t.Fatalf("original changed before recovery: %s", beforeRecovery)
	}
	if _, err := os.Stat(JournalPath(path)); err != nil {
		t.Fatalf("journal missing after crash: %v", err)
	}
	if err := RecoverLedger(context.Background(), path, 0, nil); err != nil {
		t.Fatal(err)
	}
	afterRecovery, _ := os.ReadFile(path)
	if string(afterRecovery) != "{\"value\":2}\n" {
		t.Fatalf("recovery did not install update: %s", afterRecovery)
	}
	if _, err := os.Stat(JournalPath(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("journal remains after recovery: %v", err)
	}
	assertNoTransactionTemps(t, filepath.Dir(path))
}

func TestTransactionRefusesExistingJournalAndChangedRecoveryTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte("{\"value\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := UpdateLedger(context.Background(), path, TransactionOptions{
		Mutate: func(parsed *document.Document) ([]byte, error) {
			return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/value", Value: int64(2)}})
		},
		Fault: func(stage TransactionStage) error {
			if stage == StageJournalSynced {
				return errors.New("stop")
			}
			return nil
		},
	})
	if err == nil {
		t.Fatal("fault did not stop transaction")
	}
	err = UpdateLedger(context.Background(), path, TransactionOptions{Mutate: func(parsed *document.Document) ([]byte, error) { return parsed.Raw, nil }})
	if app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("existing journal was ignored: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"value\":99}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = RecoverLedger(context.Background(), path, 0, nil)
	if app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("changed recovery target accepted: %v", err)
	}
}

func TestRecoveryCanRestoreTemporarilyMissingTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, []byte("{\"value\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := UpdateLedger(context.Background(), path, TransactionOptions{
		Mutate: func(parsed *document.Document) ([]byte, error) {
			return document.ReplaceScalars(parsed, []document.ScalarEdit{{Pointer: "/value", Value: int64(2)}})
		},
		Fault: func(stage TransactionStage) error {
			if stage == StageJournalSynced {
				return errors.New("stop")
			}
			return nil
		},
	})
	if err == nil {
		t.Fatal("fault did not stop transaction")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := RecoverLedger(context.Background(), path, 0, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "{\"value\":2}\n" {
		t.Fatalf("recovered data=%q err=%v", data, err)
	}
}

func assertNoTransactionTemps(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".forecast-ledger-") && strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("transaction temp remains: %s", entry.Name())
		}
	}
}
