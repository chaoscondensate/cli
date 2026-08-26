package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
)

func TestUpstreamFixturesPassSemanticValidation(t *testing.T) {
	for _, name := range []string{"individual-ledger.json", "team-ledger.yaml"} {
		t.Run(name, func(t *testing.T) {
			model := loadValidLedger(t, name)
			issues, err := ValidateSemantics(model, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(issues) != 0 {
				t.Fatalf("valid fixture has semantic issues: %#v", issues)
			}
		})
	}
}

func TestSemanticValidationChecksIDsReferencesChronologyAndValues(t *testing.T) {
	model := loadValidLedger(t, "individual-ledger.json")
	model.DefaultTimezone = "Not/A-Timezone"
	model.Questions[1].ID = model.Questions[0].ID
	unknown := ledger.Slug("missing")
	model.Questions[1].PlatformRefs = &[]ledger.PlatformRef{{Platform: unknown}}
	model.Questions[1].Forecasts[0].RecordedAt = "2026-08-01T00:00:00Z"
	model.Questions[1].Forecasts[0].SupersedesForecastID = &unknown
	model.Questions[1].Forecasts[0].Value.MultipleChoice.Probabilities[0].ProbabilityBP = 1
	model.Questions[2].Forecasts[0].Value.Numeric.Interval.Lower = "1000"
	model.Questions[2].Forecasts[0].Value.Numeric.Interval.Upper = "1"
	(*model.Questions[3].Forecasts[0].Value.Date.Quantiles)[0].ProbabilityBP = 9000

	issues, err := ValidateSemantics(model, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"semantic.timezone",
		"semantic.duplicate_question_id",
		"semantic.unknown_platform",
		"semantic.forecast_chronology",
		"semantic.supersedes",
		"semantic.probability_sum",
		"semantic.interval",
		"semantic.quantile_order",
	} {
		if !hasSemanticCode(issues, code) {
			t.Errorf("missing %s in %#v", code, issues)
		}
	}
}

func TestSemanticValidationChecksArtifactDigest(t *testing.T) {
	model := loadValidLedger(t, "individual-ledger.json")
	data := []byte("target bytes")
	digest := sha256.Sum256(data)
	model.Questions[0].Forecasts[0].Integrity = ledger.Integrity{Pending: &ledger.PendingIntegrity{
		Status: ledger.IntegrityPending,
		Target: ledger.ForecastTarget{
			Scope:            "forecast-envelope/v1",
			Canonicalization: "RFC8785",
			ArtifactPath:     "targets/one.json",
			Digest:           ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(hex.EncodeToString(digest[:]))},
		},
		Timestamps: []ledger.OTSTimestamp{{Type: "opentimestamps", ProofPath: "proofs/one.ots", State: ledger.OTSPending}},
	}}
	issues, err := ValidateSemantics(model, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hasSemanticCode(issues, "semantic.artifact_unavailable") || hasSemanticCode(issues, "semantic.artifact_missing") {
		t.Fatalf("model-only validation treated unavailable artifacts as invalid: %#v", issues)
	}
	artifacts := fstest.MapFS{"targets/one.json": &fstest.MapFile{Data: data}}
	issues, err = ValidateSemantics(model, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if hasSemanticCode(issues, "semantic.artifact_digest") || hasSemanticCode(issues, "semantic.artifact_missing") {
		t.Fatalf("matching artifact rejected: %#v", issues)
	}

	artifacts["targets/one.json"] = &fstest.MapFile{Data: []byte("tampered")}
	issues, err = ValidateSemantics(model, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSemanticCode(issues, "semantic.artifact_digest") {
		t.Fatalf("digest mismatch not reported: %#v", issues)
	}
}

func TestSemanticValidationChecksRevealedMirrorWithoutRejectingLateTimestamp(t *testing.T) {
	model := loadValidLedger(t, "team-ledger.yaml")
	forecast := &model.Questions[0].Forecasts[0]
	tampered := "changed after reveal"
	forecast.Rationale = &tampered
	forecast.Integrity = ledger.Integrity{Verified: &ledger.VerifiedIntegrity{
		Status: ledger.IntegrityVerified,
		Target: ledger.ForecastTarget{
			Scope: "forecast-envelope/v1", Canonicalization: "RFC8785", ArtifactPath: "target.json",
			Digest: ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(strings.Repeat("0", 64))},
		},
		Timestamps: []ledger.OTSTimestamp{{
			Type: "opentimestamps", ProofPath: "target.json.ots", State: ledger.OTSConfirmed,
			AnchoredBefore: timestampPointer("2026-09-30T15:00:00+01:00"), BitcoinBlockHeight: int64Pointer(1),
		}},
		VerifiedAt: "2026-10-01T09:00:00+01:00",
	}}
	issues, err := ValidateSemantics(model, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasSemanticCode(issues, "semantic.revealed_mirror") {
		t.Fatalf("tampered reveal mirror not reported: %#v", issues)
	}
	if hasSemanticCode(issues, "semantic.late_timestamp") {
		t.Fatalf("cryptographically valid late timestamp made the ledger invalid: %#v", issues)
	}
}

func loadValidLedger(t *testing.T, name string) *ledger.Ledger {
	t.Helper()
	data, err := fs.ReadFile(contractschema.Conformance(), name)
	if err != nil {
		t.Fatal(err)
	}
	var parsed *document.Document
	if strings.HasSuffix(name, ".json") {
		parsed, err = document.ParseJSON(strings.NewReader(string(data)), document.DefaultLimits)
	} else {
		parsed, err = document.ParseYAML(strings.NewReader(string(data)), document.DefaultLimits)
	}
	if err != nil {
		t.Fatal(err)
	}
	structural, err := DefaultStructuralValidator()
	if err != nil {
		t.Fatal(err)
	}
	issues, err := structural.Validate(parsed.Root.Any())
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("fixture is not structurally valid: %#v", issues)
	}
	model, err := DecodeLedger(parsed)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func hasSemanticCode(issues []SemanticIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func timestampPointer(value ledger.Timestamp) *ledger.Timestamp { return &value }
func int64Pointer(value int64) *int64                           { return &value }
