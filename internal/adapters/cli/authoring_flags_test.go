package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/chaoscondensate/cli/internal/buildinfo"
	"github.com/chaoscondensate/cli/internal/presentation"
	"github.com/chaoscondensate/cli/internal/service"
	urfavecli "github.com/urfave/cli/v3"
)

func TestAuthoringInventoryIsClassifiedAndOrdinaryInputIsOptional(t *testing.T) {
	root := NewCommand(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	for _, entry := range authoringInventory {
		if entry.Path == "" || entry.Schema == "" || len(entry.Fields) == 0 {
			t.Errorf("incomplete authoring inventory entry: %#v", entry)
		}
		for _, field := range entry.Fields {
			if field.Field == "" || field.Class == "" {
				t.Errorf("unclassified field in %s: %#v", entry.Path, field)
			}
			if field.Class == "service-only" {
				if field.Route != "" || strings.TrimSpace(field.Rationale) == "" {
					t.Errorf("service-only field lacks a rationale in %s: %#v", entry.Path, field)
				}
				continue
			}
			if field.Route == "" {
				t.Errorf("routed field lacks a route in %s: %#v", entry.Path, field)
			}
			if field.Class == "secret" && !entry.Protected {
				t.Errorf("secret field on ordinary command %s: %#v", entry.Path, field)
			}
		}
		command := commandAtPath(root, entry.Path)
		if command == nil {
			t.Errorf("inventory command %q is not registered", entry.Path)
			continue
		}
		for _, flag := range command.Flags {
			if flag.Names()[0] != "input" {
				continue
			}
			stringFlag, ok := flag.(*urfavecli.StringFlag)
			if !ok {
				t.Errorf("%s input flag has unexpected type %T", entry.Path, flag)
				continue
			}
			if !entry.Protected && stringFlag.Required {
				t.Errorf("ordinary authoring command %s requires --input", entry.Path)
			}
		}
	}
}

func TestEveryAuthoringLeafReachesDirectModeWithoutRequiredInput(t *testing.T) {
	tests := []struct {
		name string
		args func(path, privatePath, keyPath string) []string
	}{
		{name: "ledger update", args: func(path, _, _ string) []string {
			return []string{"ledger", "update", "--file", path, "--title", "Updated"}
		}},
		{name: "platform add", args: func(path, _, _ string) []string {
			return []string{"platform", "add", "--file", path, "--platform", "service", "--name", "Service", "--kind", "informal"}
		}},
		{name: "platform update", args: func(path, _, _ string) []string {
			return []string{"platform", "update", "--file", path, "--platform", "missing", "--name", "Updated"}
		}},
		{name: "platform remove", args: func(path, _, _ string) []string {
			return []string{"--yes", "platform", "remove", "--file", path, "--platform", "missing"}
		}},
		{name: "question add", args: func(path, _, _ string) []string {
			return []string{"question", "add", "--file", path, "--question", "q-one", "--type", "binary", "--title", "Question", "--resolution-criteria", "Objective result", "--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-01T00:00:00Z"}
		}},
		{name: "question update", args: func(path, _, _ string) []string {
			return []string{"question", "update", "--file", path, "--question", "missing", "--title", "Updated"}
		}},
		{name: "question resolve", args: func(path, _, _ string) []string {
			return []string{"--yes", "question", "resolve", "--file", path, "--question", "missing", "--outcome-boolean=true", "--outcome-known-at", "2027-01-01T00:00:00Z", "--source", "Result,https://example.com/result,2027-01-01T00:01:00Z"}
		}},
		{name: "question annul", args: func(path, _, _ string) []string {
			return []string{"--yes", "question", "annul", "--file", path, "--question", "missing", "--reason", "Cannot resolve"}
		}},
		{name: "question dispute", args: func(path, _, _ string) []string {
			return []string{"--yes", "question", "dispute", "--file", path, "--question", "missing", "--reason", "Source conflict"}
		}},
		{name: "forecast add", args: func(path, _, _ string) []string {
			return []string{"forecast", "add", "--file", path, "--question", "missing", "--forecast", "f-one", "--forecasted-at", "2026-09-01T00:00:00Z", "--value-kind", "binary", "--probability-bp", "5000"}
		}},
		{name: "forecast seal", args: func(path, privatePath, keyPath string) []string {
			return []string{"forecast", "seal", "--file", path, "--question", "missing", "--forecast", "f-sealed", "--forecasted-at", "2026-09-01T00:00:00Z", "--secret-input", privatePath, "--key-file", keyPath}
		}},
		{name: "forecast reveal", args: func(path, _, keyPath string) []string {
			return []string{"--yes", "forecast", "reveal", "--file", path, "--question", "missing", "--forecast", "f-sealed", "--key-file", keyPath}
		}},
		{name: "forecast key-hint update", args: func(path, _, _ string) []string {
			return []string{"forecast", "key-hint", "update", "--file", path, "--question", "missing", "--forecast", "f-sealed", "--key-hint", "forecast-key:f-sealed"}
		}},
	}

	initPath := filepath.Join(t.TempDir(), "direct-init.json")
	code, _, stderr := runCLI("forecast-ledger", "init", "--file", initPath, "--ledger-id", "direct-init", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner")
	if code != 0 || strings.Contains(stderr, `Required flag "input"`) {
		t.Fatalf("init did not reach direct mode: code=%d stderr=%q", code, stderr)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "ledger.json")
			code, _, stderr := runCLI("forecast-ledger", "init", "--file", path, "--ledger-id", "direct-leaf", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner")
			if code != 0 {
				t.Fatalf("setup init code=%d stderr=%q", code, stderr)
			}
			privatePath := filepath.Join(directory, "private.json")
			if err := os.WriteFile(privatePath, []byte(`{"value":{"kind":"binary","probability_bp":5000},"rationale":"private","key_factors":[],"comment":"private"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			keyPath := filepath.Join(directory, "forecast.key")
			args := append([]string{"forecast-ledger"}, test.args(path, privatePath, keyPath)...)
			_, _, stderr = runCLI(args...)
			if strings.Contains(stderr, `Required flag "input"`) {
				t.Fatalf("authoring leaf still requires --input: %q", stderr)
			}
		})
	}
}

func TestDirectBuildersMatchRetainedDocumentSchemas(t *testing.T) {
	directory := t.TempDir()
	privatePath := filepath.Join(directory, "private.json")
	if err := os.WriteFile(privatePath, []byte(`{"value":{"kind":"binary","probability_bp":6400},"rationale":"Private","key_factors":["Factor"],"comment":"Comment"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		flags       []urfavecli.Flag
		args        []string
		document    string
		schema      service.InputSchemaName
		destination any
		build       func(*urfavecli.Command) (any, error)
	}{
		{
			name: "init", flags: append(rootAuthoringFlags(), initNestedFlags()...),
			args:     []string{"--title", "Ledger", "--contact-email", "owner@example.com", "--profile", "site,https://example.com,owner", "--initial-platform", "local,Local,internal,https://example.com"},
			document: `{"title":"Ledger","contact":{"email":"owner@example.com"},"profiles":[{"service":"site","url":"https://example.com","username":"owner"}],"platforms":{"local":{"name":"Local","kind":"internal","url":"https://example.com"}}}`,
			schema:   service.InputSchemaInit, destination: &service.InitInput{},
			build: func(command *urfavecli.Command) (any, error) {
				return buildInitInput(context.Background(), command, strings.NewReader(""))
			},
		},
		{
			name: "ledger update", flags: rootPatchFlags(),
			args:     []string{"--title", "Updated", "--clear-description", "--timezone", "UTC", "--forecaster-name", "Owner", "--profile", "site,https://example.com,owner"},
			document: `{"title":"Updated","description":null,"default_timezone":"UTC","forecaster":{"name":"Owner","profiles":[{"service":"site","url":"https://example.com","username":"owner"}]}}`,
			schema:   service.InputSchemaRootMetadata, destination: &service.RootMetadataPatchInput{},
			build: func(command *urfavecli.Command) (any, error) { return buildRootPatchInput(command) },
		},
		{
			name: "platform add", flags: platformCreateFlags(),
			args:     []string{"--name", "Metaculus", "--kind", "scoring_platform", "--url", "https://www.metaculus.com", "--account-username", "owner"},
			document: `{"name":"Metaculus","kind":"scoring_platform","url":"https://www.metaculus.com","account":{"username":"owner"}}`,
			schema:   service.InputSchemaPlatformCreate, destination: &service.PlatformCreateInput{},
			build: func(command *urfavecli.Command) (any, error) { return buildPlatformCreateInput(command) },
		},
		{
			name: "platform update", flags: platformPatchFlags(),
			args:     []string{"--name", "Updated", "--clear-url", "--account-profile-url", "https://example.com/profile"},
			document: `{"name":"Updated","url":null,"account":{"profile_url":"https://example.com/profile"}}`,
			schema:   service.InputSchemaPlatformPatch, destination: &service.PlatformPatchInput{},
			build: func(command *urfavecli.Command) (any, error) { return buildPlatformPatchInput(command) },
		},
		{
			name: "question add", flags: questionCreateFlags(true),
			args:     []string{"--title", "Question", "--resolution-criteria", "Official result", "--created-at", "2026-08-30T00:00:00Z", "--opens-at", "2026-08-30T00:00:00Z", "--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-01T00:00:00Z", "--platform-ref", "local,q-1,https://example.com/q-1", "--tag", "topic", "--notes", "Note"},
			document: `{"title":"Question","resolution_criteria":"Official result","created_at":"2026-08-30T00:00:00Z","forecast_window":{"opens_at":"2026-08-30T00:00:00Z","closes_at":"2026-12-31T00:00:00Z"},"expected_resolution_at":"2027-01-01T00:00:00Z","platform_refs":[{"platform":"local","question_id":"q-1","url":"https://example.com/q-1"}],"tags":["topic"],"notes":"Note"}`,
			schema:   service.InputSchemaQuestionAdd, destination: &service.QuestionAddInput{},
			build: func(command *urfavecli.Command) (any, error) {
				return buildQuestionAddInput(context.Background(), command, strings.NewReader(""))
			},
		},
		{
			name: "question update", flags: questionPatchFlags(),
			args:     []string{"--title", "Updated", "--closes-at", "2026-11-30T00:00:00Z", "--clear-platform-refs", "--tag", "new", "--clear-notes", "--status", "closed"},
			document: `{"title":"Updated","forecast_window":{"closes_at":"2026-11-30T00:00:00Z"},"platform_refs":null,"tags":["new"],"notes":null,"status":"closed"}`,
			schema:   service.InputSchemaQuestionPatch, destination: &service.QuestionPatchInput{},
			build: func(command *urfavecli.Command) (any, error) { return buildQuestionPatchInput(command) },
		},
		{
			name: "forecast add", flags: forecastCreateFlags(),
			args:     []string{"--forecasted-at", "2026-09-01T00:00:00Z", "--recorded-at", "2026-09-01T00:01:00Z", "--value-kind", "binary", "--probability-bp", "6400", "--rationale", "Public", "--key-factor", "Factor", "--comment", "Comment", "--public-note", "Note", "--supersedes-forecast", "f-old"},
			document: `{"forecasted_at":"2026-09-01T00:00:00Z","recorded_at":"2026-09-01T00:01:00Z","value":{"kind":"binary","probability_bp":6400},"rationale":"Public","key_factors":["Factor"],"comment":"Comment","public_note":"Note","supersedes_forecast_id":"f-old"}`,
			schema:   service.InputSchemaForecastCreate, destination: &service.ForecastCreateInput{},
			build: func(command *urfavecli.Command) (any, error) { return buildForecastCreateInput(command) },
		},
		{
			name: "forecast seal", flags: forecastSealPublicFlags(),
			args:     []string{"--secret-input", privatePath, "--forecasted-at", "2026-09-01T00:00:00Z", "--recorded-at", "2026-09-01T00:01:00Z", "--public-note", "Note", "--supersedes-forecast", "f-old"},
			document: `{"forecasted_at":"2026-09-01T00:00:00Z","recorded_at":"2026-09-01T00:01:00Z","value":{"kind":"binary","probability_bp":6400},"rationale":"Private","key_factors":["Factor"],"comment":"Comment","public_note":"Note","supersedes_forecast_id":"f-old"}`,
			schema:   service.InputSchemaForecastSeal, destination: &service.SealedForecastInput{},
			build: func(command *urfavecli.Command) (any, error) {
				return buildSealedForecastInput(context.Background(), command, strings.NewReader(""))
			},
		},
		{
			name: "resolve", flags: lifecycleFlags(true),
			args:     []string{"--outcome-boolean=false", "--outcome-known-at", "2027-01-01T00:00:00Z", "--recorded-at", "2027-01-01T00:01:00Z", "--source", "Result,https://example.com/result,2027-01-01T00:02:00Z,Publisher", "--notes", "Checked"},
			document: `{"outcome":false,"outcome_known_at":"2027-01-01T00:00:00Z","recorded_at":"2027-01-01T00:01:00Z","sources":[{"title":"Result","url":"https://example.com/result","retrieved_at":"2027-01-01T00:02:00Z","publisher":"Publisher"}],"notes":"Checked"}`,
			schema:   service.InputSchemaResolution, destination: &service.ResolutionInput{},
			build: func(command *urfavecli.Command) (any, error) { return buildResolutionInput(command) },
		},
		{
			name: "annul", flags: lifecycleFlags(false),
			args:     []string{"--reason", "Cannot resolve", "--recorded-at", "2027-01-01T00:01:00Z", "--source", "Result,https://example.com/result,2027-01-01T00:02:00Z"},
			document: `{"reason":"Cannot resolve","recorded_at":"2027-01-01T00:01:00Z","sources":[{"title":"Result","url":"https://example.com/result","retrieved_at":"2027-01-01T00:02:00Z"}]}`,
			schema:   service.InputSchemaAnnul, destination: &service.AnnulInput{},
			build: func(command *urfavecli.Command) (any, error) {
				reason, recordedAt, sources, err := buildReasonInput(command)
				return service.AnnulInput{Reason: reason, RecordedAt: recordedAt, Sources: sources}, err
			},
		},
		{
			name: "dispute", flags: lifecycleFlags(false),
			args:     []string{"--reason", "Source conflict", "--recorded-at", "2027-01-01T00:01:00Z", "--source", "Counter,https://example.com/counter,2027-01-01T00:02:00Z"},
			document: `{"reason":"Source conflict","recorded_at":"2027-01-01T00:01:00Z","sources":[{"title":"Counter","url":"https://example.com/counter","retrieved_at":"2027-01-01T00:02:00Z"}]}`,
			schema:   service.InputSchemaDispute, destination: &service.DisputeInput{},
			build: func(command *urfavecli.Command) (any, error) {
				reason, recordedAt, sources, err := buildReasonInput(command)
				return service.DisputeInput{Reason: reason, RecordedAt: recordedAt, Sources: sources}, err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var direct any
			command := &urfavecli.Command{
				Name: "builder", Flags: test.flags, DisableSliceFlagSeparator: true,
				Action: func(_ context.Context, command *urfavecli.Command) error {
					var err error
					direct, err = test.build(command)
					return err
				},
			}
			if err := command.Run(context.Background(), append([]string{"builder"}, test.args...)); err != nil {
				t.Fatalf("direct builder failed: %v", err)
			}
			if err := service.DecodeOperationInput(context.Background(), "-", strings.NewReader(test.document), test.schema, test.destination); err != nil {
				t.Fatalf("document decode failed: %v", err)
			}
			document := reflect.ValueOf(test.destination).Elem().Interface()
			if !reflect.DeepEqual(direct, document) {
				t.Fatalf("direct and document modes differ\ndirect:   %#v\ndocument: %#v", direct, document)
			}
		})
	}
}

func commandAtPath(root *urfavecli.Command, path string) *urfavecli.Command {
	current := root
	for _, name := range strings.Fields(path) {
		current = current.Command(name)
		if current == nil {
			return nil
		}
	}
	return current
}

func TestFlagOnlyAuthoringWorkflowJSONAndYAML(t *testing.T) {
	for _, extension := range []string{".json", ".yaml"} {
		t.Run(extension, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ledger"+extension)
			code, stdout, stderr := runCLI("forecast-ledger", "--json", "init", "--file", path,
				"--ledger-id", "flags", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner",
				"--title", "Flag-only ledger", "--contact-website", "https://example.com")
			if code != 0 || stderr != "" || !strings.Contains(stdout, `"question_count":0`) {
				t.Fatalf("init code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			code, stdout, stderr = runCLI("forecast-ledger", "--json", "platform", "add", "--file", path,
				"--platform", "metaculus", "--name", "Metaculus", "--kind", "scoring_platform", "--url", "https://www.metaculus.com")
			if code != 0 || stderr != "" || !strings.Contains(stdout, `"platform_id":"metaculus"`) {
				t.Fatalf("platform add code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			code, stdout, stderr = runCLI("forecast-ledger", "--json", "question", "add", "--file", path,
				"--question", "q-launch", "--type", "binary", "--title", "Will it launch?", "--resolution-criteria", "Resolves yes on launch.",
				"--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-01T00:00:00Z", "--platform-ref", "metaculus,q-123,https://www.metaculus.com/questions/123", "--tag", "launch")
			if code != 0 || stderr != "" || !strings.Contains(stdout, `"question_id":"q-launch"`) {
				t.Fatalf("question add code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			code, stdout, stderr = runCLI("forecast-ledger", "--json", "forecast", "add", "--file", path,
				"--question", "q-launch", "--forecast", "f-one", "--forecasted-at", "2026-09-01T09:00:00Z", "--recorded-at", "2026-09-01T09:01:00Z",
				"--value-kind", "binary", "--probability-bp", "6500", "--rationale", "Testable rationale", "--key-factor", "Schedule")
			if code != 0 || stderr != "" || !strings.Contains(stdout, `"forecast_id":"f-one"`) {
				t.Fatalf("forecast add code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			code, stdout, stderr = runCLI("forecast-ledger", "--json", "question", "update", "--file", path,
				"--question", "q-launch", "--clear-tags", "--notes", "Updated from flags")
			if code != 0 || stderr != "" || !strings.Contains(stdout, `"changed":true`) {
				t.Fatalf("question update code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			code, stdout, stderr = runCLI("forecast-ledger", "--json", "ledger", "update", "--file", path,
				"--description", "No input documents", "--forecaster-name", "Updated Owner")
			if code != 0 || stderr != "" || !strings.Contains(stdout, `"changed":true`) {
				t.Fatalf("ledger update code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(contents, []byte("Metaculus")) || !bytes.Contains(contents, []byte("Updated from flags")) {
				t.Fatalf("flag-only writes missing from ledger:\n%s", contents)
			}
		})
	}
}

func TestFlagOnlyInitWithPublicInitialForecast(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "init", "--file", path,
		"--ledger-id", "public-init", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner",
		"--question", "q-one", "--question-type", "binary", "--question-title", "Will it happen?", "--question-resolution-criteria", "Use the official result.",
		"--question-closes-at", "2026-12-31T00:00:00Z", "--question-expected-resolution-at", "2027-01-01T00:00:00Z",
		"--initial-forecast", "f-one", "--initial-forecasted-at", "2026-09-01T00:00:00Z", "--initial-recorded-at", "2026-09-01T00:01:00Z",
		"--initial-value-kind", "binary", "--initial-probability-bp", "6000", "--initial-rationale", "Public rationale")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"question_count":1`) {
		t.Fatalf("public init code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"id": "q-one"`, `"id": "f-one"`, `"visibility": "public"`, `"probability_bp": 6000`} {
		if !bytes.Contains(stored, []byte(want)) {
			t.Errorf("public init ledger missing %s:\n%s", want, stored)
		}
	}
}

func TestFlagOnlyQuestionAndForecastVariants(t *testing.T) {
	tests := []struct {
		name         string
		questionType string
		questionArgs []string
		forecastArgs []string
	}{
		{name: "binary", questionType: "binary", forecastArgs: []string{"--value-kind", "binary", "--probability-bp", "5000"}},
		{name: "multiple choice", questionType: "multiple_choice", questionArgs: []string{"--option", "yes,Yes", "--option", "no,No"}, forecastArgs: []string{"--value-kind", "multiple_choice", "--choice-probability", "yes,4000", "--choice-probability", "no,6000"}},
		{name: "numeric", questionType: "numeric", questionArgs: []string{"--unit-name", "People", "--unit-symbol", "people"}, forecastArgs: []string{"--value-kind", "numeric", "--point", "42", "--interval", "30,60,9000", "--quantile", "5000,42"}},
		{name: "date", questionType: "date", forecastArgs: []string{"--value-kind", "date", "--point", "2026-10-01", "--interval", "2026-09-01,2026-11-01,8000"}},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ledger.json")
			code, _, stderr := runCLI("forecast-ledger", "init", "--file", path, "--ledger-id", "variants", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner")
			if code != 0 {
				t.Fatalf("init code=%d stderr=%q", code, stderr)
			}
			questionID := "q-" + test.questionType
			args := []string{"forecast-ledger", "question", "add", "--file", path, "--question", questionID, "--type", test.questionType, "--title", "Variant question", "--resolution-criteria", "Objective criterion", "--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-01T00:00:00Z"}
			args = append(args, test.questionArgs...)
			code, _, stderr = runCLI(args...)
			if code != 0 {
				t.Fatalf("question code=%d stderr=%q", code, stderr)
			}
			forecastArgs := []string{"forecast-ledger", "forecast", "add", "--file", path, "--question", questionID, "--forecast", "f-" + strconv.Itoa(index), "--forecasted-at", "2026-09-01T00:00:00Z", "--recorded-at", "2026-09-01T00:01:00Z"}
			forecastArgs = append(forecastArgs, test.forecastArgs...)
			code, _, stderr = runCLI(forecastArgs...)
			if code != 0 {
				t.Fatalf("forecast code=%d stderr=%q", code, stderr)
			}
		})
	}
}

func TestAuthoringModesAndMissingFlagsAreActionable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.json")
	code, _, stderr := runCLI("forecast-ledger", "init", "--file", path, "--ledger-id", "modes", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner")
	if code != 0 {
		t.Fatalf("init code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "platform", "add", "--file", path, "--platform", "metaculus")
	if code != 2 || !strings.Contains(stderr, "--name and --kind required") || strings.Contains(stderr, `Required flag "input"`) {
		t.Fatalf("missing flags code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLIWithStdin(`{"name":"Metaculus","kind":"scoring_platform"}`, "forecast-ledger", "platform", "add", "--file", path, "--platform", "metaculus", "--input", "-", "--name", "override")
	if code != 2 || !strings.Contains(stderr, "--input cannot be combined") {
		t.Fatalf("mixed mode code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "init", "--file", filepath.Join(t.TempDir(), "dangling.json"), "--ledger-id", "dangling", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner", "--initial-value-kind", "binary")
	if code != 2 || !strings.Contains(stderr, "--initial-value-kind requires --question") {
		t.Fatalf("dangling init flag code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "platform", "add", "--file", path, "--platform", "duplicate-check", "--name", "First", "--kind", "informal")
	if code != 0 {
		t.Fatalf("platform setup code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "question", "add", "--file", path, "--question", "q-dangling", "--type", "binary", "--title", "Question", "--resolution-criteria", "Result", "--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-01T00:00:00Z", "--initial-value-kind", "binary")
	if code != 2 || !strings.Contains(stderr, "--initial-value-kind requires --initial-forecast") {
		t.Fatalf("dangling question flag code=%d stderr=%q", code, stderr)
	}
}

func TestFlagOnlyLifecycleAndProtectedSeal(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "ledger.json")
	code, _, stderr := runCLI("forecast-ledger", "init", "--file", path, "--ledger-id", "protected", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner")
	if code != 0 {
		t.Fatalf("init code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "question", "add", "--file", path, "--question", "q-one", "--type", "binary",
		"--title", "Will it happen?", "--resolution-criteria", "Use the official result.", "--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-02T00:00:00Z")
	if code != 0 {
		t.Fatalf("question add code=%d stderr=%q", code, stderr)
	}
	privatePath := filepath.Join(directory, "private.json")
	privateJSON := `{"value":{"kind":"binary","probability_bp":7000},"rationale":"secret-rationale","key_factors":["private-factor"],"comment":"secret-comment"}`
	if err := os.WriteFile(privatePath, []byte(privateJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(directory, "forecast.key")
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "forecast", "seal", "--file", path, "--question", "q-one", "--forecast", "f-sealed",
		"--forecasted-at", "2026-09-01T00:00:00Z", "--recorded-at", "2026-09-01T00:01:00Z", "--public-note", "Public note", "--secret-input", privatePath, "--key-file", keyPath)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"forecast_id":"f-sealed"`) {
		t.Fatalf("seal code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "secret-rationale") || strings.Contains(stdout+stderr, "private-factor") || strings.Contains(stdout+stderr, "secret-comment") {
		t.Fatal("sealed private values leaked to output")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(contents, []byte("secret-rationale")) || bytes.Contains(contents, []byte("private-factor")) || bytes.Contains(contents, []byte("secret-comment")) {
		t.Fatal("sealed private values leaked to ledger")
	}

	lifecyclePath := filepath.Join(directory, "lifecycle.json")
	code, _, stderr = runCLI("forecast-ledger", "init", "--file", lifecyclePath, "--ledger-id", "lifecycle", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner",
		"--question", "q-resolve", "--question-type", "binary", "--question-title", "Did it happen?", "--question-resolution-criteria", "Use the official result.",
		"--question-closes-at", "2026-09-01T00:00:00Z", "--question-expected-resolution-at", "2026-09-02T00:00:00Z")
	if code != 0 {
		t.Fatalf("rich init code=%d stderr=%q", code, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "question", "update", "--file", lifecyclePath, "--question", "q-resolve", "--status", "closed")
	if code != 0 {
		t.Fatalf("close question code=%d stderr=%q", code, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "--yes", "question", "resolve", "--file", lifecyclePath, "--question", "q-resolve",
		"--outcome-boolean=true", "--outcome-known-at", "2026-09-02T00:00:00Z", "--recorded-at", "2026-09-02T00:01:00Z",
		"--source", "Official result,https://example.com/result,2026-09-02T00:02:00Z,Example Publisher")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"status":"resolved"`) {
		t.Fatalf("resolve code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "--yes", "question", "dispute", "--file", lifecyclePath, "--question", "q-resolve",
		"--reason", "The source is contested", "--recorded-at", "2026-09-02T00:03:00Z",
		"--source", "Counter-source,https://example.com/counter,2026-09-02T00:02:30Z")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"status":"disputed"`) {
		t.Fatalf("dispute code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	for _, questionID := range []string{"q-annul", "q-false"} {
		code, _, stderr = runCLI("forecast-ledger", "question", "add", "--file", lifecyclePath, "--question", questionID, "--type", "binary",
			"--title", "Lifecycle question", "--resolution-criteria", "Use the official result.", "--closes-at", "2026-09-01T00:00:00Z", "--expected-resolution-at", "2026-09-02T00:00:00Z")
		if code != 0 {
			t.Fatalf("add %s code=%d stderr=%q", questionID, code, stderr)
		}
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "--yes", "question", "annul", "--file", lifecyclePath, "--question", "q-annul",
		"--reason", "The criterion became unresolvable", "--recorded-at", "2026-09-02T00:01:00Z")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"status":"annulled"`) {
		t.Fatalf("annul code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, _, stderr = runCLI("forecast-ledger", "question", "update", "--file", lifecyclePath, "--question", "q-false", "--status", "closed")
	if code != 0 {
		t.Fatalf("close false question code=%d stderr=%q", code, stderr)
	}
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "--yes", "question", "resolve", "--file", lifecyclePath, "--question", "q-false",
		"--outcome-boolean=false", "--outcome-known-at", "2026-09-02T00:00:00Z", "--recorded-at", "2026-09-02T00:01:00Z",
		"--source", "Official result,https://example.com/false-result,2026-09-02T00:02:00Z")
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"status":"resolved"`) {
		t.Fatalf("false resolve code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestFlagOnlySealedInitialForecast(t *testing.T) {
	directory := t.TempDir()
	privatePath := filepath.Join(directory, "initial-private.json")
	privateJSON := `{"value":{"kind":"binary","probability_bp":7200},"rationale":"sealed rationale","key_factors":[],"comment":"sealed comment"}`
	if err := os.WriteFile(privatePath, []byte(privateJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "ledger.json")
	keyPath := filepath.Join(directory, "initial.key")
	code, stdout, stderr := runCLI("forecast-ledger", "--json", "init", "--file", path, "--ledger-id", "sealed-init", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner",
		"--question", "q-one", "--question-type", "binary", "--question-title", "Will it happen?", "--question-resolution-criteria", "Use the official result.",
		"--question-closes-at", "2026-12-31T00:00:00Z", "--question-expected-resolution-at", "2027-01-01T00:00:00Z",
		"--initial-forecast", "f-one", "--initial-visibility", "sealed", "--initial-forecasted-at", "2026-09-01T00:00:00Z", "--initial-recorded-at", "2026-09-01T00:01:00Z",
		"--initial-secret-input", privatePath, "--key-file", keyPath)
	if code != 0 || stderr != "" || !strings.Contains(stdout, `"visibility":"sealed"`) {
		t.Fatalf("sealed init code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "sealed rationale") || strings.Contains(stdout+stderr, "sealed comment") {
		t.Fatal("sealed initial values leaked to output")
	}

	backlogPath := filepath.Join(directory, "backlog.json")
	code, _, stderr = runCLI("forecast-ledger", "init", "--file", backlogPath, "--ledger-id", "sealed-add", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner")
	if code != 0 {
		t.Fatalf("backlog init code=%d stderr=%q", code, stderr)
	}
	addKeyPath := filepath.Join(directory, "added.key")
	code, stdout, stderr = runCLI("forecast-ledger", "--json", "question", "add", "--file", backlogPath, "--question", "q-added", "--type", "binary",
		"--title", "Will it happen?", "--resolution-criteria", "Use the official result.", "--closes-at", "2026-12-31T00:00:00Z", "--expected-resolution-at", "2027-01-01T00:00:00Z",
		"--initial-forecast", "f-added", "--initial-visibility", "sealed", "--initial-forecasted-at", "2026-09-01T00:00:00Z", "--initial-recorded-at", "2026-09-01T00:01:00Z",
		"--initial-secret-input", privatePath, "--key-file", addKeyPath)
	if code != 0 || stderr != "" {
		t.Fatalf("sealed question add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	stored, err := os.ReadFile(backlogPath)
	if err != nil {
		t.Fatal(err)
	}
	var ledgerDocument map[string]any
	if err := json.Unmarshal(stored, &ledgerDocument); err != nil {
		t.Fatal(err)
	}
	questions, ok := ledgerDocument["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("sealed question add persisted unexpected questions: %#v", ledgerDocument["questions"])
	}
	question, ok := questions[0].(map[string]any)
	if !ok {
		t.Fatalf("sealed question add persisted unexpected question: %#v", questions[0])
	}
	forecasts, ok := question["forecasts"].([]any)
	if !ok || len(forecasts) != 1 {
		t.Fatalf("sealed question add persisted unexpected forecasts: %#v", question["forecasts"])
	}
	forecast, ok := forecasts[0].(map[string]any)
	if !ok || forecast["visibility"] != "sealed" {
		t.Fatalf("sealed question add did not persist sealed visibility:\n%s", stored)
	}

	rejectedPath := filepath.Join(directory, "rejected.json")
	code, _, stderr = runCLI("forecast-ledger", "init", "--file", rejectedPath, "--ledger-id", "sealed-rejected", "--timezone", "UTC", "--forecaster-id", "owner", "--forecaster-name", "Owner",
		"--question", "q-rejected", "--question-type", "binary", "--question-title", "Will it happen?", "--question-resolution-criteria", "Use the official result.",
		"--question-closes-at", "2026-12-31T00:00:00Z", "--question-expected-resolution-at", "2027-01-01T00:00:00Z",
		"--initial-forecast", "f-rejected", "--initial-visibility", "sealed", "--initial-forecasted-at", "2026-09-01T00:00:00Z",
		"--initial-secret-input", privatePath, "--initial-rationale", "must-not-enter-argv", "--key-file", filepath.Join(directory, "rejected.key"))
	if code != 2 || !strings.Contains(stderr, "put private values in --initial-secret-input") {
		t.Fatalf("sealed argv-private rejection code=%d stderr=%q", code, stderr)
	}
	if _, err := os.Stat(rejectedPath); !os.IsNotExist(err) {
		t.Fatalf("rejected sealed init created a ledger: %v", err)
	}
}

func TestVersionFormattingAndStableJSON(t *testing.T) {
	info := buildinfo.Current()
	var plain bytes.Buffer
	if err := writeVersionInfo(&plain, info, presentation.ModePlain, false); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"Binary:", "Version:", "Source revision:", "Forecast Ledger schema:", "Schema SHA-256:", "MCP protocol:", "Timestamp support:", "Timestamp providers:"} {
		if !strings.Contains(plain.String(), label) {
			t.Errorf("plain version missing %q:\n%s", label, plain.String())
		}
	}
	if strings.Contains(plain.String(), "\x1b[") {
		t.Fatal("plain version contains ANSI")
	}
	var colored bytes.Buffer
	if err := writeVersionInfo(&colored, info, presentation.ModeHuman, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(colored.String(), "\x1b[36mBinary:\x1b[0m") {
		t.Fatalf("human color missing: %q", colored.String())
	}

	code, stdout, stderr := runCLI("forecast-ledger", "version", "--json")
	if code != 0 || stderr != "" || strings.Contains(stdout, "\x1b[") {
		t.Fatalf("version JSON code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var decoded buildinfo.Info
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Schema != info.Schema || decoded.Version != info.Version {
		t.Fatalf("version JSON changed values: %#v", decoded)
	}
}
