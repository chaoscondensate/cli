package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/gofrs/flock"
)

const lockRetryDelay = 25 * time.Millisecond

type LedgerLock struct {
	path string
	file *flock.Flock
	once sync.Once
	err  error
}

func LedgerLockPath(ledgerPath string) string {
	directory := filepath.Dir(ledgerPath)
	base := filepath.Base(ledgerPath)
	return filepath.Join(directory, "."+base+".forecast-ledger.lock")
}

// AcquireLedgerLock takes the same exclusive advisory lock for CLI and MCP
// mutations. A zero wait reports a conflict immediately; a positive wait is
// bounded by both wait and ctx.
func AcquireLedgerLock(ctx context.Context, ledgerPath string, wait time.Duration) (*LedgerLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	lockPath := LedgerLockPath(ledgerPath)
	file := flock.New(lockPath)
	if wait <= 0 {
		locked, err := file.TryLock()
		if err != nil {
			return nil, app.NewError(app.CodeIO, "ledger lock cannot be acquired", err)
		}
		if !locked {
			return nil, app.WithDetails(app.NewError(app.CodeConflict, "ledger is locked by another operation", nil), map[string]any{"resource": "ledger"})
		}
		return &LedgerLock{path: lockPath, file: file}, nil
	}

	waitContext, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	locked, err := file.TryLockContext(waitContext, lockRetryDelay)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, app.WithDetails(app.NewError(app.CodeConflict, "ledger remained locked until the wait limit", err), map[string]any{"resource": "ledger"})
		}
		if errors.Is(err, context.Canceled) {
			return nil, app.NewError(app.CodeInterrupted, "lock wait was interrupted", err)
		}
		return nil, app.NewError(app.CodeIO, "ledger lock cannot be acquired", err)
	}
	if !locked {
		if ctx.Err() != nil {
			return nil, app.NewError(app.CodeInterrupted, "lock wait was interrupted", ctx.Err())
		}
		return nil, app.NewError(app.CodeConflict, "ledger remained locked until the wait limit", nil)
	}
	return &LedgerLock{path: lockPath, file: file}, nil
}

func (l *LedgerLock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *LedgerLock) Release() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		if l.file == nil {
			return
		}
		if err := l.file.Unlock(); err != nil {
			l.err = app.NewError(app.CodeIO, "ledger lock cannot be released", err)
		}
	})
	return l.err
}

func (l *LedgerLock) String() string {
	if l == nil {
		return "LedgerLock(<nil>)"
	}
	return fmt.Sprintf("LedgerLock(%s)", l.path)
}
