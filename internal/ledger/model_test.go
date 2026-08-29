package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUnmarshalIndividualFixtureIntoTypedModel(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "schema", "testdata", "forecast-ledger", "v1.1.0", "individual-ledger.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got Ledger
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if got.SchemaVersion != "1.1.0" || got.LedgerID != "alex-example-forecasts" {
		t.Fatalf("unexpected ledger identity: %#v", got)
	}
	if len(got.Questions) != 4 {
		t.Fatalf("got %d questions, want 4", len(got.Questions))
	}
	if got.Questions[0].Forecasts[0].Value.Binary == nil {
		t.Fatal("binary forecast was not decoded as BinaryValue")
	}
	if got.Questions[1].Forecasts[0].Value.MultipleChoice == nil {
		t.Fatal("multiple-choice forecast was not decoded as MultipleChoiceValue")
	}
	if got.Questions[2].Forecasts[0].Value.Numeric == nil {
		t.Fatal("numeric forecast was not decoded as NumericValue")
	}
	if got.Questions[3].Forecasts[0].Value.Date == nil {
		t.Fatal("date forecast was not decoded as DateValue")
	}
	if got.Questions[0].Resolution.Resolved == nil || got.Questions[0].Resolution.Resolved.Outcome.Binary == nil {
		t.Fatal("binary resolution was not decoded into its typed state")
	}

	roundTrip, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal typed ledger: %v", err)
	}
	var again Ledger
	if err := json.Unmarshal(roundTrip, &again); err != nil {
		t.Fatalf("unmarshal round trip: %v", err)
	}
	if again.Questions[2].Forecasts[0].Value.Numeric == nil {
		t.Fatal("numeric union variant was lost in round trip")
	}
}

func TestUnmarshalRevealedCommitment(t *testing.T) {
	const input = `{
		"scheme":"forecast-seal/v1",
		"commitment_hash":{"algorithm":"sha-256","value":"379a52491319ec3a017819c09867504dae24a4a10a0e476108afc6cce4d9e391"},
		"encryption":{"algorithm":"chacha20-poly1305","nonce":"zMzMzMzMzMzMzMzM","ciphertext":"AAAAAAAAAAAAAAAAAAAAAAAA"},
		"key_hint":"secret-manager://example",
		"revealed_at":"2026-10-01T09:00:00+01:00",
		"revealed_key":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	}`
	var got Commitment
	if err := json.Unmarshal([]byte(input), &got); err != nil {
		t.Fatal(err)
	}
	if got.Revealed == nil || got.Sealed != nil {
		t.Fatalf("unexpected commitment variant: %#v", got)
	}
}

func TestUnionMarshalRejectsAmbiguousState(t *testing.T) {
	_, err := json.Marshal(ForecastValue{
		Binary:  &BinaryValue{Kind: ValueBinary, ProbabilityBP: 5000},
		Numeric: &NumericValue{Kind: ValueNumeric},
	})
	if err == nil {
		t.Fatal("expected ambiguous union state to fail")
	}
}
