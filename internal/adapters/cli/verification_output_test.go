package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/presentation"
	"github.com/chaoscondensate/cli/internal/service"
	"github.com/chaoscondensate/cli/internal/storage"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

func TestPendingVerificationKeepsExitNineInEveryOutputMode(t *testing.T) {
	path := pendingVerificationLedger(t)
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "plain-default", args: nil, want: []string{"overall\tpending", "document\tpass", "content_binding", "existence_timing"}},
		{name: "plain", args: []string{"--plain"}, want: []string{"overall\tpending", "document\tpass", "content_binding", "existence_timing"}},
		{name: "json-before", args: []string{"--json"}, want: []string{`"code":"verification.pending"`, `"overall":"pending"`, `"layers"`}},
		{name: "json-after", args: []string{"verify", "--json"}, want: []string{`"code":"verification.pending"`, `"overall":"pending"`}},
		{name: "quiet", args: []string{"--quiet"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			arguments := []string{"forecast-ledger"}
			if len(test.args) > 0 && test.args[0] == "verify" {
				arguments = append(arguments, test.args...)
				arguments = append(arguments, "--file", path, "--offline")
			} else {
				arguments = append(arguments, test.args...)
				arguments = append(arguments, "verify", "--file", path, "--offline")
			}
			code, stdout, stderr := runCLI(arguments...)
			if code != 9 || stderr != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if test.name == "quiet" && stdout != "" {
				t.Fatalf("quiet output = %q", stdout)
			}
			for _, expected := range test.want {
				if !strings.Contains(stdout, expected) {
					t.Errorf("output missing %q:\n%s", expected, stdout)
				}
			}
		})
	}
}

func TestHumanVerificationFormatterIncludesCompleteMatrix(t *testing.T) {
	path := pendingVerificationLedger(t)
	report, err := service.VerifyLedgerEvidence(context.Background(), path, service.VerificationOptions{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	output := formatVerificationReport(presentation.ModeHuman, report)
	for _, expected := range []string{"Overall: pending", "Document: pass", "Forecast: q-election-coalition / f-election-coalition-001", "content_binding: pass", "existence_timing: pending"} {
		if !strings.Contains(output, expected) {
			t.Errorf("human output missing %q:\n%s", expected, output)
		}
	}
}

func TestStoredTimingEvidenceIsVisibleInHumanAndPlainOutput(t *testing.T) {
	report := service.VerificationReport{Overall: service.VerificationPass, Document: service.VerificationLayer{Name: "document", State: service.LayerPass}, Forecasts: []service.ForecastVerification{{
		QuestionID: "q-one", ForecastID: "f-one", Layers: []service.VerificationLayer{{Name: "existence_timing", State: service.LayerPass, ReasonCodes: []string{"timing.stored_verification_consistent"}, Evidence: map[string]any{
			"bitcoin_block_height": uint64(800000), "anchored_before": "2026-08-01T00:00:00Z", "verified_at": "2026-08-02T00:00:00Z", "evidence_source": "stored_verification", "freshly_checked": false,
		}}},
	}}}
	for _, mode := range []presentation.Mode{presentation.ModeHuman, presentation.ModePlain} {
		output := formatVerificationReport(mode, report)
		for _, expected := range []string{"800000", "2026-08-01T00:00:00Z", "2026-08-02T00:00:00Z", "stored_verification", "freshly_checked"} {
			if !strings.Contains(output, expected) {
				t.Errorf("%s output missing %q: %s", mode, expected, output)
			}
		}
	}
	anchored := ledger.Timestamp("2026-08-01T00:00:00Z")
	height := int64(800000)
	verifiedAt := ledger.Timestamp("2026-08-02T00:00:00Z")
	fresh, retained := false, false
	view := service.ForecastView{Summary: service.ForecastSummary{ID: "f-one", ForecastedAt: "2026-07-01T00:00:00Z", RecordedAt: "2026-07-01T00:01:00Z", Visibility: ledger.VisibilityPublic, IntegrityStatus: ledger.IntegrityVerified}, Integrity: service.ForecastIntegrityView{
		Status: ledger.IntegrityVerified, Timestamps: []ledger.OTSTimestamp{{Type: "opentimestamps", ProofPath: "proofs/receipts/f-one.json.ots", State: ledger.OTSConfirmed, AnchoredBefore: &anchored, BitcoinBlockHeight: &height}}, VerifiedAt: &verifiedAt, EvidenceSource: "stored_verification", FreshlyChecked: &fresh, PriorSourceRetained: &retained,
	}}
	formatted := formatForecastView(presentation.ModePlain, view)
	for _, expected := range []string{"proofs/receipts/f-one.json.ots", "800000", "2026-08-01T00:00:00Z", "2026-08-02T00:00:00Z", "stored_verification"} {
		if !strings.Contains(formatted, expected) {
			t.Errorf("forecast output missing %q: %s", expected, formatted)
		}
	}
}

func TestPendingPublicationVerificationKeepsExitNineAndMatrix(t *testing.T) {
	path := pendingVerificationLedger(t)
	output := filepath.Join(t.TempDir(), "package")
	built, err := service.CommitPublicationBuild(context.Background(), path, output, false)
	if err != nil {
		t.Fatal(err)
	}
	packageLedger := filepath.Join(output, filepath.FromSlash(built.LedgerPath))
	manifest := filepath.Join(output, filepath.FromSlash(built.ManifestPath))
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{name: "plain", args: []string{"--plain"}, want: []string{"overall\tpending", "manifest\t", "file\t", "existence_timing\tpending"}},
		{name: "json", args: []string{"--json"}, want: []string{`"code":"publication.verification.pending"`, `"overall":"pending"`, `"evidence"`}},
		{name: "quiet", args: []string{"--quiet"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			arguments := append([]string{"forecast-ledger"}, test.args...)
			arguments = append(arguments, "publish", "verify", "--file", packageLedger, "--manifest", manifest)
			code, stdout, stderr := runCLI(arguments...)
			if code != 9 || stderr != "" {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if test.name == "quiet" && stdout != "" {
				t.Fatalf("quiet output = %q", stdout)
			}
			for _, expected := range test.want {
				if !strings.Contains(stdout, expected) {
					t.Errorf("output missing %q:\n%s", expected, stdout)
				}
			}
		})
	}
	result, err := service.VerifyPublicationPackage(context.Background(), packageLedger, manifest, service.PublicationVerifyOptions{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	human := formatPublicationVerification(presentation.ModeHuman, result)
	for _, expected := range []string{"Overall: pending", "Manifest SHA-256:", "File:", "existence_timing: pending"} {
		if !strings.Contains(human, expected) {
			t.Errorf("human package output missing %q:\n%s", expected, human)
		}
	}
}

func TestVerificationOutputWriterFailureIsInternal(t *testing.T) {
	path := pendingVerificationLedger(t)
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"forecast-ledger", "verify", "--file", path, "--offline"}, strings.NewReader(""), failingWriter{}, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "internal error") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestTimestampVerifySourceOutageReturnsSafeReport(t *testing.T) {
	path := confirmedVerificationLedger(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "REMOTE-PRIVATE-BODY", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	authPath := filepath.Join(t.TempDir(), "bitcoin-auth.json")
	const secret = "CLI-BITCOIN-SECRET"
	if err := storage.CreateProtectedFile(authPath, []byte(`{"username":"rpc-user","password":"`+secret+`"}`)); err != nil {
		t.Fatal(err)
	}
	arguments := []string{"forecast-ledger", "--json", "timestamp", "verify", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-001", "--bitcoin-core", server.URL, "--bitcoin-auth-file", authPath}
	code, stdout, stderr := runCLI(arguments...)
	if code != 8 || stderr != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, expected := range []string{`"code":"timestamp.verification.not_checked"`, `"state":"confirmed_unverified"`, `"state":"not_checked"`, `"timing.source_unavailable"`, `"kind":"source_unavailable"`, `"source_ids":["bitcoin-core"]`, `"mode":"bitcoin_core"`, `"http_requests":1`} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("outage JSON missing %q:\n%s", expected, stdout)
		}
	}
	for _, forbidden := range []string{"Bitcoin evidence did not verify", server.URL, secret, "REMOTE-PRIVATE-BODY"} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Errorf("outage output leaked or retained obsolete message %q: %s%s", forbidden, stdout, stderr)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("outage changed ledger: err=%v", err)
	}

	plainArguments := append([]string(nil), arguments...)
	plainArguments[1] = "--plain"
	code, stdout, stderr = runCLI(plainArguments...)
	if code != 8 || stderr != "" || !strings.Contains(stdout, "verification\tnot_checked\ttiming.source_unavailable") || !strings.Contains(stdout, "sources\tbitcoin-core") {
		t.Fatalf("plain outage code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestNoEvidenceUsesExitNineForLedgerAndPackage(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "empty.json")
	if err := os.WriteFile(ledgerPath, fixtureBytes(t, "empty-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "verify", "--file", ledgerPath, "--offline")
	if code != 9 || stderr != "" || !strings.Contains(stdout, `"code":"verification.no_evidence"`) || !strings.Contains(stdout, `"overall":"no_evidence"`) {
		t.Fatalf("empty ledger code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--plain", "verify", "--file", ledgerPath, "--offline")
	if code != 9 || stderr != "" || !strings.Contains(stdout, "overall\tno_evidence") || !strings.Contains(stdout, "document\tpass") {
		t.Fatalf("plain empty ledger code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	packageRoot := filepath.Join(directory, "package")
	if _, err := service.CommitPublicationBuild(context.Background(), ledgerPath, packageRoot, false); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "publish", "verify", "--file", filepath.Join(packageRoot, "ledger", "empty.json"), "--manifest", filepath.Join(packageRoot, "manifest.json"))
	if code != 9 || stderr != "" || !strings.Contains(stdout, `"code":"publication.verification.no_evidence"`) || !strings.Contains(stdout, `"overall":"no_evidence"`) || !strings.Contains(stdout, `"files"`) {
		t.Fatalf("empty package code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func pendingVerificationLedger(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		identity := "https://alice.btc.calendar.opentimestamps.org"
		if strings.Contains(request.URL.Host, "b.pool") {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, err := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	_, err := service.CommitTimestampStamp(context.Background(), path, "q-election-coalition", "f-election-coalition-001", service.TimestampStampOptions{
		CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func confirmedVerificationLedger(t *testing.T) string {
	t.Helper()
	path := pendingVerificationLedger(t)
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		branch, err := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationBitcoin, Height: 1}}}})
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	_, err := service.CommitTimestampUpgrade(context.Background(), path, "q-election-coalition", "f-election-coalition-001", service.TimestampUpgradeOptions{CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}}})
	if err != nil {
		t.Fatal(err)
	}
	return path
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
