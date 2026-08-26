package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
	"github.com/chaoscondensate/cli/internal/validation"
)

type PlatformFileResult struct {
	LedgerID        ledger.Slug      `json:"ledger_id"`
	PlatformID      ledger.Slug      `json:"platform_id"`
	Changed         bool             `json:"changed"`
	ChangedPointers []string         `json:"changed_pointers"`
	BeforeSHA256    string           `json:"before_sha256"`
	AfterSHA256     string           `json:"after_sha256"`
	Platform        *ledger.Platform `json:"platform,omitempty"`
}

type platformMutationBuilder func(*ledger.Ledger) (PlatformMutation, error)

func PlanPlatformAddFile(ctx context.Context, path string, id ledger.Slug, input PlatformCreateInput) (PlatformFileResult, error) {
	return planPlatformMutation(ctx, path, id, func(model *ledger.Ledger) (PlatformMutation, error) { return BuildPlatformAdd(model, id, input) })
}

func CommitPlatformAddFile(ctx context.Context, path string, id ledger.Slug, input PlatformCreateInput) (PlatformFileResult, error) {
	return commitPlatformMutation(ctx, path, id, func(model *ledger.Ledger) (PlatformMutation, error) { return BuildPlatformAdd(model, id, input) })
}

func PlanPlatformUpdateFile(ctx context.Context, path string, id ledger.Slug, input PlatformPatchInput) (PlatformFileResult, error) {
	return planPlatformMutation(ctx, path, id, func(model *ledger.Ledger) (PlatformMutation, error) { return BuildPlatformUpdate(model, id, input) })
}

func CommitPlatformUpdateFile(ctx context.Context, path string, id ledger.Slug, input PlatformPatchInput) (PlatformFileResult, error) {
	return commitPlatformMutation(ctx, path, id, func(model *ledger.Ledger) (PlatformMutation, error) { return BuildPlatformUpdate(model, id, input) })
}

func PlanPlatformRemoveFile(ctx context.Context, path string, id ledger.Slug) (PlatformFileResult, error) {
	return planPlatformMutation(ctx, path, id, func(model *ledger.Ledger) (PlatformMutation, error) { return BuildPlatformRemove(model, id) })
}

func CommitPlatformRemoveFile(ctx context.Context, path string, id ledger.Slug) (PlatformFileResult, error) {
	return commitPlatformMutation(ctx, path, id, func(model *ledger.Ledger) (PlatformMutation, error) { return BuildPlatformRemove(model, id) })
}

func planPlatformMutation(ctx context.Context, path string, id ledger.Slug, builder platformMutationBuilder) (PlatformFileResult, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return PlatformFileResult{}, err
	}
	return buildPlatformFileResult(loaded.Document, loaded.Model, id, builder)
}

func commitPlatformMutation(ctx context.Context, path string, id ledger.Slug, builder platformMutationBuilder) (PlatformFileResult, error) {
	resolved, err := storage.ResolveLedgerPath(path, true)
	if err != nil {
		return PlatformFileResult{}, err
	}
	artifacts := os.DirFS(filepath.Dir(resolved))
	var result PlatformFileResult
	err = storage.UpdateLedger(ctx, resolved, storage.TransactionOptions{
		LockWait: 0,
		Validate: func(parsed *document.Document) error { return ValidateLedgerDocument(parsed, artifacts) },
		Mutate: func(parsed *document.Document) ([]byte, error) {
			model, decodeErr := validation.DecodeLedger(parsed)
			if decodeErr != nil {
				return nil, app.NewError(app.CodeInternal, "validated ledger cannot be decoded for platform mutation", decodeErr)
			}
			built, buildErr := buildPlatformFileResult(parsed, model, id, builder)
			if buildErr != nil {
				return nil, buildErr
			}
			result = built
			mutation, buildErr := builder(model)
			if buildErr != nil {
				return nil, buildErr
			}
			return document.ApplyPatch(parsed, mutation.Patches)
		},
	})
	if err != nil {
		return PlatformFileResult{}, err
	}
	return result, nil
}

func buildPlatformFileResult(parsed *document.Document, model *ledger.Ledger, id ledger.Slug, builder platformMutationBuilder) (PlatformFileResult, error) {
	mutation, err := builder(model)
	if err != nil {
		return PlatformFileResult{}, err
	}
	patched, err := document.ApplyPatch(parsed, mutation.Patches)
	if err != nil {
		return PlatformFileResult{}, app.NewError(app.CodeInternal, "platform mutation cannot be applied to source document", err)
	}
	before := sha256.Sum256(parsed.Raw)
	after := sha256.Sum256(patched)
	result := PlatformFileResult{
		LedgerID: model.LedgerID, PlatformID: id, Changed: len(mutation.Patches) > 0,
		ChangedPointers: ChangedPointers(mutation.Patches), BeforeSHA256: hex.EncodeToString(before[:]), AfterSHA256: hex.EncodeToString(after[:]),
	}
	if platform, exists := mutation.Ledger.Platforms[id]; exists {
		copy := clonePlatform(platform)
		result.Platform = &copy
	}
	return result, nil
}

func LoadPlatformList(ctx context.Context, path string, stdin io.Reader) (ledger.Slug, []PlatformListItem, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, stdin)
	if err != nil {
		return "", nil, err
	}
	items, err := ListPlatforms(loaded.Model)
	return loaded.Model.LedgerID, items, err
}

func LoadPlatformShow(ctx context.Context, path string, stdin io.Reader, id ledger.Slug) (ledger.Slug, PlatformShowResult, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, stdin)
	if err != nil {
		return "", PlatformShowResult{}, err
	}
	result, err := ShowPlatform(loaded.Model, id)
	return loaded.Model.LedgerID, result, err
}
