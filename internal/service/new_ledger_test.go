package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
)

func TestCommitNewLedgerCreatesExclusiveJSONAndYAML(t *testing.T) {
	for _, extension := range []string{".json", ".yaml", ".yml"} {
		t.Run(extension, func(t *testing.T) {
			model := testPublicInitialLedger(t)
			path := filepath.Join(t.TempDir(), "ledger"+extension)
			resolved, err := CommitNewLedger(path, model)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := LoadAndValidateLedger(context.Background(), resolved, nil); err != nil {
				t.Fatal(err)
			}
			if _, err := CommitNewLedger(path, model); app.ErrorCodeOf(err) != app.CodeConflict {
				t.Fatalf("second create error = %v", err)
			}
		})
	}
}

func TestCommitInitialSealedFilesCreatesProtectedKeyBeforeLedger(t *testing.T) {
	build := testSealedInitialBuild(t)
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "forecast.key")
	commit, err := CommitInitialSealedFiles(context.Background(), ledgerPath, keyPath, build, InitialCommitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if commit.Recovery.State != RecoveryNone {
		t.Fatalf("recovery = %#v", commit.Recovery)
	}
	if err := storage.CheckProtectedFile(keyPath); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAndValidateLedger(context.Background(), ledgerPath, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storage.JournalPath(ledgerPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("journal remains: %v", err)
	}
}

func TestCommitInitialSealedFilesRetainsKeyAndReportsRecoveryOnLedgerFailure(t *testing.T) {
	build := testSealedInitialBuild(t)
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "forecast.key")
	commit, err := CommitInitialSealedFiles(context.Background(), ledgerPath, keyPath, build, InitialCommitOptions{Fault: func(stage InitialCommitStage) error {
		if stage == StageInitialKeyCreated {
			return errors.New("stop after key")
		}
		return nil
	}})
	if app.ErrorCodeOf(err) != app.CodeIO {
		t.Fatalf("commit error = %v", err)
	}
	if commit.Recovery.State != RecoveryRetained || len(commit.Recovery.Paths) != 1 || commit.Recovery.Paths[0] != "forecast.key" {
		t.Fatalf("recovery = %#v", commit.Recovery)
	}
	if err := storage.CheckProtectedFile(keyPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ledgerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ledger exists after injected failure: %v", err)
	}
	if _, err := os.Stat(storage.JournalPath(ledgerPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed failure left a journal: %v", err)
	}
}

func testPublicInitialLedger(t *testing.T) *ledger.Ledger {
	t.Helper()
	root, err := BuildLedgerRoot(InitRootRequest{LedgerID: "research", Timezone: "UTC", ForecasterID: "andrey", ForecasterName: "Andrey"}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	model, err := BuildInitialPublicLedger(root, binaryInitialQuestion())
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func testSealedInitialBuild(t *testing.T) SealedInitialBuild {
	t.Helper()
	root, err := BuildLedgerRoot(InitRootRequest{LedgerID: "research", Timezone: "UTC", ForecasterID: "andrey", ForecasterName: "Andrey"}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	input := binaryInitialQuestion()
	input.InitialForecast.Visibility = ledger.VisibilitySealed
	rationale, comment := "private rationale", "private comment"
	factors := []string{"base rate"}
	input.InitialForecast.Rationale, input.InitialForecast.Comment, input.InitialForecast.KeyFactors = &rationale, &comment, &factors
	built, err := BuildInitialSealedLedger(context.Background(), root, input, Effects{
		Clock:  fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x24}, 76))},
	})
	if err != nil {
		t.Fatal(err)
	}
	return built
}
