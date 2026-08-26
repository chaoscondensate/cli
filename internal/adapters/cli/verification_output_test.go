package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/presentation"
	"github.com/chaoscondensate/cli/internal/service"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
