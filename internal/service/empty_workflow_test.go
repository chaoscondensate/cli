package service

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/storage"
)

func TestEmptyAndQuestionOnlyCreationShapes(t *testing.T) {
	root, err := BuildLedgerRootAt(InitRootRequest{
		LedgerID: "empty", Timezone: "UTC", ForecasterID: "owner", ForecasterName: "Owner",
	}, "2026-08-29T10:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	shape, err := ClassifyInitInput(InitInput{})
	if err != nil || shape != CreationLedgerOnly {
		t.Fatalf("shape=%q err=%v", shape, err)
	}
	encoded, err := EncodeNewLedger(root, "ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	if result := NewInitResult(root, []SideEffect{}, Recovery{State: RecoveryNone}); result.QuestionCount != 0 || result.ForecastCount != 0 || result.QuestionID != "" || result.ForecastID != "" {
		t.Fatalf("empty result = %#v", result)
	}
	if len(encoded) == 0 {
		t.Fatal("empty ledger encoded to no bytes")
	}

	question := binaryInitialQuestion()
	question.InitialForecast = nil
	withQuestion, err := BuildInitialQuestionLedgerAt(root, question, "2026-08-29T10:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	result := NewInitResult(withQuestion, nil, Recovery{State: RecoveryNone})
	if result.QuestionCount != 1 || result.ForecastCount != 0 || result.QuestionID != question.ID || result.ForecastID != "" || len(withQuestion.Questions[0].Forecasts) != 0 {
		t.Fatalf("question-only result = %#v model=%#v", result, withQuestion)
	}
}

func TestQuestionWithoutForecastSourcePreservingJSONAndYAML(t *testing.T) {
	for _, extension := range []string{".json", ".yaml"} {
		t.Run(extension, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "ledger"+extension)
			root, err := BuildLedgerRootAt(InitRootRequest{LedgerID: "empty", Timezone: "UTC", ForecasterID: "owner", ForecasterName: "Owner"}, "2026-08-29T10:00:00Z")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := CommitNewLedger(path, root); err != nil {
				t.Fatal(err)
			}
			inputQuestion := binaryInitialQuestion()
			input := NormalizeInitialQuestion(inputQuestion)
			input.Input.InitialForecast = nil
			result, err := CommitQuestionAddEmptyFile(context.Background(), path, input, "2026-08-29T10:00:00Z")
			if err != nil || !result.Changed {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			loaded, err := LoadAndValidateLedger(context.Background(), path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(loaded.Model.Questions) != 1 || loaded.Model.Questions[0].Forecasts == nil || len(loaded.Model.Questions[0].Forecasts) != 0 {
				t.Fatalf("questions = %#v", loaded.Model.Questions)
			}
		})
	}
}

func TestBacklogQuestionAcceptsFirstForecastAndLifecycle(t *testing.T) {
	_, model := rootUpdateFixture(t, "question-without-forecasts.yaml")
	questionID := model.Questions[0].ID
	input := ForecastCreateInput{
		ForecastedAt: "2026-08-30T09:00:00Z",
		Value:        ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}},
	}
	mutation, err := BuildPublicForecastAppend(model, questionID, "f-first", input, "2026-08-30T09:01:00Z")
	if err != nil {
		t.Fatal(err)
	}
	forecast := mutation.Ledger.Questions[0].Forecasts[0]
	if forecast.SupersedesForecastID != nil {
		t.Fatalf("first forecast supersedes %#v", forecast.SupersedesForecastID)
	}
	sealed, err := PlanSealedForecastAppend(model, questionID, "f-first-sealed", SealedForecastInput{
		ForecastedAt: "2026-08-30T09:00:00Z",
		Value:        ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: ledger.ValueBinary, ProbabilityBP: 5000}},
		Rationale:    "Private rationale", KeyFactors: []string{}, Comment: "Private comment",
	}, "2026-08-30T09:01:00Z")
	if err != nil || len(sealed.Ledger.Questions[0].Forecasts) != 1 || sealed.Ledger.Questions[0].Forecasts[0].Visibility != ledger.VisibilitySealed || sealed.Ledger.Questions[0].Forecasts[0].SupersedesForecastID != nil {
		t.Fatalf("first sealed forecast=%#v err=%v", sealed, err)
	}
	missing := ledger.Slug("f-missing")
	input.SupersedesForecastID = &missing
	if _, err := BuildPublicForecastAppend(model, questionID, "f-invalid", input, "2026-08-30T09:01:00Z"); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("missing supersedes error = %v", err)
	}
	annulled, err := BuildQuestionAnnul(model, questionID, AnnulInput{Reason: "The event definition was withdrawn."}, "2026-08-30T09:02:00Z")
	if err != nil || annulled.Ledger.Questions[0].Status != ledger.QuestionAnnulled || len(annulled.Ledger.Questions[0].Forecasts) != 0 {
		t.Fatalf("annulled=%#v err=%v", annulled, err)
	}
	closed, err := BuildQuestionUpdate(model, questionID, QuestionPatchInput{Status: Optional[ledger.QuestionStatus]{Set: true, Value: ledger.QuestionClosed}})
	if err != nil {
		t.Fatal(err)
	}
	yes := true
	resolved, err := BuildQuestionResolve(closed.Ledger, questionID, ResolutionInput{
		Outcome: ResolutionOutcome{Boolean: &yes}, OutcomeKnownAt: "2027-01-02T00:00:00Z",
		Sources: []EvidenceSourceInput{{Title: "Official result", URL: "https://example.org/result", RetrievedAt: "2027-01-02T00:01:00Z"}},
	}, "2027-01-02T00:02:00Z")
	if err != nil || resolved.Ledger.Questions[0].Status != ledger.QuestionResolved || len(resolved.Ledger.Questions[0].Forecasts) != 0 {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
	disputed, err := BuildQuestionDispute(resolved.Ledger, questionID, DisputeInput{Reason: "The result is under review."}, "2027-01-03T00:00:00Z")
	if err != nil || disputed.Ledger.Questions[0].Status != ledger.QuestionDisputed || len(disputed.Ledger.Questions[0].Forecasts) != 0 {
		t.Fatalf("disputed=%#v err=%v", disputed, err)
	}
}

func TestEmptyReadsPlatformMutationAndSpecificForecastFailures(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "empty-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadAndValidateLedger(context.Background(), path, nil)
	if err != nil {
		t.Fatal(err)
	}
	status, err := StatusForLedger(loaded)
	if err != nil || status.Questions != 0 || status.Forecasts != 0 {
		t.Fatalf("status=%#v err=%v", status, err)
	}
	_, questions, err := LoadQuestionList(context.Background(), path, nil)
	if err != nil || questions == nil || len(questions) != 0 {
		t.Fatalf("questions=%#v err=%v", questions, err)
	}
	if _, err := CommitPlatformAddFile(context.Background(), path, "local", PlatformCreateInput{Name: "Local", Kind: ledger.PlatformInformal}); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitPlatformRemoveFile(context.Background(), path, "local"); err != nil {
		t.Fatal(err)
	}

	backlog, err := fs.ReadFile(contractschema.Conformance(), "question-without-forecasts.yaml")
	if err != nil {
		t.Fatal(err)
	}
	backlogPath := filepath.Join(directory, "backlog.yaml")
	if err := os.WriteFile(backlogPath, backlog, 0o600); err != nil {
		t.Fatal(err)
	}
	questionID, forecastID := ledger.Slug("q-example-backlog"), ledger.Slug("f-missing")
	if _, _, err := LoadForecastShow(context.Background(), backlogPath, nil, questionID, forecastID); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("forecast show error=%v", err)
	}
	if _, err := CheckTargets(context.Background(), backlogPath, false, questionID, forecastID); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("target error=%v", err)
	}
	if _, err := TimestampStatusFor(context.Background(), backlogPath, questionID, forecastID); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("timestamp error=%v", err)
	}
	keyPath := filepath.Join(directory, "dummy.key")
	if err := storage.CreateProtectedFile(keyPath, []byte("invalid but protected")); err != nil {
		t.Fatal(err)
	}
	beforeFailures, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanForecastRevealFile(context.Background(), backlogPath, keyPath, questionID, forecastID, "2026-08-30T09:00:00Z"); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("reveal error=%v", err)
	}
	afterFailures, err := os.ReadDir(directory)
	if err != nil || len(afterFailures) != len(beforeFailures) {
		t.Fatalf("specific failures changed directory entries=%v err=%v", afterFailures, err)
	}
	for index := range beforeFailures {
		if beforeFailures[index].Name() != afterFailures[index].Name() {
			t.Fatalf("specific failures changed directory entries before=%v after=%v", beforeFailures, afterFailures)
		}
	}
}

func TestEmptyAggregateOperationsAndPublication(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "empty-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := CommitTargetBuild(context.Background(), ledgerPath, true, "", "")
	if err != nil || targets.Targets == nil || len(targets.Targets) != 0 || len(targets.Effects) != 0 || targets.Recovery.State != RecoveryNone {
		t.Fatalf("targets=%#v err=%v", targets, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "proofs")); !os.IsNotExist(err) {
		t.Fatalf("empty target build created proofs: %v", err)
	}
	checked, err := CheckTargets(context.Background(), ledgerPath, true, "", "")
	if err != nil || checked.Targets == nil || len(checked.Targets) != 0 {
		t.Fatalf("empty target check=%#v err=%v", checked, err)
	}
	afterTargets, err := os.ReadFile(ledgerPath)
	if err != nil || string(afterTargets) != string(raw) {
		t.Fatalf("empty target build changed ledger: err=%v", err)
	}
	report, err := VerifyLedgerEvidence(context.Background(), ledgerPath, VerificationOptions{})
	if err != nil || report.Overall != VerificationNoEvidence || report.FailureCode != app.CodeIncomplete || report.Forecasts == nil || len(report.Forecasts) != 0 || report.RequestSummary.HTTPRequests != 0 {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	backlogBytes, err := fs.ReadFile(contractschema.Conformance(), "question-without-forecasts.yaml")
	if err != nil {
		t.Fatal(err)
	}
	backlogPath := filepath.Join(directory, "backlog.yaml")
	if err := os.WriteFile(backlogPath, backlogBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	backlogReport, err := VerifyLedgerEvidence(context.Background(), backlogPath, VerificationOptions{QuestionID: "q-example-backlog"})
	if err != nil || backlogReport.Overall != VerificationNoEvidence || backlogReport.FailureCode != app.CodeIncomplete || len(backlogReport.Forecasts) != 0 || backlogReport.RequestSummary.HTTPRequests != 0 {
		t.Fatalf("backlog report=%#v err=%v", backlogReport, err)
	}

	output := filepath.Join(directory, "package")
	built, err := CommitPublicationBuild(context.Background(), ledgerPath, output, false)
	if err != nil || built.FileCount != 2 || len(built.Files) != 1 || built.EvidenceState != "complete" {
		t.Fatalf("built=%#v err=%v", built, err)
	}
	verified, err := VerifyPublicationPackage(context.Background(), filepath.Join(output, "ledger", "ledger.json"), filepath.Join(output, "manifest.json"))
	if err != nil || verified.Overall != VerificationNoEvidence || verified.FailureCode != app.CodeIncomplete || verified.Evidence == nil || len(verified.Evidence) != 0 || verified.RequestSummary.HTTPRequests != 0 {
		t.Fatalf("verified=%#v err=%v", verified, err)
	}
}
