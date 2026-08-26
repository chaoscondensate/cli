package service

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

func TestLayeredVerificationLocalPassAndTamperedTarget(t *testing.T) {
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
	report, err := VerifyLedgerEvidence(context.Background(), path, VerificationOptions{Offline: true, QuestionID: "q-election-coalition", ForecastID: "f-election-coalition-001"})
	if err != nil || report.Overall != VerificationPass || report.Forecasts[0].Layers[0].State != LayerPass || len(report.Limitations) != len(verificationLimitations) {
		t.Fatalf("report = %#v, %v", report, err)
	}
	targetPath := filepath.Join(directory, "proofs", "targets", "f-election-coalition-001.json")
	if err := os.WriteFile(targetPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err = VerifyLedgerEvidence(context.Background(), path, VerificationOptions{Offline: true, QuestionID: "q-election-coalition", ForecastID: "f-election-coalition-001"})
	if err != nil || report.Overall != VerificationFail || report.FailureCode != app.CodeVerification || report.Forecasts[0].Layers[0].State != LayerFail {
		t.Fatalf("tampered report = %#v, %v", report, err)
	}
}

func TestLayeredVerificationReportsPendingReceipt(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	_ = os.WriteFile(path, raw, 0o600)
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		identity := "https://alice.btc.calendar.opentimestamps.org"
		if request.URL.Host == "b.pool.opentimestamps.org" {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, _ := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	options := TimestampStampOptions{Effects: Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{2}, 16))}}, CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}}}
	if _, err := CommitTimestampStamp(context.Background(), path, "q-election-coalition", "f-election-coalition-001", options); err != nil {
		t.Fatal(err)
	}
	report, err := VerifyLedgerEvidence(context.Background(), path, VerificationOptions{Offline: true, QuestionID: "q-election-coalition", ForecastID: "f-election-coalition-001"})
	if err != nil || report.Overall != VerificationPending || report.FailureCode != app.CodePending || report.Forecasts[0].Layers[1].State != LayerPending {
		t.Fatalf("pending report = %#v, %v", report, err)
	}
}

func TestLayeredVerificationOfflineSourceCheckOpensNoNetwork(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: timestampRoundTripper(func(*http.Request) (*http.Response, error) {
		panic("offline verification attempted an HTTP request")
	})}
	report, err := VerifyLedgerEvidence(context.Background(), path, VerificationOptions{
		Offline: true, CheckSources: true, HTTPClient: client,
		QuestionID: "q-central-bank-cut", ForecastID: "f-central-bank-cut-002",
	})
	if err != nil || report.Overall != VerificationIncomplete || report.FailureCode != app.CodeIncomplete {
		t.Fatalf("offline source report = %#v, %v", report, err)
	}
	outcome := report.Forecasts[0].Layers[3]
	if outcome.State != LayerNotChecked || len(outcome.ReasonCodes) != 1 || outcome.ReasonCodes[0] != "outcome.offline" {
		t.Fatalf("offline outcome layer = %#v", outcome)
	}
}
