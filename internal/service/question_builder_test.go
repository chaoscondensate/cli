package service

import (
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
)

func TestBuildInitialPublicLedgerProducesFullyValidLedger(t *testing.T) {
	root, err := BuildLedgerRoot(InitRootRequest{
		LedgerID: "research", Timezone: "UTC", ForecasterID: "andrey", ForecasterName: "Andrey",
	}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := BuildInitialPublicLedger(root, binaryInitialQuestion())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Questions) != 1 || len(result.Questions[0].Forecasts) != 1 {
		t.Fatalf("initial ledger = %#v", result)
	}
	forecast := result.Questions[0].Forecasts[0]
	if forecast.Visibility != ledger.VisibilityPublic || forecast.RecordedAt != root.CreatedAt || forecast.Integrity.Unanchored == nil {
		t.Fatalf("initial forecast = %#v", forecast)
	}
	if err := ValidateProspectiveLedgerModel(result); err != nil {
		t.Fatal(err)
	}
}

func TestBuildInitialPublicLedgerSupportsEveryQuestionType(t *testing.T) {
	root, err := BuildLedgerRoot(InitRootRequest{
		LedgerID: "research", Timezone: "UTC", ForecasterID: "andrey", ForecasterName: "Andrey",
	}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	point := ledger.Decimal("42.5")
	date := ledger.Date("2026-06-15")
	choices := []ledger.Option{{ID: "yes", Label: "Yes"}, {ID: "no", Label: "No"}}
	unit := ledger.Unit{Name: "items"}
	tests := []struct {
		name  string
		type_ ledger.QuestionType
		value ledger.ForecastValue
		apply func(*InitialQuestionInput)
	}{
		{name: "binary", type_: ledger.QuestionBinary, value: ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}}},
		{name: "multiple choice", type_: ledger.QuestionMultipleChoice, value: ledger.ForecastValue{MultipleChoice: &ledger.MultipleChoiceValue{Kind: ledger.ValueMultipleChoice, Probabilities: []ledger.ChoiceProbability{{OptionID: "yes", ProbabilityBP: 6000}, {OptionID: "no", ProbabilityBP: 4000}}}}, apply: func(input *InitialQuestionInput) { input.Options = &choices }},
		{name: "numeric", type_: ledger.QuestionNumeric, value: ledger.ForecastValue{Numeric: &ledger.NumericValue{Kind: ledger.ValueNumeric, Point: &point}}, apply: func(input *InitialQuestionInput) { input.Unit = &unit }},
		{name: "date", type_: ledger.QuestionDate, value: ledger.ForecastValue{Date: &ledger.DateValue{Kind: ledger.ValueDate, Point: &date}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := binaryInitialQuestion()
			input.Type = test.type_
			input.InitialForecast.Value = test.value
			if test.apply != nil {
				test.apply(&input)
			}
			result, err := BuildInitialPublicLedger(root, input)
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Questions[0].Type; got != test.type_ {
				t.Fatalf("question type = %q, want %q", got, test.type_)
			}
			if err := ValidateProspectiveLedgerModel(result); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInitialQuestionRejectsTypeWindowDistributionAndGlobalIDErrors(t *testing.T) {
	root, err := BuildLedgerRoot(InitRootRequest{LedgerID: "research", Timezone: "UTC", ForecasterID: "andrey", ForecasterName: "Andrey"}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	tests := []InitialQuestionInput{
		func() InitialQuestionInput {
			value := binaryInitialQuestion()
			value.Unit = &ledger.Unit{Name: "USD"}
			return value
		}(),
		func() InitialQuestionInput {
			value := binaryInitialQuestion()
			value.ForecastWindow.ClosesAt = "2025-01-01T00:00:00Z"
			return value
		}(),
		func() InitialQuestionInput {
			value := binaryInitialQuestion()
			value.InitialForecast.Value.Binary.ProbabilityBP = 10001
			return value
		}(),
		func() InitialQuestionInput {
			value := binaryInitialQuestion()
			value.InitialForecast.Visibility = ledger.VisibilitySealed
			return value
		}(),
	}
	for index, input := range tests {
		if _, err := BuildInitialPublicLedger(root, input); app.ErrorCodeOf(err) != app.CodeInvalidData {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
	valid, err := BuildInitialPublicLedger(root, binaryInitialQuestion())
	if err != nil {
		t.Fatal(err)
	}
	duplicate := binaryInitialQuestion()
	duplicate.ID = "q-two"
	if _, err := BuildQuestionWithInitialPublicForecast(valid, NormalizeInitialQuestion(duplicate), valid.CreatedAt); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("duplicate global forecast error = %v", err)
	}
}

func TestValidateForecastValueChecksExactNumericAndDateForms(t *testing.T) {
	decimal := func(value string) *ledger.Decimal { result := ledger.Decimal(value); return &result }
	date := func(value string) *ledger.Date { result := ledger.Date(value); return &result }
	tests := []struct {
		name      string
		type_     ledger.QuestionType
		value     ledger.ForecastValue
		wantError bool
	}{
		{name: "numeric point", type_: ledger.QuestionNumeric, value: ledger.ForecastValue{Numeric: &ledger.NumericValue{Kind: ledger.ValueNumeric, Point: decimal("1.25e+2")}}},
		{name: "numeric leading zero", type_: ledger.QuestionNumeric, value: ledger.ForecastValue{Numeric: &ledger.NumericValue{Kind: ledger.ValueNumeric, Point: decimal("01")}}, wantError: true},
		{name: "numeric reversed interval", type_: ledger.QuestionNumeric, value: ledger.ForecastValue{Numeric: &ledger.NumericValue{Kind: ledger.ValueNumeric, Interval: &ledger.NumericInterval{Lower: "10", Upper: "2", CredibilityBP: 8000}}}, wantError: true},
		{name: "numeric unordered quantiles", type_: ledger.QuestionNumeric, value: ledger.ForecastValue{Numeric: &ledger.NumericValue{Kind: ledger.ValueNumeric, Quantiles: &[]ledger.NumericQuantile{{ProbabilityBP: 9000, Value: "9"}, {ProbabilityBP: 1000, Value: "1"}}}}, wantError: true},
		{name: "numeric decreasing values", type_: ledger.QuestionNumeric, value: ledger.ForecastValue{Numeric: &ledger.NumericValue{Kind: ledger.ValueNumeric, Quantiles: &[]ledger.NumericQuantile{{ProbabilityBP: 1000, Value: "9"}, {ProbabilityBP: 9000, Value: "1"}}}}, wantError: true},
		{name: "date point", type_: ledger.QuestionDate, value: ledger.ForecastValue{Date: &ledger.DateValue{Kind: ledger.ValueDate, Point: date("2026-02-28")}}},
		{name: "invalid calendar date", type_: ledger.QuestionDate, value: ledger.ForecastValue{Date: &ledger.DateValue{Kind: ledger.ValueDate, Point: date("2026-02-30")}}, wantError: true},
		{name: "date reversed interval", type_: ledger.QuestionDate, value: ledger.ForecastValue{Date: &ledger.DateValue{Kind: ledger.ValueDate, Interval: &ledger.DateInterval{Lower: "2026-03-01", Upper: "2026-02-01", CredibilityBP: 8000}}}, wantError: true},
		{name: "date duplicate probability", type_: ledger.QuestionDate, value: ledger.ForecastValue{Date: &ledger.DateValue{Kind: ledger.ValueDate, Quantiles: &[]ledger.DateQuantile{{ProbabilityBP: 5000, Value: "2026-01-01"}, {ProbabilityBP: 5000, Value: "2026-02-01"}}}}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateForecastValue(test.type_, nil, &test.value)
			if test.wantError && app.ErrorCodeOf(err) != app.CodeInvalidData {
				t.Fatalf("error = %v", err)
			}
			if !test.wantError && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func binaryInitialQuestion() InitialQuestionInput {
	return InitialQuestionInput{
		ID: "q-one", Title: "Will it happen?", Type: ledger.QuestionBinary,
		ResolutionCriteria:   "Resolve from the named public source.",
		ForecastWindow:       ledger.ForecastWindow{ClosesAt: "2026-12-31T00:00:00Z"},
		ExpectedResolutionAt: "2027-01-01T00:00:00Z",
		InitialForecast: &InitialForecastInput{
			ID: "f-one", Visibility: ledger.VisibilityPublic,
			ForecastedAt: "2026-01-01T00:00:00Z",
			Value:        ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}},
		},
	}
}
