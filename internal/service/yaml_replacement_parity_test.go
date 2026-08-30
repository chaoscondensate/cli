package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
)

func TestQuestionReplacementLifecycleHasJSONAndYAMLParity(t *testing.T) {
	jsonPath, yamlPath := formatParityLedgers(t)
	questionID := ledger.Slug("q-election-coalition")
	questionRef, platformURL := "updated-question", "https://example.test/questions/updated"
	update := QuestionPatchInput{
		Title:                Optional[string]{Set: true, Value: "Updated coalition question"},
		ExpectedResolutionAt: Optional[ledger.Timestamp]{Set: true, Value: "2026-10-20T12:00:00+01:00"},
		PlatformRefs: Optional[[]ledger.PlatformRef]{Set: true, Value: []ledger.PlatformRef{
			{Platform: "local"},
			{Platform: "metaculus", QuestionID: &questionRef, URL: &platformURL},
		}},
		Tags:   Optional[[]ledger.Slug]{Set: true, Value: []ledger.Slug{"reviewed", "coalition"}},
		Status: Optional[ledger.QuestionStatus]{Set: true, Value: ledger.QuestionClosed},
	}
	var updateResults []QuestionFileResult
	for _, path := range []string{jsonPath, yamlPath} {
		result, err := CommitQuestionUpdateFile(context.Background(), path, questionID, update)
		if err != nil {
			t.Fatalf("question update %s: %v", filepath.Ext(path), err)
		}
		updateResults = append(updateResults, result)
	}
	if !reflect.DeepEqual(updateResults[0].ChangedPointers, updateResults[1].ChangedPointers) || updateResults[0].Status != updateResults[1].Status {
		t.Fatalf("question update results differ: JSON=%#v YAML=%#v", updateResults[0], updateResults[1])
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)

	outcome := "centre-left"
	resolution := ResolutionInput{
		Outcome:        ResolutionOutcome{Text: &outcome},
		OutcomeKnownAt: "2026-10-15T12:00:00+01:00",
		RecordedAt:     timestampPointer("2026-10-15T12:05:00+01:00"),
		Sources: []EvidenceSourceInput{{
			Title: "Official appointment", URL: "https://example.test/result", RetrievedAt: "2026-10-15T12:04:00+01:00",
		}},
	}
	for _, path := range []string{jsonPath, yamlPath} {
		if _, err := CommitQuestionResolveFile(context.Background(), path, questionID, resolution, "2026-10-15T12:05:00+01:00"); err != nil {
			t.Fatalf("question resolve %s: %v", filepath.Ext(path), err)
		}
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)

	dispute := DisputeInput{Reason: "The appointment is under review.", RecordedAt: timestampPointer("2026-10-16T00:00:00+01:00")}
	for _, path := range []string{jsonPath, yamlPath} {
		if _, err := CommitQuestionDisputeFile(context.Background(), path, questionID, dispute, "2026-10-16T00:00:00+01:00"); err != nil {
			t.Fatalf("question dispute %s: %v", filepath.Ext(path), err)
		}
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)

	annul := AnnulInput{Reason: "The event definition became invalid.", RecordedAt: timestampPointer("2026-10-17T00:00:00+01:00")}
	for _, path := range []string{jsonPath, yamlPath} {
		if _, err := CommitQuestionAnnulFile(context.Background(), path, questionID, annul, "2026-10-17T00:00:00+01:00"); err != nil {
			t.Fatalf("question annul %s: %v", filepath.Ext(path), err)
		}
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)
}

func TestRootAndPlatformReplacementsHaveJSONAndYAMLParity(t *testing.T) {
	jsonPath, yamlPath := formatParityLedgers(t)
	website := "https://example.test/updated"
	rootUpdate := RootMetadataPatchInput{
		Title:           Optional[string]{Set: true, Value: "Updated parity ledger"},
		DefaultTimezone: Optional[string]{Set: true, Value: "UTC"},
		Forecaster: Optional[ForecasterMetadataPatchInput]{Set: true, Value: ForecasterMetadataPatchInput{
			Name:    Optional[string]{Set: true, Value: "Updated Forecaster"},
			Contact: Optional[ledger.Contact]{Set: true, Value: ledger.Contact{Website: &website}},
		}},
	}
	platformUpdate := PlatformPatchInput{
		Name: Optional[string]{Set: true, Value: "Updated local platform"},
		Kind: Optional[ledger.PlatformKind]{Set: true, Value: ledger.PlatformInternal},
		Account: Optional[PlatformAccountPatchInput]{Set: true, Value: PlatformAccountPatchInput{
			Username: Optional[string]{Set: true, Value: "parity-user"},
		}},
	}
	for _, path := range []string{jsonPath, yamlPath} {
		if _, err := CommitRootMetadataFileUpdate(context.Background(), path, rootUpdate); err != nil {
			t.Fatalf("root update %s: %v", filepath.Ext(path), err)
		}
		if _, err := CommitPlatformUpdateFile(context.Background(), path, "local", platformUpdate); err != nil {
			t.Fatalf("platform update %s: %v", filepath.Ext(path), err)
		}
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)
}

func TestForecastRevealReplacementHasJSONAndYAMLParity(t *testing.T) {
	root, err := BuildLedgerRoot(InitRootRequest{LedgerID: "replacement-parity", Timezone: "UTC", ForecasterID: "me", ForecasterName: "Me"}, fixedTestClock{value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	input := binaryInitialQuestion()
	input.InitialForecast.Visibility = ledger.VisibilitySealed
	rationale, comment := "PRIVATE-REVEAL-RATIONALE", "PRIVATE-REVEAL-COMMENT"
	factors := []string{"PRIVATE-REVEAL-FACTOR"}
	input.InitialForecast.Rationale, input.InitialForecast.Comment, input.InitialForecast.KeyFactors = &rationale, &comment, &factors
	build, err := BuildInitialSealedLedger(context.Background(), root, input, Effects{
		Clock: fixedTestClock{}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x52}, 76))},
	})
	if err != nil {
		t.Fatal(err)
	}
	jsonPath, yamlPath := newFormatParityLedgers(t, build.Ledger)
	var targetBytes [][]byte
	for _, path := range []string{jsonPath, yamlPath} {
		keyPath := filepath.Join(filepath.Dir(path), "forecast.key")
		if err := storage.CreateProtectedFile(keyPath, build.KeyFile); err != nil {
			t.Fatal(err)
		}
		if _, err := CommitTargetBuild(context.Background(), path, false, "q-one", "f-one"); err != nil {
			t.Fatalf("target build %s: %v", filepath.Ext(path), err)
		}
		result, revealErr := CommitForecastRevealFile(context.Background(), path, keyPath, "q-one", "f-one", "2026-02-01T00:00:00Z")
		publicResult := fmt.Sprintf("%#v %v", result, revealErr)
		if revealErr != nil || !result.Changed {
			t.Fatalf("reveal %s: %s", filepath.Ext(path), publicResult)
		}
		for _, forbidden := range []string{rationale, comment, factors[0], keyPath, filepath.Dir(path)} {
			if strings.Contains(publicResult, forbidden) {
				t.Fatalf("reveal result leaked %q: %s", forbidden, publicResult)
			}
		}
		retained, err := os.ReadFile(filepath.Join(filepath.Dir(path), "proofs", "targets", "f-one.json"))
		if err != nil {
			t.Fatal(err)
		}
		targetBytes = append(targetBytes, retained)
	}
	if !bytes.Equal(targetBytes[0], targetBytes[1]) {
		t.Fatal("JSON and YAML reveal workflows produced different canonical target bytes")
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)
}

func TestTimestampIntegrityReplacementHasJSONAndYAMLParity(t *testing.T) {
	jsonPath, yamlPath := formatParityLedgers(t)
	tsaURL := "https://tsa.example.test/stamp"
	requestPath, _, err := TimestampEvidencePaths("f-election-coalition-001", tsaURL)
	if err != nil {
		t.Fatal(err)
	}
	var targetBytes [][]byte
	var results []TimestampArtifactResult
	for _, path := range []string{jsonPath, yamlPath} {
		directory := filepath.Dir(path)
		if err := os.WriteFile(filepath.Join(directory, "tsa.pem"), timestampFixture(t, "root.pem"), 0o600); err != nil {
			t.Fatal(err)
		}
		absoluteRequest := filepath.Join(directory, filepath.FromSlash(string(requestPath)))
		if err := os.MkdirAll(filepath.Dir(absoluteRequest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absoluteRequest, timestampFixture(t, "request.tsq"), 0o600); err != nil {
			t.Fatal(err)
		}
		transport := &countingRoundTripper{response: timestampFixture(t, "response.tsr")}
		result, err := CommitTimestampStamp(t.Context(), path, "q-election-coalition", "f-election-coalition-001", TimestampStampOptions{
			TSAURL: tsaURL, CABundlePath: "tsa.pem",
			Effects:    Effects{Clock: fixedTestClock{value: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}, Random: deterministicTestRandom{reader: bytes.NewReader(bytes.Repeat([]byte{0x42}, 64))}},
			HTTPClient: testTimestampHTTPClient(transport),
		})
		if err != nil || result.State != TimestampVerified || transport.requests != 1 {
			t.Fatalf("timestamp stamp %s: result=%#v err=%v requests=%d", filepath.Ext(path), result, err, transport.requests)
		}
		results = append(results, result)
		retained, err := os.ReadFile(filepath.Join(directory, "proofs", "targets", "f-election-coalition-001.json"))
		if err != nil {
			t.Fatal(err)
		}
		targetBytes = append(targetBytes, retained)
	}
	if results[0].State != results[1].State || !reflect.DeepEqual(results[0].Entries, results[1].Entries) {
		t.Fatalf("timestamp results differ: JSON=%#v YAML=%#v", results[0], results[1])
	}
	if !bytes.Equal(targetBytes[0], targetBytes[1]) {
		t.Fatal("JSON and YAML timestamp workflows produced different canonical target bytes")
	}
	assertFormatParityLedgers(t, jsonPath, yamlPath)
}

func formatParityLedgers(t *testing.T) (string, string) {
	t.Helper()
	_, model := rootUpdateFixture(t, "individual-ledger.json")
	return newFormatParityLedgers(t, model)
}

func newFormatParityLedgers(t *testing.T, model *ledger.Ledger) (string, string) {
	t.Helper()
	root := t.TempDir()
	paths := make([]string, 0, 2)
	for _, extension := range []string{".json", ".yaml"} {
		directory := filepath.Join(root, strings.TrimPrefix(extension, "."))
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, "ledger"+extension)
		if _, err := CommitNewLedger(path, model); err != nil {
			t.Fatalf("create %s parity ledger: %v", extension, err)
		}
		paths = append(paths, path)
	}
	return paths[0], paths[1]
}

func assertFormatParityLedgers(t *testing.T, jsonPath, yamlPath string) {
	t.Helper()
	jsonLedger, err := LoadAndValidateLedger(context.Background(), jsonPath, nil)
	if err != nil {
		t.Fatalf("load JSON ledger: %v", err)
	}
	yamlLedger, err := LoadAndValidateLedger(context.Background(), yamlPath, nil)
	if err != nil {
		t.Fatalf("load YAML ledger: %v", err)
	}
	if !reflect.DeepEqual(jsonLedger.Model, yamlLedger.Model) {
		t.Fatalf("JSON and YAML ledger models differ:\nJSON: %#v\nYAML: %#v", jsonLedger.Model, yamlLedger.Model)
	}
}
