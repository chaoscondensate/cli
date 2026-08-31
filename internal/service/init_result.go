package service

import "github.com/chaoscondensate/forecast-ledger/internal/ledger"

type InitResult struct {
	LedgerID        ledger.Slug               `json:"ledger_id"`
	SchemaVersion   ledger.SchemaVersion      `json:"schema_version"`
	QuestionCount   int                       `json:"question_count"`
	ForecastCount   int                       `json:"forecast_count"`
	QuestionID      ledger.Slug               `json:"question_id,omitempty"`
	ForecastID      ledger.Slug               `json:"forecast_id,omitempty"`
	Visibility      ledger.ForecastVisibility `json:"visibility,omitempty"`
	NormalizedTimes []TimeNormalization       `json:"normalized_times,omitempty"`
	Effects         []SideEffect              `json:"effects"`
	Recovery        Recovery                  `json:"recovery"`
}

func NewInitResult(model *ledger.Ledger, effects []SideEffect, recovery Recovery) InitResult {
	result := InitResult{Effects: effects, Recovery: recovery}
	if model == nil {
		return result
	}
	result.LedgerID = model.LedgerID
	result.SchemaVersion = model.SchemaVersion
	result.QuestionCount = len(model.Questions)
	for questionIndex := range model.Questions {
		result.ForecastCount += len(model.Questions[questionIndex].Forecasts)
	}
	if len(model.Questions) == 0 {
		return result
	}
	result.QuestionID = model.Questions[0].ID
	if len(model.Questions[0].Forecasts) == 0 {
		return result
	}
	result.ForecastID = model.Questions[0].Forecasts[0].ID
	result.Visibility = model.Questions[0].Forecasts[0].Visibility
	return result
}
