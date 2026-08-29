package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/publication"
	ledgerschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/storage"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

type PublicationFile struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	bytes  []byte
}

type PublicationBuildResult struct {
	LedgerID       ledger.Slug       `json:"ledger_id"`
	Output         string            `json:"output"`
	LedgerPath     string            `json:"ledger_path"`
	ManifestPath   string            `json:"manifest_path"`
	ManifestSHA256 string            `json:"manifest_sha256"`
	FileCount      int               `json:"file_count"`
	TotalBytes     int64             `json:"total_bytes"`
	Files          []PublicationFile `json:"files"`
	EvidenceState  string            `json:"evidence_state"`
	Effects        []SideEffect      `json:"effects,omitempty"`
	Recovery       Recovery          `json:"recovery,omitempty"`
	manifestBytes  []byte
}

type PublicationVerifyResult struct {
	LedgerID       ledger.Slug            `json:"ledger_id"`
	ManifestPath   string                 `json:"manifest_path"`
	ManifestSHA256 string                 `json:"manifest_sha256"`
	FileCount      int                    `json:"file_count"`
	TotalBytes     int64                  `json:"total_bytes"`
	Files          []PublicationFile      `json:"files"`
	Evidence       []ForecastVerification `json:"evidence"`
	Overall        VerificationOverall    `json:"overall"`
	Limitations    []string               `json:"limitations"`
	NetworkProfile NetworkProfile         `json:"network_profile"`
	RequestSummary ots.RequestSummary     `json:"request_summary,omitempty"`
	FailureCode    app.ErrorCode          `json:"-"`
}

type PublicationVerifyOptions struct {
	Online   bool
	Offline  bool
	Observer ots.BitcoinObserver
}

func PlanPublicationBuild(ctx context.Context, ledgerPath, output string) (PublicationBuildResult, error) {
	resolvedOutput, err := storage.ResolveNewFilePath(output, "package output")
	if err != nil {
		return PublicationBuildResult{}, err
	}
	loaded, err := LoadAndValidateLedger(ctx, ledgerPath, nil)
	if err != nil {
		return PublicationBuildResult{}, err
	}
	result, err := collectPublication(loaded, resolvedOutput)
	if err != nil {
		return PublicationBuildResult{}, err
	}
	for _, file := range result.Files {
		result.Effects = append(result.Effects, SideEffect{Kind: EffectPackage, Action: EffectCreate, Status: EffectDeferred, Path: file.Path, Owned: true, Rollback: RollbackCreatedPublic})
	}
	result.Effects = append(result.Effects, SideEffect{Kind: EffectPackage, Action: EffectCreate, Status: EffectDeferred, Path: "manifest.json", Owned: true, Rollback: RollbackCreatedPublic})
	return result, nil
}

func CommitPublicationBuild(ctx context.Context, ledgerPath, output string, dryRun bool) (PublicationBuildResult, error) {
	result, err := PlanPublicationBuild(ctx, ledgerPath, output)
	if err != nil || dryRun {
		return result, err
	}
	resolvedOutput, err := storage.ResolveNewFilePath(output, "package output")
	if err != nil {
		return result, err
	}
	if err := os.Mkdir(resolvedOutput, 0o755); err != nil {
		return result, app.NewError(app.CodeIO, "package output directory cannot be created", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(resolvedOutput)
		}
	}()
	for _, file := range result.Files {
		if ctx != nil && ctx.Err() != nil {
			return result, app.NewError(app.CodeInterrupted, "package build was interrupted", ctx.Err())
		}
		destination := filepath.Join(resolvedOutput, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return result, app.NewError(app.CodeIO, "package directory cannot be created", err)
		}
		if err := storage.CreateExclusive(destination, file.bytes, 0o644); err != nil {
			return result, err
		}
	}
	if err := storage.CreateExclusive(filepath.Join(resolvedOutput, "manifest.json"), result.manifestBytes, 0o644); err != nil {
		return result, err
	}
	complete = true
	result.Effects = nil
	for _, file := range result.Files {
		result.Effects = append(result.Effects, SideEffect{Kind: EffectPackage, Action: EffectCreate, Status: EffectCompleted, Path: file.Path, Owned: true, Rollback: RollbackCreatedPublic})
	}
	result.Effects = append(result.Effects, SideEffect{Kind: EffectPackage, Action: EffectCreate, Status: EffectCompleted, Path: "manifest.json", Owned: true, Rollback: RollbackCreatedPublic})
	result.Recovery = Recovery{State: RecoveryNone}
	return result, nil
}

func collectPublication(loaded *LoadedLedger, output string) (PublicationBuildResult, error) {
	root := filepath.Dir(loaded.Path)
	ledgerRelative := filepath.ToSlash(filepath.Join("ledger", filepath.Base(loaded.Path)))
	ledgerBytes := append([]byte(nil), loaded.Document.Raw...)
	files := map[string]PublicationFile{ledgerRelative: publicationFile(publication.RoleLedger, ledgerRelative, ledgerBytes)}
	evidenceState := "complete"
	for _, question := range loaded.Model.Questions {
		for _, forecast := range question.Forecasts {
			if forecast.Commitment != nil {
				hint := ""
				if forecast.Commitment.Sealed != nil {
					hint = forecast.Commitment.Sealed.KeyHint
				} else if forecast.Commitment.Revealed != nil {
					hint = forecast.Commitment.Revealed.KeyHint
				}
				if err := ValidateKeyHint(forecast.ID, hint); err != nil {
					return PublicationBuildResult{}, app.WithDetails(app.NewError(app.CodeConflict, "forecast key hint is not package-safe; run forecast key-hint update", err), map[string]any{"forecast_id": forecast.ID})
				}
			}
			var target *ledger.ForecastTarget
			var timestamps []ledger.OTSTimestamp
			switch {
			case forecast.Integrity.Pending != nil:
				target, timestamps, evidenceState = &forecast.Integrity.Pending.Target, forecast.Integrity.Pending.Timestamps, "pending"
			case forecast.Integrity.Verified != nil:
				target, timestamps = &forecast.Integrity.Verified.Target, forecast.Integrity.Verified.Timestamps
			case forecast.Integrity.Failed != nil:
				target = forecast.Integrity.Failed.Target
				if forecast.Integrity.Failed.Timestamps != nil {
					timestamps = *forecast.Integrity.Failed.Timestamps
				}
			}
			if target == nil {
				continue
			}
			artifact, err := BuildForecastTarget(loaded.Model, question.ID, forecast.ID)
			if err != nil || *target != TargetMetadataFor(artifact) {
				return PublicationBuildResult{}, app.NewError(app.CodeVerification, "recorded target metadata does not match the selected forecast", err)
			}
			actual, err := readConfinedArtifact(root, string(target.ArtifactPath), maxTargetBytes)
			if err != nil || !bytes.Equal(actual, artifact.Bytes) {
				return PublicationBuildResult{}, app.NewError(app.CodeVerification, "forecast target cannot be packaged because its bytes do not match", err)
			}
			files[string(target.ArtifactPath)] = publicationFile(publication.RoleTarget, string(target.ArtifactPath), actual)
			for _, timestamp := range timestamps {
				if timestamp.Type != "opentimestamps" {
					continue
				}
				receiptBytes, err := readConfinedArtifact(root, string(timestamp.ProofPath), maxReceiptBytes)
				if err != nil {
					return PublicationBuildResult{}, err
				}
				receipt, err := ots.ParseReceipt(receiptBytes)
				if err != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
					return PublicationBuildResult{}, app.NewError(app.CodeVerification, "OpenTimestamps receipt cannot be packaged because its binding is invalid", err)
				}
				files[string(timestamp.ProofPath)] = publicationFile(publication.RoleReceipt, string(timestamp.ProofPath), receiptBytes)
			}
			if forecast.Visibility == ledger.VisibilityRevealed {
				layer := verifyRevealLayer(question, forecast, VerificationLayer{Name: "content_binding", State: LayerPass})
				if layer.State != LayerPass {
					return PublicationBuildResult{}, app.NewError(app.CodeVerification, "revealed forecast authentication failed before package build", nil)
				}
			}
		}
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	if err := storage.DetectPortablePathCollisions(paths); err != nil {
		return PublicationBuildResult{}, err
	}
	sort.Strings(paths)
	result := PublicationBuildResult{LedgerID: loaded.Model.LedgerID, Output: filepath.Base(output), LedgerPath: ledgerRelative, ManifestPath: "manifest.json", EvidenceState: evidenceState}
	manifest := publication.Manifest{Profile: publication.ManifestProfile, LedgerSchema: publication.SchemaPin{Version: ledgerschema.Version, Commit: ledgerschema.Commit, SHA256: ledgerschema.SchemaSHA256}, LedgerPath: ledgerRelative}
	for _, path := range paths {
		file := files[path]
		result.Files = append(result.Files, file)
		result.TotalBytes += file.Size
		manifest.Entries = append(manifest.Entries, publication.Entry{Role: file.Role, Path: file.Path, Size: file.Size, Digest: publication.Digest{Algorithm: "sha-256", Value: file.SHA256}})
	}
	publication.SortEntries(manifest.Entries)
	manifestBytes, err := publication.Encode(manifest)
	if err != nil {
		return PublicationBuildResult{}, app.NewError(app.CodeInternal, "publication manifest cannot be encoded", err)
	}
	result.manifestBytes = manifestBytes
	result.ManifestSHA256 = storage.ResourceDigest(manifestBytes)
	result.FileCount = len(result.Files) + 1
	result.TotalBytes += int64(len(manifestBytes))
	return result, nil
}

func VerifyPublicationPackage(ctx context.Context, ledgerPath, manifestPath string, supplied ...PublicationVerifyOptions) (PublicationVerifyResult, error) {
	options := PublicationVerifyOptions{}
	if len(supplied) > 0 {
		options = supplied[0]
	}
	if options.Online && options.Offline {
		return PublicationVerifyResult{}, app.NewError(app.CodeUsage, "--online and --offline cannot be combined", nil)
	}
	observer := options.Observer
	if options.Online && observer == nil {
		observer = ots.NewPublicBitcoinObserver(nil)
	}
	resolvedLedger, err := storage.ResolveLedgerPath(ledgerPath, true)
	if err != nil {
		return PublicationVerifyResult{}, err
	}
	resolvedManifest, err := storage.ResolveLedgerPath(manifestPath, true)
	if err != nil {
		return PublicationVerifyResult{}, err
	}
	root := filepath.Dir(resolvedManifest)
	resolver, err := storage.NewPathResolver(root)
	if err != nil {
		return PublicationVerifyResult{}, err
	}
	manifestBytes, err := readBoundedFile(resolvedManifest, publication.MaxManifestBytes)
	if err != nil {
		return PublicationVerifyResult{}, err
	}
	manifest, err := publication.Decode(manifestBytes)
	if err != nil {
		return PublicationVerifyResult{}, app.NewError(app.CodeVerification, "publication manifest is invalid", err)
	}
	if manifest.LedgerSchema.Version != ledgerschema.Version || manifest.LedgerSchema.Commit != ledgerschema.Commit || manifest.LedgerSchema.SHA256 != ledgerschema.SchemaSHA256 {
		return PublicationVerifyResult{}, app.NewError(app.CodeVerification, "publication manifest schema pin does not match this binary", nil)
	}
	expectedLedger, err := resolver.Resolve(manifest.LedgerPath, true)
	if err != nil || expectedLedger != resolvedLedger {
		return PublicationVerifyResult{}, app.NewError(app.CodeVerification, "selected package ledger does not match manifest ledger_path", err)
	}
	result := PublicationVerifyResult{ManifestPath: "manifest.json", ManifestSHA256: storage.ResourceDigest(manifestBytes), FileCount: len(manifest.Entries) + 1, Evidence: []ForecastVerification{}, Limitations: append([]string(nil), verificationLimitations...), NetworkProfile: networkProfileForObserver(observer, !options.Online)}
	listed := map[string]struct{}{"manifest.json": {}}
	for _, entry := range manifest.Entries {
		absolute, err := resolver.Resolve(entry.Path, true)
		if err != nil {
			return result, app.NewError(app.CodeVerification, "a listed package entry is missing or unsafe", err)
		}
		data, err := readBoundedFile(absolute, 64<<20)
		if err != nil {
			return result, app.NewError(app.CodeVerification, "a listed package entry cannot be read", err)
		}
		file := publicationFile(entry.Role, entry.Path, data)
		if file.Size != entry.Size || file.SHA256 != entry.Digest.Value {
			return result, app.NewError(app.CodeVerification, "a listed package entry has different bytes", nil)
		}
		listed[entry.Path] = struct{}{}
		result.Files = append(result.Files, file)
		result.TotalBytes += file.Size
	}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return app.NewError(app.CodeVerification, "package contains an unsafe file type", nil)
		}
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		if _, ok := listed[relative]; !ok {
			return app.WithDetails(app.NewError(app.CodeVerification, "package contains an unexpected file", nil), map[string]any{"path": relative})
		}
		return nil
	}); err != nil {
		return result, err
	}
	loaded, err := LoadAndValidateLedgerWithArtifactRoot(ctx, resolvedLedger, root)
	if err != nil {
		return result, err
	}
	result.LedgerID = loaded.Model.LedgerID
	for _, question := range loaded.Model.Questions {
		for _, forecast := range question.Forecasts {
			content := verifyPackageContent(root, loaded.Model, question, forecast)
			timing := verifyPackageTiming(ctx, root, loaded.Model, question, forecast, content, options.Online, observer)
			reveal := verifyRevealLayer(question, forecast, content)
			outcome := verifyOutcomeLayer(ctx, question, VerificationOptions{Offline: true})
			result.Evidence = append(result.Evidence, ForecastVerification{QuestionID: question.ID, ForecastID: forecast.ID, Layers: []VerificationLayer{content, timing, reveal, outcome}})
		}
	}
	temporaryReport := VerificationReport{Forecasts: result.Evidence}
	result.Overall, result.FailureCode = aggregateVerification(temporaryReport)
	if observer != nil {
		result.RequestSummary = observer.Summary()
	}
	return result, nil
}

func verifyPackageContent(root string, model *ledger.Ledger, question ledger.Question, forecast ledger.Forecast) VerificationLayer {
	target := recordedForecastTarget(model, question.ID, forecast.ID)
	if target == nil {
		return VerificationLayer{Name: "content_binding", State: LayerNotApplicable, ReasonCodes: []string{"content.no_retained_target"}}
	}
	artifact, err := BuildForecastTarget(model, question.ID, forecast.ID)
	if err != nil || *target != TargetMetadataFor(artifact) {
		return failedLayer("content_binding", "content.target_metadata_mismatch", err)
	}
	data, err := readConfinedArtifact(root, string(target.ArtifactPath), maxTargetBytes)
	if err != nil || !bytes.Equal(data, artifact.Bytes) {
		return failedLayer("content_binding", "content.target_mismatch", err)
	}
	return VerificationLayer{Name: "content_binding", State: LayerPass, ReasonCodes: []string{"content.target_matches"}, Evidence: map[string]any{"path": target.ArtifactPath, "sha256": artifact.SHA256}}
}

func verifyPackageTiming(ctx context.Context, root string, model *ledger.Ledger, question ledger.Question, forecast ledger.Forecast, content VerificationLayer, online bool, observer ots.BitcoinObserver) VerificationLayer {
	if forecast.Integrity.Unanchored != nil {
		return VerificationLayer{Name: "existence_timing", State: LayerNotApplicable, ReasonCodes: []string{"timing.unanchored"}}
	}
	if content.State != LayerPass {
		return VerificationLayer{Name: "existence_timing", State: LayerNotChecked, ReasonCodes: []string{"timing.blocked_by_content"}}
	}
	var timestamps []ledger.OTSTimestamp
	if forecast.Integrity.Pending != nil {
		timestamps = forecast.Integrity.Pending.Timestamps
	} else if forecast.Integrity.Verified != nil {
		timestamps = forecast.Integrity.Verified.Timestamps
	} else {
		return failedLayer("existence_timing", "timing.imported_failed", nil)
	}
	artifact, _ := BuildForecastTarget(model, question.ID, forecast.ID)
	for _, timestamp := range timestamps {
		if timestamp.Type != "opentimestamps" {
			continue
		}
		data, err := readConfinedArtifact(root, string(timestamp.ProofPath), maxReceiptBytes)
		if err != nil {
			return failedLayer("existence_timing", "timing.receipt_missing", err)
		}
		receipt, err := ots.ParseReceipt(data)
		if err != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
			return failedLayer("existence_timing", "timing.receipt_invalid", err)
		}
		evaluated, err := receipt.Evaluate()
		if err != nil {
			return failedLayer("existence_timing", "timing.proof_invalid", err)
		}
		bitcoin := make([]ots.EvaluatedAttestation, 0)
		for _, item := range evaluated {
			if item.Attestation.Kind == ots.AttestationBitcoin {
				bitcoin = append(bitcoin, item)
			}
		}
		if err := verifiedTimestampMatchesReceipt(forecast, evaluated); err != nil {
			return failedLayer("existence_timing", "timing.stored_metadata_mismatch", err)
		}
		if online && len(bitcoin) > 0 {
			sort.Slice(bitcoin, func(i, j int) bool { return bitcoin[i].Attestation.Height < bitcoin[j].Attestation.Height })
			for _, item := range bitcoin {
				observation, observeErr := observer.Observe(ctx, item.Attestation.Height)
				if observeErr != nil {
					return VerificationLayer{Name: "existence_timing", State: LayerNotChecked, ReasonCodes: []string{"timing.source_unavailable"}, Evidence: map[string]any{"height": item.Attestation.Height}}
				}
				if verifyErr := ots.VerifyBitcoinAttestation(item, observation); verifyErr != nil {
					return failedLayer("existence_timing", "timing.bitcoin_mismatch", verifyErr)
				}
				return VerificationLayer{Name: "existence_timing", State: LayerPass, ReasonCodes: []string{"timing.bitcoin_verified"}, Evidence: map[string]any{"height": item.Attestation.Height, "block_hash": observation.Hash, "anchored_before": observation.BlockTime.Format(time.RFC3339), "source_ids": observation.SourceIDs}}
			}
		}
		if timestamp.State == ledger.OTSConfirmed && timestamp.AnchoredBefore != nil {
			return VerificationLayer{Name: "existence_timing", State: LayerPass, ReasonCodes: []string{"timing.stored_verification_consistent"}, Limitations: []string{"The prior Bitcoin source identity is not retained and was not rechecked offline."}}
		}
		return VerificationLayer{Name: "existence_timing", State: LayerPending, ReasonCodes: []string{"timing.calendar_pending"}}
	}
	return failedLayer("existence_timing", "timing.receipt_reference_missing", nil)
}

func publicationFile(role, path string, data []byte) PublicationFile {
	digest := sha256.Sum256(data)
	return PublicationFile{Role: role, Path: path, Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:]), bytes: append([]byte(nil), data...)}
}

func readConfinedArtifact(root, relative string, limit int64) ([]byte, error) {
	resolver, err := storage.NewPathResolver(root)
	if err != nil {
		return nil, err
	}
	absolute, err := resolver.Resolve(relative, true)
	if err != nil {
		return nil, err
	}
	return readBoundedFile(absolute, limit)
}
