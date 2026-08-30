package validation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"time"
	_ "time/tzdata"

	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/forecastcrypto"
	"github.com/chaoscondensate/cli/internal/ledger"
)

type SemanticIssue struct {
	Layer   string `json:"layer"`
	Code    string `json:"code"`
	Pointer string `json:"pointer"`
	Message string `json:"message"`
}

func DecodeLedger(source *document.Document) (*ledger.Ledger, error) {
	if source == nil || source.Root == nil {
		return nil, errors.New("document has no root value")
	}
	encoded, err := json.Marshal(source.Root.Any())
	if err != nil {
		return nil, fmt.Errorf("encode parsed ledger: %w", err)
	}
	var result ledger.Ledger
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, fmt.Errorf("decode typed ledger: %w", err)
	}
	return &result, nil
}

// ValidateSemantics runs cross-field checks after structural validation.
// artifacts is rooted at the ledger's artifact directory. A nil filesystem
// intentionally skips artifact-byte checks for model-only prospective
// validation and stdin inspection; path-backed validation must pass the
// confined ledger-relative filesystem so missing or changed artifacts fail.
func ValidateSemantics(model *ledger.Ledger, artifacts fs.FS) ([]SemanticIssue, error) {
	if model == nil {
		return nil, errors.New("ledger is nil")
	}
	validator := semanticValidator{model: model, artifacts: artifacts}
	validator.validate()
	sort.Slice(validator.issues, func(i, j int) bool {
		left, right := validator.issues[i], validator.issues[j]
		if left.Pointer != right.Pointer {
			return left.Pointer < right.Pointer
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return left.Message < right.Message
	})
	return validator.issues, nil
}

type semanticValidator struct {
	model     *ledger.Ledger
	artifacts fs.FS
	issues    []SemanticIssue
}

func (v *semanticValidator) validate() {
	if _, err := time.LoadLocation(v.model.DefaultTimezone); err != nil {
		v.add("semantic.timezone", "/default_timezone", "timezone is not a known IANA name")
	}
	if v.model.Forecaster.Kind == ledger.ForecasterTeam && v.model.Forecaster.Members != nil {
		memberIDs := make([]ledger.Slug, len(*v.model.Forecaster.Members))
		for index, member := range *v.model.Forecaster.Members {
			memberIDs[index] = member.ID
		}
		v.uniqueSlugs(memberIDs, "/forecaster/members", "semantic.duplicate_member_id")
	}

	questionIDs := make([]ledger.Slug, len(v.model.Questions))
	platformIDs := make(map[ledger.Slug]struct{}, len(v.model.Platforms))
	for id := range v.model.Platforms {
		platformIDs[id] = struct{}{}
	}
	globalForecastIDs := make([]ledger.Slug, 0)
	for questionIndex := range v.model.Questions {
		question := &v.model.Questions[questionIndex]
		questionIDs[questionIndex] = question.ID
		globalForecastIDs = append(globalForecastIDs, v.validateQuestion(questionIndex, question, platformIDs)...)
	}
	v.uniqueSlugs(questionIDs, "/questions", "semantic.duplicate_question_id")
	v.uniqueSlugs(globalForecastIDs, "/questions/*/forecasts", "semantic.duplicate_forecast_id")
}

func (v *semanticValidator) validateQuestion(index int, question *ledger.Question, platformIDs map[ledger.Slug]struct{}) []ledger.Slug {
	pointer := "/questions/" + strconv.Itoa(index)
	if question.PlatformRefs != nil {
		for refIndex, ref := range *question.PlatformRefs {
			if _, exists := platformIDs[ref.Platform]; !exists {
				v.add("semantic.unknown_platform", fmt.Sprintf("%s/platform_refs/%d/platform", pointer, refIndex), "platform reference does not exist")
			}
		}
	}
	if question.Type == ledger.QuestionMultipleChoice && question.Options != nil {
		optionIDs := make([]ledger.Slug, len(*question.Options))
		for optionIndex, option := range *question.Options {
			optionIDs[optionIndex] = option.ID
		}
		v.uniqueSlugs(optionIDs, pointer+"/options", "semantic.duplicate_option_id")
	}
	v.validateLifecycle(question, pointer)

	localIDs := make(map[ledger.Slug]struct{})
	forecastIDs := make([]ledger.Slug, 0, len(question.Forecasts))
	var previousRecorded time.Time
	for forecastIndex := range question.Forecasts {
		forecast := &question.Forecasts[forecastIndex]
		forecastPointer := fmt.Sprintf("%s/forecasts/%d", pointer, forecastIndex)
		forecastIDs = append(forecastIDs, forecast.ID)
		forecasted := parseTimestamp(forecast.ForecastedAt)
		recorded := parseTimestamp(forecast.RecordedAt)
		if forecasted.After(recorded) {
			v.add("semantic.forecast_chronology", forecastPointer+"/recorded_at", "recorded_at must not be before forecasted_at")
		}
		if question.ForecastWindow != nil && forecasted.Before(parseTimestamp(question.ForecastWindow.OpensAt)) {
			v.add("semantic.forecast_window", forecastPointer+"/forecasted_at", "forecasted_at must not precede forecast_window.opens_at")
		}
		if !previousRecorded.IsZero() && recorded.Before(previousRecorded) {
			v.add("semantic.forecast_order", forecastPointer, "forecasts must be ordered by recorded_at")
		}
		previousRecorded = recorded
		if forecast.SupersedesForecastID != nil {
			if _, exists := localIDs[*forecast.SupersedesForecastID]; !exists {
				v.add("semantic.supersedes", forecastPointer+"/supersedes_forecast_id", "supersedes must reference an earlier forecast in the same question")
			}
		}
		localIDs[forecast.ID] = struct{}{}
		if forecast.Value != nil {
			v.validateValue(question, forecast.Value, forecastPointer+"/value")
		}
		v.validateArtifact(forecast.Integrity, forecastPointer+"/integrity")
		v.validateTimestampChronology(question, forecast, forecastPointer+"/integrity")
		v.validateReveal(question, forecast, forecastPointer)
	}
	v.validateResolution(question, pointer)
	return forecastIDs
}

func (v *semanticValidator) validateLifecycle(question *ledger.Question, pointer string) {
	resolutionStatus := ledger.ResolutionStatus("")
	if question.Resolution != nil {
		if question.Resolution.Resolved != nil {
			resolutionStatus = question.Resolution.Resolved.Status
		} else if question.Resolution.NonResolved != nil {
			resolutionStatus = question.Resolution.NonResolved.Status
		}
	}
	switch question.Status {
	case ledger.QuestionOpen, ledger.QuestionClosed, ledger.QuestionAwaitingResolution:
		if question.Resolution != nil {
			v.add("semantic.lifecycle", pointer+"/resolution", "this question status must not have a resolution")
		}
	case ledger.QuestionResolved:
		if resolutionStatus != ledger.ResolutionResolved {
			v.add("semantic.lifecycle", pointer+"/resolution", "resolved question must have a resolved resolution")
		}
	case ledger.QuestionAnnulled:
		if resolutionStatus != ledger.ResolutionAnnulled {
			v.add("semantic.lifecycle", pointer+"/resolution", "annulled question must have an annulled resolution")
		}
	case ledger.QuestionDisputed:
		if resolutionStatus != ledger.ResolutionDisputed {
			v.add("semantic.lifecycle", pointer+"/resolution", "disputed question must have a disputed resolution")
		}
	}
}

func (v *semanticValidator) validateValue(question *ledger.Question, value *ledger.ForecastValue, pointer string) {
	kind := forecastValueKind(value)
	if ledger.QuestionType(kind) != question.Type {
		v.add("semantic.value_type", pointer, "forecast value kind must match question type")
		return
	}
	switch {
	case value.MultipleChoice != nil:
		optionIDs := make(map[ledger.Slug]struct{})
		if question.Options != nil {
			for _, option := range *question.Options {
				optionIDs[option.ID] = struct{}{}
			}
		}
		seen := make(map[ledger.Slug]struct{})
		total := int64(0)
		for probabilityIndex, probability := range value.MultipleChoice.Probabilities {
			if _, exists := seen[probability.OptionID]; exists {
				v.add("semantic.duplicate_probability", fmt.Sprintf("%s/probabilities/%d", pointer, probabilityIndex), "option probability is duplicated")
			}
			seen[probability.OptionID] = struct{}{}
			total += int64(probability.ProbabilityBP)
		}
		if len(seen) != len(optionIDs) || !sameSlugSet(seen, optionIDs) {
			v.add("semantic.option_coverage", pointer+"/probabilities", "probabilities must cover every option exactly once")
		}
		if total != 10_000 {
			v.add("semantic.probability_sum", pointer+"/probabilities", "probabilities must sum to 10000 basis points")
		}
	case value.Numeric != nil:
		v.validateNumericValue(value.Numeric, pointer)
	case value.Date != nil:
		v.validateDateValue(value.Date, pointer)
	}
}

func (v *semanticValidator) validateNumericValue(value *ledger.NumericValue, pointer string) {
	if value.Quantiles != nil {
		seen := make(map[ledger.BasisPoints]struct{})
		var previousProbability ledger.BasisPoints
		var previousValue *big.Float
		for index, quantile := range *value.Quantiles {
			if _, exists := seen[quantile.ProbabilityBP]; exists {
				v.add("semantic.quantile_probability", pointer+"/quantiles", "quantile probabilities must be unique")
			}
			seen[quantile.ProbabilityBP] = struct{}{}
			if index > 0 && quantile.ProbabilityBP < previousProbability {
				v.add("semantic.quantile_order", pointer+"/quantiles", "quantiles must be ordered by probability")
			}
			current := parseDecimal(string(quantile.Value))
			if previousValue != nil && current.Cmp(previousValue) < 0 {
				v.add("semantic.quantile_values", pointer+"/quantiles", "quantile values must be non-decreasing")
			}
			previousProbability, previousValue = quantile.ProbabilityBP, current
		}
	}
	if value.Interval != nil && parseDecimal(string(value.Interval.Lower)).Cmp(parseDecimal(string(value.Interval.Upper))) > 0 {
		v.add("semantic.interval", pointer+"/interval", "interval lower bound must not exceed upper bound")
	}
}

func (v *semanticValidator) validateDateValue(value *ledger.DateValue, pointer string) {
	if value.Quantiles != nil {
		seen := make(map[ledger.BasisPoints]struct{})
		var previousProbability ledger.BasisPoints
		var previousValue ledger.Date
		for index, quantile := range *value.Quantiles {
			if _, exists := seen[quantile.ProbabilityBP]; exists {
				v.add("semantic.quantile_probability", pointer+"/quantiles", "quantile probabilities must be unique")
			}
			seen[quantile.ProbabilityBP] = struct{}{}
			if index > 0 && quantile.ProbabilityBP < previousProbability {
				v.add("semantic.quantile_order", pointer+"/quantiles", "quantiles must be ordered by probability")
			}
			if index > 0 && quantile.Value < previousValue {
				v.add("semantic.quantile_values", pointer+"/quantiles", "quantile values must be non-decreasing")
			}
			previousProbability, previousValue = quantile.ProbabilityBP, quantile.Value
		}
	}
	if value.Interval != nil && value.Interval.Lower > value.Interval.Upper {
		v.add("semantic.interval", pointer+"/interval", "interval lower bound must not exceed upper bound")
	}
}

func (v *semanticValidator) validateArtifact(integrity ledger.Integrity, pointer string) {
	var target *ledger.ForecastTarget
	switch {
	case integrity.Pending != nil:
		target = &integrity.Pending.Target
	case integrity.Verified != nil:
		target = &integrity.Verified.Target
	default:
		return
	}
	if v.artifacts == nil {
		return
	}
	data, err := fs.ReadFile(v.artifacts, string(target.ArtifactPath))
	if err != nil {
		v.add("semantic.artifact_missing", pointer+"/target/artifact_path", "target artifact cannot be read")
		return
	}
	digest := sha256.Sum256(data)
	expected, err := hex.DecodeString(string(target.Digest.Value))
	if err != nil || !bytes.Equal(digest[:], expected) {
		v.add("semantic.artifact_digest", pointer+"/target/digest/value", "target artifact digest does not match")
	}
}

func (v *semanticValidator) validateTimestampChronology(question *ledger.Question, forecast *ledger.Forecast, pointer string) {
	if question.Resolution == nil || question.Resolution.Resolved == nil || forecast.Integrity.Verified == nil {
		return
	}
	knownAt := parseTimestamp(question.Resolution.Resolved.OutcomeKnownAt)
	for _, timestamp := range forecast.Integrity.Verified.Timestamps {
		if timestamp.State == ledger.RFC3161Verified && timestamp.GenTime != nil && parseTimestamp(*timestamp.GenTime).Before(knownAt) {
			return
		}
	}
	v.add("semantic.timestamp_chronology", pointer+"/timestamps", "verified integrity must contain a verified RFC 3161 timestamp that predates the known outcome")
}

func (v *semanticValidator) validateReveal(question *ledger.Question, forecast *ledger.Forecast, pointer string) {
	if forecast.Visibility != ledger.VisibilityRevealed || forecast.Commitment == nil || forecast.Commitment.Revealed == nil {
		return
	}
	payload, err := forecastcrypto.Reveal(question.ID, forecast.ID, *forecast.Commitment.Revealed)
	if err != nil {
		v.add("semantic.reveal_verification", pointer+"/commitment", "revealed commitment verification failed")
		return
	}
	expected := map[string]any{
		"forecasted_at": string(forecast.ForecastedAt),
		"recorded_at":   string(forecast.RecordedAt),
		"value":         modelJSONValue(forecast.Value),
		"rationale":     dereferenceString(forecast.Rationale),
		"key_factors":   dereferenceStrings(forecast.KeyFactors),
		"comment":       dereferenceString(forecast.Comment),
	}
	for _, field := range []string{"forecasted_at", "recorded_at", "value", "rationale", "key_factors", "comment"} {
		if !reflect.DeepEqual(expected[field], payload.Bundle[field]) {
			v.add("semantic.revealed_mirror", pointer+"/"+field, "field does not match the decrypted sealed bundle")
		}
	}
}

func (v *semanticValidator) validateResolution(question *ledger.Question, pointer string) {
	if question.Resolution == nil || question.Resolution.Resolved == nil {
		return
	}
	resolution := question.Resolution.Resolved
	knownAt := parseTimestamp(resolution.OutcomeKnownAt)
	if parseTimestamp(resolution.RecordedAt).Before(knownAt) {
		v.add("semantic.resolution_chronology", pointer+"/resolution/recorded_at", "resolution recorded_at must not be before outcome_known_at")
	}
	if question.Type == ledger.QuestionMultipleChoice && resolution.Outcome.Text != nil {
		found := false
		if question.Options != nil {
			for _, option := range *question.Options {
				if string(option.ID) == *resolution.Outcome.Text {
					found = true
				}
			}
		}
		if !found {
			v.add("semantic.resolution_option", pointer+"/resolution/outcome", "resolution outcome must reference a question option")
		}
	}
}

func (v *semanticValidator) uniqueSlugs(values []ledger.Slug, pointer, code string) {
	seen := make(map[ledger.Slug]struct{}, len(values))
	for index, value := range values {
		if _, exists := seen[value]; exists {
			v.add(code, fmt.Sprintf("%s/%d", pointer, index), "ID is duplicated")
		}
		seen[value] = struct{}{}
	}
}

func (v *semanticValidator) add(code, pointer, message string) {
	v.issues = append(v.issues, SemanticIssue{Layer: "semantic", Code: code, Pointer: pointer, Message: message})
}

func forecastValueKind(value *ledger.ForecastValue) ledger.ForecastValueKind {
	switch {
	case value == nil:
		return ""
	case value.Binary != nil:
		return value.Binary.Kind
	case value.MultipleChoice != nil:
		return value.MultipleChoice.Kind
	case value.Numeric != nil:
		return value.Numeric.Kind
	case value.Date != nil:
		return value.Date.Kind
	default:
		return ""
	}
}

func sameSlugSet(left, right map[ledger.Slug]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, exists := right[value]; !exists {
			return false
		}
	}
	return true
}

func parseTimestamp(value ledger.Timestamp) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, string(value))
	return parsed
}

func parseDecimal(value string) *big.Float {
	precision := uint(len(value)*4 + 64)
	parsed, _, err := big.ParseFloat(value, 10, precision, big.ToNearestEven)
	if err != nil {
		return new(big.Float).SetPrec(precision)
	}
	return parsed
}

func modelJSONValue(value any) any {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	parsed, err := document.ParseJSON(bytes.NewReader(encoded), document.DefaultLimits)
	if err != nil {
		return nil
	}
	return parsed.Root.Any()
}

func dereferenceString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func dereferenceStrings(value *[]string) []any {
	if value == nil {
		return nil
	}
	result := make([]any, len(*value))
	for index, item := range *value {
		result[index] = item
	}
	return result
}
