package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/service"
	"github.com/chaoscondensate/cli/internal/storage"
	"github.com/chaoscondensate/cli/internal/timestamp/rfc3161"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type fixedMCPClock struct{ value time.Time }

func (clock fixedMCPClock) Now() time.Time { return clock.value }

func TestMCPForecastDefaultsUseOneLedgerTimezoneObservation(t *testing.T) {
	ledgerRoot := t.TempDir()
	effects := service.ProductionEffects()
	effects.Clock = fixedMCPClock{value: time.Date(2026, 8, 30, 17, 0, 0, 0, time.UTC)}
	server, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, Timeout: time.Second, Effects: effects})
	if err != nil {
		t.Fatal(err)
	}
	client := connectClient(t, t.Context(), server)
	defer client.Close()

	calls := []struct {
		name string
		args map[string]any
	}{
		{"ledger_init", map[string]any{"file": "main:ledger.yaml", "ledger_id": "clock", "timezone": "Europe/London", "forecaster_id": "owner", "forecaster_name": "Owner"}},
		{"question_add", map[string]any{"file": "main:ledger.yaml", "question": "q-clock", "type": "binary", "title": "Question", "resolution_criteria": "Public result", "expected_resolution_at": "2030-08-10T23:59:59+01:00"}},
		{"forecast_add", map[string]any{"file": "main:ledger.yaml", "question": "q-clock", "forecast": "f-clock", "value": map[string]any{"kind": "binary", "probability_bp": 5000}}},
	}
	for _, call := range calls {
		result, callErr := callToolForTest(t, client, &sdk.CallToolParams{Name: call.name, Arguments: call.args})
		if callErr != nil || result.IsError {
			t.Fatalf("%s failed: %v %s", call.name, callErr, toolText(result))
		}
	}

	raw, err := os.ReadFile(filepath.Join(ledgerRoot, "ledger.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(raw), `"2026-08-30T18:00:00+01:00"`); count < 2 {
		t.Fatalf("forecast defaults did not share the fixed ledger-timezone observation:\n%s", raw)
	}
}

func TestMCPYAMLStructuralReplacementMutationsRemainRecoverable(t *testing.T) {
	ledgerRoot := t.TempDir()
	server, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client := connectClient(t, t.Context(), server)
	defer client.Close()

	calls := []struct {
		name string
		args map[string]any
	}{
		{"ledger_init", map[string]any{"file": "main:ledger.yaml", "ledger_id": "yaml-replacements", "timezone": "UTC", "forecaster_id": "owner", "forecaster_name": "Owner"}},
		{"platform_add", map[string]any{"file": "main:ledger.yaml", "platform": "local", "name": "Local", "kind": "self_hosted"}},
		{"question_add", map[string]any{"file": "main:ledger.yaml", "question": "q-one", "type": "binary", "title": "Will it happen?", "resolution_criteria": "Use the official result.", "expected_resolution_at": "2031-01-01T00:00:00Z", "platform_refs": []any{map[string]any{"platform": "local"}}}},
		{"platform_update", map[string]any{"file": "main:ledger.yaml", "platform": "local", "name": "Updated local", "kind": "internal"}},
		{"question_update", map[string]any{"file": "main:ledger.yaml", "question": "q-one", "title": "Updated question title", "status": "closed", "tags": []any{"reviewed", "mcp"}}},
		{"question_annul", map[string]any{"file": "main:ledger.yaml", "question": "q-one", "reason": "Question became unresolvable", "recorded_at": "2026-09-01T12:00:00Z", "confirm": true}},
		{"ledger_validate", map[string]any{"file": "main:ledger.yaml"}},
	}
	for _, call := range calls {
		result, callErr := callToolForTest(t, client, &sdk.CallToolParams{Name: call.name, Arguments: call.args})
		if callErr != nil || result.IsError || strings.Contains(toolText(result), `"code":"internal"`) {
			t.Fatalf("%s failed: err=%v result=%s", call.name, callErr, toolText(result))
		}
	}

	raw, err := os.ReadFile(filepath.Join(ledgerRoot, "ledger.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "status: annulled") || !strings.Contains(string(raw), "name: Updated local") || strings.Contains(string(raw), "{status:") {
		t.Fatalf("MCP replacements did not retain expanded valid YAML:\n%s", raw)
	}
}

func TestMCPDiscoveryClosedSchemasModesAndParityCall(t *testing.T) {
	ledgerRoot := t.TempDir()
	outputRoot := t.TempDir()
	secretRoot := t.TempDir()
	copyFixture(t, filepath.Join("..", "..", "schema", "testdata", "forecast-ledger", "v1.3.0", "individual-ledger.json"), filepath.Join(ledgerRoot, "ledger.json"))
	server, err := New(Config{
		LedgerRoots: []string{"main=" + ledgerRoot}, OutputRoots: []string{"packages=" + outputRoot}, SecretRoots: []string{"keys=" + secretRoot},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	client := connectClient(t, ctx, server)
	defer client.Close()

	listed, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
		encoded, _ := json.Marshal(tool.InputSchema)
		if !strings.Contains(string(encoded), `"additionalProperties":false`) {
			t.Errorf("tool %s schema is not closed: %s", tool.Name, encoded)
		}
		for _, removed := range []string{`"input":`, `"input_file":`} {
			if strings.Contains(string(encoded), removed) {
				t.Errorf("tool %s still exposes removed generic wrapper %s", tool.Name, removed)
			}
		}
		for _, forbidden := range []string{"calendar", "bitcoin_core", "proxy", "explorer"} {
			if strings.Contains(string(encoded), `"`+forbidden+`"`) {
				t.Errorf("tool %s exposes forbidden MCP endpoint input %q", tool.Name, forbidden)
			}
		}
		definition, ok := operationDefinitionByTool(tool.Name)
		if !ok {
			t.Errorf("tool %s has no operation definition", tool.Name)
		} else {
			expectedEffect := "effect=read-only"
			if definition.Policy.PersistentEffect {
				expectedEffect = "effect=mutating"
			}
			if !strings.Contains(tool.Description, expectedEffect) || !strings.Contains(tool.Description, "server_access=read-write") {
				t.Errorf("tool %s description does not separate effect and server mode: %q", tool.Name, tool.Description)
			}
		}
	}
	sort.Strings(names)
	if containsName(names, "forecast_reveal") {
		t.Fatal("reveal discovered without --allow-reveal")
	}
	for _, definition := range service.OperationDefinitions() {
		if definition.Name == service.OperationForecastReveal {
			continue
		}
		if !containsName(names, definition.MCPTool) {
			t.Errorf("completed tool %s is absent", definition.MCPTool)
		}
	}
	for _, tool := range listed.Tools {
		result, callErr := callToolForTest(t, client, &sdk.CallToolParams{Name: tool.Name, Arguments: minimumToolArguments(tool.Name)})
		if callErr != nil {
			t.Errorf("registered tool %s returned protocol error: %v", tool.Name, callErr)
			continue
		}
		if tool.Name == "ledger_init" {
			if result.IsError || !strings.Contains(toolText(result), `"question_count":0`) {
				t.Errorf("registered tool %s did not create an empty ledger: %s", tool.Name, toolText(result))
			}
			continue
		}
		if !result.IsError || !strings.Contains(toolText(result), `"code"`) {
			t.Errorf("registered tool %s did not return a recoverable application error: %s", tool.Name, toolText(result))
		}
	}
	if _, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "forecast_reveal", Arguments: map[string]any{}}); err == nil {
		t.Fatal("disabled reveal direct call did not return unknown-tool protocol error")
	}

	result, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:ledger.json"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("ledger_validate failed: %#v", result.Content)
	}
	encoded, _ := json.Marshal(result.StructuredContent)
	if !strings.Contains(string(encoded), `"code":"ledger.valid"`) {
		t.Fatalf("unexpected result: %s", encoded)
	}
	targetCheck, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "target_check", Arguments: map[string]any{"file": "main:ledger.json", "question": "q-election-coalition", "forecast": "f-election-coalition-001"}})
	if err != nil || targetCheck.IsError || !strings.Contains(toolText(targetCheck), `"state":"not_applicable"`) || !strings.Contains(toolText(targetCheck), "content.no_retained_target") {
		t.Fatalf("MCP unretained target result=%s err=%v", toolText(targetCheck), err)
	}
	forecastShow, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "forecast_show", Arguments: map[string]any{"file": "main:ledger.json", "question": "q-election-coalition", "forecast": "f-election-coalition-001"}})
	if err != nil || forecastShow.IsError || !strings.Contains(toolText(forecastShow), `"integrity":{"status":"unanchored"}`) {
		t.Fatalf("MCP forecast integrity result=%s err=%v", toolText(forecastShow), err)
	}
	autoPlan, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "timestamp_stamp", Arguments: map[string]any{
		"file": "main:ledger.json", "question": "q-election-coalition", "forecast": "f-election-coalition-001", "dry_run": true,
	}})
	if err != nil || autoPlan.IsError || !strings.Contains(toolText(autoPlan), `"selection_mode":"auto"`) || !strings.Contains(toolText(autoPlan), `"provider_id":"freetsa"`) || !strings.Contains(toolText(autoPlan), `"request_count":0`) {
		t.Fatalf("MCP automatic timestamp plan result=%s err=%v", toolText(autoPlan), err)
	}
	if _, statErr := os.Stat(filepath.Join(ledgerRoot, "trust")); !os.IsNotExist(statErr) {
		t.Fatalf("MCP automatic dry-run created trust: %v", statErr)
	}
	copyFixture(t, filepath.Join("..", "..", "timestamp", "rfc3161", "testdata", "root.pem"), filepath.Join(ledgerRoot, "tsa.pem"))
	tsaFailure, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "timestamp_stamp", Arguments: map[string]any{
		"file": "main:ledger.json", "question": "q-election-coalition", "forecast": "f-election-coalition-001",
		"tsa_url": "https://127.0.0.1", "ca_bundle": "tsa.pem",
	}})
	if err != nil || !tsaFailure.IsError || !strings.Contains(toolText(tsaFailure), `"code":"network"`) || !strings.Contains(toolText(tsaFailure), `"timing.tsa_unavailable"`) || !strings.Contains(toolText(tsaFailure), `"request_count":1`) {
		t.Fatalf("MCP safe TSA failure result=%s err=%v", toolText(tsaFailure), err)
	}
	sessionStillAlive, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:ledger.json"}})
	if err != nil || sessionStillAlive.IsError {
		t.Fatalf("MCP session did not survive TSA failure: result=%s err=%v", toolText(sessionStillAlive), err)
	}
	semanticFailure, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{"file": "main:ledger.json", "platform": "invalid-semantic", "name": "   ", "kind": "informal"}})
	if err != nil || !semanticFailure.IsError || strings.Contains(toolText(semanticFailure), `"line":0`) || strings.Contains(toolText(semanticFailure), `"column":0`) {
		t.Fatalf("MCP semantic diagnostic fabricated a span: %s, %v", toolText(semanticFailure), err)
	}

	lock, err := storage.AcquireLedgerLock(ctx, filepath.Join(ledgerRoot, "ledger.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	conflict, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{
		"file": "main:ledger.json", "platform": "locked-platform", "name": "Locked", "kind": "internal",
	}})
	if releaseErr := lock.Release(); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if err != nil || !conflict.IsError || !strings.Contains(toolText(conflict), string(app.CodeConflict)) || time.Since(started) > time.Second {
		t.Fatalf("same-ledger writer conflict result=%s err=%v", toolText(conflict), err)
	}

	copyFixture(t, filepath.Join("..", "..", "schema", "testdata", "forecast-ledger", "v1.3.0", "individual-ledger.json"), filepath.Join(ledgerRoot, "second.json"))
	independent, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{
		"file": "main:second.json", "platform": "independent-platform", "name": "Independent", "kind": "internal",
	}})
	if err != nil || independent.IsError {
		t.Fatalf("cross-ledger mutation was blocked: result=%s err=%v", toolText(independent), err)
	}

	unknown, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:ledger.json", "typo": true}})
	if err != nil {
		t.Fatal(err)
	}
	if !unknown.IsError || !strings.Contains(toolText(unknown), "unknown field") {
		t.Fatalf("unknown property accepted: %#v", unknown)
	}

	escape, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:../outside.json"}})
	if err != nil {
		t.Fatal(err)
	}
	escapeText := toolText(escape)
	if !escape.IsError || !strings.Contains(escapeText, `"route":"ledger:main"`) || !strings.Contains(escapeText, `"flag":"--ledger-root"`) || strings.Contains(escapeText, ledgerRoot) {
		t.Fatalf("root traversal diagnostic is incomplete or unsafe: %s", escapeText)
	}
	missingRoot, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "missing:ledger.json"}})
	if err != nil {
		t.Fatal(err)
	}
	missingRootText := toolText(missingRoot)
	if !missingRoot.IsError || !strings.Contains(missingRootText, `"root":"missing"`) || !strings.Contains(missingRootText, `"class":"ledger"`) || strings.Contains(missingRootText, ledgerRoot) {
		t.Fatalf("root diagnostic is not safe: %s", missingRootText)
	}

	templates, err := client.ListResourceTemplates(ctx, nil)
	if err != nil || len(templates.ResourceTemplates) != 1 {
		t.Fatalf("resource templates=%d err=%v", len(templates.ResourceTemplates), err)
	}
	resource, err := client.ReadResource(ctx, &sdk.ReadResourceParams{URI: "forecast-ledger://v1/ledger/main/ledger.json"})
	if err != nil || len(resource.Contents) != 1 || !strings.Contains(resource.Contents[0].Text, "ledger_id") {
		t.Fatalf("ledger resource=%#v err=%v", resource, err)
	}
}

func TestEveryMCPExistingLedgerToolRejectsV120AtAdmission(t *testing.T) {
	ledgerRoot, outputRoot, secretRoot := t.TempDir(), t.TempDir(), t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "..", "schema", "testdata", "forecast-ledger", "v1.3.0", "individual-ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	oldLedger := bytes.Replace(raw, []byte(`"schema_version": "1.3.0"`), []byte(`"schema_version": "1.2.0"`), 1)
	if err := os.WriteFile(filepath.Join(ledgerRoot, "ledger.json"), oldLedger, 0o600); err != nil {
		t.Fatal(err)
	}
	server, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, OutputRoots: []string{"packages=" + outputRoot}, SecretRoots: []string{"keys=" + secretRoot}, Mode: service.AccessMode{AllowReveal: true}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	client := connectClient(t, t.Context(), server)
	defer client.Close()
	listed, err := client.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name == "ledger_init" || tool.Name == "publication_verify" {
			continue
		}
		arguments := minimumToolArguments(tool.Name)
		arguments["file"] = "main:ledger.json"
		result, callErr := callToolForTest(t, client, &sdk.CallToolParams{Name: tool.Name, Arguments: arguments})
		if callErr != nil || !result.IsError || !strings.Contains(toolText(result), `"code":"unsupported_schema_version"`) {
			t.Errorf("tool %s bypassed v1.1.0 admission: result=%s err=%v", tool.Name, toolText(result), callErr)
		}
	}
	if _, err := os.Stat(filepath.Join(ledgerRoot, "proofs")); !os.IsNotExist(err) {
		t.Fatalf("old-schema MCP admission created artifacts: %v", err)
	}
	if entries, err := os.ReadDir(outputRoot); err != nil || len(entries) != 0 {
		t.Fatalf("old-schema MCP admission wrote output: entries=%v err=%v", entries, err)
	}
	if entries, err := os.ReadDir(secretRoot); err != nil || len(entries) != 0 {
		t.Fatalf("old-schema MCP admission wrote secrets: entries=%v err=%v", entries, err)
	}
}

func TestMCPPublicationVerifyUsesOnePackageOutputRoot(t *testing.T) {
	ledgerRoot, outputRoot := t.TempDir(), t.TempDir()
	ledgerPath := filepath.Join(ledgerRoot, "ledger.json")
	copyFixture(t, filepath.Join("..", "..", "schema", "testdata", "forecast-ledger", "v1.3.0", "individual-ledger.json"), ledgerPath)
	copyFixture(t, filepath.Join("..", "..", "timestamp", "rfc3161", "testdata", "root.pem"), filepath.Join(ledgerRoot, "tsa.pem"))
	requestPath, _, err := service.TimestampEvidencePaths("f-election-coalition-001", "https://tsa.example.test")
	if err != nil {
		t.Fatal(err)
	}
	absoluteRequest := filepath.Join(ledgerRoot, filepath.FromSlash(string(requestPath)))
	if err := os.MkdirAll(filepath.Dir(absoluteRequest), 0o755); err != nil {
		t.Fatal(err)
	}
	copyFixture(t, filepath.Join("..", "..", "timestamp", "rfc3161", "testdata", "request.tsq"), absoluteRequest)
	response, err := os.ReadFile(filepath.Join("..", "..", "timestamp", "rfc3161", "testdata", "response.tsr"))
	if err != nil {
		t.Fatal(err)
	}
	client := &rfc3161.HTTPClient{Resolver: mcpPublicResolver{}, Client: &http.Client{Transport: mcpRoundTripper(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/timestamp-reply"}}, Body: io.NopCloser(bytes.NewReader(response)), Request: request}, nil
	})}}
	if _, err := service.CommitTimestampStamp(t.Context(), ledgerPath, "q-election-coalition", "f-election-coalition-001", service.TimestampStampOptions{TSAURL: "https://tsa.example.test", CABundlePath: "tsa.pem", Effects: service.ProductionEffects(), HTTPClient: client}); err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(outputRoot, "evidence")
	if _, err := service.CommitPublicationBuild(t.Context(), ledgerPath, packageRoot, false); err != nil {
		t.Fatal(err)
	}
	server, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, OutputRoots: []string{"packages=" + outputRoot}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	mcpClient := connectClient(t, t.Context(), server)
	defer mcpClient.Close()
	result, err := callToolForTest(t, mcpClient, &sdk.CallToolParams{Name: "publication_verify", Arguments: map[string]any{
		"file": "packages:evidence/ledger/ledger.json", "manifest": "packages:evidence/manifest.json",
	}})
	if err != nil || result.IsError || !strings.Contains(toolText(result), `"overall":"pass"`) {
		t.Fatalf("MCP package verification result=%s err=%v", toolText(result), err)
	}
}

type mcpPublicResolver struct{}

func (mcpPublicResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
}

type mcpRoundTripper func(*http.Request) (*http.Response, error)

func (fn mcpRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestMCPEmptyInitAndBacklogQuestion(t *testing.T) {
	ledgerRoot := t.TempDir()
	secretRoot := t.TempDir()
	server, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, SecretRoots: []string{"keys=" + secretRoot}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	client := connectClient(t, ctx, server)
	defer client.Close()

	initialized, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "ledger_init", Arguments: map[string]any{
		"file": "main:empty.json", "ledger_id": "empty", "timezone": "UTC", "forecaster_id": "owner", "forecaster_name": "Owner",
	}})
	if err != nil || initialized.IsError || !strings.Contains(toolText(initialized), `"question_count":0`) || !strings.Contains(toolText(initialized), `"forecast_count":0`) {
		t.Fatalf("empty init result=%s err=%v", toolText(initialized), err)
	}

	questionInput := map[string]any{
		"title": "Will it happen?", "resolution_criteria": "Resolve from the named source.",
		"expected_resolution_at": "2027-01-01T00:00:00Z",
	}
	questionArguments := map[string]any{"file": "main:empty.json", "question": "q-one", "type": "binary"}
	for name, value := range questionInput {
		questionArguments[name] = value
	}
	added, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "question_add", Arguments: map[string]any{
		"file": questionArguments["file"], "question": questionArguments["question"], "type": questionArguments["type"],
		"title": questionArguments["title"], "resolution_criteria": questionArguments["resolution_criteria"], "expected_resolution_at": questionArguments["expected_resolution_at"],
	}})
	if err != nil || added.IsError || !strings.Contains(toolText(added), `"message":"Question was added"`) {
		t.Fatalf("question add result=%s err=%v", toolText(added), err)
	}
	listed, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "forecast_list", Arguments: map[string]any{"file": "main:empty.json", "question": "q-one"}})
	if err != nil || listed.IsError || !strings.Contains(toolText(listed), `"forecasts":[]`) {
		t.Fatalf("forecast list result=%s err=%v", toolText(listed), err)
	}

	sealedInput := map[string]any{
		"title": "Secret", "resolution_criteria": "Resolve from the named source.",
		"expected_resolution_at": "2027-01-01T00:00:00Z",
		"initial_forecast":       map[string]any{"id": "f-secret", "visibility": "sealed", "forecasted_at": "2026-08-30T00:00:00Z", "value": map[string]any{"kind": "binary", "probability_bp": 5000}, "rationale": "private", "key_factors": []any{}, "comment": "private"},
	}
	sealed, err := callToolForTest(t, client, &sdk.CallToolParams{Name: "question_add", Arguments: map[string]any{
		"file": "main:empty.json", "question": "q-secret", "type": "binary", "title": sealedInput["title"],
		"resolution_criteria": sealedInput["resolution_criteria"], "expected_resolution_at": sealedInput["expected_resolution_at"], "initial_forecast": sealedInput["initial_forecast"],
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !sealed.IsError || !strings.Contains(toolText(sealed), "initial_secret_input_file") {
		t.Fatalf("inline sealed input was not rejected safely: %s", toolText(sealed))
	}
}

func TestMCPRevealDiscoveryReadOnlyOfflineAndRootValidation(t *testing.T) {
	ledgerRoot, secretRoot := t.TempDir(), t.TempDir()
	if _, err := New(Config{LedgerRoots: []string{ledgerRoot}}); err == nil {
		t.Fatal("unnamed ledger root succeeded")
	} else if applicationErr, ok := err.(*app.Error); !ok || applicationErr.Details["class"] != service.RootLedger || applicationErr.Details["flag"] != "--ledger-root" {
		t.Fatalf("unnamed root diagnostic = %#v", err)
	}
	reveal, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, SecretRoots: []string{"keys=" + secretRoot}, Mode: service.AccessMode{AllowReveal: true}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := connectClient(t, ctx, reveal)
	defer client.Close()
	listed, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tool := range listed.Tools {
		found = found || tool.Name == "forecast_reveal"
	}
	if !found {
		t.Fatal("reveal was not discovered after explicit enablement")
	}

	if _, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, SecretRoots: []string{"keys=" + secretRoot}, Mode: service.AccessMode{AllowReveal: true, ReadOnly: true}}); err == nil {
		t.Fatal("contradictory reveal/read-only configuration succeeded")
	}
	if _, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, Mode: service.AccessMode{AllowReveal: true}}); err == nil {
		t.Fatal("reveal without secret root succeeded")
	}
	inside := filepath.Join(ledgerRoot, "inside")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, OutputRoots: []string{"packages=" + inside}}); err == nil {
		t.Fatal("overlapping roots succeeded")
	} else if applicationErr, ok := err.(*app.Error); !ok || applicationErr.Details["first_route"] != "ledger:main" || applicationErr.Details["second_route"] != "output:packages" {
		t.Fatalf("overlap diagnostic = %#v", err)
	}

	readOnly, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, Mode: service.AccessMode{ReadOnly: true, Offline: true}})
	if err != nil {
		t.Fatal(err)
	}
	client2 := connectClient(t, ctx, readOnly)
	defer client2.Close()
	readOnlyTools, err := client2.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range readOnlyTools.Tools {
		definition, ok := operationDefinitionByTool(tool.Name)
		if !ok || definition.Policy.PersistentEffect {
			t.Fatalf("mutating or unknown tool %s discovered in read-only mode", tool.Name)
		}
		if !strings.Contains(tool.Description, "effect=read-only") || !strings.Contains(tool.Description, "server_access=read-only") {
			t.Fatalf("read-only tool description is ambiguous: %q", tool.Description)
		}
	}
	if _, err := client2.CallTool(ctx, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{"file": "main:missing.json", "platform": "x", "name": "X", "kind": "informal"}}); err == nil {
		t.Fatal("read-only mutation direct call did not return unknown-tool protocol error")
	}

	limited, err := New(Config{LedgerRoots: []string{"main=" + ledgerRoot}, MaxConcurrent: 1, MaxToolBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	client3 := connectClient(t, ctx, limited)
	defer client3.Close()
	limited.sem <- struct{}{}
	busy, err := client3.CallTool(ctx, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:missing.json"}})
	<-limited.sem
	if err != nil || !busy.IsError || !strings.Contains(toolText(busy), "concurrent request limit") {
		t.Fatalf("concurrency limit result=%s err=%v", toolText(busy), err)
	}
	tooLarge, err := client3.CallTool(ctx, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:" + strings.Repeat("a", 2048)}})
	if err != nil || !tooLarge.IsError || !strings.Contains(toolText(tooLarge), "size limit") {
		t.Fatalf("argument limit result=%s err=%v", toolText(tooLarge), err)
	}
}

func TestMCPRealProcessStdioIsProtocolCleanAndShutsDownOnEOF(t *testing.T) {
	root := t.TempDir()
	command := exec.Command(os.Args[0], "-test.run=TestMCPHelperProcess")
	command.Env = append(os.Environ(), "FORECAST_LEDGER_MCP_HELPER=1", "FORECAST_LEDGER_MCP_ROOT="+root)
	var stderr strings.Builder
	command.Stderr = &stderr
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := sdk.NewClient(&sdk.Implementation{Name: "real-process-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.CommandTransport{Command: command, TerminateDuration: 10 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) == 0 {
		t.Fatalf("tools=%d err=%v", len(listed.Tools), err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected server diagnostic output: %q", stderr.String())
	}
}

func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("FORECAST_LEDGER_MCP_HELPER") != "1" {
		return
	}
	server, err := New(Config{LedgerRoots: []string{"main=" + os.Getenv("FORECAST_LEDGER_MCP_ROOT")}, Stderr: os.Stderr})
	if err == nil {
		err = server.ServeStdio(context.Background())
	}
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func FuzzMCPToolArguments(f *testing.F) {
	f.Add([]byte(`{"file":"main:ledger.json"}`))
	f.Add([]byte(`{"file":null,"unknown":true}`))
	allowed := map[string]bool{"file": true, "question": true, "forecast": true, "dry_run": true}
	definition := service.OperationDefinition{Name: service.OperationLedgerValidate}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64<<10 {
			return
		}
		_, _ = decodeToolArguments(data, 64<<10, definition, allowed, []string{"file"})
	})
}

func callToolForTest(t *testing.T, client *sdk.ClientSession, params *sdk.CallToolParams) (*sdk.CallToolResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	return client.CallTool(ctx, params)
}

func connectClient(t *testing.T, ctx context.Context, server *Server) *sdk.ClientSession {
	t.Helper()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverSession, err := server.SDK().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := sdk.NewClient(&sdk.Implementation{Name: "forecast-ledger-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func copyFixture(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsName(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func operationDefinitionByTool(name string) (service.OperationDefinition, bool) {
	for _, definition := range service.OperationDefinitions() {
		if definition.MCPTool == name {
			return definition, true
		}
	}
	return service.OperationDefinition{}, false
}

func toolText(result *sdk.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if text, ok := result.Content[0].(*sdk.TextContent); ok {
		return text.Text
	}
	return ""
}

func minimumToolArguments(name string) map[string]any {
	arguments := map[string]any{"file": "main:missing.json"}
	if strings.HasPrefix(name, "platform_") {
		arguments["platform"] = "p-one"
	}
	if strings.HasPrefix(name, "question_") {
		arguments["question"] = "q-one"
	}
	if strings.HasPrefix(name, "forecast_") || strings.HasPrefix(name, "timestamp_") {
		arguments["question"], arguments["forecast"] = "q-one", "f-one"
	}
	switch name {
	case "ledger_init":
		arguments["file"], arguments["ledger_id"], arguments["timezone"] = "main:new.json", "ledger-one", "UTC"
		arguments["forecaster_id"], arguments["forecaster_name"] = "me", "Me"
	case "platform_add":
		arguments["name"], arguments["kind"] = "Platform", "internal"
	case "question_add":
		arguments["title"], arguments["resolution_criteria"], arguments["expected_resolution_at"] = "Question", "Criteria", "2030-01-01T00:00:00Z"
	case "question_resolve":
		arguments["outcome"], arguments["outcome_known_at"], arguments["sources"] = true, "2030-01-01T00:00:00Z", []any{}
	case "question_annul", "question_dispute":
		arguments["reason"] = "Reason"
	case "forecast_add":
		arguments["value"] = map[string]any{"kind": "binary", "probability_bp": 5000}
	case "forecast_seal":
		arguments["secret_input_file"], arguments["key_file"] = "keys:missing.json", "keys:new.key"
	case "forecast_reveal":
		arguments["key_file"], arguments["confirm"] = "keys:missing.key", true
	case "forecast_key_hint_update":
		arguments["key_hint"] = "vault:item"
	case "publication_build":
		arguments["output"] = "packages:new-package"
	case "publication_verify":
		arguments["manifest"] = "packages:missing-manifest.json"
	case "timestamp_stamp":
		arguments["tsa_url"], arguments["ca_bundle"] = "https://tsa.example.test", "tsa.pem"
	}
	if name == "platform_list" {
		delete(arguments, "platform")
	}
	if name == "question_list" {
		delete(arguments, "question")
	}
	if name == "forecast_list" {
		delete(arguments, "forecast")
	}
	if name == "question_add" {
		arguments["type"] = "binary"
	}
	if name == "platform_remove" || name == "question_resolve" || name == "question_annul" || name == "question_dispute" {
		arguments["confirm"] = true
	}
	return arguments
}
