package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/forecastcrypto"
	"github.com/chaoscondensate/cli/internal/ledger"
)

type SealedInitialBuild struct {
	Ledger  *ledger.Ledger
	KeyFile []byte
}

type SealedQuestionBuild struct {
	Mutation QuestionMutation
	KeyFile  []byte
}

// BuildInitialSealedLedger constructs a complete schema-valid initial ledger
// and the separate protected key bytes without performing filesystem effects.
// Callers must persist KeyFile before publishing Ledger.
func BuildInitialSealedLedger(ctx context.Context, root *ledger.Ledger, input InitialQuestionInput, effects Effects) (SealedInitialBuild, error) {
	if root == nil {
		return SealedInitialBuild{}, app.NewError(app.CodeInternal, "ledger root is nil", nil)
	}
	return BuildInitialSealedLedgerAt(ctx, root, input, root.CreatedAt, effects)
}

func BuildInitialSealedLedgerAt(ctx context.Context, root *ledger.Ledger, input InitialQuestionInput, observedAt ledger.Timestamp, effects Effects) (SealedInitialBuild, error) {
	var result SealedInitialBuild
	if err := effects.Validate(); err != nil {
		return result, app.NewError(app.CodeInternal, "sealing effects are not configured", err)
	}
	question, private, recordedAt, err := prepareInitialSealedLedger(root, input, observedAt)
	if err != nil {
		return result, err
	}
	sealed, err := forecastcrypto.Seal(ctx, question.ID, private.ID, forecastcrypto.PrivateBundle{
		ForecastedAt: private.ForecastedAt, RecordedAt: recordedAt, Value: private.Value,
		Rationale: *private.Rationale, KeyFactors: append([]string{}, (*private.KeyFactors)...), Comment: *private.Comment,
	}, "forecast-key:"+string(private.ID), effects.Random)
	if err != nil {
		return result, app.NewError(app.CodeIO, "sealed forecast could not be created", err)
	}
	question.Forecasts = []ledger.Forecast{sealedForecastRecord(private, recordedAt, sealed.Commitment)}
	prospective, err := appendProspectiveQuestion(root, question)
	if err != nil {
		return result, err
	}
	result.Ledger = prospective
	result.KeyFile = sealed.KeyFile
	return result, nil
}

// PlanInitialSealedLedger validates the complete prospective shape without
// reading entropy or returning cryptographic bytes. The placeholder commitment
// exists only in memory to exercise the pinned ledger schema during dry-run.
func PlanInitialSealedLedger(root *ledger.Ledger, input InitialQuestionInput) (*ledger.Ledger, error) {
	if root == nil {
		return nil, app.NewError(app.CodeInternal, "ledger root is nil", nil)
	}
	return PlanInitialSealedLedgerAt(root, input, root.CreatedAt)
}

func PlanInitialSealedLedgerAt(root *ledger.Ledger, input InitialQuestionInput, observedAt ledger.Timestamp) (*ledger.Ledger, error) {
	question, private, recordedAt, err := prepareInitialSealedLedger(root, input, observedAt)
	if err != nil {
		return nil, err
	}
	placeholder := ledger.SealedCommitment{
		Scheme:         forecastcrypto.SealScheme,
		CommitmentHash: ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(strings.Repeat("0", 64))},
		Encryption:     ledger.Encryption{Algorithm: forecastcrypto.EncryptionProfile, Nonce: "AAAAAAAAAAAAAAAA", Ciphertext: "AAAAAAAAAAAAAAAAAAAAAAAA"},
		KeyHint:        "forecast-key:" + string(private.ID),
	}
	question.Forecasts = []ledger.Forecast{sealedForecastRecord(private, recordedAt, placeholder)}
	return appendProspectiveQuestion(root, question)
}

// BuildQuestionAddSealed constructs one append-only question mutation and its
// separate key bytes. The caller must protect the key before committing the
// ledger patch.
func BuildQuestionAddSealed(ctx context.Context, model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp, effects Effects) (SealedQuestionBuild, error) {
	var result SealedQuestionBuild
	if err := effects.Validate(); err != nil {
		return result, app.NewError(app.CodeInternal, "sealing effects are not configured", err)
	}
	question, private, recordedAt, err := prepareQuestionAddSealed(model, input, observedAt)
	if err != nil {
		return result, err
	}
	sealed, err := forecastcrypto.Seal(ctx, question.ID, private.ID, forecastcrypto.PrivateBundle{
		ForecastedAt: private.ForecastedAt, RecordedAt: recordedAt, Value: private.Value,
		Rationale: *private.Rationale, KeyFactors: append([]string{}, (*private.KeyFactors)...), Comment: *private.Comment,
	}, "forecast-key:"+string(private.ID), effects.Random)
	if err != nil {
		return result, app.NewError(app.CodeIO, "sealed forecast could not be created", err)
	}
	question.Forecasts = []ledger.Forecast{sealedForecastRecord(private, recordedAt, sealed.Commitment)}
	prospective, err := appendProspectiveQuestion(model, question)
	if err != nil {
		return result, err
	}
	value, err := jsonPatchValue(question)
	if err != nil {
		return result, err
	}
	result.Mutation = QuestionMutation{Ledger: prospective, Patches: []document.PatchOperation{{Kind: document.PatchAdd, Pointer: "/questions/-", Value: value}}}
	result.KeyFile = sealed.KeyFile
	return result, nil
}

// PlanQuestionAddSealed performs every local check without consuming entropy.
func PlanQuestionAddSealed(model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp) (QuestionMutation, error) {
	question, private, recordedAt, err := prepareQuestionAddSealed(model, input, observedAt)
	if err != nil {
		return QuestionMutation{}, err
	}
	placeholder := ledger.SealedCommitment{
		Scheme: forecastcrypto.SealScheme, CommitmentHash: ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(strings.Repeat("0", 64))},
		Encryption: ledger.Encryption{Algorithm: forecastcrypto.EncryptionProfile, Nonce: "AAAAAAAAAAAAAAAA", Ciphertext: "AAAAAAAAAAAAAAAAAAAAAAAA"},
		KeyHint:    "forecast-key:" + string(private.ID),
	}
	question.Forecasts = []ledger.Forecast{sealedForecastRecord(private, recordedAt, placeholder)}
	prospective, err := appendProspectiveQuestion(model, question)
	if err != nil {
		return QuestionMutation{}, err
	}
	value, err := jsonPatchValue(question)
	if err != nil {
		return QuestionMutation{}, err
	}
	return QuestionMutation{Ledger: prospective, Patches: []document.PatchOperation{{Kind: document.PatchAdd, Pointer: "/questions/-", Value: value}}}, nil
}

func prepareQuestionAddSealed(model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp) (ledger.Question, InitialForecastInput, ledger.Timestamp, error) {
	question, index, err := buildQuestionShell(model, input, observedAt)
	if err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	private := input.Input.InitialForecast
	if private.Visibility != ledger.VisibilitySealed {
		return ledger.Question{}, InitialForecastInput{}, "", invalidField("initial_forecast.visibility", "this builder requires a sealed initial forecast")
	}
	if err := ValidateSlug(private.ID, "initial_forecast.id"); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	if _, exists := index.Forecast(private.ID); exists {
		return ledger.Question{}, InitialForecastInput{}, "", app.WithDetails(app.NewError(app.CodeConflict, "forecast ID already exists", nil), map[string]any{"forecast_id": private.ID})
	}
	if private.SupersedesForecastID != nil {
		return ledger.Question{}, InitialForecastInput{}, "", invalidField("initial_forecast.supersedes_forecast_id", "a question's first forecast cannot supersede another forecast")
	}
	if private.Rationale == nil || private.KeyFactors == nil || private.Comment == nil {
		return ledger.Question{}, InitialForecastInput{}, "", invalidField("initial_forecast", "a sealed forecast requires rationale, key_factors, and comment")
	}
	if err := validateOptionalKeyFactors(private.KeyFactors); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	if err := ValidateForecastValue(input.Type, input.Input.Options, &private.Value); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	recordedAt := observedAt
	if private.RecordedAt != nil {
		recordedAt = *private.RecordedAt
	}
	if err := validateForecastChronology(question.ForecastWindow, private.ForecastedAt, recordedAt); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	return question, private, recordedAt, nil
}

func prepareInitialSealedLedger(root *ledger.Ledger, input InitialQuestionInput, observedAt ledger.Timestamp) (ledger.Question, InitialForecastInput, ledger.Timestamp, error) {
	if root == nil {
		return ledger.Question{}, InitialForecastInput{}, "", app.NewError(app.CodeInternal, "ledger root is nil", nil)
	}
	normalized := NormalizeInitialQuestion(input)
	question, index, err := buildQuestionShell(root, normalized, observedAt)
	if err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	private := input.InitialForecast
	if private.Visibility != ledger.VisibilitySealed {
		return ledger.Question{}, InitialForecastInput{}, "", invalidField("initial_forecast.visibility", "this builder requires a sealed initial forecast")
	}
	if err := ValidateSlug(private.ID, "initial_forecast.id"); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	if _, exists := index.Forecast(private.ID); exists {
		return ledger.Question{}, InitialForecastInput{}, "", app.WithDetails(app.NewError(app.CodeConflict, "forecast ID already exists", nil), map[string]any{"forecast_id": private.ID})
	}
	if private.SupersedesForecastID != nil {
		return ledger.Question{}, InitialForecastInput{}, "", invalidField("initial_forecast.supersedes_forecast_id", "a question's first forecast cannot supersede another forecast")
	}
	if private.Rationale == nil || private.KeyFactors == nil || private.Comment == nil {
		return ledger.Question{}, InitialForecastInput{}, "", invalidField("initial_forecast", "a sealed forecast requires rationale, key_factors, and comment")
	}
	for position, factor := range *private.KeyFactors {
		if strings.TrimSpace(factor) == "" {
			return ledger.Question{}, InitialForecastInput{}, "", invalidField(fmt.Sprintf("initial_forecast.key_factors.%d", position), "key factor must not be empty")
		}
	}
	if err := ValidateForecastValue(input.Type, input.Options, &private.Value); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	recordedAt := observedAt
	if private.RecordedAt != nil {
		recordedAt = *private.RecordedAt
	}
	if err := validateForecastChronology(question.ForecastWindow, private.ForecastedAt, recordedAt); err != nil {
		return ledger.Question{}, InitialForecastInput{}, "", err
	}
	return question, private, recordedAt, nil
}

func sealedForecastRecord(private InitialForecastInput, recordedAt ledger.Timestamp, commitment ledger.SealedCommitment) ledger.Forecast {
	return ledger.Forecast{
		ID: private.ID, ForecastedAt: private.ForecastedAt, RecordedAt: recordedAt,
		Visibility: ledger.VisibilitySealed, PublicNote: cloneString(private.PublicNote),
		Commitment: &ledger.Commitment{Sealed: &commitment},
		Integrity:  ledger.Integrity{Unanchored: &ledger.UnanchoredIntegrity{Status: ledger.IntegrityUnanchored}},
	}
}

func validateForecastChronology(window ledger.ForecastWindow, forecastedAt, recordedAt ledger.Timestamp) error {
	if window.OpensAt == nil {
		return invalidField("forecast_window.opens_at", "forecast window opening is required after normalization")
	}
	if err := ValidateChronology(forecastedAt, "forecasted_at", recordedAt, "recorded_at", true); err != nil {
		return err
	}
	if err := ValidateChronology(*window.OpensAt, "forecast_window.opens_at", forecastedAt, "forecasted_at", true); err != nil {
		return err
	}
	return ValidateChronology(forecastedAt, "forecasted_at", window.ClosesAt, "forecast_window.closes_at", true)
}
