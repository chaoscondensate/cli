package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
	"go.yaml.in/yaml/v3"
)

type InitialCommitStage string

const (
	StageInitialKeyCreated    InitialCommitStage = "key_created"
	StageInitialLedgerCreate  InitialCommitStage = "before_ledger_create"
	StageInitialLedgerCreated InitialCommitStage = "ledger_created"
)

type InitialCommitOptions struct {
	Fault func(InitialCommitStage) error
}

type InitialFileCommit struct {
	LedgerPath string
	KeyPath    string
	Recovery   Recovery
}

func EncodeNewLedger(model *ledger.Ledger, path string) ([]byte, error) {
	if err := ValidateProspectiveLedgerModel(model); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		return nil, app.NewError(app.CodeInternal, "new ledger cannot be encoded", err)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		buffer := &bytes.Buffer{}
		if err := json.Indent(buffer, encoded, "", "  "); err != nil {
			return nil, app.NewError(app.CodeInternal, "new JSON ledger cannot be formatted", err)
		}
		return append(buffer.Bytes(), '\n'), nil
	case ".yaml", ".yml":
		parsed, err := document.ParseJSON(strings.NewReader(string(encoded)), document.DefaultLimits)
		if err != nil {
			return nil, app.NewError(app.CodeInternal, "new ledger cannot be converted to YAML", err)
		}
		result, err := yaml.Marshal(parsed.Root.Any())
		if err != nil {
			return nil, app.NewError(app.CodeInternal, "new YAML ledger cannot be formatted", err)
		}
		return result, nil
	default:
		return nil, app.NewError(app.CodeUsage, "ledger filename must end in .json, .yaml, or .yml", nil)
	}
}

func CommitNewLedger(path string, model *ledger.Ledger) (string, error) {
	resolved, err := storage.ResolveNewFilePath(path, "ledger file")
	if err != nil {
		return "", err
	}
	bytes, err := EncodeNewLedger(model, resolved)
	if err != nil {
		return "", err
	}
	if err := storage.CreateExclusive(resolved, bytes, 0o644); err != nil {
		return "", err
	}
	return resolved, nil
}

// CommitInitialSealedFiles preserves the only key copy under every failure.
// The returned Recovery is meaningful even when err is non-nil.
func CommitInitialSealedFiles(ctx context.Context, ledgerPath, keyPath string, build SealedInitialBuild, options InitialCommitOptions) (InitialFileCommit, error) {
	var result InitialFileCommit
	resolvedLedger, err := storage.ResolveNewFilePath(ledgerPath, "ledger file")
	if err != nil {
		return result, err
	}
	resolvedKey, err := storage.ResolveNewFilePath(keyPath, "key file")
	if err != nil {
		return result, err
	}
	if resolvedLedger == resolvedKey {
		return result, app.NewError(app.CodeConflict, "ledger and key destinations must be different", nil)
	}
	ledgerBytes, err := EncodeNewLedger(build.Ledger, resolvedLedger)
	if err != nil {
		return result, err
	}
	if len(build.KeyFile) == 0 {
		return result, app.NewError(app.CodeInternal, "sealed initialization has no key bytes", nil)
	}
	result.LedgerPath, result.KeyPath = resolvedLedger, resolvedKey
	journalPath := storage.JournalPath(resolvedLedger)
	plan, err := storage.NewResourcePlan(journalPath, string(OperationLedgerInit), []storage.ResourceEntry{
		{Kind: storage.ResourceKey, Type: storage.ResourceFile, Path: resolvedKey, Owned: true, Rollback: storage.ResourceRollbackRetainSecret, State: storage.ResourcePlanned},
		{Kind: storage.ResourceLedger, Type: storage.ResourceFile, Path: resolvedLedger, Owned: true, Rollback: storage.ResourceRollbackRemoveOwned, State: storage.ResourcePlanned},
	})
	if err != nil {
		return result, err
	}
	if err := plan.Begin(); err != nil {
		return result, err
	}
	if err := storage.CreateProtectedFile(resolvedKey, build.KeyFile); err != nil {
		_ = plan.Finish()
		return result, err
	}
	result.Recovery = retainedKeyRecovery(resolvedKey)
	if err := plan.MarkCreated(resolvedKey, storage.ResourceDigest(build.KeyFile)); err != nil {
		result.Recovery.State = RecoveryRequired
		return result, err
	}
	if err := initialCommitFault(options, StageInitialKeyCreated); err != nil {
		_ = plan.Finish()
		return result, err
	}
	if err := initialCommitFault(options, StageInitialLedgerCreate); err != nil {
		_ = plan.Finish()
		return result, err
	}
	if ctx != nil && ctx.Err() != nil {
		_ = plan.Finish()
		return result, app.NewError(app.CodeInterrupted, "initialization was interrupted after key creation", ctx.Err())
	}
	if err := storage.CreateExclusive(resolvedLedger, ledgerBytes, 0o644); err != nil {
		_ = plan.Finish()
		return result, err
	}
	if err := plan.MarkCreated(resolvedLedger, storage.ResourceDigest(ledgerBytes)); err != nil {
		result.Recovery.State = RecoveryRequired
		return result, err
	}
	if err := initialCommitFault(options, StageInitialLedgerCreated); err != nil {
		result.Recovery = Recovery{
			State: RecoveryRequired, Message: "The protected key and ledger were created, but initialization cleanup did not finish.",
			Paths:   []string{filepath.Base(resolvedKey), filepath.Base(resolvedLedger), filepath.Base(journalPath)},
			Actions: []string{"Run recovery for the retained initialization journal before another write."},
		}
		return result, err
	}
	if err := plan.MarkCommitted(resolvedKey); err != nil {
		result.Recovery.State = RecoveryRequired
		return result, err
	}
	if err := plan.MarkCommitted(resolvedLedger); err != nil {
		result.Recovery.State = RecoveryRequired
		return result, err
	}
	if err := plan.Finish(); err != nil {
		result.Recovery.State = RecoveryRequired
		return result, err
	}
	result.Recovery = Recovery{State: RecoveryNone}
	return result, nil
}

func retainedKeyRecovery(path string) Recovery {
	return Recovery{
		State: RecoveryRetained, Message: "The protected key was created and retained, but the ledger was not committed.",
		Paths: []string{filepath.Base(path)}, Actions: []string{"Keep the key file, resolve the ledger destination problem, then retry with a new key destination or securely remove the unused key."},
	}
}

func initialCommitFault(options InitialCommitOptions, stage InitialCommitStage) error {
	if options.Fault == nil {
		return nil
	}
	if err := options.Fault(stage); err != nil {
		return app.NewError(app.CodeIO, fmt.Sprintf("initialization stopped after %s", stage), err)
	}
	return nil
}
