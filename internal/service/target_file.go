package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
)

const maxTargetBytes = 16 << 20

type TargetResult struct {
	QuestionID ledger.Slug                `json:"question_id"`
	ForecastID ledger.Slug                `json:"forecast_id"`
	Path       ledger.RelativePath        `json:"path"`
	SHA256     string                     `json:"sha256"`
	Size       int                        `json:"size"`
	State      storage.DeterministicState `json:"state,omitempty"`
	Valid      *bool                      `json:"valid,omitempty"`
}

type TargetOperationResult struct {
	LedgerID ledger.Slug    `json:"ledger_id"`
	Targets  []TargetResult `json:"targets"`
	Effects  []SideEffect   `json:"effects,omitempty"`
	Recovery Recovery       `json:"recovery,omitempty"`
}

func PlanTargetBuild(ctx context.Context, path string, all bool, questionID, forecastID ledger.Slug) (TargetOperationResult, error) {
	loaded, artifacts, err := loadSelectedTargets(ctx, path, all, questionID, forecastID)
	if err != nil {
		return TargetOperationResult{}, err
	}
	root := filepath.Dir(loaded.Path)
	if _, _, err := inspectTargetDirectories(root); err != nil {
		return TargetOperationResult{}, err
	}
	result := TargetOperationResult{LedgerID: loaded.Model.LedgerID, Targets: make([]TargetResult, len(artifacts))}
	for index, artifact := range artifacts {
		state, err := preflightTargetFile(root, artifact)
		if err != nil {
			return TargetOperationResult{}, err
		}
		result.Targets[index] = targetResult(artifact, state, nil)
		status := EffectDeferred
		if state == storage.DeterministicUnchanged {
			status = EffectUnchanged
		}
		result.Effects = append(result.Effects, SideEffect{Kind: EffectTarget, Action: EffectCreate, Status: status, Path: string(artifact.RelativePath), Owned: state != storage.DeterministicUnchanged, Rollback: RollbackCreatedPublic})
	}
	return result, nil
}

func CommitTargetBuild(ctx context.Context, path string, all bool, questionID, forecastID ledger.Slug) (TargetOperationResult, error) {
	resolvedLedger, err := storage.ResolveLedgerPath(path, true)
	if err != nil {
		return TargetOperationResult{}, err
	}
	lock, err := storage.AcquireLedgerLock(ctx, resolvedLedger, 0)
	if err != nil {
		return TargetOperationResult{}, err
	}
	defer lock.Release()
	planned, err := PlanTargetBuild(ctx, resolvedLedger, all, questionID, forecastID)
	if err != nil {
		return TargetOperationResult{}, err
	}
	loaded, artifacts, err := loadSelectedTargets(ctx, resolvedLedger, all, questionID, forecastID)
	if err != nil {
		return TargetOperationResult{}, err
	}
	root := filepath.Dir(loaded.Path)
	createProofs, createTargets, err := inspectTargetDirectories(root)
	if err != nil {
		return TargetOperationResult{}, err
	}
	proofsPath := filepath.Join(root, "proofs")
	targetsPath := filepath.Join(proofsPath, "targets")
	entries := make([]storage.ResourceEntry, 0, len(artifacts)+2)
	if createProofs {
		entries = append(entries, storage.ResourceEntry{Kind: storage.ResourceTarget, Type: storage.ResourceDirectory, Path: proofsPath, Owned: true, Rollback: storage.ResourceRollbackRemoveOwned, State: storage.ResourcePlanned})
	}
	if createTargets {
		entries = append(entries, storage.ResourceEntry{Kind: storage.ResourceTarget, Type: storage.ResourceDirectory, Path: targetsPath, Owned: true, Rollback: storage.ResourceRollbackRemoveOwned, State: storage.ResourcePlanned})
	}
	for index, artifact := range artifacts {
		path := filepath.Join(root, filepath.FromSlash(string(artifact.RelativePath)))
		owned := planned.Targets[index].State != storage.DeterministicUnchanged
		rollback := storage.ResourceRollbackNone
		if owned {
			rollback = storage.ResourceRollbackRemoveOwned
		}
		entries = append(entries, storage.ResourceEntry{Kind: storage.ResourceTarget, Type: storage.ResourceFile, Path: path, Owned: owned, Rollback: rollback, State: storage.ResourcePlanned})
	}
	journal := filepath.Join(root, "."+filepath.Base(loaded.Path)+".target-build-resources.json")
	plan, err := storage.NewResourcePlan(journal, string(OperationTargetBuild), entries)
	if err != nil {
		return TargetOperationResult{}, err
	}
	if err := plan.Begin(); err != nil {
		return TargetOperationResult{}, err
	}
	fail := func(cause error) (TargetOperationResult, error) {
		recovery, recoveryErr := storage.RecoverResourcePlan(context.Background(), journal)
		if recoveryErr != nil {
			planned.Recovery = Recovery{State: RecoveryRequired, Message: "Target creation did not finish and automatic cleanup needs attention.", Paths: []string{filepath.Base(journal)}}
			return planned, app.NewError(app.CodeIO, "target creation failed and recovery was incomplete", errors.Join(cause, recoveryErr))
		}
		planned.Recovery = Recovery{State: RecoveryNone}
		_ = recovery
		return planned, cause
	}
	if createProofs {
		if err := os.Mkdir(proofsPath, 0o755); err != nil {
			return fail(app.NewError(app.CodeIO, "proofs directory cannot be created", err))
		}
		if err := plan.MarkCreated(proofsPath, ""); err != nil {
			return fail(err)
		}
	}
	if createTargets {
		if err := os.Mkdir(targetsPath, 0o755); err != nil {
			return fail(app.NewError(app.CodeIO, "target directory cannot be created", err))
		}
		if err := plan.MarkCreated(targetsPath, ""); err != nil {
			return fail(err)
		}
	}
	result := TargetOperationResult{LedgerID: loaded.Model.LedgerID, Targets: make([]TargetResult, len(artifacts))}
	for index, artifact := range artifacts {
		if ctx != nil && ctx.Err() != nil {
			return fail(app.NewError(app.CodeInterrupted, "target build was interrupted", ctx.Err()))
		}
		absolute := filepath.Join(root, filepath.FromSlash(string(artifact.RelativePath)))
		written, err := storage.EnsureDeterministicFile(absolute, artifact.Bytes, 0o644, maxTargetBytes)
		if err != nil {
			return fail(err)
		}
		if written.State == storage.DeterministicCreated {
			if err := plan.MarkCreated(absolute, artifact.SHA256); err != nil {
				return fail(err)
			}
		}
		result.Targets[index] = targetResult(artifact, written.State, nil)
		status := EffectCompleted
		if written.State == storage.DeterministicUnchanged {
			status = EffectUnchanged
		}
		result.Effects = append(result.Effects, SideEffect{Kind: EffectTarget, Action: EffectCreate, Status: status, Path: string(artifact.RelativePath), Owned: written.State == storage.DeterministicCreated, Rollback: RollbackCreatedPublic})
	}
	for _, entry := range entries {
		if err := plan.MarkCommitted(entry.Path); err != nil {
			result.Recovery = Recovery{State: RecoveryRequired, Message: "Targets were created, but target resource journal cleanup needs attention.", Paths: []string{filepath.Base(journal)}}
			return result, err
		}
	}
	if err := plan.Finish(); err != nil {
		result.Recovery = Recovery{State: RecoveryRequired, Message: "Targets were created, but target resource journal cleanup needs attention.", Paths: []string{filepath.Base(journal)}}
		return result, err
	}
	result.Recovery = Recovery{State: RecoveryNone}
	return result, nil
}

func CheckTargets(ctx context.Context, path string, all bool, questionID, forecastID ledger.Slug) (TargetOperationResult, error) {
	loaded, artifacts, err := loadSelectedTargets(ctx, path, all, questionID, forecastID)
	if err != nil {
		return TargetOperationResult{}, err
	}
	root := filepath.Dir(loaded.Path)
	resolver, err := storage.NewPathResolver(root)
	if err != nil {
		return TargetOperationResult{}, err
	}
	result := TargetOperationResult{LedgerID: loaded.Model.LedgerID, Targets: make([]TargetResult, len(artifacts))}
	for index, artifact := range artifacts {
		absolute, err := resolver.Resolve(string(artifact.RelativePath), true)
		if err != nil {
			return TargetOperationResult{}, err
		}
		actual, err := readBoundedFile(absolute, maxTargetBytes)
		if err != nil {
			return TargetOperationResult{}, err
		}
		if !bytes.Equal(actual, artifact.Bytes) {
			return TargetOperationResult{}, app.WithDetails(app.NewError(app.CodeVerification, "forecast target bytes do not match the ledger", nil), map[string]any{"forecast_id": artifact.ForecastID, "path": artifact.RelativePath, "expected_sha256": artifact.SHA256, "actual_sha256": storage.ResourceDigest(actual)})
		}
		if target := recordedForecastTarget(loaded.Model, artifact.QuestionID, artifact.ForecastID); target != nil {
			expected := TargetMetadataFor(artifact)
			if *target != expected {
				return TargetOperationResult{}, app.WithDetails(app.NewError(app.CodeVerification, "recorded target metadata does not match the deterministic target", nil), map[string]any{"forecast_id": artifact.ForecastID})
			}
		}
		valid := true
		result.Targets[index] = targetResult(artifact, storage.DeterministicUnchanged, &valid)
	}
	return result, nil
}

func loadSelectedTargets(ctx context.Context, path string, all bool, questionID, forecastID ledger.Slug) (*LoadedLedger, []TargetArtifact, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return nil, nil, err
	}
	var artifacts []TargetArtifact
	if all {
		artifacts, err = BuildAllForecastTargets(loaded.Model)
	} else {
		var artifact TargetArtifact
		artifact, err = BuildForecastTarget(loaded.Model, questionID, forecastID)
		artifacts = []TargetArtifact{artifact}
	}
	if err != nil {
		return nil, nil, err
	}
	return loaded, artifacts, nil
}

func inspectTargetDirectories(root string) (createProofs, createTargets bool, err error) {
	proofs := filepath.Join(root, "proofs")
	targets := filepath.Join(proofs, "targets")
	proofsInfo, err := os.Lstat(proofs)
	if errors.Is(err, fs.ErrNotExist) {
		if collision, collisionErr := caseFoldName(root, "proofs"); collisionErr != nil {
			return false, false, collisionErr
		} else if collision != "" {
			return false, false, app.NewError(app.CodeConflict, "proofs directory has a portable case collision", nil)
		}
		return true, true, nil
	}
	if err != nil {
		return false, false, app.NewError(app.CodeIO, "proofs directory cannot be inspected", err)
	}
	if proofsInfo.Mode()&os.ModeSymlink != 0 || !proofsInfo.IsDir() {
		return false, false, app.NewError(app.CodeConflict, "proofs path must be a real directory", nil)
	}
	targetInfo, err := os.Lstat(targets)
	if errors.Is(err, fs.ErrNotExist) {
		if collision, collisionErr := caseFoldName(proofs, "targets"); collisionErr != nil {
			return false, false, collisionErr
		} else if collision != "" {
			return false, false, app.NewError(app.CodeConflict, "target directory has a portable case collision", nil)
		}
		return false, true, nil
	}
	if err != nil {
		return false, false, app.NewError(app.CodeIO, "target directory cannot be inspected", err)
	}
	if targetInfo.Mode()&os.ModeSymlink != 0 || !targetInfo.IsDir() {
		return false, false, app.NewError(app.CodeConflict, "target path must be a real directory", nil)
	}
	return false, false, nil
}

func preflightTargetFile(root string, artifact TargetArtifact) (storage.DeterministicState, error) {
	absolute := filepath.Join(root, filepath.FromSlash(string(artifact.RelativePath)))
	info, err := os.Lstat(absolute)
	if errors.Is(err, fs.ErrNotExist) {
		return storage.DeterministicCreated, nil
	}
	if err != nil {
		return "", app.NewError(app.CodeIO, "target path cannot be inspected", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", app.NewError(app.CodeConflict, "target destination is not a regular file", nil)
	}
	actual, err := readBoundedFile(absolute, maxTargetBytes)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(actual, artifact.Bytes) {
		return "", app.WithDetails(app.NewError(app.CodeConflict, "target destination contains different bytes", nil), map[string]any{"path": artifact.RelativePath, "expected_sha256": artifact.SHA256, "actual_sha256": storage.ResourceDigest(actual)})
	}
	return storage.DeterministicUnchanged, nil
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, app.NewError(app.CodeNotFound, "target file does not exist", err)
		}
		return nil, app.NewError(app.CodeIO, "target file cannot be opened", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, app.NewError(app.CodeIO, "target file cannot be read", err)
	}
	if int64(len(data)) > limit {
		return nil, app.NewError(app.CodeVerification, "target file exceeds its size limit", nil)
	}
	return data, nil
}

func recordedForecastTarget(model *ledger.Ledger, questionID, forecastID ledger.Slug) *ledger.ForecastTarget {
	_, question, err := selectQuestion(model, questionID)
	if err != nil {
		return nil
	}
	for _, forecast := range question.Forecasts {
		if forecast.ID != forecastID {
			continue
		}
		switch {
		case forecast.Integrity.Pending != nil:
			return &forecast.Integrity.Pending.Target
		case forecast.Integrity.Verified != nil:
			return &forecast.Integrity.Verified.Target
		case forecast.Integrity.Failed != nil:
			return forecast.Integrity.Failed.Target
		}
	}
	return nil
}

func targetResult(artifact TargetArtifact, state storage.DeterministicState, valid *bool) TargetResult {
	return TargetResult{QuestionID: artifact.QuestionID, ForecastID: artifact.ForecastID, Path: artifact.RelativePath, SHA256: artifact.SHA256, Size: artifact.Size, State: state, Valid: valid}
}

func caseFoldName(directory, requested string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", app.NewError(app.CodeIO, "artifact directory cannot be read", err)
	}
	for _, entry := range entries {
		if entry.Name() != requested && strings.EqualFold(entry.Name(), requested) {
			return entry.Name(), nil
		}
	}
	return "", nil
}

func sortTargetResults(items []TargetResult) {
	sort.Slice(items, func(i, j int) bool { return items[i].ForecastID < items[j].ForecastID })
}
