package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
)

func TestLedgerLockIsExclusiveAndDeterministic(t *testing.T) {
	ledgerPath := filepath.Join(t.TempDir(), "ledger.json")
	first, err := AcquireLedgerLock(context.Background(), ledgerPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Release() })
	if first.Path() != LedgerLockPath(ledgerPath) {
		t.Fatalf("lock path = %q", first.Path())
	}

	second, err := AcquireLedgerLock(context.Background(), ledgerPath, 0)
	if second != nil || app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("second lock = %#v, err=%v", second, err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("idempotent release failed: %v", err)
	}
	third, err := AcquireLedgerLock(context.Background(), ledgerPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := third.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(LedgerLockPath(ledgerPath)); err != nil {
		t.Fatalf("stable lock file was not retained: %v", err)
	}
}

func TestLedgerLockWaitHonorsDeadlineAndCancellation(t *testing.T) {
	ledgerPath := filepath.Join(t.TempDir(), "ledger.json")
	first, err := AcquireLedgerLock(context.Background(), ledgerPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()

	started := time.Now()
	_, err = AcquireLedgerLock(context.Background(), ledgerPath, 60*time.Millisecond)
	if app.ErrorCodeOf(err) != app.CodeConflict || time.Since(started) < 40*time.Millisecond {
		t.Fatalf("bounded wait err=%v duration=%s", err, time.Since(started))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = AcquireLedgerLock(ctx, ledgerPath, time.Second)
	if app.ErrorCodeOf(err) != app.CodeInterrupted {
		t.Fatalf("canceled lock wait: %v", err)
	}
}
