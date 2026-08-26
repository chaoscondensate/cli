package service

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestPublicForecastAppendEnforcesOrderIdentityAndSupersession(t *testing.T) {
	_, model := rootUpdateFixture(t, "individual-ledger.json")
	input := coalitionForecastInput()
	input.SupersedesForecastID = slugPointer("f-election-coalition-001")
	mutation, err := BuildPublicForecastAppend(model, "q-election-coalition", "f-election-coalition-002", input, "2026-09-01T09:01:00+01:00")
	if err != nil {
		t.Fatal(err)
	}
	question, _, _ := selectQuestion(mutation.Ledger, "q-election-coalition")
	appended := mutation.Ledger.Questions[question].Forecasts[1]
	if appended.Visibility != ledger.VisibilityPublic || appended.Integrity.Unanchored == nil || appended.RecordedAt != "2026-09-01T09:01:00+01:00" {
		t.Fatalf("appended forecast = %#v", appended)
	}
	if _, err := BuildPublicForecastAppend(model, "q-election-coalition", "f-central-bank-cut-001", input, "2026-09-01T09:01:00+01:00"); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("global duplicate error = %v", err)
	}
	wrongQuestion := slugPointer("f-quarterly-revenue-001")
	input.SupersedesForecastID = wrongQuestion
	if _, err := BuildPublicForecastAppend(model, "q-election-coalition", "f-election-coalition-002", input, "2026-09-01T09:01:00+01:00"); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("cross-question supersession error = %v", err)
	}
	input.SupersedesForecastID = nil
	input.RecordedAt = timestampPointer("2026-08-01T00:00:00+01:00")
	if _, err := BuildPublicForecastAppend(model, "q-election-coalition", "f-election-coalition-002", input, "2026-09-01T09:01:00+01:00"); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("out-of-order error = %v", err)
	}
	input.RecordedAt = timestampPointer("2026-09-01T09:01:00+01:00")
	input.ForecastedAt = "2026-11-01T09:00:00+00:00"
	if _, err := BuildPublicForecastAppend(model, "q-election-coalition", "f-election-coalition-002", input, "2026-09-01T09:01:00+01:00"); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("outside-window error = %v", err)
	}
}

func TestForecastListPreservesHistoryAndShowRedactsSealedSecrets(t *testing.T) {
	_, model := rootUpdateFixture(t, "individual-ledger.json")
	items, err := ListForecasts(model, "q-central-bank-cut")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "f-central-bank-cut-001" || items[1].ID != "f-central-bank-cut-002" {
		t.Fatalf("list = %#v", items)
	}
	public, err := ShowForecast(model, "q-election-coalition", "f-election-coalition-001")
	if err != nil || public.Value == nil {
		t.Fatalf("public show = %#v, %v", public, err)
	}
	sealed := model.Questions[1].Forecasts[0]
	sealed.Visibility = ledger.VisibilitySealed
	sealed.Value = nil
	sealed.Rationale = nil
	sealed.KeyFactors = nil
	sealed.Comment = nil
	sealed.Commitment = &ledger.Commitment{Sealed: &ledger.SealedCommitment{
		Scheme: "forecast-seal/v1", CommitmentHash: ledger.Digest{Algorithm: "sha256", Value: ledger.Hex32("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")},
		Encryption: ledger.Encryption{Algorithm: "chacha20-poly1305", Nonce: ledger.Base64Nonce12("AAAAAAAAAAAAAAAA"), Ciphertext: ledger.Base64Ciphertext("PRIVATE-CANARY")}, KeyHint: "forecast-key:f-one",
	}}
	model.Questions[1].Forecasts[0] = sealed
	view, err := ShowForecast(model, model.Questions[1].ID, sealed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Value != nil || view.Rationale != nil || view.Commitment == nil || view.Commitment.Encryption.Ciphertext != "" || view.Commitment.Encryption.Nonce != "" {
		t.Fatalf("sealed view leaked private fields: %#v", view)
	}
}

func TestForecastFileMutationIsMinimalAndStdinReadsWork(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	input := coalitionForecastInput()
	input.RecordedAt = timestampPointer("2026-09-01T09:01:00+01:00")
	result, err := CommitPublicForecastAddFile(context.Background(), path, "q-election-coalition", "f-election-coalition-002", input, "2026-09-01T09:01:00+01:00")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || len(result.ChangedPointers) != 1 || result.ChangedPointers[0] != "/questions/1/forecasts/-" {
		t.Fatalf("result = %#v", result)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(updated, []byte(`"id":"f-election-coalition-002"`)) {
		t.Fatal("appended forecast is missing")
	}
	ledgerID, items, err := LoadForecastList(context.Background(), "-", bytes.NewReader(updated), "q-election-coalition")
	if err != nil || ledgerID == "" || len(items) != 2 {
		t.Fatalf("stdin list = %q %#v, %v", ledgerID, items, err)
	}
	_, shown, err := LoadForecastShow(context.Background(), "-", bytes.NewReader(updated), "q-election-coalition", "f-election-coalition-002")
	if err != nil || shown.Value == nil {
		t.Fatalf("stdin show = %#v, %v", shown, err)
	}
}

func coalitionForecastInput() ForecastCreateInput {
	return ForecastCreateInput{
		ForecastedAt: "2026-09-01T09:00:00+01:00",
		Value: ledger.ForecastValue{MultipleChoice: &ledger.MultipleChoiceValue{Kind: ledger.ValueMultipleChoice, Probabilities: []ledger.ChoiceProbability{
			{OptionID: "centre-left", ProbabilityBP: 5000}, {OptionID: "centre-right", ProbabilityBP: 3500}, {OptionID: "other", ProbabilityBP: 1500},
		}}},
	}
}

func slugPointer(value ledger.Slug) *ledger.Slug { return &value }

func timestampPointer(value ledger.Timestamp) *ledger.Timestamp { return &value }
