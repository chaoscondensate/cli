package service

import (
	"bytes"
	"context"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/forecastcrypto"
	"github.com/chaoscondensate/cli/internal/ledger"
)

var keyHintPattern = regexp.MustCompile(`^([a-z][a-z0-9+.-]*):([A-Za-z0-9._~+-]+)$`)

func ValidateKeyHint(forecastID ledger.Slug, keyHint string) error {
	match := keyHintPattern.FindStringSubmatch(keyHint)
	if len(match) != 3 || match[1] == "file" || match[1] == "forecast-key" && match[2] != string(forecastID) {
		return invalidField("key_hint", "key hint must be a non-location scheme:opaque value; forecast-key must name the selected forecast")
	}
	return nil
}

type SealedForecastBuild struct {
	Mutation ForecastMutation
	KeyFile  []byte
}

func BuildSealedForecastAppend(ctx context.Context, model *ledger.Ledger, questionID, forecastID ledger.Slug, input SealedForecastInput, observedAt ledger.Timestamp, effects Effects) (SealedForecastBuild, error) {
	var result SealedForecastBuild
	if err := effects.Validate(); err != nil {
		return result, app.NewError(app.CodeInternal, "sealing effects are not configured", err)
	}
	input.ForecastedAt, observedAt = DefaultForecastTimes(input.ForecastedAt, input.RecordedAt, observedAt)
	input.RecordedAt = &observedAt
	questionPosition, question, recordedAt, err := prepareSealedForecastAppend(model, questionID, forecastID, input, observedAt)
	if err != nil {
		return result, err
	}
	sealed, err := forecastcrypto.Seal(ctx, questionID, forecastID, forecastcrypto.PrivateBundle{
		ForecastedAt: input.ForecastedAt, RecordedAt: recordedAt, Value: input.Value,
		Rationale: input.Rationale, KeyFactors: append([]string{}, input.KeyFactors...), Comment: input.Comment,
	}, "forecast-key:"+string(forecastID), effects.Random)
	if err != nil {
		return result, app.NewError(app.CodeIO, "sealed forecast could not be created", err)
	}
	forecast := sealedAppendRecord(forecastID, input, recordedAt, sealed.Commitment)
	mutation, err := appendSealedForecastMutation(model, questionPosition, question, forecast)
	if err != nil {
		return result, err
	}
	result.Mutation, result.KeyFile = mutation, sealed.KeyFile
	return result, nil
}

func PlanSealedForecastAppend(model *ledger.Ledger, questionID, forecastID ledger.Slug, input SealedForecastInput, observedAt ledger.Timestamp) (ForecastMutation, error) {
	input.ForecastedAt, observedAt = DefaultForecastTimes(input.ForecastedAt, input.RecordedAt, observedAt)
	input.RecordedAt = &observedAt
	questionPosition, question, recordedAt, err := prepareSealedForecastAppend(model, questionID, forecastID, input, observedAt)
	if err != nil {
		return ForecastMutation{}, err
	}
	placeholder := ledger.SealedCommitment{
		Scheme: forecastcrypto.SealScheme, CommitmentHash: ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(strings.Repeat("0", 64))},
		Encryption: ledger.Encryption{Algorithm: forecastcrypto.EncryptionProfile, Nonce: "AAAAAAAAAAAAAAAA", Ciphertext: "AAAAAAAAAAAAAAAAAAAAAAAA"},
		KeyHint:    "forecast-key:" + string(forecastID),
	}
	return appendSealedForecastMutation(model, questionPosition, question, sealedAppendRecord(forecastID, input, recordedAt, placeholder))
}

func prepareSealedForecastAppend(model *ledger.Ledger, questionID, forecastID ledger.Slug, input SealedForecastInput, observedAt ledger.Timestamp) (int, ledger.Question, ledger.Timestamp, error) {
	forecastedAt, recordedAt := DefaultForecastTimes(input.ForecastedAt, input.RecordedAt, observedAt)
	input.ForecastedAt = forecastedAt
	questionPosition, question, err := prepareForecastAppend(model, questionID, forecastID, forecastedAt, recordedAt, input.SupersedesForecastID)
	if err != nil {
		return 0, ledger.Question{}, "", err
	}
	if err := ValidateForecastValue(question.Type, question.Options, &input.Value); err != nil {
		return 0, ledger.Question{}, "", err
	}
	factors := input.KeyFactors
	if err := validateOptionalKeyFactors(&factors); err != nil {
		return 0, ledger.Question{}, "", err
	}
	return questionPosition, question, recordedAt, nil
}

func sealedAppendRecord(id ledger.Slug, input SealedForecastInput, recordedAt ledger.Timestamp, commitment ledger.SealedCommitment) ledger.Forecast {
	return ledger.Forecast{
		ID: id, ForecastedAt: input.ForecastedAt, RecordedAt: recordedAt, Visibility: ledger.VisibilitySealed,
		PublicNote: cloneString(input.PublicNote), SupersedesForecastID: cloneSlug(input.SupersedesForecastID),
		Commitment: &ledger.Commitment{Sealed: &commitment}, Integrity: ledger.Integrity{Unanchored: &ledger.UnanchoredIntegrity{Status: ledger.IntegrityUnanchored}},
	}
}

func appendSealedForecastMutation(model *ledger.Ledger, questionPosition int, question ledger.Question, forecast ledger.Forecast) (ForecastMutation, error) {
	prospective, err := cloneLedger(model)
	if err != nil {
		return ForecastMutation{}, err
	}
	prospective.Questions[questionPosition].Forecasts = append(prospective.Questions[questionPosition].Forecasts, forecast)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return ForecastMutation{}, err
	}
	value, err := jsonPatchValue(forecast)
	if err != nil {
		return ForecastMutation{}, err
	}
	_ = question
	return ForecastMutation{Ledger: prospective, Patches: []document.PatchOperation{{Kind: document.PatchAdd, Pointer: questionForecastAppendPointer(questionPosition), Value: value}}}, nil
}

func BuildForecastReveal(model *ledger.Ledger, questionID, forecastID ledger.Slug, keyFile []byte, revealedAt ledger.Timestamp) (ForecastMutation, error) {
	if _, err := ParseTimestamp(revealedAt, "revealed_at"); err != nil {
		return ForecastMutation{}, err
	}
	questionPosition, question, forecastPosition, forecast, err := selectForecast(model, questionID, forecastID)
	if err != nil {
		return ForecastMutation{}, err
	}
	sealed, alreadyRevealed, err := originalSealedCommitment(forecast)
	if err != nil {
		return ForecastMutation{}, err
	}
	opened, err := forecastcrypto.Open(keyFile, questionID, forecastID, sealed)
	if err != nil {
		return ForecastMutation{}, app.NewError(app.CodeVerification, "sealed forecast authentication failed", err)
	}
	if err := validateRevealedBundle(question, forecast, opened.Bundle); err != nil {
		return ForecastMutation{}, app.NewError(app.CodeVerification, "authenticated bundle does not match the selected forecast", err)
	}
	if alreadyRevealed {
		if forecast.Commitment.Revealed.RevealedKey != opened.KeyHex || !revealedMirrorMatches(forecast, opened.Bundle) {
			return ForecastMutation{}, app.NewError(app.CodeVerification, "revealed forecast mirror or disclosed key does not match the authenticated seal", nil)
		}
		return ForecastMutation{Ledger: model}, nil
	}
	originalTarget, err := BuildForecastTarget(model, questionID, forecastID)
	if err != nil {
		return ForecastMutation{}, err
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return ForecastMutation{}, err
	}
	updated := &prospective.Questions[questionPosition].Forecasts[forecastPosition]
	value := opened.Bundle.Value
	rationale := opened.Bundle.Rationale
	factors := append([]string{}, opened.Bundle.KeyFactors...)
	comment := opened.Bundle.Comment
	updated.Visibility = ledger.VisibilityRevealed
	updated.Value, updated.Rationale, updated.KeyFactors, updated.Comment = &value, &rationale, &factors, &comment
	updated.Commitment = &ledger.Commitment{Revealed: &ledger.RevealedCommitment{
		Scheme: sealed.Scheme, CommitmentHash: sealed.CommitmentHash, Encryption: sealed.Encryption, KeyHint: sealed.KeyHint,
		RevealedAt: revealedAt, RevealedKey: opened.KeyHex,
	}}
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return ForecastMutation{}, err
	}
	revealedTarget, err := BuildForecastTarget(prospective, questionID, forecastID)
	if err != nil {
		return ForecastMutation{}, err
	}
	if !bytes.Equal(originalTarget.Bytes, revealedTarget.Bytes) {
		return ForecastMutation{}, app.NewError(app.CodeVerification, "reveal would change the original sealed forecast target", nil)
	}
	base := "/questions/" + strconv.Itoa(questionPosition) + "/forecasts/" + strconv.Itoa(forecastPosition)
	patchValue := func(value any) any { normalized, _ := jsonPatchValue(value); return normalized }
	return ForecastMutation{Ledger: prospective, Patches: []document.PatchOperation{
		replacePatch(base+"/visibility", ledger.VisibilityRevealed),
		{Kind: document.PatchAdd, Pointer: base + "/value", Value: patchValue(value)},
		{Kind: document.PatchAdd, Pointer: base + "/rationale", Value: rationale},
		{Kind: document.PatchAdd, Pointer: base + "/key_factors", Value: patchValue(factors)},
		{Kind: document.PatchAdd, Pointer: base + "/comment", Value: comment},
		{Kind: document.PatchAdd, Pointer: base + "/commitment/revealed_at", Value: string(revealedAt)},
		{Kind: document.PatchAdd, Pointer: base + "/commitment/revealed_key", Value: string(opened.KeyHex)},
	}}, nil
}

func BuildForecastKeyHintUpdate(model *ledger.Ledger, questionID, forecastID ledger.Slug, keyHint string) (ForecastMutation, error) {
	questionPosition, _, forecastPosition, forecast, err := selectForecast(model, questionID, forecastID)
	if err != nil {
		return ForecastMutation{}, err
	}
	if err := ValidateKeyHint(forecastID, keyHint); err != nil {
		return ForecastMutation{}, err
	}
	if forecast.Commitment == nil || forecast.Commitment.Sealed == nil && forecast.Commitment.Revealed == nil {
		return ForecastMutation{}, app.NewError(app.CodeConflict, "key hint can be changed only on a sealed or revealed forecast", nil)
	}
	current := ""
	if forecast.Commitment.Sealed != nil {
		current = forecast.Commitment.Sealed.KeyHint
	} else {
		current = forecast.Commitment.Revealed.KeyHint
	}
	if current == keyHint {
		return ForecastMutation{Ledger: model}, nil
	}
	before, err := BuildForecastTarget(model, questionID, forecastID)
	if err != nil {
		return ForecastMutation{}, err
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return ForecastMutation{}, err
	}
	updated := &prospective.Questions[questionPosition].Forecasts[forecastPosition]
	if updated.Commitment.Sealed != nil {
		updated.Commitment.Sealed.KeyHint = keyHint
	} else {
		updated.Commitment.Revealed.KeyHint = keyHint
	}
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return ForecastMutation{}, err
	}
	after, err := BuildForecastTarget(prospective, questionID, forecastID)
	if err != nil || !bytes.Equal(before.Bytes, after.Bytes) {
		return ForecastMutation{}, app.NewError(app.CodeInternal, "key hint update changed forecast target bytes", err)
	}
	base := "/questions/" + strconv.Itoa(questionPosition) + "/forecasts/" + strconv.Itoa(forecastPosition)
	return ForecastMutation{Ledger: prospective, Patches: []document.PatchOperation{replacePatch(base+"/commitment/key_hint", keyHint)}}, nil
}

func selectForecast(model *ledger.Ledger, questionID, forecastID ledger.Slug) (int, ledger.Question, int, ledger.Forecast, error) {
	questionPosition, question, err := selectQuestion(model, questionID)
	if err != nil {
		return 0, ledger.Question{}, 0, ledger.Forecast{}, err
	}
	for index, forecast := range question.Forecasts {
		if forecast.ID == forecastID {
			return questionPosition, question, index, forecast, nil
		}
	}
	return 0, ledger.Question{}, 0, ledger.Forecast{}, app.WithDetails(app.NewError(app.CodeNotFound, "forecast was not found in the selected question", nil), map[string]any{"question_id": questionID, "forecast_id": forecastID})
}

func originalSealedCommitment(forecast ledger.Forecast) (ledger.SealedCommitment, bool, error) {
	if forecast.Commitment == nil {
		return ledger.SealedCommitment{}, false, app.NewError(app.CodeConflict, "forecast has no sealed commitment", nil)
	}
	if forecast.Visibility == ledger.VisibilitySealed && forecast.Commitment.Sealed != nil {
		return *forecast.Commitment.Sealed, false, nil
	}
	if forecast.Visibility == ledger.VisibilityRevealed && forecast.Commitment.Revealed != nil {
		revealed := forecast.Commitment.Revealed
		return ledger.SealedCommitment{Scheme: revealed.Scheme, CommitmentHash: revealed.CommitmentHash, Encryption: revealed.Encryption, KeyHint: revealed.KeyHint}, true, nil
	}
	return ledger.SealedCommitment{}, false, app.NewError(app.CodeConflict, "forecast is not sealed or revealed", nil)
}

func validateRevealedBundle(question ledger.Question, forecast ledger.Forecast, bundle forecastcrypto.PrivateBundle) error {
	if bundle.ForecastedAt != forecast.ForecastedAt || bundle.RecordedAt != forecast.RecordedAt {
		return app.NewError(app.CodeVerification, "authenticated forecast times do not match the public record", nil)
	}
	if err := ValidateForecastValue(question.Type, question.Options, &bundle.Value); err != nil {
		return err
	}
	factors := bundle.KeyFactors
	if err := validateOptionalKeyFactors(&factors); err != nil {
		return err
	}
	return validateForecastChronology(question.ForecastWindow, bundle.ForecastedAt, bundle.RecordedAt)
}

func revealedMirrorMatches(forecast ledger.Forecast, bundle forecastcrypto.PrivateBundle) bool {
	if forecast.Value == nil || forecast.Rationale == nil || forecast.KeyFactors == nil || forecast.Comment == nil {
		return false
	}
	left, _ := jsonPatchValue(struct {
		Value      ledger.ForecastValue
		Rationale  string
		KeyFactors []string
		Comment    string
	}{*forecast.Value, *forecast.Rationale, *forecast.KeyFactors, *forecast.Comment})
	right, _ := jsonPatchValue(struct {
		Value      ledger.ForecastValue
		Rationale  string
		KeyFactors []string
		Comment    string
	}{bundle.Value, bundle.Rationale, bundle.KeyFactors, bundle.Comment})
	return reflect.DeepEqual(left, right)
}
