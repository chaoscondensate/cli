package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/buildinfo"
	"github.com/chaoscondensate/cli/internal/service"
	urfavecli "github.com/urfave/cli/v3"
)

func NewCommand(stdin io.Reader, stdout, stderr io.Writer) *urfavecli.Command {
	info := buildinfo.Current()
	root := &urfavecli.Command{
		Name:                  "forecast-ledger",
		Usage:                 "Create and verify portable forecast evidence",
		Description:           "Manage Forecast Ledger files without requiring Git or a hosted service.\n\nExamples:\n  forecast-ledger validate --file ledger.yaml\n  forecast-ledger status --file ledger.yaml\n  forecast-ledger completion bash",
		Version:               info.Version,
		Suggest:               true,
		EnableShellCompletion: true,
		ConfigureShellCompletionCommand: func(command *urfavecli.Command) {
			command.Hidden = false
		},
		Reader:    stdin,
		Writer:    stdout,
		ErrWriter: stderr,
		Flags: []urfavecli.Flag{
			&urfavecli.BoolFlag{Name: "json", Usage: "Write stable JSON output"},
			&urfavecli.BoolFlag{Name: "plain", Usage: "Write plain text without decoration"},
			&urfavecli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "Suppress successful output"},
			&urfavecli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Write additional diagnostics to stderr"},
			&urfavecli.BoolFlag{Name: "no-input", Usage: "Never prompt; fail when input is missing"},
			&urfavecli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "Limit network or wait operations"},
		},
		Commands: []*urfavecli.Command{
			initCommand(),
			ledgerReadCommand("validate", "Validate a ledger locally", true),
			ledgerReadCommand("status", "Show ledger and evidence status", true),
			platformCommand(), questionCommand(), forecastCommand(), targetCommand(), timestampCommand(),
			verifyCommand(), publishCommand(), mcpCommand(), versionCommand(),
		},
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			if command.NArg() > 0 {
				err := urfavecli.ShowCommandHelp(ctx, command, command.Args().First())
				return app.NewError(app.CodeUsage, err.Error(), err)
			}
			return urfavecli.ShowRootCommandHelp(command)
		},
		Before: func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
			if err := contextApplicationError(ctx); err != nil {
				return ctx, err
			}
			modes := 0
			for _, name := range []string{"json", "plain", "quiet"} {
				if command.Bool(name) {
					modes++
				}
			}
			if modes > 1 {
				return ctx, app.NewError(app.CodeUsage, "--json, --plain, and --quiet cannot be combined", nil)
			}
			return ctx, nil
		},
		ExitErrHandler: func(context.Context, *urfavecli.Command, error) {},
	}
	_ = root.Walk(func(command *urfavecli.Command) error {
		command.OnUsageError = func(_ context.Context, _ *urfavecli.Command, err error, _ bool) error {
			return app.NewError(app.CodeUsage, "invalid command input: "+err.Error(), err)
		}
		return nil
	})
	return root
}

func initCommand() *urfavecli.Command {
	return leaf("init", "Create a new ledger", "forecast-ledger init --file ledger.yaml --ledger-id my-forecasts --timezone Europe/London", false,
		[]urfavecli.Flag{fileFlag(false),
			&urfavecli.StringFlag{Name: "ledger-id", Required: true, Usage: "Stable ledger ID"},
			&urfavecli.StringFlag{Name: "timezone", Required: true, Usage: "IANA timezone name"},
			&urfavecli.StringFlag{Name: "forecaster-id", Required: true, Usage: "Stable forecaster ID"},
			&urfavecli.StringFlag{Name: "forecaster-name", Required: true, Usage: "Forecaster display name"},
			&urfavecli.StringFlag{Name: "forecaster-kind", Value: "individual", Usage: "Forecaster kind: individual or team"}})
}

func ledgerReadCommand(name, usage string, stdinAllowed bool) *urfavecli.Command {
	command := leaf(name, usage, fmt.Sprintf("forecast-ledger %s --file ledger.yaml", name), true, []urfavecli.Flag{fileFlag(stdinAllowed)})
	command.Action = func(ctx context.Context, command *urfavecli.Command) error {
		runtime := RuntimeFromCommand(command)
		operationContext, cancel := runtime.Context(ctx)
		defer cancel()
		loaded, err := service.LoadAndValidateLedger(operationContext, command.String("file"), command.Root().Reader)
		if err != nil {
			return err
		}
		presenter := presenterFor(command)
		if name == "validate" {
			return presenter.Success("ledger.valid", "Ledger is valid", map[string]any{"ledger_id": loaded.Model.LedgerID, "schema_version": loaded.Model.SchemaVersion})
		}
		status, err := service.StatusForLedger(loaded)
		if err != nil {
			return app.NewError(app.CodeInternal, "ledger status could not be built", err)
		}
		message := fmt.Sprintf("%s: %d questions, %d forecasts; integrity: %d unanchored, %d pending, %d verified, %d failed",
			status.LedgerID, status.Questions, status.Forecasts, status.Unanchored, status.Pending, status.Verified, status.Failed)
		return presenter.Success("ledger.status", message, status)
	}
	return command
}

func platformCommand() *urfavecli.Command {
	return group("platform", "Manage platform records",
		leaf("add", "Add a platform", "forecast-ledger platform add --file ledger.yaml --platform metaculus", false, []urfavecli.Flag{fileFlag(false), platformFlag()}),
		leaf("update", "Update a platform", "forecast-ledger platform update --file ledger.yaml --platform metaculus", false, []urfavecli.Flag{fileFlag(false), platformFlag()}),
		leaf("list", "List platforms", "forecast-ledger platform list --file ledger.yaml", true, []urfavecli.Flag{fileFlag(true)}),
		leaf("show", "Show a platform", "forecast-ledger platform show --file ledger.yaml --platform metaculus", true, []urfavecli.Flag{fileFlag(true), platformFlag()}),
		leaf("remove", "Remove an unused platform", "forecast-ledger platform remove --file ledger.yaml --platform metaculus", false, []urfavecli.Flag{fileFlag(false), platformFlag()}))
}

func questionCommand() *urfavecli.Command {
	return group("question", "Manage forecast questions",
		leaf("add", "Add a question", "forecast-ledger question add --file ledger.yaml --question q-launch --type binary", false, []urfavecli.Flag{fileFlag(false), questionFlag(), &urfavecli.StringFlag{Name: "type", Required: true, Usage: "Question type"}}),
		leaf("update", "Update a question", "forecast-ledger question update --file ledger.yaml --question q-launch", false, []urfavecli.Flag{fileFlag(false), questionFlag()}),
		leaf("list", "List questions", "forecast-ledger question list --file ledger.yaml", true, []urfavecli.Flag{fileFlag(true)}),
		leaf("show", "Show a question", "forecast-ledger question show --file ledger.yaml --question q-launch", true, []urfavecli.Flag{fileFlag(true), questionFlag()}),
		leaf("resolve", "Resolve a question", "forecast-ledger question resolve --file ledger.yaml --question q-launch", false, []urfavecli.Flag{fileFlag(false), questionFlag()}),
		leaf("annul", "Annul a question", "forecast-ledger question annul --file ledger.yaml --question q-launch", false, []urfavecli.Flag{fileFlag(false), questionFlag()}),
		leaf("dispute", "Dispute a resolution", "forecast-ledger question dispute --file ledger.yaml --question q-launch", false, []urfavecli.Flag{fileFlag(false), questionFlag()}))
}

func forecastCommand() *urfavecli.Command {
	return group("forecast", "Manage append-only forecast records",
		leaf("add", "Add a public forecast", "forecast-ledger forecast add --file ledger.yaml --question q-launch --forecast f-001", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag()}),
		leaf("list", "List forecasts", "forecast-ledger forecast list --file ledger.yaml --question q-launch", true, []urfavecli.Flag{fileFlag(true), questionFlag()}),
		leaf("show", "Show a forecast", "forecast-ledger forecast show --file ledger.yaml --question q-launch --forecast f-001", true, []urfavecli.Flag{fileFlag(true), questionFlag(), forecastFlag()}),
		leaf("seal", "Create and append a sealed forecast", "forecast-ledger forecast seal --file ledger.yaml --question q-launch --forecast f-001 --key-file secret.key", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), secretOutputFlag()}),
		leaf("reveal", "Verify and reveal a sealed forecast", "forecast-ledger forecast reveal --file ledger.yaml --question q-launch --forecast f-001 --key-file secret.key", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), &urfavecli.StringFlag{Name: "key-file", Required: true, TakesFile: true, Usage: "Protected key file"}}))
}

func targetCommand() *urfavecli.Command {
	return group("target", "Build or check canonical forecast targets", targetLeaf("build", "Build target artifacts", false), targetLeaf("check", "Check target bytes and digests", true))
}

func targetLeaf(name, usage string, readOnly bool) *urfavecli.Command {
	flags := []urfavecli.Flag{fileFlag(readOnly), &urfavecli.StringFlag{Name: "question", Usage: "Question ID"}, &urfavecli.StringFlag{Name: "forecast", Usage: "Forecast ID"}, &urfavecli.BoolFlag{Name: "all", Usage: "Select every forecast"}}
	command := leaf(name, usage, fmt.Sprintf("forecast-ledger target %s --file ledger.yaml --question q-launch --forecast f-001", name), readOnly, flags)
	command.Before = requireTargetSelection
	return command
}

func timestampCommand() *urfavecli.Command {
	return group("timestamp", "Manage OpenTimestamps receipts",
		timestampLeaf("stamp", "Submit a target to configured OTS calendars", false), timestampLeaf("upgrade", "Upgrade a pending OTS receipt", false),
		timestampLeaf("status", "Show OTS receipt status", true), timestampLeaf("verify", "Verify an OTS receipt", true))
}

func timestampLeaf(name, usage string, readOnly bool) *urfavecli.Command {
	return leaf(name, usage, fmt.Sprintf("forecast-ledger timestamp %s --file ledger.yaml --question q-launch --forecast f-001", name), readOnly, []urfavecli.Flag{fileFlag(readOnly), questionFlag(), forecastFlag()})
}

func verifyCommand() *urfavecli.Command {
	command := leaf("verify", "Run layered verification", "forecast-ledger verify --file ledger.yaml", true, []urfavecli.Flag{fileFlag(true), &urfavecli.StringFlag{Name: "question", Usage: "Optional question ID"}, &urfavecli.StringFlag{Name: "forecast", Usage: "Optional forecast ID"}})
	command.Before = func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
		if command.String("forecast") != "" && command.String("question") == "" {
			return ctx, app.NewError(app.CodeUsage, "--forecast requires --question", nil)
		}
		return ctx, nil
	}
	return command
}

func publishCommand() *urfavecli.Command {
	return group("publish", "Build or verify portable evidence packages",
		leaf("build", "Build an evidence package", "forecast-ledger publish build --file ledger.yaml --output evidence", false, []urfavecli.Flag{fileFlag(false), &urfavecli.StringFlag{Name: "output", Required: true, TakesFile: true, Usage: "New package directory"}}),
		leaf("verify", "Verify an evidence package", "forecast-ledger publish verify --file package/ledger.yaml --manifest package/manifest.json", true, []urfavecli.Flag{fileFlag(false), &urfavecli.StringFlag{Name: "manifest", Required: true, TakesFile: true, Usage: "Package manifest file"}}))
}

func mcpCommand() *urfavecli.Command {
	return group("mcp", "Run the MCP adapter", leaf("serve", "Serve MCP over stdio", "forecast-ledger mcp serve --ledger-root /data/ledgers", true, []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "ledger-root", Required: true, TakesFile: true, Usage: "Allowed ledger root"},
		&urfavecli.StringFlag{Name: "output-root", TakesFile: true, Usage: "Allowed package output root"},
		&urfavecli.StringFlag{Name: "secret-root", TakesFile: true, Usage: "Allowed protected secret root"},
		&urfavecli.BoolFlag{Name: "allow-write", Usage: "Grant ledger write tools"}, &urfavecli.BoolFlag{Name: "allow-network", Usage: "Grant network tools"}, &urfavecli.BoolFlag{Name: "allow-reveal", Usage: "Grant reveal tools"}}))
}

func versionCommand() *urfavecli.Command {
	return &urfavecli.Command{Name: "version", Usage: "Show build and contract versions", Description: "Example:\n  forecast-ledger version --json",
		Flags: []urfavecli.Flag{&urfavecli.BoolFlag{Name: "json", Usage: "Write stable JSON metadata", Local: true}},
		Action: func(_ context.Context, command *urfavecli.Command) error {
			info := buildinfo.Current()
			if command.Bool("json") {
				encoder := json.NewEncoder(command.Root().Writer)
				encoder.SetEscapeHTML(false)
				return encoder.Encode(info)
			}
			_, err := fmt.Fprintf(command.Root().Writer, "%s %s\nsource revision: %s\ngo: %s\nforecast ledger schema: %s (%s, sha256:%s)\nmcp protocol: %s\n", info.Binary, info.Version, info.SourceRevision, info.GoVersion, info.Schema.Version, info.Schema.Commit, info.Schema.SHA256, info.MCPProtocol)
			return err
		}}
}

func group(name, usage string, children ...*urfavecli.Command) *urfavecli.Command {
	return &urfavecli.Command{Name: name, Usage: usage, Commands: children, Action: func(ctx context.Context, command *urfavecli.Command) error {
		if command.NArg() > 0 {
			err := urfavecli.ShowCommandHelp(ctx, command, command.Args().First())
			return app.NewError(app.CodeUsage, err.Error(), err)
		}
		return urfavecli.ShowSubcommandHelp(command)
	}}
}

func leaf(name, usage, example string, readOnly bool, flags []urfavecli.Flag) *urfavecli.Command {
	if !readOnly {
		flags = append(flags, &urfavecli.BoolFlag{Name: "dry-run", Usage: "Validate and show the planned change without writing"})
	}
	return &urfavecli.Command{Name: name, Usage: usage, Description: "Example:\n  " + example, Flags: flags, Action: unavailableAction}
}

func fileFlag(allowStdin bool) *urfavecli.StringFlag {
	usage := "Ledger file path"
	if allowStdin {
		usage += "; use - for stdin"
	}
	return &urfavecli.StringFlag{Name: "file", Aliases: []string{"f"}, Required: true, OnlyOnce: true, TakesFile: true, Usage: usage,
		Validator: func(value string) error {
			if strings.TrimSpace(value) == "" {
				return errors.New("file path is empty")
			}
			if value == "-" && !allowStdin {
				return errors.New("--file - is only available for eligible read-only commands")
			}
			return nil
		}}
}

func questionFlag() *urfavecli.StringFlag {
	return &urfavecli.StringFlag{Name: "question", Required: true, OnlyOnce: true, Usage: "Stable question ID"}
}
func forecastFlag() *urfavecli.StringFlag {
	return &urfavecli.StringFlag{Name: "forecast", Required: true, OnlyOnce: true, Usage: "Stable forecast ID"}
}
func platformFlag() *urfavecli.StringFlag {
	return &urfavecli.StringFlag{Name: "platform", Required: true, OnlyOnce: true, Usage: "Stable platform ID"}
}
func secretOutputFlag() *urfavecli.StringFlag {
	return &urfavecli.StringFlag{Name: "key-file", Required: true, OnlyOnce: true, TakesFile: true, Usage: "New protected key file"}
}
func unavailableAction(_ context.Context, _ *urfavecli.Command) error {
	return app.NewError(app.CodeInternal, "command service is not implemented yet", nil)
}

func requireTargetSelection(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
	all := command.Bool("all")
	question := command.String("question")
	forecast := command.String("forecast")
	if all && (question != "" || forecast != "") {
		return ctx, app.NewError(app.CodeUsage, "--all cannot be combined with --question or --forecast", nil)
	}
	if !all && (question == "" || forecast == "") {
		return ctx, app.NewError(app.CodeUsage, "use --all or both --question and --forecast", nil)
	}
	return ctx, nil
}
