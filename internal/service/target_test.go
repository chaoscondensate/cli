package service

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/document"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	contractschema "github.com/chaoscondensate/forecast-ledger/internal/schema"
	"github.com/chaoscondensate/forecast-ledger/internal/storage"
)

func TestForecastTargetUsesExactClosedProjectionAndExcludedStatus(t *testing.T) {
	_, model := rootUpdateFixture(t, "individual-ledger.json")
	target, err := BuildForecastTarget(model, "q-election-coalition", "f-election-coalition-001")
	if err != nil {
		t.Fatal(err)
	}
	if target.SHA256 != "b360d603065bfcc064392cf364f1cc599650ff6e924a244427eca40e76e8f3bb" || target.Size != 421 {
		t.Fatalf("pinned target vector changed: sha256=%s size=%d", target.SHA256, target.Size)
	}
	parsed, err := document.ParseJSON(bytes.NewReader(target.Bytes), document.DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	root := parsed.Root.Any().(map[string]any)
	if len(root) != 3 || root["schema"] != ForecastEnvelopeSchema || root["question_id"] != "q-election-coalition" {
		t.Fatalf("target root = %#v", root)
	}
	forecast := root["forecast"].(map[string]any)
	if _, exists := forecast["integrity"]; exists {
		t.Fatal("target forecast contains integrity")
	}
	changed, err := cloneLedger(model)
	if err != nil {
		t.Fatal(err)
	}
	changed.Questions[1].Status = ledger.QuestionClosed
	changedTarget, err := BuildForecastTarget(changed, "q-election-coalition", "f-election-coalition-001")
	if err != nil || !bytes.Equal(target.Bytes, changedTarget.Bytes) {
		t.Fatalf("excluded status changed target: %v", err)
	}
	changed.Questions[1].Forecasts[0].RecordedAt = "2026-08-06T13:03:00+01:00"
	changedTarget, err = BuildForecastTarget(changed, "q-election-coalition", "f-election-coalition-001")
	if err != nil || bytes.Equal(target.Bytes, changedTarget.Bytes) {
		t.Fatalf("included forecast field did not change target: %v", err)
	}
}

func TestRevealedTargetContinuesOriginalSealedBytes(t *testing.T) {
	root, err := BuildLedgerRoot(InitRootRequest{LedgerID: "research", Timezone: "UTC", ForecasterID: "me", ForecasterName: "Me"}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	input := binaryInitialQuestion()
	input.InitialForecast.Visibility = ledger.VisibilitySealed
	rationale, comment := "private rationale", "private comment"
	factors := []string{"factor"}
	input.InitialForecast.Rationale, input.InitialForecast.Comment, input.InitialForecast.KeyFactors = &rationale, &comment, &factors
	build, err := BuildInitialSealedLedger(context.Background(), root, input, Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x23}, 76))}})
	if err != nil {
		t.Fatal(err)
	}
	sealedTarget, err := BuildForecastTarget(build.Ledger, "q-one", "f-one")
	if err != nil {
		t.Fatal(err)
	}
	revealed, err := cloneLedger(build.Ledger)
	if err != nil {
		t.Fatal(err)
	}
	forecast := &revealed.Questions[0].Forecasts[0]
	sealed := forecast.Commitment.Sealed
	forecast.Visibility = ledger.VisibilityRevealed
	forecast.Value = &input.InitialForecast.Value
	forecast.Rationale, forecast.Comment, forecast.KeyFactors = &rationale, &comment, &factors
	forecast.Commitment = &ledger.Commitment{Revealed: &ledger.RevealedCommitment{Scheme: sealed.Scheme, CommitmentHash: sealed.CommitmentHash, Encryption: sealed.Encryption, KeyHint: "different-hint", RevealedAt: "2026-02-01T00:00:00Z", RevealedKey: ledger.Hex32("2323232323232323232323232323232323232323232323232323232323232323")}}
	revealedTarget, err := BuildForecastTarget(revealed, "q-one", "f-one")
	if err != nil || !bytes.Equal(sealedTarget.Bytes, revealedTarget.Bytes) {
		t.Fatalf("revealed target changed: %v\nsealed=%s\nrevealed=%s", err, sealedTarget.Bytes, revealedTarget.Bytes)
	}
}

func TestTargetBuildCheckIdempotencyCollisionAndDryRun(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanTargetBuild(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001")
	if err != nil || len(plan.Targets) != 1 {
		t.Fatalf("plan = %#v, %v", plan, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "proofs")); !os.IsNotExist(err) {
		t.Fatalf("plan created proofs directory: %v", err)
	}
	created, err := CommitTargetBuild(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001")
	if err != nil || created.Targets[0].State != storage.DeterministicCreated {
		t.Fatalf("created = %#v, %v", created, err)
	}
	checked, err := CheckTargets(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001")
	if err != nil || checked.Targets[0].Valid == nil || !*checked.Targets[0].Valid {
		t.Fatalf("checked = %#v, %v", checked, err)
	}
	retried, err := CommitTargetBuild(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001")
	if err != nil || retried.Targets[0].State != storage.DeterministicUnchanged {
		t.Fatalf("retry = %#v, %v", retried, err)
	}
	targetPath := filepath.Join(directory, "proofs", "targets", "f-election-coalition-001.json")
	if err := os.WriteFile(targetPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	inspection, inspectErr := InspectTargets(context.Background(), path, true, "", "")
	if inspectErr != nil || inspection.FailureCode != app.CodeVerification || len(inspection.Targets) < 3 {
		t.Fatalf("tampered all inspection = %#v, %v", inspection, inspectErr)
	}
	var failedRows int
	for _, row := range inspection.Targets {
		if row.State == storage.DeterministicState(LayerFail) {
			failedRows++
		}
	}
	if failedRows != 1 {
		t.Fatalf("tampered all rows = %#v", inspection.Targets)
	}
	if _, err := CheckTargets(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001"); app.ErrorCodeOf(err) != app.CodeVerification {
		t.Fatalf("tampered check error = %v", err)
	}
	if _, err := CommitTargetBuild(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001"); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("tampered build error = %v", err)
	}
}

func TestTargetInspectionReportsUnretainedRowsAndKeepsLedgerOrder(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitTargetBuild(context.Background(), path, false, "q-election-coalition", "f-election-coalition-001"); err != nil {
		t.Fatal(err)
	}
	result, err := InspectTargets(context.Background(), path, true, "", "")
	if err != nil || result.FailureCode != "" || len(result.Targets) < 3 {
		t.Fatalf("inspection = %#v, %v", result, err)
	}
	if result.Targets[0].ForecastID != "f-central-bank-cut-001" || result.Targets[0].State != storage.DeterministicState(LayerNotApplicable) {
		t.Fatalf("first target = %#v", result.Targets[0])
	}
	var built, unretained *TargetResult
	for index := range result.Targets {
		if result.Targets[index].ForecastID == "f-election-coalition-001" {
			built = &result.Targets[index]
		}
		if result.Targets[index].State == storage.DeterministicState(LayerNotApplicable) {
			unretained = &result.Targets[index]
		}
	}
	if built == nil || built.State != storage.DeterministicState(LayerPass) {
		t.Fatalf("built target row = %#v", built)
	}
	if unretained == nil || unretained.ReasonCodes[0] != "content.no_retained_target" || unretained.Guidance == "" {
		t.Fatalf("unretained target = %#v", unretained)
	}
	if _, err := CheckTargets(context.Background(), path, false, "q-quarterly-revenue", "f-quarterly-revenue-001"); app.ErrorCodeOf(err) != app.CodeNotFound {
		t.Fatalf("strict target check error = %v", err)
	}
}

func TestTargetBuildAllPreflightsBeforeCreatingAnything(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "proofs", "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "proofs", "targets", "f-quarterly-revenue-001.json"), []byte("collision"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitTargetBuild(context.Background(), path, true, "", ""); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("all collision error = %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(directory, "proofs", "targets"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "f-quarterly-revenue-001.json" {
		t.Fatalf("all build created partial artifacts: %#v, %v", entries, err)
	}
}

func TestTargetBuildCancellationCreatesNothing(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CommitTargetBuild(ctx, path, false, "q-election-coalition", "f-election-coalition-001"); app.ErrorCodeOf(err) != app.CodeInterrupted {
		t.Fatalf("canceled target build error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "proofs")); !os.IsNotExist(err) {
		t.Fatalf("canceled target build created directory: %v", err)
	}
}
