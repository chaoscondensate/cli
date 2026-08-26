package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/storage"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

func TestTimestampStampPendingStatusAndRetry(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		identity := "https://unknown.example"
		switch request.URL.Host {
		case "a.pool.opentimestamps.org":
			identity = "https://alice.btc.calendar.opentimestamps.org"
		case "b.pool.opentimestamps.org":
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, err := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	client := &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}}
	options := TimestampStampOptions{Effects: Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x42}, 16))}}, CalendarClient: client}
	result, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", options)
	if err != nil || result.State != TimestampPending || requests.Load() != 4 || len(result.CalendarSourceIDs) != 2 {
		t.Fatalf("stamp = %#v, requests=%d, err=%v", result, requests.Load(), err)
	}
	status, err := TimestampStatusFor(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001")
	if err != nil || status.State != TimestampPending || !status.TargetPresent || !status.ReceiptPresent {
		t.Fatalf("status = %#v, %v", status, err)
	}
	if _, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", options); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 4 {
		t.Fatalf("retry submitted duplicate calendar requests: %d", requests.Load())
	}
	if _, err := CommitTimestampVerify(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", TimestampVerifyOptions{}); app.ErrorCodeOf(err) != app.CodePending {
		t.Fatalf("pending verify error = %v", err)
	}
	loaded, err := LoadAndValidateLedger(context.Background(), ledgerPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, forecast, err := selectForecast(loaded.Model, "q-election-coalition", "f-election-coalition-001")
	if err != nil || forecast.Integrity.Pending == nil || len(forecast.Integrity.Pending.Timestamps) != 1 {
		t.Fatalf("pending ledger state = %#v, %v", forecast.Integrity, err)
	}
}

func TestAuthoringLifecyclePreservesRetainedTimestampEvidence(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		identity := "https://alice.btc.calendar.opentimestamps.org"
		if strings.Contains(request.URL.Host, "b.pool") {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, serializeErr := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		if serializeErr != nil {
			t.Fatal(serializeErr)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	options := TimestampStampOptions{
		Effects:        Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x63}, 16))}},
		CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}},
	}
	const questionID, stampedForecastID = "q-election-coalition", "f-election-coalition-001"
	stamped, err := CommitTimestampStamp(context.Background(), ledgerPath, questionID, stampedForecastID, options)
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(directory, filepath.FromSlash(string(stamped.TargetPath)))
	receiptPath := filepath.Join(directory, filepath.FromSlash(string(stamped.ReceiptPath)))
	targetBefore, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	receiptBefore, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}

	input := coalitionForecastInput()
	input.ForecastedAt = "2026-09-02T09:00:00+01:00"
	input.RecordedAt = timestampPointer("2026-09-02T09:01:00+01:00")
	prior := ledger.Slug(stampedForecastID)
	input.SupersedesForecastID = &prior
	if _, err := CommitPublicForecastAddFile(context.Background(), ledgerPath, questionID, "f-election-coalition-002", input, *input.RecordedAt); err != nil {
		t.Fatalf("append after timestamp: %v", err)
	}
	notes := Optional[string]{Set: true, Value: "Reviewed after the second forecast."}
	if _, err := CommitQuestionUpdateFile(context.Background(), ledgerPath, questionID, QuestionPatchInput{Notes: notes}); err != nil {
		t.Fatalf("metadata update after timestamp: %v", err)
	}
	closed := Optional[ledger.QuestionStatus]{Set: true, Value: ledger.QuestionClosed}
	if _, err := CommitQuestionUpdateFile(context.Background(), ledgerPath, questionID, QuestionPatchInput{Status: closed}); err != nil {
		t.Fatalf("close after timestamp: %v", err)
	}
	outcome := "centre-left"
	recordedAt := ledger.Timestamp("2026-10-15T12:01:00+01:00")
	resolution := ResolutionInput{
		Outcome: ResolutionOutcome{Text: &outcome}, OutcomeKnownAt: "2026-10-15T12:00:00+01:00", RecordedAt: &recordedAt,
		Sources: []EvidenceSourceInput{{Title: "Official result", URL: "https://example.org/result", RetrievedAt: "2026-10-15T12:00:30+01:00"}},
	}
	if _, err := CommitQuestionResolveFile(context.Background(), ledgerPath, questionID, resolution, recordedAt); err != nil {
		t.Fatalf("resolve after timestamp: %v", err)
	}

	targetAfter, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	receiptAfter, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(targetBefore, targetAfter) || !bytes.Equal(receiptBefore, receiptAfter) {
		t.Fatal("authoring changed retained target or receipt bytes")
	}
	loaded, err := LoadAndValidateLedger(context.Background(), ledgerPath, nil)
	if err != nil {
		t.Fatalf("final ledger with retained evidence is invalid: %v", err)
	}
	_, question, err := selectQuestion(loaded.Model, questionID)
	if err != nil || question.Status != ledger.QuestionResolved || len(question.Forecasts) != 2 {
		t.Fatalf("final lifecycle state = %#v, %v", question, err)
	}
}

func TestTimestampDryRunAndOfflineHaveNoEffects(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), raw...)
	plan, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", TimestampStampOptions{DryRun: true})
	if err != nil || len(plan.Effects) == 0 {
		t.Fatalf("dry run = %#v, %v", plan, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "proofs")); !os.IsNotExist(err) {
		t.Fatalf("dry run created artifacts: %v", err)
	}
	var requests atomic.Int32
	spyClient := &ots.CalendarClient{HTTPClient: &http.Client{Transport: timestampRoundTripper(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, context.Canceled
	})}}
	if _, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", TimestampStampOptions{Offline: true, CalendarClient: spyClient}); app.ErrorCodeOf(err) != app.CodeNetworkDisabled {
		t.Fatalf("offline stamp error = %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("offline stamp opened %d network requests", requests.Load())
	}
	after, _ := os.ReadFile(ledgerPath)
	if !bytes.Equal(before, after) {
		t.Fatal("dry-run/offline stamp changed ledger")
	}
}

func TestTimestampStatusDetectsMissingReceipt(t *testing.T) {
	// Reuse the first test setup through a minimal pending mutation and then
	// remove the receipt to assert the documented inconsistent state.
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	_ = os.WriteFile(ledgerPath, raw, 0o600)
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		identity := "https://alice.btc.calendar.opentimestamps.org"
		if strings.Contains(request.URL.Host, "b.pool") {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, _ := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	options := TimestampStampOptions{Effects: Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{1}, 16))}}, CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}}}
	if _, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", options); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(directory, "proofs", "receipts", "f-election-coalition-001.json.ots")); err != nil {
		t.Fatal(err)
	}
	status, err := TimestampStatusFor(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001")
	if app.ErrorCodeOf(err) != app.CodeVerification || status.State != TimestampInconsistent {
		t.Fatalf("missing receipt status = %#v, %v", status, err)
	}
}

func TestTimestampStampRecoveryReusesRetainedArtifacts(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		identity := "https://alice.btc.calendar.opentimestamps.org"
		if strings.Contains(request.URL.Host, "b.pool") {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, _ := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	options := TimestampStampOptions{Effects: Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{5}, 16))}}, CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}}}
	lock, err := storage.AcquireLedgerLock(context.Background(), ledgerPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	result, stampErr := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", options)
	if releaseErr := lock.Release(); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if app.ErrorCodeOf(stampErr) != app.CodeConflict || result.Recovery.State != RecoveryRetained || requests.Load() != 4 {
		t.Fatalf("retained recovery = %#v, requests=%d, err=%v", result, requests.Load(), stampErr)
	}
	if _, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", options); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 4 {
		t.Fatalf("recovery retry repeated calendar submission: %d", requests.Load())
	}
}

func TestBitcoinCoreCredentialErrorsDoNotExposeSecrets(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "bitcoin-auth.json")
	const canary = "TOP-SECRET-BITCOIN-PASSWORD"
	if err := storage.CreateProtectedFile(secretPath, []byte(`{"username":"rpc-user","password":"`+canary+`"}`)); err != nil {
		t.Fatal(err)
	}
	_, err := ProtectedCoreObserver("not-a-url", secretPath)
	if err == nil || strings.Contains(err.Error(), canary) {
		t.Fatalf("Bitcoin Core configuration error exposed credentials: %v", err)
	}
}

func TestTimestampUpgradeVerifyAndVerifiedStatus(t *testing.T) {
	const questionID, forecastID = "q-central-bank-cut", "f-central-bank-cut-002"
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		var branch []byte
		if request.Method == http.MethodPost {
			identity := "https://alice.btc.calendar.opentimestamps.org"
			if strings.Contains(request.URL.Host, "b.pool") {
				identity = "https://bob.btc.calendar.opentimestamps.org"
			}
			branch, _ = ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		} else {
			branch, _ = ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationBitcoin, Height: 1}}}})
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	client := &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}}
	stamp := TimestampStampOptions{Effects: Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{4}, 16))}}, CalendarClient: client}
	if _, err := CommitTimestampStamp(context.Background(), ledgerPath, questionID, forecastID, stamp); err != nil {
		t.Fatal(err)
	}
	upgraded, err := CommitTimestampUpgrade(context.Background(), ledgerPath, questionID, forecastID, TimestampUpgradeOptions{CalendarClient: client})
	if err != nil || upgraded.State != TimestampConfirmedUnverified || upgraded.BitcoinHeight == nil {
		t.Fatalf("upgrade = %#v, %v", upgraded, err)
	}
	receiptBytes, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(string(upgraded.ReceiptPath))))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := ots.ParseReceipt(receiptBytes)
	if err != nil {
		t.Fatal(err)
	}
	evaluated, err := receipt.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	var bitcoin ots.EvaluatedAttestation
	for _, item := range evaluated {
		if item.Attestation.Kind == ots.AttestationBitcoin {
			bitcoin = item
			break
		}
	}
	if bitcoin.Attestation.Height == 0 {
		t.Fatal("upgraded receipt has no Bitcoin attestation")
	}
	observation := mineTestObservation(t, bitcoin)
	verifiedAt := ledger.Timestamp("2026-01-02T03:04:05Z")
	verified, err := CommitTimestampVerify(context.Background(), ledgerPath, questionID, forecastID, TimestampVerifyOptions{VerifiedAt: verifiedAt, Observer: fixedBitcoinObserver{observation: observation}})
	if err != nil || verified.State != TimestampVerified || verified.BitcoinHeight == nil || verified.AnchoredBefore == nil {
		t.Fatalf("verify = %#v, %v", verified, err)
	}
	if len(verified.Warnings) != 1 || verified.Warnings[0].Code != "timestamp.valid_but_too_late" {
		t.Fatalf("late timestamp warning = %#v", verified.Warnings)
	}
	status, err := TimestampStatusFor(context.Background(), ledgerPath, questionID, forecastID)
	if err != nil || status.State != TimestampVerified || status.BitcoinHeight == nil || *status.BitcoinHeight != 1 {
		t.Fatalf("verified status = %#v, %v", status, err)
	}
	report, err := VerifyLedgerEvidence(context.Background(), ledgerPath, VerificationOptions{Offline: true, QuestionID: questionID, ForecastID: forecastID})
	if err != nil || report.Overall != VerificationFail || report.Forecasts[0].Layers[1].State != LayerFail || report.Forecasts[0].Layers[1].ReasonCodes[0] != "timing.not_before_outcome" {
		t.Fatalf("late offline verification = %#v, %v", report, err)
	}
}

type fixedBitcoinObserver struct {
	observation ots.BlockObservation
}

func (observer fixedBitcoinObserver) Observe(context.Context, uint64) (ots.BlockObservation, error) {
	return observer.observation, nil
}

func (fixedBitcoinObserver) Summary() ots.RequestSummary {
	return ots.RequestSummary{UniqueHeights: 1, HTTPRequests: 0, MaxHeights: 1, MaxRequests: 1, MaxConcurrent: 1}
}

func mineTestObservation(t *testing.T, attestation ots.EvaluatedAttestation) ots.BlockObservation {
	t.Helper()
	if len(attestation.Message) != 32 {
		t.Fatalf("Bitcoin proof message length = %d", len(attestation.Message))
	}
	header := make([]byte, 80)
	binary.LittleEndian.PutUint32(header[0:4], 1)
	copy(header[36:68], attestation.Message)
	blockTime := time.Date(2027, time.January, 2, 3, 4, 5, 0, time.UTC)
	binary.LittleEndian.PutUint32(header[68:72], uint32(blockTime.Unix()))
	binary.LittleEndian.PutUint32(header[72:76], 0x207fffff)
	for nonce := uint32(0); ; nonce++ {
		binary.LittleEndian.PutUint32(header[76:80], nonce)
		first := sha256.Sum256(header)
		second := sha256.Sum256(first[:])
		reversed := make([]byte, len(second))
		for index := range second {
			reversed[index] = second[len(second)-1-index]
		}
		observation := ots.BlockObservation{
			Height: attestation.Attestation.Height, Hash: hex.EncodeToString(reversed), HeaderHex: hex.EncodeToString(header),
			MerkleRoot: hex.EncodeToString(header[36:68]), BlockTime: blockTime, SourceIDs: []string{"test-bitcoin"},
		}
		if ots.VerifyBitcoinAttestation(attestation, observation) == nil {
			return observation
		}
	}
}

type timestampRoundTripper func(*http.Request) (*http.Response, error)

func (function timestampRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
