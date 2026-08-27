package mcp

import (
	"context"
	"encoding/json"
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
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPDiscoveryClosedSchemasModesAndParityCall(t *testing.T) {
	ledgerRoot := t.TempDir()
	outputRoot := t.TempDir()
	secretRoot := t.TempDir()
	copyFixture(t, filepath.Join("..", "..", "schema", "testdata", "forecast-ledger", "v1.0.0", "individual-ledger.json"), filepath.Join(ledgerRoot, "ledger.json"))
	server, err := New(Config{
		LedgerRoots: []string{"main=" + ledgerRoot}, OutputRoots: []string{"packages=" + outputRoot}, SecretRoots: []string{"keys=" + secretRoot},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
		result, callErr := client.CallTool(ctx, &sdk.CallToolParams{Name: tool.Name, Arguments: minimumToolArguments(tool.Name)})
		if callErr != nil {
			t.Errorf("registered tool %s returned protocol error: %v", tool.Name, callErr)
			continue
		}
		if !result.IsError || !strings.Contains(toolText(result), `"code"`) {
			t.Errorf("registered tool %s did not return a recoverable application error: %s", tool.Name, toolText(result))
		}
	}
	if _, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "forecast_reveal", Arguments: map[string]any{}}); err == nil {
		t.Fatal("disabled reveal direct call did not return unknown-tool protocol error")
	}

	result, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:ledger.json"}})
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
	targetCheck, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "target_check", Arguments: map[string]any{"file": "main:ledger.json", "question": "q-election-coalition", "forecast": "f-election-coalition-001"}})
	if err != nil || targetCheck.IsError || !strings.Contains(toolText(targetCheck), `"state":"not_applicable"`) || !strings.Contains(toolText(targetCheck), "content.no_retained_target") {
		t.Fatalf("MCP unretained target result=%s err=%v", toolText(targetCheck), err)
	}
	forecastShow, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "forecast_show", Arguments: map[string]any{"file": "main:ledger.json", "question": "q-election-coalition", "forecast": "f-election-coalition-001"}})
	if err != nil || forecastShow.IsError || !strings.Contains(toolText(forecastShow), `"integrity":{"status":"unanchored"}`) {
		t.Fatalf("MCP forecast integrity result=%s err=%v", toolText(forecastShow), err)
	}
	semanticFailure, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{"file": "main:ledger.json", "platform": "invalid-semantic", "input": map[string]any{"name": "   ", "kind": "informal"}}})
	if err != nil || !semanticFailure.IsError || strings.Contains(toolText(semanticFailure), `"line":0`) || strings.Contains(toolText(semanticFailure), `"column":0`) {
		t.Fatalf("MCP semantic diagnostic fabricated a span: %s, %v", toolText(semanticFailure), err)
	}

	lock, err := storage.AcquireLedgerLock(ctx, filepath.Join(ledgerRoot, "ledger.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	conflict, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{
		"file": "main:ledger.json", "platform": "locked-platform", "input": map[string]any{"name": "Locked", "kind": "internal"},
	}})
	if releaseErr := lock.Release(); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if err != nil || !conflict.IsError || !strings.Contains(toolText(conflict), string(app.CodeConflict)) || time.Since(started) > time.Second {
		t.Fatalf("same-ledger writer conflict result=%s err=%v", toolText(conflict), err)
	}

	copyFixture(t, filepath.Join("..", "..", "schema", "testdata", "forecast-ledger", "v1.0.0", "individual-ledger.json"), filepath.Join(ledgerRoot, "second.json"))
	independent, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{
		"file": "main:second.json", "platform": "independent-platform", "input": map[string]any{"name": "Independent", "kind": "internal"},
	}})
	if err != nil || independent.IsError {
		t.Fatalf("cross-ledger mutation was blocked: result=%s err=%v", toolText(independent), err)
	}

	unknown, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:ledger.json", "typo": true}})
	if err != nil {
		t.Fatal(err)
	}
	if !unknown.IsError || !strings.Contains(toolText(unknown), "unknown field") {
		t.Fatalf("unknown property accepted: %#v", unknown)
	}

	escape, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "main:../outside.json"}})
	if err != nil {
		t.Fatal(err)
	}
	escapeText := toolText(escape)
	if !escape.IsError || !strings.Contains(escapeText, `"route":"ledger:main"`) || !strings.Contains(escapeText, `"flag":"--ledger-root"`) || strings.Contains(escapeText, ledgerRoot) {
		t.Fatalf("root traversal diagnostic is incomplete or unsafe: %s", escapeText)
	}
	missingRoot, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "ledger_validate", Arguments: map[string]any{"file": "missing:ledger.json"}})
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
	if _, err := client2.CallTool(ctx, &sdk.CallToolParams{Name: "platform_add", Arguments: map[string]any{"file": "main:missing.json", "platform": "x", "input": map[string]any{"name": "X", "kind": "informal"}}}); err == nil {
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
	allowed := map[string]bool{"file": true, "question": true, "forecast": true, "dry_run": true, "input": true}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64<<10 {
			return
		}
		_, _ = decodeToolArguments(data, 64<<10, allowed, []string{"file"})
	})
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
		arguments["forecaster_id"], arguments["forecaster_name"], arguments["input"] = "me", "Me", map[string]any{}
	case "ledger_update", "platform_add", "platform_update", "question_add", "question_update", "question_resolve", "question_annul", "question_dispute", "forecast_add":
		arguments["input"] = map[string]any{}
	case "forecast_seal":
		arguments["input_file"], arguments["key_file"] = "keys:missing.json", "keys:new.key"
	case "forecast_key_hint_update":
		arguments["key_hint"] = "vault:item"
	case "publication_build":
		arguments["output"] = "packages:new-package"
	case "publication_verify":
		arguments["manifest"] = "packages:missing-manifest.json"
	}
	if name == "question_add" {
		arguments["type"] = "binary"
	}
	if name == "platform_remove" || name == "question_resolve" || name == "question_annul" || name == "question_dispute" {
		arguments["confirm"] = true
	}
	return arguments
}
