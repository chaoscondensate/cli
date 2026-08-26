package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
	"github.com/chaoscondensate/cli/internal/validation"
)

const maxReceiptBytes = ots.MaxReceiptBytes

type TimestampState string

const (
	TimestampUnanchored          TimestampState = "unanchored"
	TimestampPending             TimestampState = "pending"
	TimestampConfirmedUnverified TimestampState = "confirmed_unverified"
	TimestampVerified            TimestampState = "verified"
	TimestampFailed              TimestampState = "failed"
	TimestampInconsistent        TimestampState = "inconsistent"
)

type TimestampArtifactResult struct {
	QuestionID        ledger.Slug         `json:"question_id"`
	ForecastID        ledger.Slug         `json:"forecast_id"`
	State             TimestampState      `json:"state"`
	TargetPath        ledger.RelativePath `json:"target_path"`
	TargetSHA256      string              `json:"target_sha256"`
	ReceiptPath       ledger.RelativePath `json:"receipt_path"`
	TargetPresent     bool                `json:"target_present"`
	ReceiptPresent    bool                `json:"receipt_present"`
	CalendarSourceIDs []string            `json:"calendar_source_ids,omitempty"`
	CalendarIdentity  []string            `json:"calendar_identities,omitempty"`
	BitcoinHeight     *uint64             `json:"bitcoin_height,omitempty"`
	AnchoredBefore    *ledger.Timestamp   `json:"anchored_before,omitempty"`
	NetworkProfile    NetworkProfile      `json:"network_profile"`
	RequestSummary    ots.RequestSummary  `json:"request_summary,omitempty"`
	NextActions       []string            `json:"next_actions,omitempty"`
	Warnings          []Warning           `json:"warnings,omitempty"`
	Effects           []SideEffect        `json:"effects,omitempty"`
	Recovery          Recovery            `json:"recovery,omitempty"`
}

type TimestampStampOptions struct {
	DryRun                bool
	Offline               bool
	CustomCalendars       []string
	CalendarMinimum       int
	Effects               Effects
	CalendarClient        *ots.CalendarClient
	ResolvedCalendarInput []string
}

type TimestampVerifyOptions struct {
	DryRun     bool
	Offline    bool
	VerifiedAt ledger.Timestamp
	Observer   ots.BitcoinObserver
}

type TimestampUpgradeOptions struct {
	DryRun          bool
	Offline         bool
	CustomCalendars []string
	CalendarMinimum int
	CalendarClient  *ots.CalendarClient
}

func ReceiptRelativePath(forecastID ledger.Slug) ledger.RelativePath {
	return ledger.RelativePath(storage.DeterministicRelativePath("proofs/receipts", string(forecastID)+".json.ots"))
}

func ProtectedCoreObserver(endpoint, authPath string) (ots.BitcoinObserver, error) {
	if strings.TrimSpace(endpoint) == "" && strings.TrimSpace(authPath) == "" {
		return nil, nil
	}
	if strings.TrimSpace(endpoint) == "" || strings.TrimSpace(authPath) == "" {
		return nil, app.NewError(app.CodeUsage, "--bitcoin-core and --bitcoin-auth-file must be supplied together", nil)
	}
	data, err := storage.ReadProtectedFile(authPath, 4096)
	if err != nil {
		return nil, err
	}
	defer clear(data)
	auth, err := ots.DecodeCoreAuth(data)
	if err != nil {
		return nil, app.NewError(app.CodeInvalidData, "Bitcoin Core auth file is invalid", err)
	}
	observer, err := ots.NewCoreObserver(endpoint, auth, nil)
	if err != nil {
		return nil, app.NewError(app.CodeInvalidData, "Bitcoin Core configuration is invalid", err)
	}
	return observer, nil
}

func TimestampStatusFor(ctx context.Context, path string, questionID, forecastID ledger.Slug) (TimestampArtifactResult, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	artifact, err := BuildForecastTarget(loaded.Model, questionID, forecastID)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	_, _, _, forecast, err := selectForecast(loaded.Model, questionID, forecastID)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	result := baseTimestampResult(artifact)
	root := filepath.Dir(loaded.Path)
	result.TargetPresent = regularFileExists(filepath.Join(root, filepath.FromSlash(string(artifact.RelativePath))))
	result.ReceiptPresent = regularFileExists(filepath.Join(root, filepath.FromSlash(string(result.ReceiptPath))))
	switch {
	case forecast.Integrity.Unanchored != nil:
		result.State = TimestampUnanchored
		result.NextActions = []string{"timestamp stamp"}
		return result, nil
	case forecast.Integrity.Failed != nil:
		result.State = TimestampFailed
		result.NextActions = []string{"forecast add --supersedes-forecast-id " + string(forecastID)}
		return result, nil
	case forecast.Integrity.Pending == nil && forecast.Integrity.Verified == nil:
		result.State = TimestampInconsistent
		return result, app.NewError(app.CodeVerification, "forecast integrity state is inconsistent", nil)
	}
	if !result.TargetPresent || !result.ReceiptPresent {
		result.State = TimestampInconsistent
		return result, app.NewError(app.CodeVerification, "timestamp metadata references a missing artifact", nil)
	}
	if _, err := CheckTargets(ctx, loaded.Path, false, questionID, forecastID); err != nil {
		result.State = TimestampInconsistent
		return result, err
	}
	receiptBytes, err := readBoundedFile(filepath.Join(root, filepath.FromSlash(string(result.ReceiptPath))), maxReceiptBytes)
	if err != nil {
		result.State = TimestampInconsistent
		return result, err
	}
	receipt, err := ots.ParseReceipt(receiptBytes)
	if err != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
		result.State = TimestampInconsistent
		return result, app.NewError(app.CodeVerification, "OpenTimestamps receipt does not bind the selected target", err)
	}
	evaluated, err := receipt.Evaluate()
	if err != nil {
		result.State = TimestampInconsistent
		return result, app.NewError(app.CodeVerification, "OpenTimestamps receipt cannot be evaluated", err)
	}
	for _, item := range evaluated {
		if item.Attestation.Kind == ots.AttestationBitcoin {
			height := item.Attestation.Height
			result.BitcoinHeight = &height
		}
	}
	if forecast.Integrity.Verified != nil {
		if err := verifiedTimestampMatchesReceipt(forecast, evaluated); err != nil {
			result.State = TimestampInconsistent
			return result, err
		}
		result.State = TimestampVerified
		for _, timestamp := range forecast.Integrity.Verified.Timestamps {
			if timestamp.Type == "opentimestamps" && timestamp.AnchoredBefore != nil {
				bound := *timestamp.AnchoredBefore
				result.AnchoredBefore = &bound
			}
		}
		result.NextActions = []string{"verify --offline", "timestamp verify"}
	} else if result.BitcoinHeight != nil {
		result.State = TimestampConfirmedUnverified
		result.NextActions = []string{"timestamp verify"}
	} else {
		result.State = TimestampPending
		result.NextActions = []string{"timestamp upgrade", "timestamp status"}
	}
	return result, nil
}

func PlanTimestampStamp(ctx context.Context, path string, questionID, forecastID ledger.Slug) (TimestampArtifactResult, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	artifact, forecast, err := timestampPreflight(loaded.Model, questionID, forecastID)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	if forecast.Integrity.Unanchored == nil {
		status, statusErr := TimestampStatusFor(ctx, path, questionID, forecastID)
		return status, statusErr
	}
	root := filepath.Dir(loaded.Path)
	if _, err := preflightTargetFile(root, artifact); err != nil {
		return TimestampArtifactResult{}, err
	}
	receiptPath := filepath.Join(root, filepath.FromSlash(string(ReceiptRelativePath(forecastID))))
	if info, err := os.Lstat(receiptPath); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return TimestampArtifactResult{}, app.NewError(app.CodeConflict, "receipt destination is not a regular file", nil)
		}
		data, err := readBoundedFile(receiptPath, maxReceiptBytes)
		if err != nil {
			return TimestampArtifactResult{}, err
		}
		receipt, err := ots.ParseReceipt(data)
		if err != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
			return TimestampArtifactResult{}, app.NewError(app.CodeConflict, "existing receipt does not match the selected target", err)
		}
	} else if !os.IsNotExist(err) {
		return TimestampArtifactResult{}, app.NewError(app.CodeIO, "receipt destination cannot be inspected", err)
	}
	result := baseTimestampResult(artifact)
	result.Effects = []SideEffect{{Kind: EffectTarget, Action: EffectCreate, Status: EffectDeferred, Path: string(result.TargetPath), Rollback: RollbackCreatedPublic}, {Kind: EffectNetwork, Action: EffectContact, Status: EffectDeferred, SourceID: ots.PublicProfileID}, {Kind: EffectReceipt, Action: EffectCreate, Status: EffectDeferred, Path: string(result.ReceiptPath), Rollback: RollbackCreatedPublic}, {Kind: EffectLedger, Action: EffectReplace, Status: EffectDeferred, Path: filepath.Base(loaded.Path)}}
	return result, nil
}

func CommitTimestampStamp(ctx context.Context, path string, questionID, forecastID ledger.Slug, options TimestampStampOptions) (TimestampArtifactResult, error) {
	if options.Offline {
		return TimestampArtifactResult{}, app.NewError(app.CodeNetworkDisabled, "timestamp stamp requires network access", nil)
	}
	if options.DryRun {
		return PlanTimestampStamp(ctx, path, questionID, forecastID)
	}
	if err := options.Effects.Validate(); err != nil {
		options.Effects = ProductionEffects()
	}
	planned, err := PlanTimestampStamp(ctx, path, questionID, forecastID)
	if err != nil {
		return planned, err
	}
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return planned, err
	}
	artifact, _, err := timestampPreflight(loaded.Model, questionID, forecastID)
	if err != nil {
		return planned, err
	}
	root := filepath.Dir(loaded.Path)
	receiptAbsolute := filepath.Join(root, filepath.FromSlash(string(planned.ReceiptPath)))
	var receiptBytes []byte
	if existing, readErr := readBoundedFile(receiptAbsolute, maxReceiptBytes); readErr == nil {
		receipt, parseErr := ots.ParseReceipt(existing)
		if parseErr != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
			return planned, app.NewError(app.CodeConflict, "existing receipt does not match the selected target", parseErr)
		}
		receiptBytes = existing
	} else if app.ErrorCodeOf(readErr) != app.CodeNotFound {
		return planned, readErr
	}
	if len(receiptBytes) == 0 {
		var nonce [16]byte
		if err := options.Effects.Random.ReadFull(ctx, nonce[:]); err != nil {
			return planned, app.NewError(app.CodeIO, "timestamp nonce could not be generated", err)
		}
		client := options.CalendarClient
		if client == nil {
			client = ots.NewCalendarClient()
		}
		commitment := ots.Blind(mustDigest32(artifact.SHA256), nonce)
		var responses []ots.CalendarResult
		if len(options.CustomCalendars) > 0 {
			responses, err = client.StampCustom(ctx, options.CustomCalendars, options.CalendarMinimum, commitment)
			planned.NetworkProfile = customCalendarProfile(responses, options.CalendarMinimum)
		} else {
			responses, err = client.StampPublic(ctx, commitment)
		}
		for _, response := range responses {
			if response.Err == nil {
				planned.CalendarSourceIDs = append(planned.CalendarSourceIDs, response.SourceID)
				planned.CalendarIdentity = append(planned.CalendarIdentity, response.Identity)
			}
		}
		if err != nil {
			return planned, app.NewError(app.CodeNetwork, "calendar submission threshold was not reached", err)
		}
		branches := make([]ots.Sequence, 0, len(responses))
		for _, response := range responses {
			if response.Err == nil {
				branches = append(branches, response.Branch)
			}
		}
		receipt, err := ots.NewPendingReceipt(mustDigest32(artifact.SHA256), nonce, branches)
		if err != nil {
			return planned, app.NewError(app.CodeVerification, "calendar receipts could not be merged", err)
		}
		receiptBytes, err = receipt.Serialize()
		if err != nil {
			return planned, app.NewError(app.CodeVerification, "OpenTimestamps receipt could not be encoded", err)
		}
	}
	return commitTimestampPending(ctx, loaded.Path, questionID, forecastID, artifact, receiptBytes, planned)
}

func CommitTimestampUpgrade(ctx context.Context, path string, questionID, forecastID ledger.Slug, options TimestampUpgradeOptions) (TimestampArtifactResult, error) {
	if options.Offline {
		return TimestampArtifactResult{}, app.NewError(app.CodeNetworkDisabled, "timestamp upgrade requires network access", nil)
	}
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	artifact, forecast, err := timestampPreflight(loaded.Model, questionID, forecastID)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	result := baseTimestampResult(artifact)
	if forecast.Integrity.Pending == nil {
		return result, app.NewError(app.CodeConflict, "timestamp upgrade requires pending forecast integrity", nil)
	}
	if _, err := CheckTargets(ctx, loaded.Path, false, questionID, forecastID); err != nil {
		return result, err
	}
	receiptAbsolute := filepath.Join(filepath.Dir(loaded.Path), filepath.FromSlash(string(result.ReceiptPath)))
	receiptBytes, err := readBoundedFile(receiptAbsolute, maxReceiptBytes)
	if err != nil {
		return result, err
	}
	receipt, err := ots.ParseReceipt(receiptBytes)
	if err != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
		return result, app.NewError(app.CodeVerification, "receipt does not bind the selected target", err)
	}
	evaluated, err := receipt.Evaluate()
	if err != nil {
		return result, app.NewError(app.CodeVerification, "receipt cannot be evaluated", err)
	}
	allowed := make(map[string]struct{})
	if len(options.CustomCalendars) > 0 {
		validated, validationErr := ots.ValidateCustomCalendars(ctx, nil, options.CustomCalendars, options.CalendarMinimum)
		if validationErr != nil {
			return result, app.NewError(app.CodeInvalidData, "custom calendar configuration is invalid", validationErr)
		}
		for _, endpoint := range validated {
			allowed[strings.TrimRight(endpoint, "/")] = struct{}{}
		}
		result.NetworkProfile = NetworkProfile{Mode: NetworkCustom, ID: "custom", SourceIDs: validated, MinimumSuccess: options.CalendarMinimum, TrustLimitations: []string{"Custom calendar endpoints are caller-selected and are not reviewed by the built-in profile."}, PrivacyLimitations: []string{"Calendar services learn request timing and blinded commitments."}}
	} else {
		for _, source := range ots.Profile().Calendars {
			for _, identity := range source.AcceptedIdentities {
				allowed[strings.TrimRight(identity, "/")] = struct{}{}
			}
		}
	}
	client := options.CalendarClient
	if client == nil {
		client = ots.NewCalendarClient()
	}
	additions := make([]ots.Sequence, 0)
	for _, item := range evaluated {
		if item.Attestation.Kind != ots.AttestationPending {
			continue
		}
		calendar := strings.TrimRight(item.Attestation.Calendar, "/")
		if _, ok := allowed[calendar]; !ok {
			result.Warnings = append(result.Warnings, Warning{Code: "calendar.not_checked", Message: "A pending calendar identity is outside the selected profile."})
			continue
		}
		if options.DryRun {
			result.Effects = append(result.Effects, SideEffect{Kind: EffectNetwork, Action: EffectContact, Status: EffectDeferred, SourceID: calendar})
			continue
		}
		tails, upgradeErr := client.Upgrade(ctx, calendar, item.Message)
		if upgradeErr != nil {
			continue
		}
		prefix := cloneOTSSequenceWithoutAttestation(item.Sequence)
		for _, tail := range tails {
			candidate := append(prefix, tail...)
			additions = append(additions, candidate)
		}
	}
	if options.DryRun {
		result.Effects = append(result.Effects, SideEffect{Kind: EffectReceipt, Action: EffectReplace, Status: EffectDeferred, Path: string(result.ReceiptPath)})
		return result, nil
	}
	if len(additions) == 0 {
		return result, app.NewError(app.CodePending, "no calendar upgrade is ready", nil)
	}
	addition := &ots.Receipt{Digest: receipt.Digest, Sequences: additions}
	merged, err := ots.Merge(receipt, addition)
	if err != nil || !ots.IsSemanticSuperset(merged, receipt) {
		return result, app.NewError(app.CodeVerification, "calendar upgrade is not a semantic proof superset", err)
	}
	updated, err := merged.Serialize()
	if err != nil {
		return result, app.NewError(app.CodeVerification, "upgraded receipt cannot be encoded", err)
	}
	if bytes.Equal(updated, receiptBytes) {
		return result, app.NewError(app.CodePending, "no new calendar proof branch was available", nil)
	}
	if _, err := storage.ReplaceDeterministicFile(receiptAbsolute, updated, 0o644, maxReceiptBytes); err != nil {
		return result, err
	}
	result.ReceiptPresent = true
	result.Effects = []SideEffect{{Kind: EffectReceipt, Action: EffectReplace, Status: EffectCompleted, Path: string(result.ReceiptPath)}}
	for _, sequence := range additions {
		if len(sequence) > 0 && sequence[len(sequence)-1].Attestation != nil && sequence[len(sequence)-1].Attestation.Kind == ots.AttestationBitcoin {
			height := sequence[len(sequence)-1].Attestation.Height
			result.BitcoinHeight = &height
			result.State = TimestampConfirmedUnverified
		}
	}
	if result.BitcoinHeight == nil {
		result.State = TimestampPending
	}
	return result, nil
}

func CommitTimestampVerify(ctx context.Context, path string, questionID, forecastID ledger.Slug, options TimestampVerifyOptions) (TimestampArtifactResult, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	artifact, _, err := timestampPreflight(loaded.Model, questionID, forecastID)
	if err != nil {
		return TimestampArtifactResult{}, err
	}
	result := baseTimestampResult(artifact)
	if _, err := CheckTargets(ctx, loaded.Path, false, questionID, forecastID); err != nil {
		return result, err
	}
	receiptBytes, err := readBoundedFile(filepath.Join(filepath.Dir(loaded.Path), filepath.FromSlash(string(result.ReceiptPath))), maxReceiptBytes)
	if err != nil {
		return result, err
	}
	receipt, err := ots.ParseReceipt(receiptBytes)
	if err != nil || receipt.VerifyBinding(artifact.Bytes) != nil {
		return result, app.NewError(app.CodeVerification, "receipt does not bind the selected target", err)
	}
	evaluated, err := receipt.Evaluate()
	if err != nil {
		return result, app.NewError(app.CodeVerification, "receipt cannot be evaluated", err)
	}
	bitcoin := make([]ots.EvaluatedAttestation, 0)
	for _, item := range evaluated {
		if item.Attestation.Kind == ots.AttestationBitcoin {
			bitcoin = append(bitcoin, item)
		}
	}
	if len(bitcoin) == 0 {
		result.State = TimestampPending
		return result, app.NewError(app.CodePending, "OpenTimestamps receipt is still pending", nil)
	}
	if options.DryRun {
		result.State = TimestampConfirmedUnverified
		result.Effects = []SideEffect{{Kind: EffectNetwork, Action: EffectContact, Status: EffectDeferred, SourceID: result.NetworkProfile.ID}, {Kind: EffectLedger, Action: EffectReplace, Status: EffectDeferred, Path: filepath.Base(loaded.Path)}}
		return result, nil
	}
	if options.Offline {
		result.State = TimestampConfirmedUnverified
		return result, app.NewError(app.CodePending, "Bitcoin block evidence was not checked in offline mode", nil)
	}
	observer := options.Observer
	if observer == nil {
		observer = ots.NewPublicBitcoinObserver(nil)
	}
	result.NetworkProfile = networkProfileForObserver(observer, false)
	sort.Slice(bitcoin, func(i, j int) bool { return bitcoin[i].Attestation.Height < bitcoin[j].Attestation.Height })
	var verified ots.EvaluatedAttestation
	var observation ots.BlockObservation
	for _, item := range bitcoin {
		observation, err = observer.Observe(ctx, item.Attestation.Height)
		if err == nil {
			err = ots.VerifyBitcoinAttestation(item, observation)
		}
		if err == nil {
			verified = item
			break
		}
	}
	result.RequestSummary = observer.Summary()
	if err != nil || verified.Attestation.Height == 0 {
		return result, app.NewError(app.CodeVerification, "Bitcoin evidence did not verify", err)
	}
	height := verified.Attestation.Height
	bound := ledger.Timestamp(observation.BlockTime.Format(time.RFC3339))
	result.BitcoinHeight, result.AnchoredBefore, result.State = &height, &bound, TimestampVerified
	_, question, selectErr := selectQuestion(loaded.Model, questionID)
	if selectErr == nil && question.Resolution != nil && question.Resolution.Resolved != nil {
		outcomeKnownAt, parseErr := ParseTimestamp(question.Resolution.Resolved.OutcomeKnownAt, "outcome_known_at")
		if parseErr == nil && !observation.BlockTime.Before(outcomeKnownAt) {
			result.Warnings = append(result.Warnings, Warning{Code: "timestamp.valid_but_too_late", Message: "The Bitcoin evidence is valid, but its conservative time bound is not before the recorded outcome became known."})
		}
	}
	if options.VerifiedAt == "" {
		options.VerifiedAt = ledger.Timestamp(time.Now().Format(time.RFC3339))
	}
	return commitTimestampVerified(ctx, loaded.Path, questionID, forecastID, artifact, verified.Attestation.Height, bound, options.VerifiedAt, result)
}

func baseTimestampResult(artifact TargetArtifact) TimestampArtifactResult {
	return TimestampArtifactResult{QuestionID: artifact.QuestionID, ForecastID: artifact.ForecastID, State: TimestampPending, TargetPath: artifact.RelativePath, TargetSHA256: artifact.SHA256, ReceiptPath: ReceiptRelativePath(artifact.ForecastID), NetworkProfile: networkProfileForObserver(nil, false)}
}

func networkProfileForObserver(observer ots.BitcoinObserver, offline bool) NetworkProfile {
	profile := ots.Profile()
	if offline {
		return NetworkProfile{Mode: NetworkOffline, ID: profile.ID, MaxUniqueHeights: profile.MaximumUniqueHeights, MaxRequests: profile.MaximumHTTPRequests, MaxConcurrent: profile.MaximumConcurrentHTTP, TrustLimitations: []string{profile.TrustLimitation}, PrivacyLimitations: []string{profile.PrivacyLimitation}}
	}
	if _, core := observer.(*ots.CoreObserver); core {
		return NetworkProfile{Mode: NetworkCore, ID: "bitcoin-core", SourceIDs: []string{"bitcoin-core"}, MaxUniqueHeights: profile.MaximumUniqueHeights, MaxRequests: profile.MaximumHTTPRequests, MaxConcurrent: 1, TrustLimitations: []string{"Bitcoin timing is checked against the caller-selected Bitcoin Core node."}, PrivacyLimitations: []string{"The selected Bitcoin Core operator can observe requested block heights unless the node is operated locally."}}
	}
	sourceIDs := make([]string, len(profile.BitcoinSources))
	for index := range profile.BitcoinSources {
		sourceIDs[index] = profile.BitcoinSources[index].ID
	}
	return NetworkProfile{Mode: NetworkBuiltin, ID: profile.ID, SourceIDs: sourceIDs, MinimumSuccess: profile.CalendarMinimum, MaxUniqueHeights: profile.MaximumUniqueHeights, MaxRequests: profile.MaximumHTTPRequests, MaxConcurrent: profile.MaximumConcurrentHTTP, TrustLimitations: []string{profile.TrustLimitation}, PrivacyLimitations: []string{profile.PrivacyLimitation}}
}

func customCalendarProfile(results []ots.CalendarResult, minimum int) NetworkProfile {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.SourceID)
	}
	sort.Strings(ids)
	return NetworkProfile{Mode: NetworkCustom, ID: "custom", SourceIDs: ids, MinimumSuccess: minimum, TrustLimitations: []string{"Custom calendar endpoints are caller-selected and are not reviewed by the built-in profile."}, PrivacyLimitations: []string{"Calendar services learn request timing and blinded commitments."}}
}

func timestampPreflight(model *ledger.Ledger, questionID, forecastID ledger.Slug) (TargetArtifact, ledger.Forecast, error) {
	artifact, err := BuildForecastTarget(model, questionID, forecastID)
	if err != nil {
		return TargetArtifact{}, ledger.Forecast{}, err
	}
	_, _, _, forecast, err := selectForecast(model, questionID, forecastID)
	if err != nil {
		return TargetArtifact{}, ledger.Forecast{}, err
	}
	if forecast.Integrity.Failed != nil {
		return TargetArtifact{}, ledger.Forecast{}, app.NewError(app.CodeConflict, "failed integrity is terminal; append a new forecast revision", nil)
	}
	return artifact, forecast, nil
}

func commitTimestampPending(ctx context.Context, path string, questionID, forecastID ledger.Slug, artifact TargetArtifact, receiptBytes []byte, result TimestampArtifactResult) (TimestampArtifactResult, error) {
	root := filepath.Dir(path)
	if err := os.MkdirAll(filepath.Join(root, "proofs", "targets"), 0o755); err != nil {
		return result, app.NewError(app.CodeIO, "target directory cannot be created", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "proofs", "receipts"), 0o755); err != nil {
		return result, app.NewError(app.CodeIO, "receipt directory cannot be created", err)
	}
	targetAbsolute := filepath.Join(root, filepath.FromSlash(string(artifact.RelativePath)))
	receiptAbsolute := filepath.Join(root, filepath.FromSlash(string(result.ReceiptPath)))
	if _, err := storage.EnsureDeterministicFile(targetAbsolute, artifact.Bytes, 0o644, maxTargetBytes); err != nil {
		return result, err
	}
	if _, err := storage.EnsureDeterministicFile(receiptAbsolute, receiptBytes, 0o644, maxReceiptBytes); err != nil {
		return result, err
	}
	artifacts := os.DirFS(root)
	err := storage.UpdateLedger(ctx, path, storage.TransactionOptions{Validate: func(parsed *document.Document) error { return ValidateLedgerDocument(parsed, artifacts) }, Mutate: func(parsed *document.Document) ([]byte, error) {
		model, err := validation.DecodeLedger(parsed)
		if err != nil {
			return nil, err
		}
		questionPosition, _, forecastPosition, forecast, err := selectForecast(model, questionID, forecastID)
		if err != nil {
			return nil, err
		}
		if forecast.Integrity.Pending != nil {
			return parsed.Raw, nil
		}
		if forecast.Integrity.Unanchored == nil {
			return nil, app.NewError(app.CodeConflict, "forecast integrity is not unanchored", nil)
		}
		integrity := ledger.PendingIntegrity{Status: ledger.IntegrityPending, Target: TargetMetadataFor(artifact), Timestamps: []ledger.OTSTimestamp{{Type: "opentimestamps", ProofPath: result.ReceiptPath, State: ledger.OTSPending}}}
		value, err := jsonPatchValue(integrity)
		if err != nil {
			return nil, err
		}
		pointer := fmt.Sprintf("/questions/%d/forecasts/%d/integrity", questionPosition, forecastPosition)
		return document.ApplyPatch(parsed, []document.PatchOperation{{Kind: document.PatchReplace, Pointer: pointer, Value: value}})
	}})
	if err != nil {
		result.Recovery = Recovery{State: RecoveryRetained, Message: "Target and receipt are durable and can be reused by a retry.", Paths: []string{string(result.TargetPath), string(result.ReceiptPath)}, Actions: []string{"retry timestamp stamp"}}
		return result, err
	}
	result.TargetPresent, result.ReceiptPresent, result.State = true, true, TimestampPending
	result.Effects = []SideEffect{{Kind: EffectTarget, Action: EffectCreate, Status: EffectCompleted, Path: string(result.TargetPath)}, {Kind: EffectReceipt, Action: EffectCreate, Status: EffectCompleted, Path: string(result.ReceiptPath)}, {Kind: EffectLedger, Action: EffectReplace, Status: EffectCompleted, Path: filepath.Base(path)}}
	return result, nil
}

func commitTimestampVerified(ctx context.Context, path string, questionID, forecastID ledger.Slug, artifact TargetArtifact, height uint64, anchoredBefore, verifiedAt ledger.Timestamp, result TimestampArtifactResult) (TimestampArtifactResult, error) {
	root := filepath.Dir(path)
	artifacts := os.DirFS(root)
	err := storage.UpdateLedger(ctx, path, storage.TransactionOptions{Validate: func(parsed *document.Document) error { return ValidateLedgerDocument(parsed, artifacts) }, Mutate: func(parsed *document.Document) ([]byte, error) {
		model, err := validation.DecodeLedger(parsed)
		if err != nil {
			return nil, err
		}
		questionPosition, _, forecastPosition, forecast, err := selectForecast(model, questionID, forecastID)
		if err != nil {
			return nil, err
		}
		if forecast.Integrity.Verified != nil {
			return parsed.Raw, nil
		}
		if forecast.Integrity.Pending == nil {
			return nil, app.NewError(app.CodeConflict, "only pending integrity can become verified", nil)
		}
		pending := forecast.Integrity.Pending
		if pending.Target != TargetMetadataFor(artifact) {
			return nil, app.NewError(app.CodeVerification, "pending target metadata changed before verification", nil)
		}
		timestamps := append([]ledger.OTSTimestamp(nil), pending.Timestamps...)
		found := false
		for index := range timestamps {
			if timestamps[index].Type == "opentimestamps" && timestamps[index].ProofPath == ReceiptRelativePath(forecastID) {
				heightValue := int64(height)
				timestamps[index].State = ledger.OTSConfirmed
				timestamps[index].AnchoredBefore = &anchoredBefore
				timestamps[index].BitcoinBlockHeight = &heightValue
				found = true
			}
		}
		if !found {
			return nil, app.NewError(app.CodeVerification, "pending integrity has no matching OpenTimestamps entry", nil)
		}
		verified := ledger.VerifiedIntegrity{Status: ledger.IntegrityVerified, Target: pending.Target, Timestamps: timestamps, VerifiedAt: verifiedAt, ExternalAnchors: pending.ExternalAnchors}
		value, err := jsonPatchValue(verified)
		if err != nil {
			return nil, err
		}
		pointer := fmt.Sprintf("/questions/%d/forecasts/%d/integrity", questionPosition, forecastPosition)
		return document.ApplyPatch(parsed, []document.PatchOperation{{Kind: document.PatchReplace, Pointer: pointer, Value: value}})
	}})
	if err != nil {
		return result, err
	}
	result.Effects = []SideEffect{{Kind: EffectLedger, Action: EffectReplace, Status: EffectCompleted, Path: filepath.Base(path)}}
	return result, nil
}

func regularFileExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func cloneOTSSequenceWithoutAttestation(sequence ots.Sequence) ots.Sequence {
	if len(sequence) == 0 {
		return nil
	}
	result := make(ots.Sequence, 0, len(sequence)-1)
	for _, step := range sequence[:len(sequence)-1] {
		if step.Operation == nil {
			continue
		}
		operation := *step.Operation
		operation.Argument = append([]byte(nil), operation.Argument...)
		result = append(result, ots.Step{Operation: &operation})
	}
	return result
}

func verifiedTimestampMatchesReceipt(forecast ledger.Forecast, evaluated []ots.EvaluatedAttestation) error {
	if forecast.Integrity.Verified == nil {
		return nil
	}
	if _, err := ParseTimestamp(forecast.Integrity.Verified.VerifiedAt, "verified_at"); err != nil {
		return app.NewError(app.CodeVerification, "stored timestamp verification time is invalid", err)
	}
	for _, timestamp := range forecast.Integrity.Verified.Timestamps {
		if timestamp.Type != "opentimestamps" || timestamp.ProofPath != ReceiptRelativePath(forecast.ID) {
			continue
		}
		if timestamp.State != ledger.OTSConfirmed || timestamp.AnchoredBefore == nil || timestamp.BitcoinBlockHeight == nil || *timestamp.BitcoinBlockHeight < 0 {
			return app.NewError(app.CodeVerification, "stored verified timestamp metadata is incomplete", nil)
		}
		if _, err := ParseTimestamp(*timestamp.AnchoredBefore, "anchored_before"); err != nil {
			return app.NewError(app.CodeVerification, "stored timestamp bound is invalid", err)
		}
		for _, item := range evaluated {
			if item.Attestation.Kind == ots.AttestationBitcoin && item.Attestation.Height == uint64(*timestamp.BitcoinBlockHeight) {
				return nil
			}
		}
		return app.NewError(app.CodeVerification, "stored Bitcoin height is absent from the receipt", nil)
	}
	return app.NewError(app.CodeVerification, "verified integrity has no matching OpenTimestamps entry", nil)
}

func mustDigest32(value string) [32]byte {
	var result [32]byte
	decoded, _ := hexDecode(value)
	copy(result[:], decoded)
	return result
}

func hexDecode(value string) ([]byte, error) {
	if len(value) != 64 || strings.ToLower(value) != value {
		return nil, errors.New("digest is not lowercase SHA-256")
	}
	return hex.DecodeString(value)
}
