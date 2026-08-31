package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/document"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	"github.com/chaoscondensate/forecast-ledger/internal/storage"
	"github.com/chaoscondensate/forecast-ledger/internal/validation"
)

type RootMetadataFileResult struct {
	LedgerID        ledger.Slug `json:"ledger_id"`
	Changed         bool        `json:"changed"`
	ChangedPointers []string    `json:"changed_pointers"`
	BeforeSHA256    string      `json:"before_sha256"`
	AfterSHA256     string      `json:"after_sha256"`
	Warnings        []Warning   `json:"warnings"`
}

func PlanRootMetadataFileUpdate(ctx context.Context, path string, input RootMetadataPatchInput) (RootMetadataFileResult, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return RootMetadataFileResult{}, err
	}
	update, err := BuildRootMetadataUpdate(loaded.Model, input)
	if err != nil {
		return RootMetadataFileResult{}, err
	}
	if err := validateProspectiveFileMutation(loaded, update.Patches); err != nil {
		return RootMetadataFileResult{}, err
	}
	return buildRootMetadataFileResult(loaded.Document, loaded.Model, input)
}

func CommitRootMetadataFileUpdate(ctx context.Context, path string, input RootMetadataPatchInput) (RootMetadataFileResult, error) {
	resolved, err := storage.ResolveLedgerPath(path, true)
	if err != nil {
		return RootMetadataFileResult{}, err
	}
	artifacts := os.DirFS(filepath.Dir(resolved))
	var result RootMetadataFileResult
	err = storage.UpdateLedger(ctx, resolved, storage.TransactionOptions{
		LockWait: 0,
		Validate: func(parsed *document.Document) error {
			return ValidateLedgerDocument(parsed, artifacts)
		},
		Mutate: func(parsed *document.Document) ([]byte, error) {
			model, decodeErr := validation.DecodeLedger(parsed)
			if decodeErr != nil {
				return nil, app.NewError(app.CodeInternal, "validated ledger cannot be decoded for metadata update", decodeErr)
			}
			built, buildErr := buildRootMetadataFileResult(parsed, model, input)
			if buildErr != nil {
				return nil, buildErr
			}
			result = built
			if !built.Changed {
				return append([]byte(nil), parsed.Raw...), nil
			}
			update, buildErr := BuildRootMetadataUpdate(model, input)
			if buildErr != nil {
				return nil, buildErr
			}
			return document.ApplyPatch(parsed, update.Patches)
		},
	})
	if err != nil {
		return RootMetadataFileResult{}, err
	}
	return result, nil
}

func buildRootMetadataFileResult(parsed *document.Document, model *ledger.Ledger, input RootMetadataPatchInput) (RootMetadataFileResult, error) {
	if parsed == nil || model == nil {
		return RootMetadataFileResult{}, app.NewError(app.CodeInternal, "metadata update source is missing", nil)
	}
	update, err := BuildRootMetadataUpdate(model, input)
	if err != nil {
		return RootMetadataFileResult{}, err
	}
	patched, err := document.ApplyPatch(parsed, update.Patches)
	if err != nil {
		return RootMetadataFileResult{}, app.NewError(app.CodeInternal, "metadata update cannot be applied to the source document", err)
	}
	before := sha256.Sum256(parsed.Raw)
	after := sha256.Sum256(patched)
	return RootMetadataFileResult{
		LedgerID: model.LedgerID, Changed: len(update.Patches) > 0,
		ChangedPointers: ChangedPointers(update.Patches), BeforeSHA256: hex.EncodeToString(before[:]), AfterSHA256: hex.EncodeToString(after[:]),
		Warnings: update.Warnings,
	}, nil
}
