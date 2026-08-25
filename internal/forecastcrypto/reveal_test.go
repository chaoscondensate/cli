package forecastcrypto

import (
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestRevealMatchesPinnedSealVector(t *testing.T) {
	data, err := fs.ReadFile(contractschema.Conformance(), "forecast-seal-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		QuestionID string `json:"question_id"`
		ForecastID string `json:"forecast_id"`
		Material   struct {
			KeyHex string `json:"key_hex"`
		} `json:"material"`
		Expected struct {
			CanonicalPlaintext string `json:"canonical_plaintext"`
			Commitment         struct {
				Scheme         string            `json:"scheme"`
				CommitmentHash ledger.Digest     `json:"commitment_hash"`
				Encryption     ledger.Encryption `json:"encryption"`
				KeyHint        string            `json:"key_hint"`
			} `json:"commitment"`
		} `json:"expected"`
	}
	if err := json.Unmarshal(data, &vector); err != nil {
		t.Fatal(err)
	}
	commitment := ledger.RevealedCommitment{
		Scheme:         vector.Expected.Commitment.Scheme,
		CommitmentHash: vector.Expected.Commitment.CommitmentHash,
		Encryption:     vector.Expected.Commitment.Encryption,
		KeyHint:        vector.Expected.Commitment.KeyHint,
		RevealedKey:    ledger.Hex32(vector.Material.KeyHex),
	}
	payload, err := Reveal(ledger.Slug(vector.QuestionID), ledger.Slug(vector.ForecastID), commitment)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload.Plaintext) != vector.Expected.CanonicalPlaintext {
		t.Fatalf("plaintext differs:\n%s", payload.Plaintext)
	}
	if payload.Bundle["rationale"] == nil {
		t.Fatal("bundle was not returned")
	}
}

func TestRevealRejectsWrongBindingWithoutLeakingKey(t *testing.T) {
	commitment := ledger.RevealedCommitment{RevealedKey: ledger.Hex32("secret-not-hex")}
	_, err := Reveal("question", "forecast", commitment)
	if err == nil {
		t.Fatal("invalid key accepted")
	}
	if got := err.Error(); got == "" || got == "secret-not-hex" {
		t.Fatalf("unsafe error: %q", got)
	}
}
