package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
)

type ForecastMutation struct {
	Ledger  *ledger.Ledger
	Patches []document.PatchOperation
}

type ForecastSummary struct {
	ID                   ledger.Slug               `json:"id"`
	ForecastedAt         ledger.Timestamp          `json:"forecasted_at"`
	RecordedAt           ledger.Timestamp          `json:"recorded_at"`
	Visibility           ledger.ForecastVisibility `json:"visibility"`
	SupersedesForecastID *ledger.Slug              `json:"supersedes_forecast_id,omitempty"`
	IntegrityStatus      ledger.IntegrityStatus    `json:"integrity_status"`
	ValueSummary         string                    `json:"value_summary,omitempty"`
}

type CommitmentView struct {
	Scheme              string            `json:"scheme"`
	CommitmentHash      ledger.Digest     `json:"commitment_hash"`
	Encryption          ledger.Encryption `json:"encryption"`
	KeyHint             string            `json:"key_hint"`
	RevealedAt          *ledger.Timestamp `json:"revealed_at,omitempty"`
	RevealedKeyRedacted bool              `json:"revealed_key_redacted,omitempty"`
}

type ForecastView struct {
	Summary    ForecastSummary       `json:"summary"`
	Value      *ledger.ForecastValue `json:"value,omitempty"`
	Rationale  *string               `json:"rationale,omitempty"`
	KeyFactors *[]string             `json:"key_factors,omitempty"`
	Comment    *string               `json:"comment,omitempty"`
	PublicNote *string               `json:"public_note,omitempty"`
	Commitment *CommitmentView       `json:"commitment,omitempty"`
	Integrity  ForecastIntegrityView `json:"integrity"`
}

type ForecastIntegrityView struct {
	Status        ledger.IntegrityStatus    `json:"status"`
	Target        *ledger.ForecastTarget    `json:"target,omitempty"`
	Timestamps    []ledger.RFC3161Timestamp `json:"timestamps,omitempty"`
	VerifiedAt    *ledger.Timestamp         `json:"verified_at,omitempty"`
	FailureReason string                    `json:"failure_reason,omitempty"`
	StoredOnly    bool                      `json:"stored_only,omitempty"`
}

func BuildPublicForecastAppend(model *ledger.Ledger, questionID, forecastID ledger.Slug, input ForecastCreateInput, observedAt ledger.Timestamp) (ForecastMutation, error) {
	var result ForecastMutation
	questionPosition, question, recordedAt, err := prepareForecastAppend(model, questionID, forecastID, input.ForecastedAt, input.RecordedAt, input.SupersedesForecastID, observedAt)
	if err != nil {
		return result, err
	}
	if err := ValidateForecastValue(question.Type, question.Options, &input.Value); err != nil {
		return result, err
	}
	if err := validateOptionalKeyFactors(input.KeyFactors); err != nil {
		return result, err
	}
	value := input.Value
	forecast := ledger.Forecast{
		ID: forecastID, ForecastedAt: input.ForecastedAt, RecordedAt: recordedAt,
		Visibility: ledger.VisibilityPublic, Value: &value,
		Rationale: cloneString(input.Rationale), KeyFactors: cloneStrings(input.KeyFactors), Comment: cloneString(input.Comment), PublicNote: cloneString(input.PublicNote),
		SupersedesForecastID: cloneSlug(input.SupersedesForecastID),
		Integrity:            ledger.Integrity{Unanchored: &ledger.UnanchoredIntegrity{Status: ledger.IntegrityUnanchored}},
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return result, err
	}
	prospective.Questions[questionPosition].Forecasts = append(prospective.Questions[questionPosition].Forecasts, forecast)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return result, err
	}
	valuePatch, err := jsonPatchValue(forecast)
	if err != nil {
		return result, err
	}
	result.Ledger = prospective
	result.Patches = []document.PatchOperation{{Kind: document.PatchAdd, Pointer: questionForecastAppendPointer(questionPosition), Value: valuePatch}}
	return result, nil
}

func prepareForecastAppend(model *ledger.Ledger, questionID, forecastID ledger.Slug, forecastedAt ledger.Timestamp, explicitRecordedAt *ledger.Timestamp, supersedes *ledger.Slug, observedAt ledger.Timestamp) (int, ledger.Question, ledger.Timestamp, error) {
	if model == nil {
		return 0, ledger.Question{}, "", app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	if err := ValidateSlug(questionID, "question"); err != nil {
		return 0, ledger.Question{}, "", err
	}
	if err := ValidateSlug(forecastID, "forecast"); err != nil {
		return 0, ledger.Question{}, "", err
	}
	index, err := ledger.BuildIndex(model)
	if err != nil {
		return 0, ledger.Question{}, "", app.NewError(app.CodeInvalidData, "ledger indexes are invalid", err)
	}
	questionPosition, exists := index.Question(questionID)
	if !exists {
		return 0, ledger.Question{}, "", app.WithDetails(app.NewError(app.CodeNotFound, "question was not found", nil), map[string]any{"question_id": questionID})
	}
	question := model.Questions[questionPosition]
	if question.Status != ledger.QuestionOpen {
		return 0, ledger.Question{}, "", app.WithDetails(app.NewError(app.CodeConflict, "forecasts can be added only to an open question", nil), map[string]any{"question_id": questionID, "status": question.Status})
	}
	if _, exists := index.Forecast(forecastID); exists {
		return 0, ledger.Question{}, "", app.WithDetails(app.NewError(app.CodeConflict, "forecast ID already exists", nil), map[string]any{"forecast_id": forecastID})
	}
	recordedAt := observedAt
	if explicitRecordedAt != nil {
		recordedAt = *explicitRecordedAt
	}
	window := question.ForecastWindow
	if window.OpensAt == nil {
		opening := question.CreatedAt
		window.OpensAt = &opening
	}
	if err := validateForecastChronology(window, forecastedAt, recordedAt); err != nil {
		return 0, ledger.Question{}, "", err
	}
	if len(question.Forecasts) > 0 {
		last := question.Forecasts[len(question.Forecasts)-1]
		if err := ValidateChronology(last.RecordedAt, "previous.recorded_at", recordedAt, "recorded_at", true); err != nil {
			return 0, ledger.Question{}, "", invalidField("recorded_at", "forecast records must remain ordered by recorded time")
		}
	}
	if supersedes != nil {
		location, exists := index.Forecast(*supersedes)
		if !exists || location.QuestionID != questionID {
			return 0, ledger.Question{}, "", invalidField("supersedes_forecast_id", "supersedes must identify an earlier forecast in the same question")
		}
	}
	return questionPosition, question, recordedAt, nil
}

func ListForecasts(model *ledger.Ledger, questionID ledger.Slug) ([]ForecastSummary, error) {
	_, question, err := selectQuestion(model, questionID)
	if err != nil {
		return nil, err
	}
	result := make([]ForecastSummary, len(question.Forecasts))
	for index, forecast := range question.Forecasts {
		result[index] = summarizeForecast(forecast)
	}
	return result, nil
}

func ShowForecast(model *ledger.Ledger, questionID, forecastID ledger.Slug) (ForecastView, error) {
	_, question, err := selectQuestion(model, questionID)
	if err != nil {
		return ForecastView{}, err
	}
	var selected *ledger.Forecast
	for index := range question.Forecasts {
		if question.Forecasts[index].ID == forecastID {
			selected = &question.Forecasts[index]
			break
		}
	}
	if selected == nil {
		return ForecastView{}, app.WithDetails(app.NewError(app.CodeNotFound, "forecast was not found in the selected question", nil), map[string]any{"question_id": questionID, "forecast_id": forecastID})
	}
	view := ForecastView{Summary: summarizeForecast(*selected), PublicNote: cloneString(selected.PublicNote), Integrity: forecastIntegrityView(selected.Integrity)}
	if selected.Visibility != ledger.VisibilitySealed {
		view.Value = cloneForecastValue(selected.Value)
		view.Rationale = cloneString(selected.Rationale)
		view.KeyFactors = cloneStrings(selected.KeyFactors)
		view.Comment = cloneString(selected.Comment)
	}
	if selected.Commitment != nil {
		view.Commitment = commitmentView(selected.Commitment)
		if selected.Visibility == ledger.VisibilitySealed && view.Commitment != nil {
			view.Commitment.Encryption.Ciphertext = ""
			view.Commitment.Encryption.Nonce = ""
		}
	}
	return view, nil
}

func forecastIntegrityView(value ledger.Integrity) ForecastIntegrityView {
	view := ForecastIntegrityView{Status: integrityStatus(value)}
	switch {
	case value.Pending != nil:
		target := value.Pending.Target
		view.Target = &target
		view.Timestamps = append([]ledger.RFC3161Timestamp(nil), value.Pending.Timestamps...)
	case value.Verified != nil:
		target, verifiedAt := value.Verified.Target, value.Verified.VerifiedAt
		view.Target = &target
		view.Timestamps = append([]ledger.RFC3161Timestamp(nil), value.Verified.Timestamps...)
		view.VerifiedAt = &verifiedAt
		view.StoredOnly = true
	case value.Failed != nil:
		view.FailureReason = value.Failed.FailureReason
		if value.Failed.Target != nil {
			target := *value.Failed.Target
			view.Target = &target
		}
		if value.Failed.Timestamps != nil {
			view.Timestamps = append([]ledger.RFC3161Timestamp(nil), (*value.Failed.Timestamps)...)
		}
	}
	return view
}

func selectQuestion(model *ledger.Ledger, id ledger.Slug) (int, ledger.Question, error) {
	if model == nil {
		return 0, ledger.Question{}, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	index, err := ledger.BuildIndex(model)
	if err != nil {
		return 0, ledger.Question{}, app.NewError(app.CodeInvalidData, "ledger indexes are invalid", err)
	}
	position, exists := index.Question(id)
	if !exists {
		return 0, ledger.Question{}, app.WithDetails(app.NewError(app.CodeNotFound, "question was not found", nil), map[string]any{"question_id": id})
	}
	return position, model.Questions[position], nil
}

func summarizeForecast(forecast ledger.Forecast) ForecastSummary {
	return ForecastSummary{
		ID: forecast.ID, ForecastedAt: forecast.ForecastedAt, RecordedAt: forecast.RecordedAt,
		Visibility: forecast.Visibility, SupersedesForecastID: cloneSlug(forecast.SupersedesForecastID), IntegrityStatus: integrityStatus(forecast.Integrity),
		ValueSummary: forecastValueSummary(forecast.Value),
	}
}

func forecastValueSummary(value *ledger.ForecastValue) string {
	if value == nil {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func integrityStatus(value ledger.Integrity) ledger.IntegrityStatus {
	switch {
	case value.Unanchored != nil:
		return value.Unanchored.Status
	case value.Pending != nil:
		return value.Pending.Status
	case value.Verified != nil:
		return value.Verified.Status
	case value.Failed != nil:
		return value.Failed.Status
	default:
		return ""
	}
}

func commitmentView(value *ledger.Commitment) *CommitmentView {
	if value == nil {
		return nil
	}
	if value.Sealed != nil {
		sealed := value.Sealed
		return &CommitmentView{Scheme: sealed.Scheme, CommitmentHash: sealed.CommitmentHash, Encryption: sealed.Encryption, KeyHint: sealed.KeyHint}
	}
	if value.Revealed != nil {
		revealed := value.Revealed
		at := revealed.RevealedAt
		return &CommitmentView{Scheme: revealed.Scheme, CommitmentHash: revealed.CommitmentHash, Encryption: revealed.Encryption, KeyHint: revealed.KeyHint, RevealedAt: &at, RevealedKeyRedacted: true}
	}
	return nil
}

func cloneForecastValue(value *ledger.ForecastValue) *ledger.ForecastValue {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneSlug(value *ledger.Slug) *ledger.Slug {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func validateOptionalKeyFactors(value *[]string) error {
	if value == nil {
		return nil
	}
	for _, factor := range *value {
		if strings.TrimSpace(factor) == "" {
			return invalidField("key_factors", "key factors must not contain an empty item")
		}
	}
	return nil
}

func questionForecastAppendPointer(questionPosition int) string {
	return "/questions/" + strconv.Itoa(questionPosition) + "/forecasts/-"
}
