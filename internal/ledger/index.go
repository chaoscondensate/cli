package ledger

import (
	"fmt"
	"sort"
)

type IndexErrorCode string

const (
	IndexDuplicateQuestion IndexErrorCode = "duplicate_question_id"
	IndexDuplicateForecast IndexErrorCode = "duplicate_forecast_id"
	IndexUnknownPlatform   IndexErrorCode = "unknown_platform_reference"
	IndexUnknownSuperseded IndexErrorCode = "unknown_superseded_forecast"
	IndexCrossQuestionLink IndexErrorCode = "cross_question_supersession"
	IndexForwardLink       IndexErrorCode = "forward_supersession"
)

type IndexError struct {
	Code       IndexErrorCode
	QuestionID Slug
	ForecastID Slug
	Reference  Slug
}

func (e *IndexError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ledger index %s at question %q forecast %q reference %q", e.Code, e.QuestionID, e.ForecastID, e.Reference)
}

type ForecastLocation struct {
	QuestionIndex int
	ForecastIndex int
	QuestionID    Slug
	ForecastID    Slug
}

// Index is an immutable lookup snapshot for one validated ledger. It stores
// positions rather than mutable record pointers so append-only forecast code
// cannot accidentally replace historical records through a selector.
type Index struct {
	PlatformIDs         map[Slug]struct{}
	QuestionPositions   map[Slug]int
	ForecastLocations   map[Slug]ForecastLocation
	QuestionForecastIDs map[Slug][]Slug
	PlatformQuestionIDs map[Slug][]Slug
	Supersedes          map[Slug]Slug
	SupersededBy        map[Slug][]Slug
}

func BuildIndex(model *Ledger) (*Index, error) {
	if model == nil {
		return nil, fmt.Errorf("ledger is nil")
	}
	index := &Index{
		PlatformIDs:         make(map[Slug]struct{}, len(model.Platforms)),
		QuestionPositions:   make(map[Slug]int, len(model.Questions)),
		ForecastLocations:   make(map[Slug]ForecastLocation),
		QuestionForecastIDs: make(map[Slug][]Slug, len(model.Questions)),
		PlatformQuestionIDs: make(map[Slug][]Slug, len(model.Platforms)),
		Supersedes:          make(map[Slug]Slug),
		SupersededBy:        make(map[Slug][]Slug),
	}
	for id := range model.Platforms {
		index.PlatformIDs[id] = struct{}{}
	}

	for questionPosition := range model.Questions {
		question := &model.Questions[questionPosition]
		if _, exists := index.QuestionPositions[question.ID]; exists {
			return nil, &IndexError{Code: IndexDuplicateQuestion, QuestionID: question.ID}
		}
		index.QuestionPositions[question.ID] = questionPosition
		if question.PlatformRefs != nil {
			seen := make(map[Slug]struct{}, len(*question.PlatformRefs))
			for _, reference := range *question.PlatformRefs {
				if _, exists := index.PlatformIDs[reference.Platform]; !exists {
					return nil, &IndexError{Code: IndexUnknownPlatform, QuestionID: question.ID, Reference: reference.Platform}
				}
				if _, duplicate := seen[reference.Platform]; duplicate {
					continue
				}
				seen[reference.Platform] = struct{}{}
				index.PlatformQuestionIDs[reference.Platform] = append(index.PlatformQuestionIDs[reference.Platform], question.ID)
			}
		}
		for forecastPosition := range question.Forecasts {
			forecast := &question.Forecasts[forecastPosition]
			if _, exists := index.ForecastLocations[forecast.ID]; exists {
				return nil, &IndexError{Code: IndexDuplicateForecast, QuestionID: question.ID, ForecastID: forecast.ID}
			}
			index.ForecastLocations[forecast.ID] = ForecastLocation{
				QuestionIndex: questionPosition, ForecastIndex: forecastPosition,
				QuestionID: question.ID, ForecastID: forecast.ID,
			}
			index.QuestionForecastIDs[question.ID] = append(index.QuestionForecastIDs[question.ID], forecast.ID)
		}
	}

	for questionPosition := range model.Questions {
		question := &model.Questions[questionPosition]
		for forecastPosition := range question.Forecasts {
			forecast := &question.Forecasts[forecastPosition]
			if forecast.SupersedesForecastID == nil {
				continue
			}
			reference := *forecast.SupersedesForecastID
			location, exists := index.ForecastLocations[reference]
			if !exists {
				return nil, &IndexError{Code: IndexUnknownSuperseded, QuestionID: question.ID, ForecastID: forecast.ID, Reference: reference}
			}
			if location.QuestionID != question.ID {
				return nil, &IndexError{Code: IndexCrossQuestionLink, QuestionID: question.ID, ForecastID: forecast.ID, Reference: reference}
			}
			if location.ForecastIndex >= forecastPosition {
				return nil, &IndexError{Code: IndexForwardLink, QuestionID: question.ID, ForecastID: forecast.ID, Reference: reference}
			}
			index.Supersedes[forecast.ID] = reference
			index.SupersededBy[reference] = append(index.SupersededBy[reference], forecast.ID)
		}
	}
	for platform := range index.PlatformQuestionIDs {
		sort.Slice(index.PlatformQuestionIDs[platform], func(i, j int) bool {
			return index.PlatformQuestionIDs[platform][i] < index.PlatformQuestionIDs[platform][j]
		})
	}
	for forecast := range index.SupersededBy {
		sort.Slice(index.SupersededBy[forecast], func(i, j int) bool {
			return index.SupersededBy[forecast][i] < index.SupersededBy[forecast][j]
		})
	}
	return index, nil
}

func (i *Index) Question(id Slug) (int, bool) {
	if i == nil {
		return 0, false
	}
	position, ok := i.QuestionPositions[id]
	return position, ok
}

func (i *Index) Forecast(id Slug) (ForecastLocation, bool) {
	if i == nil {
		return ForecastLocation{}, false
	}
	location, ok := i.ForecastLocations[id]
	return location, ok
}
