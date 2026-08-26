package service

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chaoscondensate/cli/internal/app"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

func TestPublicationBuildVerifyDeterministicAndRejectExtraFile(t *testing.T) {
	raw, err := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "adjacent.key"), []byte("PRIVATE-ADJACENT-CANARY"), 0o600); err != nil {
		t.Fatal(err)
	}
	firstRoot, secondRoot := filepath.Join(directory, "package-one"), filepath.Join(directory, "package-two")
	first, err := CommitPublicationBuild(context.Background(), ledgerPath, firstRoot, false)
	if err != nil || first.FileCount != 2 || first.EvidenceState != "complete" {
		t.Fatalf("first build = %#v, %v", first, err)
	}
	second, err := CommitPublicationBuild(context.Background(), ledgerPath, secondRoot, false)
	if err != nil || first.ManifestSHA256 != second.ManifestSHA256 {
		t.Fatalf("second build = %#v, %v", second, err)
	}
	firstManifest, _ := os.ReadFile(filepath.Join(firstRoot, "manifest.json"))
	secondManifest, _ := os.ReadFile(filepath.Join(secondRoot, "manifest.json"))
	if !bytes.Equal(firstManifest, secondManifest) {
		t.Fatal("identical evidence produced different manifest bytes")
	}
	if _, err := os.Stat(filepath.Join(firstRoot, "adjacent.key")); !os.IsNotExist(err) {
		t.Fatalf("adjacent key was copied: %v", err)
	}
	verified, err := VerifyPublicationPackage(context.Background(), filepath.Join(firstRoot, "ledger", "ledger.json"), filepath.Join(firstRoot, "manifest.json"))
	if err != nil || verified.Overall != VerificationPass || verified.LedgerID == "" {
		t.Fatalf("verify = %#v, %v", verified, err)
	}
	if err := os.Remove(ledgerPath); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPublicationPackage(context.Background(), filepath.Join(secondRoot, "ledger", "ledger.json"), filepath.Join(secondRoot, "manifest.json")); err != nil {
		t.Fatalf("standalone package depended on removed source ledger: %v", err)
	}
	if err := os.WriteFile(filepath.Join(firstRoot, "secret.key"), []byte("PRIVATE-CANARY"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPublicationPackage(context.Background(), filepath.Join(firstRoot, "ledger", "ledger.json"), filepath.Join(firstRoot, "manifest.json")); app.ErrorCodeOf(err) != app.CodeVerification {
		t.Fatalf("extra file error = %v", err)
	}
}

func TestPublicationDryRunAndExistingOutput(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath, output := filepath.Join(directory, "ledger.json"), filepath.Join(directory, "package")
	_ = os.WriteFile(ledgerPath, raw, 0o600)
	plan, err := CommitPublicationBuild(context.Background(), ledgerPath, output, true)
	if err != nil || len(plan.Effects) == 0 || plan.ManifestSHA256 == "" {
		t.Fatalf("plan = %#v, %v", plan, err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("dry run created package: %v", err)
	}
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitPublicationBuild(context.Background(), ledgerPath, output, false); app.ErrorCodeOf(err) != app.CodeConflict {
		t.Fatalf("existing output error = %v", err)
	}
}

func TestPublicationVerifyRejectsTamperTraversalAndLinks(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath, output := filepath.Join(directory, "ledger.json"), filepath.Join(directory, "package")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CommitPublicationBuild(context.Background(), ledgerPath, output, false); err != nil {
		t.Fatal(err)
	}
	packageLedger := filepath.Join(output, "ledger", "ledger.json")
	manifest := filepath.Join(output, "manifest.json")
	originalLedger, err := os.ReadFile(packageLedger)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(packageLedger, append(originalLedger, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPublicationPackage(context.Background(), packageLedger, manifest); app.ErrorCodeOf(err) != app.CodeVerification {
		t.Fatalf("tampered package error = %v", err)
	}
	if err := os.WriteFile(packageLedger, originalLedger, 0o600); err != nil {
		t.Fatal(err)
	}

	manifestBytes, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	traversal := bytes.Replace(manifestBytes, []byte(`"ledger_path":"ledger/ledger.json"`), []byte(`"ledger_path":"../ledger.json"`), 1)
	if bytes.Equal(traversal, manifestBytes) {
		t.Fatal("manifest traversal fixture was not changed")
	}
	if err := os.WriteFile(manifest, traversal, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPublicationPackage(context.Background(), packageLedger, manifest); app.ErrorCodeOf(err) != app.CodeVerification {
		t.Fatalf("traversal manifest error = %v", err)
	}
	if err := os.WriteFile(manifest, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Symlink(packageLedger, filepath.Join(output, "linked-ledger.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyPublicationPackage(context.Background(), packageLedger, manifest); app.ErrorCodeOf(err) != app.CodeVerification {
			t.Fatalf("linked package entry error = %v", err)
		}
	}
}

func TestPublicationBuildCancellationRemovesOnlyNewOutput(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath, output := filepath.Join(directory, "ledger.json"), filepath.Join(directory, "package")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CommitPublicationBuild(ctx, ledgerPath, output, false); app.ErrorCodeOf(err) != app.CodeInterrupted {
		t.Fatalf("cancelled package build error = %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("cancelled package output remained: %v", err)
	}
	if _, err := os.Stat(ledgerPath); err != nil {
		t.Fatalf("source ledger was removed: %v", err)
	}
}

func TestPublicationPackagesPendingEvidenceAndClassifiesMissingReceipt(t *testing.T) {
	raw, _ := fs.ReadFile(contractschema.Conformance(), "individual-ledger.json")
	directory := t.TempDir()
	ledgerPath, output := filepath.Join(directory, "ledger.json"), filepath.Join(directory, "package")
	if err := os.WriteFile(ledgerPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	transport := timestampRoundTripper(func(request *http.Request) (*http.Response, error) {
		identity := "https://alice.btc.calendar.opentimestamps.org"
		if request.URL.Host == "b.pool.opentimestamps.org" {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		branch, _ := ots.SerializeCalendarResponse([]ots.Sequence{{{Attestation: &ots.Attestation{Kind: ots.AttestationPending, Calendar: identity}}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(branch)), Header: make(http.Header)}, nil
	})
	stamp := TimestampStampOptions{
		Effects:        Effects{Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{3}, 16))}},
		CalendarClient: &ots.CalendarClient{HTTPClient: &http.Client{Transport: transport}},
	}
	if _, err := CommitTimestampStamp(context.Background(), ledgerPath, "q-election-coalition", "f-election-coalition-001", stamp); err != nil {
		t.Fatal(err)
	}
	built, err := CommitPublicationBuild(context.Background(), ledgerPath, output, false)
	if err != nil || built.EvidenceState != "pending" || built.FileCount != 4 {
		t.Fatalf("pending package = %#v, %v", built, err)
	}
	packageLedger := filepath.Join(output, "ledger", "ledger.json")
	manifest := filepath.Join(output, "manifest.json")
	verified, err := VerifyPublicationPackage(context.Background(), packageLedger, manifest)
	if err != nil || verified.Overall != VerificationPending || verified.FailureCode != app.CodePending {
		t.Fatalf("pending package verify = %#v, %v", verified, err)
	}
	receipt := filepath.Join(output, filepath.FromSlash(string(ReceiptRelativePath("f-election-coalition-001"))))
	if err := os.Remove(receipt); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPublicationPackage(context.Background(), packageLedger, manifest); app.ErrorCodeOf(err) != app.CodeVerification {
		t.Fatalf("missing listed receipt error = %v", err)
	}
}
