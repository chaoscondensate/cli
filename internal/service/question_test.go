package service

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestQuestionAddUpdateListAndShow(t *testing.T) {
	_, model := rootUpdateFixture(t, "individual-ledger.json")
	add := NormalizedQuestionCreate{ID: "q-new", Type: ledger.QuestionBinary, Input: QuestionAddInput{
		Title: "Will the new event happen?", ResolutionCriteria: "Resolve from the named source.",
		CreatedAt:      timestampPointer("2026-08-20T00:00:00Z"),
		ForecastWindow: ledger.ForecastWindow{}, ExpectedResolutionAt: "2026-12-02T00:00:00Z",
		InitialForecast: &InitialForecastInput{ID: "f-new-001", Visibility: ledger.VisibilityPublic, ForecastedAt: "2026-08-20T00:00:00Z", Value: ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5500}}},
	}}
	mutation, err := BuildQuestionAddPublic(model, add, "2026-08-20T00:01:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(mutation.Patches) != 1 || mutation.Patches[0].Pointer != "/questions/-" || len(mutation.Ledger.Questions) != len(model.Questions)+1 {
		t.Fatalf("add mutation = %#v", mutation)
	}

	newTitle := Optional[string]{Set: true, Value: "Updated question title"}
	newTags := Optional[[]ledger.Slug]{Set: true, Value: []ledger.Slug{"reviewed"}}
	updated, err := BuildQuestionUpdate(model, "q-election-coalition", QuestionPatchInput{Title: newTitle, Tags: newTags})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Ledger.Questions[1].Title != "Updated question title" || updated.Ledger.Questions[1].Tags == nil || len(updated.Patches) != 2 {
		t.Fatalf("update mutation = %#v", updated)
	}
	opening := Optional[ledger.Timestamp]{Set: true, Value: "2026-08-07T12:00:00+01:00"}
	if _, err := BuildQuestionUpdate(model, "q-election-coalition", QuestionPatchInput{ForecastWindow: Optional[ForecastWindowPatchInput]{Set: true, Value: ForecastWindowPatchInput{OpensAt: opening}}}); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("moved opening error = %v", err)
	}
	model.Questions[1].Forecasts[0].Integrity = ledger.Integrity{Failed: &ledger.FailedIntegrity{Status: ledger.IntegrityFailed, FailureReason: "imported", Target: &ledger.ForecastTarget{}}}
	if _, err := BuildQuestionUpdate(model, "q-election-coalition", QuestionPatchInput{Title: newTitle}); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("frozen target error = %v", err)
	}

	items, err := ListQuestions(model)
	if err != nil || len(items) != 4 || items[0].ID != "q-central-bank-cut" {
		t.Fatalf("question list = %#v, %v", items, err)
	}
	view, err := ShowQuestion(model, "q-election-coalition")
	if err != nil || len(view.Forecasts) != 1 || view.Forecasts[0].Summary.ID == "" {
		t.Fatalf("question view = %#v, %v", view, err)
	}
}

func TestQuestionAddSealedCommitsProtectedKeyBeforeValidLedger(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "f-secret.key")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	rationale, comment := "PRIVATE-RATIONALE-CANARY", "PRIVATE-COMMENT-CANARY"
	factors := []string{"PRIVATE-FACTOR-CANARY"}
	input := NormalizedQuestionCreate{ID: "q-secret", Type: ledger.QuestionBinary, Input: QuestionAddInput{
		Title: "Secret forecast question", ResolutionCriteria: "Resolve from the named source.", CreatedAt: timestampPointer("2026-08-20T00:00:00Z"),
		ForecastWindow: ledger.ForecastWindow{}, ExpectedResolutionAt: "2026-12-02T00:00:00Z",
		InitialForecast: &InitialForecastInput{ID: "f-secret", Visibility: ledger.VisibilitySealed, ForecastedAt: "2026-08-20T00:00:00Z", RecordedAt: timestampPointer("2026-08-20T00:01:00Z"), Value: ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5100}}, Rationale: &rationale, KeyFactors: &factors, Comment: &comment},
	}}
	plan, err := PlanQuestionAddSealedFile(context.Background(), ledgerPath, keyPath, input, "2026-08-20T00:01:00Z")
	if err != nil || !plan.Changed || plan.Recovery.State != "" {
		t.Fatalf("plan = %#v, %v", plan, err)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run created key: %v", err)
	}
	effects := Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x42}, 76))}}
	result, err := CommitQuestionAddSealedFile(context.Background(), ledgerPath, keyPath, input, "2026-08-20T00:01:00Z", effects)
	if err != nil {
		t.Fatal(err)
	}
	if result.Recovery.State != RecoveryNone || len(result.Effects) != 2 {
		t.Fatalf("result = %#v", result)
	}
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil || !bytes.Contains(keyBytes, []byte(`"schema":"forecast-key/v1"`)) {
		t.Fatalf("key = %q, %v", keyBytes, err)
	}
	ledgerBytes, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{rationale, comment, factors[0]} {
		if strings.Contains(string(ledgerBytes), secret) {
			t.Fatalf("ledger leaked %q", secret)
		}
	}
	loaded, err := LoadAndValidateLedger(context.Background(), ledgerPath, nil)
	if err != nil || loaded.Model.Questions[len(loaded.Model.Questions)-1].Forecasts[0].Visibility != ledger.VisibilitySealed {
		t.Fatalf("sealed ledger = %#v, %v", loaded, err)
	}
}

func TestQuestionPublicFileAddThenUpdate(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	input := NormalizedQuestionCreate{ID: "q-new", Type: ledger.QuestionBinary, Input: QuestionAddInput{
		Title: "New", ResolutionCriteria: "Official source", CreatedAt: timestampPointer("2026-08-20T00:00:00Z"),
		ForecastWindow: ledger.ForecastWindow{}, ExpectedResolutionAt: "2026-12-02T00:00:00Z",
		InitialForecast: &InitialForecastInput{ID: "f-new", Visibility: ledger.VisibilityPublic, ForecastedAt: "2026-08-20T00:00:00Z", RecordedAt: timestampPointer("2026-08-20T00:01:00Z"), Value: ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}}},
	}}
	if _, err := CommitQuestionAddPublicFile(context.Background(), path, input, "2026-08-20T00:01:00Z"); err != nil {
		t.Fatal(err)
	}
	beforeUpdate, err := LoadAndValidateLedger(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	retainedForecast := beforeUpdate.Model.Questions[len(beforeUpdate.Model.Questions)-1].Forecasts[0]
	if _, err := CommitQuestionUpdateFile(context.Background(), path, "q-new", QuestionPatchInput{Status: Optional[ledger.QuestionStatus]{Set: true, Value: ledger.QuestionClosed}}); err != nil {
		t.Fatalf("update after add: %v; cause: %v", err, errors.Unwrap(err))
	}
	afterUpdate, err := LoadAndValidateLedger(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(retainedForecast, afterUpdate.Model.Questions[len(afterUpdate.Model.Questions)-1].Forecasts[0]) {
		t.Fatal("question status update changed retained forecast")
	}
	targetDirectory := filepath.Join(filepath.Dir(path), "proofs", "targets")
	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDirectory, "f-new.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanQuestionUpdateFile(context.Background(), path, "q-new", QuestionPatchInput{Title: Optional[string]{Set: true, Value: "Changed meaning"}}); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("precomputed target conflict = %v", err)
	}
}

func TestQuestionLifecycleTypedOutcomesAndReplacement(t *testing.T) {
	_, model := rootUpdateFixture(t, "individual-ledger.json")
	closed, err := BuildQuestionUpdate(model, "q-election-coalition", QuestionPatchInput{Status: Optional[ledger.QuestionStatus]{Set: true, Value: ledger.QuestionClosed}})
	if err != nil {
		t.Fatal(err)
	}
	outcome := "centre-left"
	resolved, err := BuildQuestionResolve(closed.Ledger, "q-election-coalition", ResolutionInput{
		Outcome: ResolutionOutcome{Text: &outcome}, OutcomeKnownAt: "2026-10-15T12:00:00+01:00", RecordedAt: timestampPointer("2026-10-15T12:05:00+01:00"),
		Sources: []EvidenceSourceInput{{Title: "Official appointment", URL: "https://example.org/result", RetrievedAt: "2026-10-15T12:04:00+01:00"}},
	}, "2026-10-15T12:05:00+01:00")
	if err != nil {
		t.Fatal(err)
	}
	question := resolved.Ledger.Questions[1]
	if question.Status != ledger.QuestionResolved || question.Resolution == nil || question.Resolution.Resolved == nil || question.Resolution.Resolved.Outcome.Text == nil {
		t.Fatalf("resolved question = %#v", question)
	}
	disputed, err := BuildQuestionDispute(resolved.Ledger, question.ID, DisputeInput{Reason: "The appointment is under review.", RecordedAt: timestampPointer("2026-10-16T00:00:00+01:00")}, "2026-10-16T00:00:00+01:00")
	if err != nil {
		t.Fatal(err)
	}
	if disputed.PriorStatus != ledger.QuestionResolved || disputed.Ledger.Questions[1].Status != ledger.QuestionDisputed {
		t.Fatalf("disputed = %#v", disputed)
	}
	annulled, err := BuildQuestionAnnul(disputed.Ledger, question.ID, AnnulInput{Reason: "The event definition became invalid."}, "2026-10-17T00:00:00+01:00")
	if err != nil || annulled.Ledger.Questions[1].Status != ledger.QuestionAnnulled {
		t.Fatalf("annulled = %#v, %v", annulled, err)
	}
	if _, err := BuildQuestionDispute(model, "q-election-coalition", DisputeInput{Reason: "too early"}, "2026-10-17T00:00:00+01:00"); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("unresolved dispute error = %v", err)
	}
	badOutcome := "missing-option"
	if _, err := BuildQuestionResolve(closed.Ledger, "q-election-coalition", ResolutionInput{Outcome: ResolutionOutcome{Text: &badOutcome}, OutcomeKnownAt: "2026-10-15T12:00:00+01:00", Sources: []EvidenceSourceInput{{Title: "Source", URL: "https://example.org", RetrievedAt: "2026-10-15T12:00:00+01:00"}}}, "2026-10-15T12:05:00+01:00"); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("invalid typed outcome error = %v", err)
	}
}
