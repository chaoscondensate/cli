package service

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/forecastcrypto"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	contractschema "github.com/chaoscondensate/forecast-ledger/internal/schema"
	"github.com/chaoscondensate/forecast-ledger/internal/storage"
)

func TestForecastSealRevealAndKeyHintLifecycle(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "f-election-coalition-002.key")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	input := SealedForecastInput{
		ForecastedAt: "2026-08-25T09:00:00+01:00", RecordedAt: timestampPointer("2026-08-25T09:01:00+01:00"),
		Value:     ledger.ForecastValue{MultipleChoice: &ledger.MultipleChoiceValue{Kind: ledger.ValueMultipleChoice, Probabilities: []ledger.ChoiceProbability{{OptionID: "centre-left", ProbabilityBP: 5000}, {OptionID: "centre-right", ProbabilityBP: 3500}, {OptionID: "other", ProbabilityBP: 1500}}}},
		Rationale: "PRIVATE-SEALED-RATIONALE", KeyFactors: []string{}, Comment: "PRIVATE-SEALED-COMMENT",
		SupersedesForecastID: slugPointer("f-election-coalition-001"),
	}
	effects := ProductionEffects()
	plan, err := PlanForecastSealFile(context.Background(), ledgerPath, keyPath, "q-election-coalition", "f-election-coalition-002", input, "2026-08-25T09:01:00+01:00")
	if err != nil || !plan.Changed {
		t.Fatalf("seal plan = %#v, %v", plan, err)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("seal plan created key: %v", err)
	}
	sealedResult, err := CommitForecastSealFile(context.Background(), ledgerPath, keyPath, "q-election-coalition", "f-election-coalition-002", input, "2026-08-25T09:01:00+01:00", effects)
	if err != nil || sealedResult.Recovery.State != RecoveryNone {
		t.Fatalf("seal result = %#v, %v", sealedResult, err)
	}
	sealedLedgerBytes, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{input.Rationale, input.Comment} {
		if bytes.Contains(sealedLedgerBytes, []byte(secret)) {
			t.Fatalf("sealed ledger leaked %q", secret)
		}
	}
	if _, err := CommitTargetBuild(context.Background(), ledgerPath, false, "q-election-coalition", "f-election-coalition-002"); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(directory, "proofs", "targets", "f-election-coalition-002.json")
	targetBefore, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	revealTime := ledger.Timestamp("2026-09-01T12:00:00Z")
	revealed, err := CommitForecastRevealFile(context.Background(), ledgerPath, keyPath, "q-election-coalition", "f-election-coalition-002", revealTime)
	if err != nil || !revealed.Changed {
		t.Fatalf("reveal = %#v, %v", revealed, err)
	}
	loaded, err := LoadAndValidateLedger(context.Background(), ledgerPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, forecast, err := selectForecast(loaded.Model, "q-election-coalition", "f-election-coalition-002")
	if err != nil || forecast.Visibility != ledger.VisibilityRevealed || forecast.Value == nil || forecast.Rationale == nil || *forecast.Rationale != input.Rationale || forecast.Commitment.Revealed == nil {
		t.Fatalf("revealed forecast = %#v, %v", forecast, err)
	}
	targetAfter, err := os.ReadFile(targetPath)
	if err != nil || !bytes.Equal(targetBefore, targetAfter) {
		t.Fatalf("reveal changed target bytes: %v", err)
	}
	if _, err := CheckTargets(context.Background(), ledgerPath, false, "q-election-coalition", "f-election-coalition-002"); err != nil {
		t.Fatal(err)
	}
	verification, err := VerifyLedgerEvidence(context.Background(), ledgerPath, VerificationOptions{Offline: true, QuestionID: "q-election-coalition", ForecastID: "f-election-coalition-002"})
	if err != nil || verification.Overall != VerificationPass || verification.Forecasts[0].Layers[2].State != LayerPass {
		t.Fatalf("revealed verification = %#v, %v", verification, err)
	}
	repeated, err := VerifyLedgerEvidence(context.Background(), ledgerPath, VerificationOptions{Offline: true, QuestionID: "q-election-coalition", ForecastID: "f-election-coalition-002"})
	if err != nil || !reflect.DeepEqual(verification, repeated) {
		t.Fatalf("verification output is not deterministic:\nfirst=%#v\nsecond=%#v\nerr=%v", verification, repeated, err)
	}
	beforeRetry, _ := os.ReadFile(ledgerPath)
	idempotent, err := CommitForecastRevealFile(context.Background(), ledgerPath, keyPath, "q-election-coalition", "f-election-coalition-002", "2026-10-03T00:00:00+01:00")
	if err != nil || idempotent.Changed {
		t.Fatalf("idempotent reveal = %#v, %v", idempotent, err)
	}
	afterRetry, _ := os.ReadFile(ledgerPath)
	if !bytes.Equal(beforeRetry, afterRetry) {
		t.Fatal("idempotent reveal changed ledger")
	}
	hint, err := CommitForecastKeyHintUpdateFile(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-002", "vault:item-42")
	if err != nil || !hint.Changed {
		t.Fatalf("key hint update = %#v, %v", hint, err)
	}
	if _, err := CheckTargets(context.Background(), ledgerPath, false, "q-election-coalition", "f-election-coalition-002"); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanForecastKeyHintUpdateFile(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-002", "file:secret.key"); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("file-like key hint error = %v", err)
	}
}

func TestForecastRevealWrongKeyPreservesLedger(t *testing.T) {
	root, err := BuildLedgerRoot(InitRootRequest{LedgerID: "research", Timezone: "UTC", ForecasterID: "me", ForecasterName: "Me"}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	input := binaryInitialQuestion()
	input.InitialForecast.Visibility = ledger.VisibilitySealed
	rationale, comment := "private", "private"
	factors := []string{}
	input.InitialForecast.Rationale, input.InitialForecast.Comment, input.InitialForecast.KeyFactors = &rationale, &comment, &factors
	build, err := BuildInitialSealedLedger(context.Background(), root, input, Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x42}, 76))}})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	wrongPath := filepath.Join(directory, "wrong.key")
	if _, err := CommitNewLedger(ledgerPath, build.Ledger); err != nil {
		t.Fatal(err)
	}
	wrongBytes, err := forecastcrypto.EncodeKeyFile("q-one", "f-one", bytes.Repeat([]byte{0x24}, 32))
	if err != nil || storage.CreateProtectedFile(wrongPath, wrongBytes) != nil {
		t.Fatalf("wrong key setup: %v", err)
	}
	before, _ := os.ReadFile(ledgerPath)
	if _, err := CommitForecastRevealFile(context.Background(), ledgerPath, wrongPath, "q-one", "f-one", "2026-02-01T00:00:00Z"); app.ErrorCodeOf(err) != app.CodeVerification {
		t.Fatalf("wrong key reveal error = %v", err)
	}
	after, _ := os.ReadFile(ledgerPath)
	if !bytes.Equal(before, after) {
		t.Fatal("wrong key reveal changed ledger")
	}
}
