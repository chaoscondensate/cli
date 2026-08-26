package ledger

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildIndexCreatesStableLookupsAndReferenceLists(t *testing.T) {
	first := Slug("f-first")
	references := []PlatformRef{{Platform: "zeta"}, {Platform: "alpha"}, {Platform: "zeta"}}
	model := &Ledger{
		Platforms: map[Slug]Platform{"zeta": {}, "alpha": {}},
		Questions: []Question{
			{ID: "q-two", PlatformRefs: &references, Forecasts: []Forecast{{ID: first}, {ID: "f-second", SupersedesForecastID: &first}}},
			{ID: "q-one", Forecasts: []Forecast{{ID: "f-third"}}},
		},
	}
	index, err := BuildIndex(model)
	if err != nil {
		t.Fatal(err)
	}
	if got := index.PlatformQuestionIDs["zeta"]; !reflect.DeepEqual(got, []Slug{"q-two"}) {
		t.Fatalf("platform references = %#v", got)
	}
	if got := index.QuestionForecastIDs["q-two"]; !reflect.DeepEqual(got, []Slug{"f-first", "f-second"}) {
		t.Fatalf("question order = %#v", got)
	}
	location, ok := index.Forecast("f-third")
	if !ok || location.QuestionID != "q-one" || location.ForecastIndex != 0 {
		t.Fatalf("forecast location = %#v, %v", location, ok)
	}
	if index.Supersedes["f-second"] != "f-first" {
		t.Fatalf("supersession index = %#v", index.Supersedes)
	}
}

func TestBuildIndexRejectsInvalidIdentityAndLinks(t *testing.T) {
	earlier := Slug("f-earlier")
	tests := []struct {
		name  string
		model *Ledger
		code  IndexErrorCode
	}{
		{name: "duplicate question", model: &Ledger{Questions: []Question{{ID: "q"}, {ID: "q"}}}, code: IndexDuplicateQuestion},
		{name: "global duplicate forecast", model: &Ledger{Questions: []Question{{ID: "q-a", Forecasts: []Forecast{{ID: "f"}}}, {ID: "q-b", Forecasts: []Forecast{{ID: "f"}}}}}, code: IndexDuplicateForecast},
		{name: "unknown platform", model: &Ledger{Questions: []Question{{ID: "q", PlatformRefs: &[]PlatformRef{{Platform: "missing"}}}}}, code: IndexUnknownPlatform},
		{name: "unknown superseded", model: &Ledger{Questions: []Question{{ID: "q", Forecasts: []Forecast{{ID: "f", SupersedesForecastID: &earlier}}}}}, code: IndexUnknownSuperseded},
		{name: "forward superseded", model: &Ledger{Questions: []Question{{ID: "q", Forecasts: []Forecast{{ID: "f", SupersedesForecastID: &earlier}, {ID: earlier}}}}}, code: IndexForwardLink},
		{name: "cross question", model: &Ledger{Questions: []Question{{ID: "q-a", Forecasts: []Forecast{{ID: earlier}}}, {ID: "q-b", Forecasts: []Forecast{{ID: "f", SupersedesForecastID: &earlier}}}}}, code: IndexCrossQuestionLink},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildIndex(test.model)
			var indexErr *IndexError
			if !errors.As(err, &indexErr) || indexErr.Code != test.code {
				t.Fatalf("error = %#v, want %q", err, test.code)
			}
		})
	}
}

func FuzzBuildIndexSelectors(f *testing.F) {
	f.Add("q-one", "f-one", "f-two")
	f.Add("same", "same", "same")
	f.Fuzz(func(t *testing.T, questionID, firstID, secondID string) {
		first := Slug(firstID)
		model := &Ledger{Questions: []Question{{ID: Slug(questionID), Forecasts: []Forecast{{ID: first}, {ID: Slug(secondID), SupersedesForecastID: &first}}}}}
		index, err := BuildIndex(model)
		if err != nil {
			return
		}
		if location, ok := index.Forecast(Slug(secondID)); !ok || location.QuestionID != Slug(questionID) {
			t.Fatalf("built selector index lost forecast location: %#v, %v", location, ok)
		}
	})
}
