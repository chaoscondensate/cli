package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/buildinfo"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
	"github.com/chaoscondensate/cli/internal/storage"
	urfavecli "github.com/urfave/cli/v3"
)

func TestCommandTreeAndLeafLocalFileFlags(t *testing.T) {
	root := NewCommand(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	for _, name := range []string{"init", "ledger", "validate", "status", "platform", "question", "forecast", "target", "timestamp", "verify", "publish", "mcp", "version"} {
		if root.Command(name) == nil {
			t.Errorf("root command %q missing", name)
		}
	}
	for _, name := range []string{"init", "ledger", "platform", "question", "forecast", "target", "timestamp", "verify", "publish", "mcp", "validate", "status", "version"} {
		if root.Command(name).Hidden {
			t.Errorf("working root command %q is hidden", name)
		}
	}
	expectedGroups := map[string][]string{
		"ledger":    {"update"},
		"platform":  {"add", "update", "list", "show", "remove"},
		"question":  {"add", "update", "list", "show", "resolve", "annul", "dispute"},
		"forecast":  {"add", "list", "show", "seal", "reveal"},
		"target":    {"build", "check"},
		"timestamp": {"stamp", "upgrade", "status", "verify"},
		"publish":   {"build", "verify"},
		"mcp":       {"serve"},
	}
	for groupName, children := range expectedGroups {
		group := root.Command(groupName)
		for _, childName := range children {
			child := group.Command(childName)
			if child == nil {
				t.Errorf("command %s %s missing", groupName, childName)
				continue
			}
			if groupName != "mcp" {
				assertRequiredFileFlag(t, child)
			}
		}
	}
	for _, leafName := range []string{"init", "validate", "status", "verify"} {
		assertRequiredFileFlag(t, root.Command(leafName))
	}
	for _, groupName := range []string{"ledger", "platform", "question", "forecast", "target", "timestamp", "publish"} {
		for _, flag := range root.Command(groupName).Flags {
			if flag.Names()[0] == "file" {
				t.Errorf("--file leaked onto parent command %s", groupName)
			}
		}
	}
}

func TestEveryVisibleLeafHasARealAction(t *testing.T) {
	root := NewCommand(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err := root.Walk(func(command *urfavecli.Command) error {
		if command == root || command.Hidden || len(command.Commands) > 0 {
			return nil
		}
		if command.Action == nil {
			t.Errorf("visible leaf %q has no action", command.FullName())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEveryVisibleLeafHasSafeEnglishHelpAndExample(t *testing.T) {
	command := NewCommand(strings.NewReader(""), io.Discard, io.Discard)
	globalFlags := map[string]bool{}
	for _, flag := range command.Flags {
		for _, name := range flag.Names() {
			globalFlags[name] = true
		}
	}
	for _, required := range []string{"json", "plain", "quiet", "no-color", "no-input", "yes", "timeout"} {
		if !globalFlags[required] {
			t.Errorf("root help is missing --%s", required)
		}
	}
	var visit func(prefix string, current *urfavecli.Command)
	visit = func(prefix string, current *urfavecli.Command) {
		path := strings.TrimSpace(prefix + " " + current.Name)
		if len(current.Commands) > 0 {
			for _, child := range current.Commands {
				if !child.Hidden {
					visit(path, child)
				}
			}
			return
		}
		if current.Hidden {
			return
		}
		if !strings.Contains(current.Description, "Example:\n  forecast-ledger ") {
			t.Errorf("%s has no concrete English example", path)
		}
		lower := strings.ToLower(current.Description + " " + current.Usage)
		for _, unsafe := range []string{"password=", "key_hex", "private key", "bearer "} {
			if strings.Contains(lower, unsafe) {
				t.Errorf("%s help contains unsafe secret-like text %q", path, unsafe)
			}
		}
		if path == "forecast-ledger version" || path == "forecast-ledger mcp serve" {
			return
		}
		hasFile := false
		for _, flag := range current.Flags {
			for _, name := range flag.Names() {
				hasFile = hasFile || name == "file"
			}
		}
		if !hasFile {
			t.Errorf("%s help has no leaf-local --file", path)
		}
	}
	for _, child := range command.Commands {
		if !child.Hidden {
			visit("forecast-ledger", child)
		}
	}
}

func TestRequiredFileDashAndTargetSelectionUseUrfave(t *testing.T) {
	code, _, stderr := runCLI("forecast-ledger", "validate")
	if code != 2 || !strings.Contains(stderr, `Required flag "file" not set`) {
		t.Fatalf("missing file code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "init", "--file", "-")
	if code != 2 || !strings.Contains(stderr, "only available for eligible read-only commands") {
		t.Fatalf("mutating stdin code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "validate", "--file", "-")
	if code != 3 || !strings.Contains(stderr, "ledger cannot be parsed") {
		t.Fatalf("read-only stdin did not reach action: code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "target", "check", "--file", "ledger.yaml")
	if code != 2 || !strings.Contains(stderr, "use --all or both") {
		t.Fatalf("missing selector code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "target", "check", "--file", "ledger.yaml", "--all", "--question", "q")
	if code != 2 || !strings.Contains(stderr, "--all cannot be combined") {
		t.Fatalf("mixed selector code=%d stderr=%q", code, stderr)
	}
}

func TestValidateAndStatusFromFilesAndStdin(t *testing.T) {
	jsonFixture := fixtureBytes(t, "individual-ledger.json")
	yamlFixture := fixtureBytes(t, "team-ledger.yaml")
	temporary := t.TempDir()
	jsonPath := filepath.Join(temporary, "ledger.json")
	yamlPath := filepath.Join(temporary, "ledger.yaml")
	if err := os.WriteFile(jsonPath, jsonFixture, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(yamlPath, yamlFixture, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{name: "json file", args: []string{"forecast-ledger", "validate", "--file", jsonPath}, want: "Ledger is valid"},
		{name: "yaml file", args: []string{"forecast-ledger", "validate", "--file", yamlPath}, want: "Ledger is valid"},
		{name: "json stdin", stdin: string(jsonFixture), args: []string{"forecast-ledger", "validate", "--file", "-"}, want: "Ledger is valid"},
		{name: "yaml stdin", stdin: string(yamlFixture), args: []string{"forecast-ledger", "status", "--file", "-"}, want: "example-research-team: 1 questions, 1 forecasts"},
		{name: "status", args: []string{"forecast-ledger", "status", "--file", jsonPath}, want: "alex-example-forecasts: 4 questions, 5 forecasts"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runCLIWithStdin(test.stdin, test.args...)
			if code != 0 || stderr != "" || !strings.Contains(stdout, test.want) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
		})
	}
}

func TestValidateJSONOutputAndStableInvalidData(t *testing.T) {
	valid := string(fixtureBytes(t, "team-ledger.yaml"))
	code, stdout, stderr := runCLIWithStdin(valid, "forecast-ledger", "--json", "validate", "--file", "-")
	if code != 0 || stderr != "" {
		t.Fatalf("valid JSON mode code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var success struct {
		OK   bool           `json:"ok"`
		Code string         `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &success); err != nil {
		t.Fatal(err)
	}
	if !success.OK || success.Code != "ledger.valid" || success.Data["ledger_id"] != "example-research-team" {
		t.Fatalf("unexpected success envelope: %#v", success)
	}

	code, stdout, stderr = runCLIWithStdin(`{"schema_version":"1.0.0","secret":"do-not-print"}`, "forecast-ledger", "--json", "validate", "--file", "-")
	if code != 3 || stdout != "" {
		t.Fatalf("invalid JSON mode code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var failure struct {
		OK      bool           `json:"ok"`
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal([]byte(stderr), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.OK || failure.Code != "invalid_data" || failure.Details["issues"] == nil {
		t.Fatalf("unexpected failure envelope: %#v", failure)
	}
	if strings.Contains(stdout+stderr, "do-not-print") {
		t.Fatal("invalid ledger contents leaked into output")
	}

	code, stdout, stderr = runCLIWithStdin(`{}`, "forecast-ledger", "validate", "--file", "-")
	if code != 3 || stdout != "" || !strings.Contains(stderr, "only 1.0.0") {
		t.Fatalf("unsupported schema message missing: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestForecastAddListShowAndDryRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"forecasted_at":"2026-09-01T09:00:00+01:00","recorded_at":"2026-09-01T09:01:00+01:00","value":{"kind":"multiple_choice","probabilities":[{"option_id":"centre-left","probability_bp":5000},{"option_id":"centre-right","probability_bp":3500},{"option_id":"other","probability_bp":1500}]},"supersedes_forecast_id":"f-election-coalition-001"}`
	code, stdout, stderr := runCLIWithStdin(input, "forecast-ledger", "--json", "forecast", "add", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-002", "--input", "-")
	if code != 0 || stderr != "" {
		t.Fatalf("add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var added struct {
		Code string `json:"code"`
		Data struct {
			RecordedAt string `json:"recorded_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil || added.Code != "forecast.added" || added.Data.RecordedAt != "2026-09-01T09:01:00+01:00" {
		t.Fatalf("add envelope error=%v value=%#v", err, added)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "forecast", "list", "--file", path, "--question", "q-election-coalition")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "f-election-coalition-001") || !strings.Contains(stdout, "f-election-coalition-002") {
		t.Fatalf("list code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLIWithStdin(string(updated), "forecast-ledger", "--json", "forecast", "show", "--file", "-", "--question", "q-election-coalition", "--forecast", "f-election-coalition-002")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"visibility":"public"`) || !strings.Contains(stdout, `"probability_bp":5000`) {
		t.Fatalf("show code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	dryInput := strings.Replace(input, "f-election-coalition-001", "f-election-coalition-002", 1)
	before := append([]byte(nil), updated...)
	code, stdout, stderr = runCLIWithStdin(dryInput, "forecast-ledger", "forecast", "add", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-003", "--input", "-", "--dry-run")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "no file was changed") {
		t.Fatalf("dry-run code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("forecast dry-run changed ledger")
	}
}

func TestQuestionAuthoringAndLifecycleCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	addInput := `{"title":"Will the new event happen?","resolution_criteria":"Resolve from the official notice.","created_at":"2026-08-20T00:00:00Z","forecast_window":{"closes_at":"2026-12-01T00:00:00Z"},"expected_resolution_at":"2026-12-02T00:00:00Z","initial_forecast":{"id":"f-new-001","visibility":"public","forecasted_at":"2026-08-20T00:00:00Z","recorded_at":"2026-08-20T00:01:00Z","value":{"kind":"binary","probability_bp":5500}}}`
	code, stdout, stderr := runCLIWithStdin(addInput, "forecast-ledger", "--json", "question", "add", "--file", path, "--question", "q-new", "--type", "binary", "--input", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"question.added"`) {
		t.Fatalf("question add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "question", "list", "--file", path)
	if code != 0 || stderr != "" || !strings.Contains(stdout, "q-new\tbinary\topen\t1") {
		t.Fatalf("question list code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLIWithStdin(string(updated), "forecast-ledger", "--json", "question", "show", "--file", "-", "--question", "q-new")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"forecast_count"`) && !strings.Contains(stdout, `"f-new-001"`) {
		t.Fatalf("question show code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLIWithStdin(`{"status":"closed"}`, "forecast-ledger", "question", "update", "--file", path, "--question", "q-new", "--input", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "updated") {
		t.Fatalf("question close code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	resolution := `{"outcome":true,"outcome_known_at":"2026-12-02T00:00:00Z","recorded_at":"2026-12-02T00:01:00Z","sources":[{"title":"Official notice","url":"https://example.org/notice","retrieved_at":"2026-12-02T00:00:30Z"}]}`
	code, stdout, stderr = runCLIWithStdin(resolution, "forecast-ledger", "--json", "question", "resolve", "--file", path, "--question", "q-new", "--input", "-", "--yes")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"question.resolved"`) || !strings.Contains(stdout, `"status":"resolved"`) {
		t.Fatalf("question resolve code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	dispute := `{"reason":"The notice is being reviewed.","recorded_at":"2026-12-03T00:00:00Z"}`
	code, stdout, stderr = runCLIWithStdin(dispute, "forecast-ledger", "--json", "question", "dispute", "--file", path, "--question", "q-new", "--input", "-", "--yes")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"prior_status":"resolved"`) || !strings.Contains(stdout, `"resolution_history_external"`) {
		t.Fatalf("question dispute code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestQuestionAddSealedDoesNotLeakInput(t *testing.T) {
	const canary = "PRIVATE-QUESTION-CANARY"
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "f-secret.key")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"title":"Secret forecast question","resolution_criteria":"Resolve from the official notice.","created_at":"2026-08-20T00:00:00Z","forecast_window":{"closes_at":"2026-12-01T00:00:00Z"},"expected_resolution_at":"2026-12-02T00:00:00Z","initial_forecast":{"id":"f-secret","visibility":"sealed","forecasted_at":"2026-08-20T00:00:00Z","recorded_at":"2026-08-20T00:01:00Z","value":{"kind":"binary","probability_bp":5500},"rationale":"` + canary + `","key_factors":[],"comment":"private"}}`
	code, stdout, stderr := runCLIWithStdin(input, "forecast-ledger", "--json", "question", "add", "--file", path, "--question", "q-secret", "--type", "binary", "--input", "-", "--key-file", keyPath)
	if code != 0 || stderr != "" || strings.Contains(stdout+stderr, canary) {
		t.Fatalf("sealed add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	ledgerBytes, _ := os.ReadFile(path)
	if strings.Contains(string(ledgerBytes), canary) {
		t.Fatal("sealed question plaintext leaked into ledger")
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("sealed question key missing: %v", err)
	}
}

func TestTargetBuildAndCheckCommands(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	selector := []string{"--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-001"}
	dryArgs := append([]string{"forecast-ledger", "target", "build"}, selector...)
	dryArgs = append(dryArgs, "--dry-run")
	code, stdout, stderr := runCLI(dryArgs...)
	if code != 0 || stderr != "" || !strings.Contains(stdout, "no files were written") {
		t.Fatalf("target dry-run code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(directory, "proofs")); !os.IsNotExist(err) {
		t.Fatalf("target dry-run created directory: %v", err)
	}
	buildArgs := append([]string{"forecast-ledger", "--json", "target", "build"}, selector...)
	code, stdout, stderr = runCLI(buildArgs...)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"target.built"`) || !strings.Contains(stdout, `"path":"proofs/targets/f-election-coalition-001.json"`) {
		t.Fatalf("target build code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	checkArgs := append([]string{"forecast-ledger", "--json", "target", "check"}, selector...)
	code, stdout, stderr = runCLI(checkArgs...)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"target.valid"`) || !strings.Contains(stdout, `"valid":true`) {
		t.Fatalf("target check code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "target", "check", "--file", "-", "--question", "q", "--forecast", "f")
	if code != 2 || !strings.Contains(stderr, "only available for eligible read-only commands") {
		t.Fatalf("target stdin code=%d stderr=%q", code, stderr)
	}
}

func TestForecastSealRevealAndKeyHintCommands(t *testing.T) {
	const canary = "PRIVATE-FORECAST-CANARY"
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "f-election-coalition-002.key")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	input := `{"forecasted_at":"2026-08-25T09:00:00+01:00","recorded_at":"2026-08-25T09:01:00+01:00","value":{"kind":"multiple_choice","probabilities":[{"option_id":"centre-left","probability_bp":5000},{"option_id":"centre-right","probability_bp":3500},{"option_id":"other","probability_bp":1500}]},"rationale":"` + canary + `","key_factors":[],"comment":"private","supersedes_forecast_id":"f-election-coalition-001"}`
	code, stdout, stderr := runCLIWithStdin(input, "forecast-ledger", "--json", "forecast", "seal", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-002", "--input", "-", "--key-file", keyPath)
	if code != 0 || stderr != "" || strings.Contains(stdout+stderr, canary) || !strings.Contains(stdout, `"code":"forecast.sealed"`) {
		t.Fatalf("forecast seal code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	ledgerBytes, _ := os.ReadFile(path)
	if strings.Contains(string(ledgerBytes), canary) {
		t.Fatal("sealed forecast plaintext leaked into ledger")
	}
	code, _, stderr = runCLI("forecast-ledger", "forecast", "reveal", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-002", "--key-file", keyPath)
	if code != 2 || !strings.Contains(stderr, "use --yes") {
		t.Fatalf("unapproved reveal code=%d stderr=%q", code, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "forecast", "reveal", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-002", "--key-file", keyPath, "--revealed-at", "2026-08-26T16:00:00+01:00", "--yes")
	if code != 0 || stderr != "" || strings.Contains(stdout+stderr, canary) || !strings.Contains(stdout, `"code":"forecast.revealed"`) {
		t.Fatalf("forecast reveal code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "forecast", "key-hint", "update", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-002", "--key-hint", "vault:item-42")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"forecast.key_hint.updated"`) {
		t.Fatalf("key hint update code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	data, err := fs.ReadFile(contractschema.Conformance(), name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHelpSuggestionsCompletionAndVersionJSON(t *testing.T) {
	code, stdout, stderr := runCLI("forecast-ledger", "forecast", "show", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("help code=%d stderr=%q", code, stderr)
	}
	for _, expected := range []string{"Example:", "--file string, -f string", "--question string", "--forecast string"} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("help missing %q:\n%s", expected, stdout)
		}
	}

	code, stdout, stderr = runCLI("forecast-ledger", "validte")
	if code == 0 || (!strings.Contains(stdout+stderr, "validate") && !strings.Contains(stdout+stderr, "Did you mean")) {
		t.Fatalf("typo guidance missing: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, shell := range []string{"bash", "zsh", "fish", "pwsh"} {
		code, stdout, stderr = runCLI("forecast-ledger", "completion", shell)
		if code != 0 || stdout == "" || stderr != "" {
			t.Errorf("completion %s code=%d stdout=%d bytes stderr=%q", shell, code, len(stdout), stderr)
		}
		if shell == "bash" && (!strings.Contains(stdout, "init") || !strings.Contains(stdout, "ledger")) {
			t.Errorf("completion %s omits a working authoring command", shell)
		}
	}

	for _, args := range [][]string{{"forecast-ledger", "--json", "version"}, {"forecast-ledger", "version", "--json"}} {
		code, stdout, stderr = runCLI(args...)
		if code != 0 || stderr != "" {
			t.Fatalf("version %v code=%d stderr=%q", args, code, stderr)
		}
		var got buildinfo.Info
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatal(err)
		}
		if got.Binary != "forecast-ledger" || got.Schema.Version != "1.0.0" {
			t.Fatalf("unexpected version metadata: %#v", got)
		}
	}
	code, _, stderr = runCLI("forecast-ledger", "--plain", "version", "--json")
	if code != 2 || !strings.Contains(stderr, "cannot be combined") {
		t.Fatalf("mixed global/local output modes code=%d stderr=%q", code, stderr)
	}
}

func TestRootHelpShowsOnlyWorkingCommandsAndUnavailableHasStableExit(t *testing.T) {
	code, stdout, stderr := runCLI("forecast-ledger", "--help")
	if code != 0 || stderr != "" {
		t.Fatalf("root help code=%d stderr=%q", code, stderr)
	}
	for _, visible := range []string{"init", "ledger", "platform", "question", "forecast", "target", "timestamp", "verify", "publish", "mcp", "validate", "status", "version", "completion"} {
		if !strings.Contains(stdout, visible) {
			t.Errorf("root help missing working command %q", visible)
		}
	}
}

func TestLayeredVerifyCLIProducesStableReport(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "verify", "--file", path, "--question", "q-election-coalition", "--forecast", "f-election-coalition-001", "--offline")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"verification.pass"`) || !strings.Contains(stdout, `"limitations"`) {
		t.Fatalf("verify code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestTimestampStatusOfflineFailureAndPublicationCLI(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.json")
	if err := os.WriteFile(ledgerPath, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	selector := []string{"--file", ledgerPath, "--question", "q-election-coalition", "--forecast", "f-election-coalition-001"}
	args := append([]string{"forecast-ledger", "--json", "timestamp", "status"}, selector...)
	code, stdout, stderr := runCLI(args...)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"state":"unanchored"`) {
		t.Fatalf("timestamp status code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	args = append([]string{"forecast-ledger", "timestamp", "stamp"}, selector...)
	args = append(args, "--offline")
	code, stdout, stderr = runCLI(args...)
	if code != 8 || stdout != "" || !strings.Contains(stderr, "network access") {
		t.Fatalf("offline stamp code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	output := filepath.Join(directory, "package")
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "publish", "build", "--file", ledgerPath, "--output", output)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"publication.built"`) {
		t.Fatalf("publish build code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	packageLedger := filepath.Join(output, "ledger", "ledger.json")
	manifest := filepath.Join(output, "manifest.json")
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "publish", "verify", "--file", packageLedger, "--manifest", manifest)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"overall":"pass"`) {
		t.Fatalf("publish verify code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestInitCreatesValidPublicLedgerAndSupportsDryRun(t *testing.T) {
	input := `{"created_at":"2026-01-01T00:00:00Z","question":{"id":"q-one","title":"Will it happen?","type":"binary","resolution_criteria":"Resolve from the named source.","forecast_window":{"closes_at":"2026-12-31T00:00:00Z"},"expected_resolution_at":"2027-01-01T00:00:00Z","initial_forecast":{"id":"f-one","visibility":"public","forecasted_at":"2026-01-01T00:00:00Z","value":{"kind":"binary","probability_bp":5000}}}}`
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	args := []string{"forecast-ledger", "--json", "init", "--file", path, "--ledger-id", "research", "--timezone", "UTC", "--forecaster-id", "andrey", "--forecaster-name", "Andrey", "--input", "-"}
	code, stdout, stderr := runCLIWithStdin(input, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("init code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var envelope struct {
		Code string `json:"code"`
		Data struct {
			LedgerID   string `json:"ledger_id"`
			Visibility string `json:"visibility"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != "ledger.initialized" || envelope.Data.LedgerID != "research" || envelope.Data.Visibility != "public" {
		t.Fatalf("init envelope = %#v", envelope)
	}
	code, _, stderr = runCLI("forecast-ledger", "validate", "--file", path)
	if code != 0 || stderr != "" {
		t.Fatalf("created ledger validation code=%d stderr=%q", code, stderr)
	}
	dryPath := filepath.Join(directory, "dry.yaml")
	dryArgs := []string{"forecast-ledger", "init", "--file", dryPath, "--ledger-id", "research-two", "--timezone", "UTC", "--forecaster-id", "andrey", "--forecaster-name", "Andrey", "--input", "-", "--dry-run"}
	code, stdout, stderr = runCLIWithStdin(input, dryArgs...)
	if code != 0 || stderr != "" || !strings.Contains(stdout, "no files were written") {
		t.Fatalf("dry-run code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Stat(dryPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run created ledger: %v", err)
	}
}

func TestInitSealedCreatesProtectedKeyWithoutLeakingPrivateInput(t *testing.T) {
	const secret = "PRIVATE-CANARY-DO-NOT-PRINT"
	input := `{"created_at":"2026-01-01T00:00:00Z","question":{"id":"q-one","title":"Will it happen?","type":"binary","resolution_criteria":"Resolve from the named source.","forecast_window":{"closes_at":"2026-12-31T00:00:00Z"},"expected_resolution_at":"2027-01-01T00:00:00Z","initial_forecast":{"id":"f-one","visibility":"sealed","forecasted_at":"2026-01-01T00:00:00Z","value":{"kind":"binary","probability_bp":5000},"rationale":"` + secret + `","key_factors":[],"comment":"private"}}}`
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ledger.yaml")
	keyPath := filepath.Join(directory, "f-one.key")
	code, stdout, stderr := runCLIWithStdin(input, "forecast-ledger", "--json", "init", "--file", ledgerPath, "--ledger-id", "research", "--timezone", "UTC", "--forecaster-id", "andrey", "--forecaster-name", "Andrey", "--input", "-", "--key-file", keyPath)
	if code != 0 || stderr != "" || strings.Contains(stdout+stderr, secret) {
		t.Fatalf("sealed init code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	ledgerBytes, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ledgerBytes), secret) {
		t.Fatal("sealed ledger contains private input")
	}
	if err := storage.CheckProtectedFile(keyPath); err != nil {
		t.Fatal(err)
	}
	code, _, stderr = runCLI("forecast-ledger", "validate", "--file", ledgerPath)
	if code != 0 || stderr != "" {
		t.Fatalf("sealed ledger validation code=%d stderr=%q", code, stderr)
	}
}

func TestLedgerUpdateCommitsClosedPatchAndDryRunDoesNotWrite(t *testing.T) {
	raw := fixtureBytes(t, "individual-ledger.json")
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLIWithStdin(`{"title":"Updated ledger","description":null}`, "forecast-ledger", "--json", "ledger", "update", "--file", path, "--input", "-")
	if code != 0 || stderr != "" {
		t.Fatalf("update code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var envelope struct {
		Code string `json:"code"`
		Data struct {
			Changed         bool     `json:"changed"`
			ChangedPointers []string `json:"changed_pointers"`
			Warnings        []struct {
				Code string `json:"code"`
			} `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != "ledger.updated" || !envelope.Data.Changed || len(envelope.Data.ChangedPointers) != 2 || len(envelope.Data.Warnings) != 2 {
		t.Fatalf("update envelope = %#v", envelope)
	}
	afterUpdate, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(afterUpdate), `"title": "Updated ledger"`) || strings.Contains(string(afterUpdate), `"description":`) {
		t.Fatalf("patch not applied:\n%s", afterUpdate)
	}
	code, stdout, stderr = runCLIWithStdin(`{"default_timezone":"UTC"}`, "forecast-ledger", "ledger", "update", "--file", path, "--input", "-", "--dry-run")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "no file was changed") {
		t.Fatalf("update dry-run code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterDryRun, afterUpdate) {
		t.Fatal("ledger update dry-run changed file")
	}
}

func TestLedgerUpdateReturnsConflictWhenLedgerIsLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := storage.AcquireLedgerLock(context.Background(), path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	code, stdout, stderr := runCLIWithStdin(`{"title":"Blocked"}`, "forecast-ledger", "ledger", "update", "--file", path, "--input", "-")
	if code != 5 || stdout != "" || !strings.Contains(stderr, "locked by another operation") {
		t.Fatalf("locked update code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestPlatformCommandsCoverMutationReadsStdinAndApproval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := os.WriteFile(path, fixtureBytes(t, "individual-ledger.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLIWithStdin(`{"name":"New platform","kind":"internal"}`, "forecast-ledger", "platform", "add", "--file", path, "--platform", "new-platform", "--input", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "Platform was added") {
		t.Fatalf("platform add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLIWithStdin(`{"name":"Renamed platform","url":"https://example.net/platform"}`, "forecast-ledger", "--json", "platform", "update", "--file", path, "--platform", "new-platform", "--input", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"code":"platform.updated"`) || !strings.Contains(stdout, `"name":"Renamed platform"`) {
		t.Fatalf("platform update code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	ledgerBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLIWithStdin(string(ledgerBytes), "forecast-ledger", "platform", "list", "--file", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "local\tself_hosted") || !strings.Contains(stdout, "new-platform\tinternal") {
		t.Fatalf("platform list code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLIWithStdin(string(ledgerBytes), "forecast-ledger", "--json", "platform", "show", "--file", "-", "--platform", "new-platform")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"platform_id"`) && !strings.Contains(stdout, `"id":"new-platform"`) {
		t.Fatalf("platform show code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "platform", "remove", "--file", path, "--platform", "new-platform")
	if code != 2 || !strings.Contains(stderr, "use --yes") {
		t.Fatalf("unapproved remove code=%d stderr=%q", code, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "platform", "remove", "--file", path, "--platform", "new-platform", "--yes")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "Platform was removed") {
		t.Fatalf("platform remove code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "platform", "remove", "--file", path, "--platform", "metaculus", "--yes")
	if code != 5 || stdout != "" || !strings.Contains(stderr, `"question_ids"`) {
		t.Fatalf("referenced remove code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func assertRequiredFileFlag(t *testing.T, command *urfavecli.Command) {
	t.Helper()
	for _, flag := range command.Flags {
		if flag.Names()[0] == "file" {
			requiredFlag, ok := flag.(interface{ IsRequired() bool })
			if !ok || !requiredFlag.IsRequired() || !contains(flag.Names(), "f") {
				t.Errorf("%s file flag required=%v names=%v", command.FullName(), ok && requiredFlag.IsRequired(), flag.Names())
			}
			return
		}
	}
	t.Errorf("%s has no --file flag", command.FullName())
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func runCLI(arguments ...string) (int, string, string) {
	return runCLIWithStdin("", arguments...)
}

func runCLIWithStdin(stdin string, arguments ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), arguments, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
