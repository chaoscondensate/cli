package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
)

var exactDecimalPattern = regexp.MustCompile(`^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?$`)

func BuildInitialPublicLedger(root *ledger.Ledger, input InitialQuestionInput) (*ledger.Ledger, error) {
	if root == nil {
		return nil, app.NewError(app.CodeInternal, "ledger root is nil", nil)
	}
	return BuildQuestionWithInitialPublicForecast(root, NormalizeInitialQuestion(input), root.CreatedAt)
}

func BuildInitialPublicLedgerAt(root *ledger.Ledger, input InitialQuestionInput, observedAt ledger.Timestamp) (*ledger.Ledger, error) {
	if root == nil {
		return nil, app.NewError(app.CodeInternal, "ledger root is nil", nil)
	}
	return BuildQuestionWithInitialPublicForecast(root, NormalizeInitialQuestion(input), observedAt)
}

// BuildInitialQuestionLedgerAt appends a validated question shell without a
// forecast. The stored forecasts member remains an explicit empty array.
func BuildInitialQuestionLedgerAt(root *ledger.Ledger, input InitialQuestionInput, observedAt ledger.Timestamp) (*ledger.Ledger, error) {
	if root == nil {
		return nil, app.NewError(app.CodeInternal, "ledger root is nil", nil)
	}
	question, _, err := buildQuestionShell(root, NormalizeInitialQuestion(input), observedAt)
	if err != nil {
		return nil, err
	}
	return appendProspectiveQuestion(root, question)
}

// BuildQuestionWithoutForecast appends a validated question shell and emits a
// source-preserving add patch for the complete question object.
func BuildQuestionWithoutForecast(model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp) (QuestionMutation, error) {
	question, _, err := buildQuestionShell(model, input, observedAt)
	if err != nil {
		return QuestionMutation{}, err
	}
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

func BuildQuestionWithInitialPublicForecast(model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp) (*ledger.Ledger, error) {
	question, index, err := buildQuestionShell(model, input, observedAt)
	if err != nil {
		return nil, err
	}
	if input.Input.InitialForecast == nil {
		return nil, invalidField("initial_forecast", "this builder requires an initial forecast")
	}
	forecast, err := buildInitialPublicForecast(*input.Input.InitialForecast, input.Type, input.Input.Options, question.ForecastWindow, observedAt)
	if err != nil {
		return nil, err
	}
	if input.Input.InitialForecast.SupersedesForecastID != nil {
		return nil, invalidField("initial_forecast.supersedes_forecast_id", "a question's first forecast cannot supersede another forecast")
	}
	if _, exists := index.Forecast(forecast.ID); exists {
		return nil, app.WithDetails(app.NewError(app.CodeConflict, "forecast ID already exists", nil), map[string]any{"forecast_id": forecast.ID})
	}
	question.Forecasts = []ledger.Forecast{forecast}
	return appendProspectiveQuestion(model, question)
}

func buildQuestionShell(model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp) (ledger.Question, *ledger.Index, error) {
	if model == nil {
		return ledger.Question{}, nil, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	if err := ValidateSlug(input.ID, "question"); err != nil {
		return ledger.Question{}, nil, err
	}
	if input.Type != ledger.QuestionBinary && input.Type != ledger.QuestionMultipleChoice && input.Type != ledger.QuestionNumeric && input.Type != ledger.QuestionDate {
		return ledger.Question{}, nil, invalidField("type", "question type is not supported")
	}
	if strings.TrimSpace(input.Input.Title) == "" || strings.TrimSpace(input.Input.ResolutionCriteria) == "" {
		return ledger.Question{}, nil, invalidField("question", "question title and resolution criteria must not be empty")
	}
	if _, err := ParseTimestamp(observedAt, "observed_at"); err != nil {
		return ledger.Question{}, nil, err
	}
	createdAt := observedAt
	if input.Input.CreatedAt != nil {
		createdAt = *input.Input.CreatedAt
		if _, err := ParseTimestamp(createdAt, "question.created_at"); err != nil {
			return ledger.Question{}, nil, err
		}
	}
	window := input.Input.ForecastWindow
	var storedWindow *ledger.ForecastWindow
	if window.OpensAt != "" {
		if _, err := ParseTimestamp(window.OpensAt, "forecast_window.opens_at"); err != nil {
			return ledger.Question{}, nil, err
		}
		storedWindow = &window
	}
	if _, err := ParseTimestamp(input.Input.ExpectedResolutionAt, "expected_resolution_at"); err != nil {
		return ledger.Question{}, nil, err
	}
	if err := validateQuestionShape(model, input); err != nil {
		return ledger.Question{}, nil, err
	}
	index, err := ledger.BuildIndex(model)
	if err != nil {
		return ledger.Question{}, nil, app.NewError(app.CodeInvalidData, "existing ledger indexes are invalid", err)
	}
	if _, exists := index.Question(input.ID); exists {
		return ledger.Question{}, nil, app.WithDetails(app.NewError(app.CodeConflict, "question ID already exists", nil), map[string]any{"question_id": input.ID})
	}
	question := ledger.Question{
		ID: input.ID, Title: input.Input.Title, Type: input.Type, Status: ledger.QuestionOpen,
		ResolutionCriteria: input.Input.ResolutionCriteria, CreatedAt: createdAt, ForecastWindow: storedWindow,
		ExpectedResolutionAt: input.Input.ExpectedResolutionAt,
		Options:              cloneOptions(input.Input.Options), Unit: cloneUnit(input.Input.Unit),
		PlatformRefs: clonePlatformRefs(input.Input.PlatformRefs), Tags: cloneSlugs(input.Input.Tags),
		Notes: cloneString(input.Input.Notes), Forecasts: []ledger.Forecast{},
	}
	return question, index, nil
}

func appendProspectiveQuestion(model *ledger.Ledger, question ledger.Question) (*ledger.Ledger, error) {
	prospective, err := cloneLedger(model)
	if err != nil {
		return nil, err
	}
	prospective.Questions = append(prospective.Questions, question)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return nil, err
	}
	return prospective, nil
}

func buildInitialPublicForecast(input InitialForecastInput, questionType ledger.QuestionType, options *[]ledger.Option, window *ledger.ForecastWindow, observedAt ledger.Timestamp) (ledger.Forecast, error) {
	if input.Visibility != ledger.VisibilityPublic {
		return ledger.Forecast{}, invalidField("initial_forecast.visibility", "this builder requires a public initial forecast")
	}
	if err := ValidateSlug(input.ID, "initial_forecast.id"); err != nil {
		return ledger.Forecast{}, err
	}
	if err := ValidateForecastValue(questionType, options, &input.Value); err != nil {
		return ledger.Forecast{}, err
	}
	forecastedAt, recordedAt := DefaultForecastTimes(input.ForecastedAt, input.RecordedAt, observedAt)
	if err := validateForecastChronology(window, forecastedAt, recordedAt); err != nil {
		return ledger.Forecast{}, err
	}
	value := input.Value
	return ledger.Forecast{
		ID: input.ID, ForecastedAt: forecastedAt, RecordedAt: recordedAt,
		Visibility: ledger.VisibilityPublic, Value: &value,
		Rationale: cloneString(input.Rationale), KeyFactors: cloneStrings(input.KeyFactors),
		Comment: cloneString(input.Comment), PublicNote: cloneString(input.PublicNote),
		Integrity: ledger.Integrity{Unanchored: &ledger.UnanchoredIntegrity{Status: ledger.IntegrityUnanchored}},
	}, nil
}

func ValidateForecastValue(questionType ledger.QuestionType, options *[]ledger.Option, value *ledger.ForecastValue) error {
	if value == nil {
		return invalidField("value", "forecast value is required")
	}
	set := 0
	if value.Binary != nil {
		set++
		if questionType != ledger.QuestionBinary || value.Binary.Kind != ledger.ValueBinary || value.Binary.ProbabilityBP < 0 || value.Binary.ProbabilityBP > 10000 {
			return invalidField("value", "binary forecast value is not valid for this question")
		}
	}
	if value.MultipleChoice != nil {
		set++
		if questionType != ledger.QuestionMultipleChoice || value.MultipleChoice.Kind != ledger.ValueMultipleChoice {
			return invalidField("value", "multiple-choice forecast value is not valid for this question")
		}
		optionSet := make(map[ledger.Slug]struct{})
		if options != nil {
			for _, option := range *options {
				optionSet[option.ID] = struct{}{}
			}
		}
		seen := make(map[ledger.Slug]struct{}, len(value.MultipleChoice.Probabilities))
		total := int64(0)
		for _, probability := range value.MultipleChoice.Probabilities {
			if probability.ProbabilityBP < 0 || probability.ProbabilityBP > 10000 {
				return invalidField("value.probabilities", "probability must be from 0 through 10000 basis points")
			}
			if _, duplicate := seen[probability.OptionID]; duplicate {
				return invalidField("value.probabilities", "option probability is duplicated")
			}
			seen[probability.OptionID] = struct{}{}
			total += int64(probability.ProbabilityBP)
		}
		if total != 10000 || !sameSlugKeys(seen, optionSet) {
			return invalidField("value.probabilities", "probabilities must cover every option exactly once and total 10000 basis points")
		}
	}
	if value.Numeric != nil {
		set++
		if questionType != ledger.QuestionNumeric || value.Numeric.Kind != ledger.ValueNumeric {
			return invalidField("value", "numeric forecast value is not valid for this question")
		}
		if err := validateNumericForecastValue(value.Numeric); err != nil {
			return err
		}
	}
	if value.Date != nil {
		set++
		if questionType != ledger.QuestionDate || value.Date.Kind != ledger.ValueDate {
			return invalidField("value", "date forecast value is not valid for this question")
		}
		if err := validateDateForecastValue(value.Date); err != nil {
			return err
		}
	}
	if set != 1 {
		return invalidField("value", "forecast value must contain exactly one supported type")
	}
	return nil
}

func validateNumericForecastValue(value *ledger.NumericValue) error {
	if value == nil || value.Point == nil && value.Interval == nil && value.Quantiles == nil {
		return invalidField("value", "numeric value requires point, interval, or quantiles")
	}
	if value.Point != nil && !validExactDecimal(*value.Point) {
		return invalidField("value.point", "numeric point must be an exact decimal string")
	}
	if value.Interval != nil {
		interval := value.Interval
		if !validExactDecimal(interval.Lower) || !validExactDecimal(interval.Upper) || interval.CredibilityBP < 1 || interval.CredibilityBP > 10000 {
			return invalidField("value.interval", "numeric interval bounds or credibility are invalid")
		}
		if compareExactDecimals(interval.Lower, interval.Upper) > 0 {
			return invalidField("value.interval", "numeric interval lower bound must not exceed upper bound")
		}
	}
	if value.Quantiles != nil {
		if len(*value.Quantiles) == 0 {
			return invalidField("value.quantiles", "numeric quantiles must not be empty")
		}
		var previousProbability ledger.BasisPoints
		var previousValue ledger.Decimal
		for index, quantile := range *value.Quantiles {
			if quantile.ProbabilityBP < 1 || quantile.ProbabilityBP > 9999 || !validExactDecimal(quantile.Value) {
				return invalidField("value.quantiles", "numeric quantile probability or value is invalid")
			}
			if index > 0 && quantile.ProbabilityBP <= previousProbability {
				return invalidField("value.quantiles", "numeric quantile probabilities must be unique and ordered")
			}
			if index > 0 && compareExactDecimals(quantile.Value, previousValue) < 0 {
				return invalidField("value.quantiles", "numeric quantile values must be non-decreasing")
			}
			previousProbability, previousValue = quantile.ProbabilityBP, quantile.Value
		}
	}
	return nil
}

func validateDateForecastValue(value *ledger.DateValue) error {
	if value == nil || value.Point == nil && value.Interval == nil && value.Quantiles == nil {
		return invalidField("value", "date value requires point, interval, or quantiles")
	}
	if value.Point != nil && !validFullDate(*value.Point) {
		return invalidField("value.point", "date point must be a valid full date")
	}
	if value.Interval != nil {
		interval := value.Interval
		if !validFullDate(interval.Lower) || !validFullDate(interval.Upper) || interval.CredibilityBP < 1 || interval.CredibilityBP > 10000 {
			return invalidField("value.interval", "date interval bounds or credibility are invalid")
		}
		if interval.Lower > interval.Upper {
			return invalidField("value.interval", "date interval lower bound must not exceed upper bound")
		}
	}
	if value.Quantiles != nil {
		if len(*value.Quantiles) == 0 {
			return invalidField("value.quantiles", "date quantiles must not be empty")
		}
		var previousProbability ledger.BasisPoints
		var previousValue ledger.Date
		for index, quantile := range *value.Quantiles {
			if quantile.ProbabilityBP < 1 || quantile.ProbabilityBP > 9999 || !validFullDate(quantile.Value) {
				return invalidField("value.quantiles", "date quantile probability or value is invalid")
			}
			if index > 0 && quantile.ProbabilityBP <= previousProbability {
				return invalidField("value.quantiles", "date quantile probabilities must be unique and ordered")
			}
			if index > 0 && quantile.Value < previousValue {
				return invalidField("value.quantiles", "date quantile values must be non-decreasing")
			}
			previousProbability, previousValue = quantile.ProbabilityBP, quantile.Value
		}
	}
	return nil
}

func validExactDecimal(value ledger.Decimal) bool {
	if !exactDecimalPattern.MatchString(string(value)) {
		return false
	}
	_, _, err := big.ParseFloat(string(value), 10, uint(len(value)*4+64), big.ToNearestEven)
	return err == nil
}

func compareExactDecimals(left, right ledger.Decimal) int {
	precision := uint((len(left)+len(right))*4 + 128)
	leftValue, _, _ := big.ParseFloat(string(left), 10, precision, big.ToNearestEven)
	rightValue, _, _ := big.ParseFloat(string(right), 10, precision, big.ToNearestEven)
	return leftValue.Cmp(rightValue)
}

func validFullDate(value ledger.Date) bool {
	parsed, err := time.Parse("2006-01-02", string(value))
	return err == nil && parsed.Format("2006-01-02") == string(value)
}

func validateQuestionShape(model *ledger.Ledger, input NormalizedQuestionCreate) error {
	switch input.Type {
	case ledger.QuestionBinary, ledger.QuestionDate:
		if input.Input.Options != nil || input.Input.Unit != nil {
			return invalidField("question", "binary and date questions do not accept options or unit")
		}
	case ledger.QuestionMultipleChoice:
		if input.Input.Options == nil || len(*input.Input.Options) < 2 || input.Input.Unit != nil {
			return invalidField("options", "multiple-choice questions require at least two options and no unit")
		}
		seen := make(map[ledger.Slug]struct{}, len(*input.Input.Options))
		for index, option := range *input.Input.Options {
			if err := ValidateSlug(option.ID, fmt.Sprintf("options.%d.id", index)); err != nil {
				return err
			}
			if strings.TrimSpace(option.Label) == "" {
				return invalidField(fmt.Sprintf("options.%d.label", index), "option label must not be empty")
			}
			if _, duplicate := seen[option.ID]; duplicate {
				return invalidField("options", "option IDs must be unique")
			}
			seen[option.ID] = struct{}{}
		}
	case ledger.QuestionNumeric:
		if input.Input.Unit == nil || strings.TrimSpace(input.Input.Unit.Name) == "" || input.Input.Options != nil {
			return invalidField("unit", "numeric questions require a non-empty unit and no options")
		}
	}
	if input.Input.Tags != nil {
		seen := make(map[ledger.Slug]struct{}, len(*input.Input.Tags))
		for _, tag := range *input.Input.Tags {
			if err := ValidateSlug(tag, "tags"); err != nil {
				return err
			}
			if _, duplicate := seen[tag]; duplicate {
				return invalidField("tags", "tags must be unique")
			}
			seen[tag] = struct{}{}
		}
	}
	if input.Input.PlatformRefs != nil {
		for _, reference := range *input.Input.PlatformRefs {
			if _, exists := model.Platforms[reference.Platform]; !exists {
				return invalidField("platform_refs", "platform reference does not exist")
			}
		}
	}
	return nil
}

func ValidateProspectiveLedgerModel(model *ledger.Ledger) error {
	encoded, err := json.Marshal(model)
	if err != nil {
		return app.NewError(app.CodeInternal, "prospective ledger cannot be encoded", err)
	}
	parsed, err := document.ParseJSON(bytes.NewReader(encoded), document.DefaultLimits)
	if err != nil {
		return app.NewError(app.CodeInvalidData, "prospective ledger cannot be parsed", err)
	}
	if err := ValidateLedgerDocument(parsed, nil); err != nil {
		return preserveValidationDetails(app.CodeInvalidData, "prospective ledger is not valid", err)
	}
	return nil
}

func cloneLedger(model *ledger.Ledger) (*ledger.Ledger, error) {
	encoded, err := json.Marshal(model)
	if err != nil {
		return nil, app.NewError(app.CodeInternal, "ledger cannot be copied", err)
	}
	var result ledger.Ledger
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, app.NewError(app.CodeInternal, "ledger copy cannot be decoded", err)
	}
	return &result, nil
}

func sameSlugKeys(left, right map[ledger.Slug]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if _, exists := right[key]; !exists {
			return false
		}
	}
	return true
}

func cloneOptions(value *[]ledger.Option) *[]ledger.Option {
	if value == nil {
		return nil
	}
	copy := append([]ledger.Option{}, (*value)...)
	return &copy
}

func cloneUnit(value *ledger.Unit) *ledger.Unit {
	if value == nil {
		return nil
	}
	return &ledger.Unit{Name: value.Name, Symbol: cloneString(value.Symbol), UCUMCode: cloneString(value.UCUMCode)}
}

func clonePlatformRefs(value *[]ledger.PlatformRef) *[]ledger.PlatformRef {
	if value == nil {
		return nil
	}
	copy := make([]ledger.PlatformRef, len(*value))
	for index, reference := range *value {
		copy[index] = reference
		copy[index].QuestionID = cloneString(reference.QuestionID)
		copy[index].URL = cloneString(reference.URL)
	}
	return &copy
}

func cloneSlugs(value *[]ledger.Slug) *[]ledger.Slug {
	if value == nil {
		return nil
	}
	copy := append([]ledger.Slug{}, (*value)...)
	return &copy
}

func cloneStrings(value *[]string) *[]string {
	if value == nil {
		return nil
	}
	copy := append([]string{}, (*value)...)
	return &copy
}
