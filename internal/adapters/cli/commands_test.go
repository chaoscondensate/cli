package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/buildinfo"
	contractschema "github.com/chaoscondensate/cli/internal/schema"
	urfavecli "github.com/urfave/cli/v3"
)

func TestCommandTreeAndLeafLocalFileFlags(t *testing.T) {
	root := NewCommand(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	for _, name := range []string{"init", "validate", "status", "platform", "question", "forecast", "target", "timestamp", "verify", "publish", "mcp", "version"} {
		if root.Command(name) == nil {
			t.Errorf("root command %q missing", name)
		}
	}
	for _, name := range []string{"init", "platform", "question", "forecast", "target", "timestamp", "verify", "publish", "mcp"} {
		if !root.Command(name).Hidden {
			t.Errorf("planned root command %q is visible", name)
		}
	}
	for _, name := range []string{"validate", "status", "version"} {
		if root.Command(name).Hidden {
			t.Errorf("working root command %q is hidden", name)
		}
	}
	expectedGroups := map[string][]string{
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
	for _, groupName := range []string{"platform", "question", "forecast", "target", "timestamp", "publish"} {
		for _, flag := range root.Command(groupName).Flags {
			if flag.Names()[0] == "file" {
				t.Errorf("--file leaked onto parent command %s", groupName)
			}
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

	code, stdout, stderr = runCLIWithStdin(`{"secret":"do-not-print"}`, "forecast-ledger", "--json", "validate", "--file", "-")
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
	if code != 3 || stdout != "" || !strings.Contains(stderr, "schema.required") || !strings.Contains(stderr, "/") {
		t.Fatalf("human validation details missing: code=%d stdout=%q stderr=%q", code, stdout, stderr)
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
	for _, visible := range []string{"validate", "status", "version", "completion"} {
		if !strings.Contains(stdout, visible) {
			t.Errorf("root help missing working command %q", visible)
		}
	}
	for _, hidden := range []string{"platform", "question", "forecast", "timestamp", "publish", "mcp"} {
		if strings.Contains(stdout, "\n   "+hidden+" ") {
			t.Errorf("root help advertises planned command %q", hidden)
		}
	}

	code, stdout, stderr = runCLI("forecast-ledger", "init", "--file", "ledger.yaml", "--ledger-id", "ledger", "--timezone", "UTC", "--forecaster-id", "forecaster", "--forecaster-name", "Forecaster")
	if code != 10 || stdout != "" || !strings.Contains(stderr, "not available in this preview release") {
		t.Fatalf("unavailable command code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	code, stdout, stderr = runCLI("forecast-ledger", "--json", "init", "--file", "ledger.yaml", "--ledger-id", "ledger", "--timezone", "UTC", "--forecaster-id", "forecaster", "--forecaster-name", "Forecaster")
	if code != 10 || stdout != "" {
		t.Fatalf("JSON unavailable command code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil || envelope.Code != "unavailable" {
		t.Fatalf("JSON unavailable error=%v envelope=%#v raw=%q", err, envelope, stderr)
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
