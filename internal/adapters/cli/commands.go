package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	mcpadapter "github.com/chaoscondensate/cli/internal/adapters/mcp"
	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/buildinfo"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/presentation"
	"github.com/chaoscondensate/cli/internal/service"
	"github.com/chaoscondensate/cli/internal/storage"
	urfavecli "github.com/urfave/cli/v3"
)

func NewCommand(stdin io.Reader, stdout, stderr io.Writer) *urfavecli.Command {
	return newCommandWithEffects(stdin, stdout, stderr, service.ProductionEffects())
}

func newCommandWithEffects(stdin io.Reader, stdout, stderr io.Writer, effects service.Effects) *urfavecli.Command {
	info := buildinfo.Current()
	root := &urfavecli.Command{
		Name:                      "forecast-ledger",
		Usage:                     "Create and verify portable forecast evidence",
		Description:               "Manage Forecast Ledger files without requiring Git or a hosted service.\n\nExamples:\n  forecast-ledger validate --file ledger.yaml\n  forecast-ledger status --file ledger.yaml\n  forecast-ledger completion bash",
		Version:                   info.Version,
		Suggest:                   true,
		EnableShellCompletion:     true,
		DisableSliceFlagSeparator: true,
		ConfigureShellCompletionCommand: func(command *urfavecli.Command) {
			command.Hidden = false
		},
		Reader:    stdin,
		Writer:    stdout,
		ErrWriter: stderr,
		Metadata:  map[string]any{"effects": effects},
		Flags: []urfavecli.Flag{
			&urfavecli.BoolFlag{Name: "json", Usage: "Write stable JSON output"},
			&urfavecli.BoolFlag{Name: "plain", Usage: "Write plain text without decoration"},
			&urfavecli.BoolFlag{Name: "quiet", Aliases: []string{"q"}, Usage: "Suppress successful output"},
			&urfavecli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "Write additional diagnostics to stderr"},
			&urfavecli.BoolFlag{Name: "no-color", Usage: "Disable color and interactive decoration"},
			&urfavecli.BoolFlag{Name: "no-input", Usage: "Never prompt; fail when input is missing"},
			&urfavecli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Approve a change that requires confirmation"},
			&urfavecli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "Limit operation work; ledger lock conflicts remain immediate"},
		},
		Commands: []*urfavecli.Command{
			initCommand(),
			ledgerCommand(),
			ledgerReadCommand("validate", "Validate a ledger locally", true),
			ledgerReadCommand("status", "Show ledger and evidence status", true),
			platformCommand(), questionCommand(), forecastCommand(),
			targetCommand(), timestampCommand(), verifyCommand(),
			publishCommand(), mcpCommand(), versionCommand(),
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

func commandEffects(command *urfavecli.Command) service.Effects {
	if effects, ok := command.Root().Metadata["effects"].(service.Effects); ok {
		return effects
	}
	return service.ProductionEffects()
}

func presentOperationOutcome(command *urfavecli.Command, operation service.OperationName, dryRun bool, data any, operationErr error, humanMessage string) error {
	outcome := service.ClassifyOperationOutcome(operation, service.OutcomeInput{DryRun: dryRun, Data: data, Err: operationErr})
	if operationErr != nil && !outcome.HasData {
		return operationErr
	}
	presenter := presenterFor(command)
	message := outcome.Message
	if humanMessage != "" && presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = humanMessage
	}
	if err := presenter.Success(outcome.Code, message, data); err != nil {
		return err
	}
	if operationErr != nil {
		return presentedApplicationError{operationErr}
	}
	if outcome.FailureCode != "" {
		return presentedApplicationError{app.NewError(outcome.FailureCode, outcome.Message, nil)}
	}
	return nil
}

func ledgerCommand() *urfavecli.Command {
	authorFlags := rootPatchFlags()
	flags := append([]urfavecli.Flag{fileFlag(false)}, authorFlags...)
	command := leaf("update", "Update ledger and current forecaster metadata", "forecast-ledger ledger update --file ledger.yaml --title 'Forecast archive' --timezone Europe/London", false, flags)
	command.Description += "\n\nUse direct flags for authoring."
	command.Action = ledgerUpdateAction
	return group("ledger", "Manage ledger metadata", command)
}

func ledgerUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	input, err := buildRootPatchInput(command)
	if err != nil {
		return err
	}
	var result service.RootMetadataFileResult
	if runtime.DryRun {
		result, err = service.PlanRootMetadataFileUpdate(operationContext, command.String("file"), input)
	} else {
		result, err = service.CommitRootMetadataFileUpdate(operationContext, command.String("file"), input)
	}
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationLedgerUpdate, runtime.DryRun, result, nil, "")
}

func initCommand() *urfavecli.Command {
	authorFlags := append(rootAuthoringFlags(), initNestedFlags()...)
	flags := []urfavecli.Flag{fileFlag(false),
		&urfavecli.StringFlag{Name: "ledger-id", Required: true, Usage: "Stable ledger ID"},
		&urfavecli.StringFlag{Name: "timezone", Required: true, Usage: "IANA timezone name"},
		&urfavecli.StringFlag{Name: "forecaster-id", Required: true, Usage: "Stable forecaster ID"},
		&urfavecli.StringFlag{Name: "forecaster-name", Required: true, Usage: "Forecaster display name"},
		&urfavecli.StringFlag{Name: "forecaster-kind", Value: "individual", Usage: "Forecaster kind: individual or team"},
		&urfavecli.StringFlag{Name: "key-file", OnlyOnce: true, TakesFile: true, Usage: "New protected key file; required only for a sealed first forecast"},
	}
	flags = append(flags, authorFlags...)
	command := leaf("init", "Create a new ledger", "forecast-ledger init --file ledger.yaml --ledger-id my-forecasts --timezone Europe/London --forecaster-id me --forecaster-name 'My Name'", false,
		flags)
	command.Action = initAction
	command.Description += "\n\nAll non-secret ledger, platform, question, and public initial-forecast data are supplied through flags. Omitted initial times use one operation-clock observation."
	return command
}

func initAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	effects := commandEffects(command)
	operationAt, err := formatOperationTime(effects.Clock.Now(), command.String("timezone"))
	if err != nil {
		return err
	}
	if command.IsSet("initial-forecast") && !command.IsSet("initial-forecasted-at") {
		if err := command.Set("initial-forecasted-at", string(operationAt)); err != nil {
			return err
		}
	}
	var normalizedTimes []service.TimeNormalization
	for _, item := range []struct {
		name   string
		policy dateOnlyPolicy
	}{{"created-at", dateOnlyRejected}, {"question-created-at", dateOnlyRejected}, {"question-opens-at", dateOnlyStart}, {"question-expected-resolution-at", dateOnlyEnd}, {"initial-forecasted-at", dateOnlyRejected}, {"initial-recorded-at", dateOnlyRejected}} {
		normalized, err := normalizeSetTimeWithMetadata(command, item.name, command.String("timezone"), item.policy)
		if err != nil {
			return err
		}
		normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	}
	input, err := buildInitInput(operationContext, command, command.Root().Reader)
	if err != nil {
		return err
	}
	root, err := service.BuildLedgerRootAt(service.InitRootRequest{
		LedgerID: ledger.Slug(command.String("ledger-id")), Timezone: command.String("timezone"),
		ForecasterID: ledger.Slug(command.String("forecaster-id")), ForecasterName: command.String("forecaster-name"),
		ForecasterKind: ledger.ForecasterKind(command.String("forecaster-kind")), Input: input,
	}, operationAt)
	if err != nil {
		return err
	}
	shape, err := service.ClassifyInitInput(input)
	if err != nil {
		return err
	}
	keyPath := command.String("key-file")
	if shape != service.CreationSealedForecast && keyPath != "" {
		return app.NewError(app.CodeUsage, "--key-file is only valid for a sealed initial forecast", nil)
	}
	var model *ledger.Ledger
	var sealed service.SealedInitialBuild
	switch shape {
	case service.CreationLedgerOnly:
		model = root
	case service.CreationQuestionOnly:
		model, err = service.BuildInitialQuestionLedgerAt(root, *input.Question, operationAt)
	case service.CreationPublicForecast:
		model, err = service.BuildInitialPublicLedgerAt(root, *input.Question, operationAt)
	case service.CreationSealedForecast:
		protectedInputPath := command.String("initial-secret-input")
		protectedArgument := "--initial-secret-input"
		if protectedInputPath != "-" {
			if err := storage.CheckProtectedFile(protectedInputPath); err != nil {
				return protectedArgumentError(err, protectedArgument)
			}
		}
		if strings.TrimSpace(keyPath) == "" {
			return app.NewError(app.CodeUsage, "--key-file is required for a sealed initial forecast", nil)
		}
		if runtime.DryRun {
			model, err = service.PlanInitialSealedLedgerAt(root, *input.Question, operationAt)
		} else {
			sealed, err = service.BuildInitialSealedLedgerAt(operationContext, root, *input.Question, operationAt, effects)
			model = sealed.Ledger
		}
	}
	if err != nil {
		return err
	}
	ledgerPath := command.String("file")
	if runtime.DryRun {
		resolvedLedger, resolveErr := storage.ResolveNewFilePath(ledgerPath, "ledger file")
		if resolveErr != nil {
			return resolveErr
		}
		if _, encodeErr := service.EncodeNewLedger(model, resolvedLedger); encodeErr != nil {
			return encodeErr
		}
		effects := []service.SideEffect{{Kind: service.EffectLedger, Action: service.EffectCreate, Status: service.EffectDeferred, Path: filepath.Base(resolvedLedger), Owned: true, Rollback: service.RollbackCreatedPublic}}
		if shape == service.CreationSealedForecast {
			resolvedKey, keyErr := storage.ResolveNewFilePath(keyPath, "key file")
			if keyErr != nil {
				return keyErr
			}
			if resolvedKey == resolvedLedger {
				return app.NewError(app.CodeConflict, "ledger and key destinations must be different", nil)
			}
			effects = append([]service.SideEffect{{Kind: service.EffectKey, Action: service.EffectCreate, Status: service.EffectDeferred, Path: filepath.Base(resolvedKey), Owned: true, Rollback: service.RollbackRetainSecret}}, effects...)
		}
		result := service.NewInitResult(model, effects, service.Recovery{State: service.RecoveryNone})
		result.NormalizedTimes = normalizedTimes
		return presentOperationOutcome(command, service.OperationLedgerInit, true, result, nil, "")
	}
	recovery := service.Recovery{State: service.RecoveryNone}
	if shape == service.CreationSealedForecast {
		commit, commitErr := service.CommitInitialSealedFiles(operationContext, ledgerPath, keyPath, sealed, service.InitialCommitOptions{})
		recovery = commit.Recovery
		if commitErr != nil {
			return withRecovery(commitErr, recovery)
		}
	} else {
		if _, err := service.CommitNewLedger(ledgerPath, model); err != nil {
			return err
		}
	}
	completedEffects := []service.SideEffect{{Kind: service.EffectLedger, Action: service.EffectCreate, Status: service.EffectCompleted, Path: filepath.Base(ledgerPath), Owned: true, Rollback: service.RollbackCreatedPublic}}
	if shape == service.CreationSealedForecast {
		completedEffects = append([]service.SideEffect{{Kind: service.EffectKey, Action: service.EffectCreate, Status: service.EffectCompleted, Path: filepath.Base(keyPath), Owned: true, Rollback: service.RollbackRetainSecret}}, completedEffects...)
	}
	result := service.NewInitResult(model, completedEffects, recovery)
	result.NormalizedTimes = normalizedTimes
	return presentOperationOutcome(command, service.OperationLedgerInit, false, result, nil, "")
}

func withRecovery(err error, recovery service.Recovery) error {
	if recovery.State == "" || recovery.State == service.RecoveryNone {
		return err
	}
	var applicationErr *app.Error
	if !errors.As(err, &applicationErr) {
		applicationErr = app.NewError(app.CodeInternal, "operation failed and recovery information is available", err)
	}
	details := make(map[string]any, len(applicationErr.Details)+1)
	for key, value := range applicationErr.Details {
		details[key] = value
	}
	details["recovery"] = recovery
	return app.WithDetails(applicationErr, details)
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
		if name == "validate" {
			return presentOperationOutcome(command, service.OperationLedgerValidate, false, map[string]any{"ledger_id": loaded.Model.LedgerID, "schema_version": loaded.Model.SchemaVersion}, nil, "")
		}
		status, err := service.StatusForLedger(loaded)
		if err != nil {
			return app.NewError(app.CodeInternal, "ledger status could not be built", err)
		}
		message := fmt.Sprintf("%s: %d questions, %d forecasts; integrity: %d unanchored, %d pending, %d verified, %d failed",
			status.LedgerID, status.Questions, status.Forecasts, status.Unanchored, status.Pending, status.Verified, status.Failed)
		return presentOperationOutcome(command, service.OperationLedgerStatus, false, status, nil, message)
	}
	return command
}

func platformCommand() *urfavecli.Command {
	addFlags := append([]urfavecli.Flag{fileFlag(false), platformFlag()}, platformCreateFlags()...)
	add := leaf("add", "Add a platform", "forecast-ledger platform add --file ledger.yaml --platform metaculus --name Metaculus --kind scoring_platform", false, addFlags)
	add.Description += "\n\nUse direct flags for authoring."
	add.Action = platformAddAction
	updateFlags := append([]urfavecli.Flag{fileFlag(false), platformFlag()}, platformPatchFlags()...)
	update := leaf("update", "Update a platform", "forecast-ledger platform update --file ledger.yaml --platform metaculus --url https://www.metaculus.com", false, updateFlags)
	update.Description += "\n\nOmitted fields are unchanged. Use --clear-url or --clear-account for explicit removal."
	update.Action = platformUpdateAction
	list := leaf("list", "List platforms", "forecast-ledger platform list --file ledger.yaml", true, []urfavecli.Flag{fileFlag(true)})
	list.Action = platformListAction
	show := leaf("show", "Show a platform", "forecast-ledger platform show --file ledger.yaml --platform metaculus", true, []urfavecli.Flag{fileFlag(true), platformFlag()})
	show.Action = platformShowAction
	remove := leaf("remove", "Remove an unused platform", "forecast-ledger platform remove --file ledger.yaml --platform old-platform --yes", false, []urfavecli.Flag{fileFlag(false), platformFlag()})
	remove.Action = platformRemoveAction
	return group("platform", "Manage platform records", add, update, list, show, remove)
}

func platformAddAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	input, err := buildPlatformCreateInput(command)
	if err != nil {
		return err
	}
	id := ledger.Slug(command.String("platform"))
	var result service.PlatformFileResult
	if runtime.DryRun {
		result, err = service.PlanPlatformAddFile(operationContext, command.String("file"), id, input)
	} else {
		result, err = service.CommitPlatformAddFile(operationContext, command.String("file"), id, input)
	}
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationPlatformAdd, runtime.DryRun, result, nil, "")
}

func platformUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	input, err := buildPlatformPatchInput(command)
	if err != nil {
		return err
	}
	id := ledger.Slug(command.String("platform"))
	var result service.PlatformFileResult
	if runtime.DryRun {
		result, err = service.PlanPlatformUpdateFile(operationContext, command.String("file"), id, input)
	} else {
		result, err = service.CommitPlatformUpdateFile(operationContext, command.String("file"), id, input)
	}
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationPlatformUpdate, runtime.DryRun, result, nil, "")
}

func platformListAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	ledgerID, items, err := service.LoadPlatformList(operationContext, command.String("file"), command.Root().Reader)
	if err != nil {
		return err
	}
	var lines strings.Builder
	for index, item := range items {
		if index > 0 {
			lines.WriteByte('\n')
		}
		fmt.Fprintf(&lines, "%s\t%s\t%s\t%d", item.ID, item.Kind, item.Name, item.ReferenceCount)
	}
	message := lines.String()
	if message == "" {
		message = "No platforms"
	}
	return presentOperationOutcome(command, service.OperationPlatformList, false, map[string]any{"ledger_id": ledgerID, "platforms": items}, nil, message)
}

func platformShowAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	ledgerID, result, err := service.LoadPlatformShow(operationContext, command.String("file"), command.Root().Reader, ledger.Slug(command.String("platform")))
	if err != nil {
		return err
	}
	message := fmt.Sprintf("%s\t%s\t%s\t%d", result.ID, result.Platform.Kind, result.Platform.Name, len(result.ReferencingQuestionIDs))
	return presentOperationOutcome(command, service.OperationPlatformShow, false, map[string]any{"ledger_id": ledgerID, "platform": result}, nil, message)
}

func platformRemoveAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	id := ledger.Slug(command.String("platform"))
	if runtime.DryRun {
		result, err := service.PlanPlatformRemoveFile(operationContext, command.String("file"), id)
		if err != nil {
			return err
		}
		return presentOperationOutcome(command, service.OperationPlatformRemove, true, result, nil, "")
	}
	approved, err := runtime.Confirm(operationContext, fmt.Sprintf("Remove platform %s?", id))
	if err != nil {
		return err
	}
	if !approved {
		return app.NewError(app.CodeConflict, "platform removal was not approved", nil)
	}
	result, err := service.CommitPlatformRemoveFile(operationContext, command.String("file"), id)
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationPlatformRemove, false, result, nil, "")
}

func questionCommand() *urfavecli.Command {
	addFlags := []urfavecli.Flag{fileFlag(false), questionFlag(), &urfavecli.StringFlag{Name: "type", Required: true, Usage: "Question type: binary, multiple_choice, numeric, or date"}, &urfavecli.StringFlag{Name: "key-file", OnlyOnce: true, TakesFile: true, Usage: "New protected key file; required only for a sealed first forecast"}}
	addFlags = append(addFlags, questionCreateFlags(true)...)
	add := leaf("add", "Add a question, optionally with its first forecast", "forecast-ledger question add --file ledger.yaml --question q-launch --type binary --title 'Will it launch?' --resolution-criteria 'Resolves yes on launch' --expected-resolution-at '2 Feb 2027'", false, addFlags)
	add.Description += "\n\nUse direct flags for authoring. Repeated structured values use the documented CSV field order; sealed private data remains protected."
	add.Action = questionAddAction
	updateFlags := append([]urfavecli.Flag{fileFlag(false), questionFlag()}, questionPatchFlags()...)
	update := leaf("update", "Update allowed question fields", "forecast-ledger question update --file ledger.yaml --question q-launch --title 'Updated title' --tag launch --tag space", false, updateFlags)
	update.Description += "\n\nOmitted fields are unchanged; --clear-* removes optional values."
	update.Action = questionUpdateAction
	list := leaf("list", "List questions", "forecast-ledger question list --file ledger.yaml", true, []urfavecli.Flag{fileFlag(true)})
	list.Action = questionListAction
	show := leaf("show", "Show a question", "forecast-ledger question show --file ledger.yaml --question q-launch", true, []urfavecli.Flag{fileFlag(true), questionFlag()})
	show.Description += "\n\nNormal human and plain output includes public business fields and redacted forecast summaries."
	show.Action = questionShowAction
	resolveFlags := append([]urfavecli.Flag{fileFlag(false), questionFlag()}, lifecycleFlags(true)...)
	resolve := leaf("resolve", "Resolve a question", "forecast-ledger question resolve --file ledger.yaml --question q-launch --outcome-boolean=true --outcome-known-at 2027-01-02T00:00:00Z --source 'Official result,https://example.com/result,2027-01-02T00:10:00Z' --yes", false, resolveFlags)
	resolve.Action = questionResolveAction
	annulFlags := append([]urfavecli.Flag{fileFlag(false), questionFlag()}, lifecycleFlags(false)...)
	annul := leaf("annul", "Annul a question", "forecast-ledger question annul --file ledger.yaml --question q-launch --reason 'Question became unresolvable' --yes", false, annulFlags)
	annul.Action = questionAnnulAction
	disputeFlags := append([]urfavecli.Flag{fileFlag(false), questionFlag()}, lifecycleFlags(false)...)
	dispute := leaf("dispute", "Dispute a resolution", "forecast-ledger question dispute --file ledger.yaml --question q-launch --reason 'Source conflicts with the recorded outcome' --yes", false, disputeFlags)
	dispute.Action = questionDisputeAction
	for _, command := range []*urfavecli.Command{resolve, annul, dispute} {
		command.Description += "\n\nUse repeated --source values as title,url,retrieved-at[,publisher[,published-at[,sha256]]]."
	}
	return group("question", "Manage forecast questions", add, update, list, show, resolve, annul, dispute)
}

func questionAddAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	timezone, err := mutationTimezone(operationContext, command)
	if err != nil {
		return err
	}
	observedAt, err := formatOperationTime(commandEffects(command).Clock.Now(), timezone)
	if err != nil {
		return err
	}
	if command.IsSet("initial-forecast") && !command.IsSet("initial-forecasted-at") {
		if err := command.Set("initial-forecasted-at", string(observedAt)); err != nil {
			return err
		}
	}
	var normalizedTimes []service.TimeNormalization
	for _, item := range []struct {
		name   string
		policy dateOnlyPolicy
	}{{"created-at", dateOnlyRejected}, {"opens-at", dateOnlyStart}, {"expected-resolution-at", dateOnlyEnd}, {"initial-forecasted-at", dateOnlyRejected}, {"initial-recorded-at", dateOnlyRejected}} {
		normalized, err := normalizeSetTimeWithMetadata(command, item.name, timezone, item.policy)
		if err != nil {
			return err
		}
		normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	}
	input, err := buildQuestionAddInput(operationContext, command, command.Root().Reader)
	if err != nil {
		return err
	}
	normalized := service.NormalizedQuestionCreate{ID: ledger.Slug(command.String("question")), Type: ledger.QuestionType(command.String("type")), Input: input}
	keyPath := command.String("key-file")
	shape, err := service.ClassifyQuestionAddInput(input)
	if err != nil {
		return err
	}
	var result service.QuestionFileResult
	humanMessage := ""
	if shape != service.CreationSealedForecast && keyPath != "" {
		return app.NewError(app.CodeUsage, "--key-file is only valid for a sealed initial forecast", nil)
	}
	switch shape {
	case service.CreationQuestionOnly:
		if runtime.DryRun {
			result, err = service.PlanQuestionAddEmptyFile(operationContext, command.String("file"), normalized, observedAt)
		} else {
			result, err = service.CommitQuestionAddEmptyFile(operationContext, command.String("file"), normalized, observedAt)
		}
	case service.CreationPublicForecast:
		humanMessage = "Question and initial forecast were added"
		if runtime.DryRun {
			result, err = service.PlanQuestionAddPublicFile(operationContext, command.String("file"), normalized, observedAt)
		} else {
			result, err = service.CommitQuestionAddPublicFile(operationContext, command.String("file"), normalized, observedAt)
		}
	case service.CreationSealedForecast:
		humanMessage = "Question and initial forecast were added"
		protectedInputPath := command.String("initial-secret-input")
		protectedArgument := "--initial-secret-input"
		if protectedInputPath != "-" {
			if err := storage.CheckProtectedFile(protectedInputPath); err != nil {
				return protectedArgumentError(err, protectedArgument)
			}
		}
		if strings.TrimSpace(keyPath) == "" {
			return app.NewError(app.CodeUsage, "--key-file is required for a sealed initial forecast", nil)
		}
		if runtime.DryRun {
			result, err = service.PlanQuestionAddSealedFile(operationContext, command.String("file"), keyPath, normalized, observedAt)
		} else {
			result, err = service.CommitQuestionAddSealedFile(operationContext, command.String("file"), keyPath, normalized, observedAt, commandEffects(command))
		}
	}
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	result.NormalizedTimes = normalizedTimes
	if runtime.DryRun {
		humanMessage = ""
	}
	return presentOperationOutcome(command, service.OperationQuestionAdd, runtime.DryRun, result, nil, humanMessage)
}

func questionUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	timezone, err := mutationTimezone(operationContext, command)
	if err != nil {
		return err
	}
	var normalizedTimes []service.TimeNormalization
	normalized, err := normalizeSetTimeWithMetadata(command, "opens-at", timezone, dateOnlyStart)
	if err != nil {
		return err
	}
	normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	normalized, err = normalizeSetTimeWithMetadata(command, "expected-resolution-at", timezone, dateOnlyEnd)
	if err != nil {
		return err
	}
	normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	input, err := buildQuestionPatchInput(command)
	if err != nil {
		return err
	}
	id := ledger.Slug(command.String("question"))
	var result service.QuestionFileResult
	if runtime.DryRun {
		result, err = service.PlanQuestionUpdateFile(operationContext, command.String("file"), id, input)
	} else {
		result, err = service.CommitQuestionUpdateFile(operationContext, command.String("file"), id, input)
	}
	if err != nil {
		return err
	}
	result.NormalizedTimes = normalizedTimes
	return presentOperationOutcome(command, service.OperationQuestionUpdate, runtime.DryRun, result, nil, "")
}

func questionListAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	ledgerID, items, err := service.LoadQuestionList(operationContext, command.String("file"), command.Root().Reader)
	if err != nil {
		return err
	}
	var lines strings.Builder
	for index, item := range items {
		if index > 0 {
			lines.WriteByte('\n')
		}
		fmt.Fprintf(&lines, "%s\t%s\t%s\t%s\t%d\t%s", item.ID, item.Title, item.Type, item.Status, item.ForecastCount, item.ExpectedResolutionAt)
	}
	message := lines.String()
	if message == "" {
		message = "No questions"
	}
	return presentOperationOutcome(command, service.OperationQuestionList, false, map[string]any{"ledger_id": ledgerID, "questions": items}, nil, message)
}

func questionShowAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	id := ledger.Slug(command.String("question"))
	ledgerID, result, err := service.LoadQuestionShow(operationContext, command.String("file"), command.Root().Reader, id)
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	message := "Question was found"
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = formatQuestionView(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationQuestionShow, false, map[string]any{"ledger_id": ledgerID, "question": result}, nil, message)
}

func questionResolveAction(ctx context.Context, command *urfavecli.Command) error {
	var input service.ResolutionInput
	return questionTerminalAction(ctx, command, func(command *urfavecli.Command) error {
		value, err := buildResolutionInput(command)
		input = value
		return err
	}, "resolve", func(operationContext context.Context, id ledger.Slug, observedAt ledger.Timestamp, dryRun bool) (service.QuestionFileResult, error) {
		if dryRun {
			return service.PlanQuestionResolveFile(operationContext, command.String("file"), id, input, observedAt)
		}
		return service.CommitQuestionResolveFile(operationContext, command.String("file"), id, input, observedAt)
	})
}

func questionAnnulAction(ctx context.Context, command *urfavecli.Command) error {
	var input service.AnnulInput
	return questionTerminalAction(ctx, command, func(command *urfavecli.Command) error {
		reason, recordedAt, sources, err := buildReasonInput(command)
		input = service.AnnulInput{Reason: reason, RecordedAt: recordedAt, Sources: sources}
		return err
	}, "annul", func(operationContext context.Context, id ledger.Slug, observedAt ledger.Timestamp, dryRun bool) (service.QuestionFileResult, error) {
		if dryRun {
			return service.PlanQuestionAnnulFile(operationContext, command.String("file"), id, input, observedAt)
		}
		return service.CommitQuestionAnnulFile(operationContext, command.String("file"), id, input, observedAt)
	})
}

func questionDisputeAction(ctx context.Context, command *urfavecli.Command) error {
	var input service.DisputeInput
	return questionTerminalAction(ctx, command, func(command *urfavecli.Command) error {
		reason, recordedAt, sources, err := buildReasonInput(command)
		input = service.DisputeInput{Reason: reason, RecordedAt: recordedAt, Sources: sources}
		return err
	}, "dispute", func(operationContext context.Context, id ledger.Slug, observedAt ledger.Timestamp, dryRun bool) (service.QuestionFileResult, error) {
		if dryRun {
			return service.PlanQuestionDisputeFile(operationContext, command.String("file"), id, input, observedAt)
		}
		return service.CommitQuestionDisputeFile(operationContext, command.String("file"), id, input, observedAt)
	})
}

func questionTerminalAction(ctx context.Context, command *urfavecli.Command, buildDirect func(*urfavecli.Command) error, verb string, execute func(context.Context, ledger.Slug, ledger.Timestamp, bool) (service.QuestionFileResult, error)) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	timezone, err := mutationTimezone(operationContext, command)
	if err != nil {
		return err
	}
	var normalizedTimes []service.TimeNormalization
	for _, name := range []string{"outcome-known-at", "recorded-at"} {
		normalized, err := normalizeSetTimeWithMetadata(command, name, timezone, dateOnlyRejected)
		if err != nil {
			return err
		}
		normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	}
	if err := buildDirect(command); err != nil {
		return err
	}
	id := ledger.Slug(command.String("question"))
	if !runtime.DryRun {
		approved, err := runtime.Confirm(operationContext, fmt.Sprintf("%s question %s?", strings.ToUpper(verb[:1])+verb[1:], id))
		if err != nil {
			return err
		}
		if !approved {
			return app.NewError(app.CodeConflict, "question lifecycle change was not approved", nil)
		}
	}
	observedAt := ledger.Timestamp(commandEffects(command).Clock.Now().Format(time.RFC3339))
	result, err := execute(operationContext, id, observedAt, runtime.DryRun)
	if err != nil {
		return err
	}
	result.NormalizedTimes = normalizedTimes
	operation := service.OperationQuestionDispute
	humanMessage := ""
	if verb == "resolve" {
		operation = service.OperationQuestionResolve
	} else if verb == "annul" {
		operation = service.OperationQuestionAnnul
		if !runtime.DryRun {
			humanMessage = "Question was annulled; the question and forecasts were retained"
		}
	}
	return presentOperationOutcome(command, operation, runtime.DryRun, result, nil, humanMessage)
}

func forecastCommand() *urfavecli.Command {
	addFlags := append([]urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag()}, forecastCreateFlags()...)
	add := leaf("add", "Add a public forecast", "forecast-ledger forecast add --file ledger.yaml --question q-launch --forecast f-002 --forecasted-at 2026-12-01T12:00:00Z --value-kind binary --probability-bp 6500", false, addFlags)
	add.Description += "\n\nUse --value-kind with --probability-bp, repeated --choice-probability, --point, --interval, or repeated --quantile as applicable."
	add.Action = forecastAddAction
	list := leaf("list", "List forecasts", "forecast-ledger forecast list --file ledger.yaml --question q-launch", true, []urfavecli.Flag{fileFlag(true), questionFlag()})
	list.Action = forecastListAction
	show := leaf("show", "Show a forecast", "forecast-ledger forecast show --file ledger.yaml --question q-launch --forecast f-001", true, []urfavecli.Flag{fileFlag(true), questionFlag(), forecastFlag()})
	show.Description += "\n\nNormal human and plain output includes type-aware public values and safe stored integrity evidence; sealed private fields stay redacted. No network check is performed."
	show.Action = forecastShowAction
	sealFlags := append([]urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), secretOutputFlag()}, forecastSealPublicFlags()...)
	seal := leaf("seal", "Create and append a sealed forecast", "forecast-ledger forecast seal --file ledger.yaml --question q-launch --forecast f-002 --forecasted-at 2026-12-01T12:00:00Z --secret-input private.yaml --key-file secret.key", false, sealFlags)
	seal.Description += "\n\nValue, rationale, key factors, and comment stay in protected --secret-input while public times, note, and supersedes ID use flags."
	seal.Action = forecastSealAction
	reveal := leaf("reveal", "Verify and reveal a sealed forecast", "forecast-ledger forecast reveal --file ledger.yaml --question q-launch --forecast f-002 --key-file secret.key --yes", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), &urfavecli.StringFlag{Name: "key-file", Required: true, TakesFile: true, Usage: "Protected key file"}, &urfavecli.StringFlag{Name: "revealed-at", OnlyOnce: true, Usage: "Explicit RFC 3339 reveal time; defaults to the current clock"}})
	reveal.Action = forecastRevealAction
	hintUpdate := leaf("update", "Change a non-location key hint", "forecast-ledger forecast key-hint update --file ledger.yaml --question q-launch --forecast f-002 --key-hint forecast-key:f-002", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), &urfavecli.StringFlag{Name: "key-hint", Required: true, OnlyOnce: true, Usage: "Safe scheme:opaque logical hint"}})
	hintUpdate.Action = forecastKeyHintUpdateAction
	return group("forecast", "Manage append-only forecast records", add, list, show, seal, reveal, group("key-hint", "Manage non-authoritative key hints", hintUpdate))
}

func forecastAddAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	timezone, err := mutationTimezone(operationContext, command)
	if err != nil {
		return err
	}
	observedAt, err := formatOperationTime(commandEffects(command).Clock.Now(), timezone)
	if err != nil {
		return err
	}
	if !command.IsSet("forecasted-at") {
		if err := command.Set("forecasted-at", string(observedAt)); err != nil {
			return err
		}
	}
	var normalizedTimes []service.TimeNormalization
	for _, name := range []string{"forecasted-at", "recorded-at"} {
		normalized, err := normalizeSetTimeWithMetadata(command, name, timezone, dateOnlyRejected)
		if err != nil {
			return err
		}
		normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	}
	input, err := buildForecastCreateInput(command)
	if err != nil {
		return err
	}
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	var result service.ForecastFileResult
	if runtime.DryRun {
		result, err = service.PlanPublicForecastAddFile(operationContext, command.String("file"), questionID, forecastID, input, observedAt)
	} else {
		result, err = service.CommitPublicForecastAddFile(operationContext, command.String("file"), questionID, forecastID, input, observedAt)
	}
	if err != nil {
		return err
	}
	result.NormalizedTimes = normalizedTimes
	return presentOperationOutcome(command, service.OperationForecastAdd, runtime.DryRun, result, nil, "")
}

func forecastListAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	questionID := ledger.Slug(command.String("question"))
	ledgerID, items, err := service.LoadForecastList(operationContext, command.String("file"), command.Root().Reader, questionID)
	if err != nil {
		return err
	}
	var lines strings.Builder
	for index, item := range items {
		if index > 0 {
			lines.WriteByte('\n')
		}
		fmt.Fprintf(&lines, "%s\t%s\t%s\t%s\t%s\t%s", item.ID, item.ForecastedAt, item.RecordedAt, item.Visibility, item.IntegrityStatus, item.ValueSummary)
	}
	message := lines.String()
	if message == "" {
		message = "No forecasts"
	}
	return presentOperationOutcome(command, service.OperationForecastList, false, map[string]any{"ledger_id": ledgerID, "question_id": questionID, "forecasts": items}, nil, message)
}

func forecastShowAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	ledgerID, result, err := service.LoadForecastShow(operationContext, command.String("file"), command.Root().Reader, questionID, forecastID)
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	message := "Forecast was found"
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = formatForecastView(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationForecastShow, false, map[string]any{"ledger_id": ledgerID, "question_id": questionID, "forecast": result}, nil, message)
}

func forecastSealAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	timezone, err := mutationTimezone(operationContext, command)
	if err != nil {
		return err
	}
	observedAt, err := formatOperationTime(commandEffects(command).Clock.Now(), timezone)
	if err != nil {
		return err
	}
	if !command.IsSet("forecasted-at") {
		if err := command.Set("forecasted-at", string(observedAt)); err != nil {
			return err
		}
	}
	var normalizedTimes []service.TimeNormalization
	for _, name := range []string{"forecasted-at", "recorded-at"} {
		normalized, err := normalizeSetTimeWithMetadata(command, name, timezone, dateOnlyRejected)
		if err != nil {
			return err
		}
		normalizedTimes = appendTimeNormalization(normalizedTimes, normalized)
	}
	input, err := buildSealedForecastInput(operationContext, command, command.Root().Reader)
	if err != nil {
		return err
	}
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	var result service.ForecastFileResult
	if runtime.DryRun {
		result, err = service.PlanForecastSealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, input, observedAt)
	} else {
		result, err = service.CommitForecastSealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, input, observedAt, commandEffects(command))
	}
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	result.NormalizedTimes = normalizedTimes
	return presentOperationOutcome(command, service.OperationForecastSeal, runtime.DryRun, result, nil, "")
}

func forecastRevealAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	approved, err := runtime.Confirm(operationContext, fmt.Sprintf("Reveal forecast %s and publish its private fields and key?", forecastID))
	if err != nil {
		return err
	}
	if !approved {
		return app.NewError(app.CodeConflict, "forecast reveal was not approved", nil)
	}
	revealedAt := ledger.Timestamp(command.String("revealed-at"))
	if revealedAt == "" {
		revealedAt = ledger.Timestamp(commandEffects(command).Clock.Now().Format(time.RFC3339))
	}
	var result service.ForecastFileResult
	if runtime.DryRun {
		result, err = service.PlanForecastRevealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, revealedAt)
	} else {
		result, err = service.CommitForecastRevealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, revealedAt)
	}
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationForecastReveal, runtime.DryRun, result, nil, "")
}

func forecastKeyHintUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	keyHint := command.String("key-hint")
	var result service.ForecastFileResult
	var err error
	if runtime.DryRun {
		result, err = service.PlanForecastKeyHintUpdateFile(operationContext, command.String("file"), questionID, forecastID, keyHint)
	} else {
		result, err = service.CommitForecastKeyHintUpdateFile(operationContext, command.String("file"), questionID, forecastID, keyHint)
	}
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationForecastKeyHintUpdate, runtime.DryRun, result, nil, "")
}

func decodePrivateOperationInputForArgument(ctx context.Context, path string, stdin io.Reader, schema service.InputSchemaName, destination any, argument string) error {
	if path == "-" {
		return service.DecodeOperationInput(ctx, path, stdin, schema, destination)
	}
	data, err := storage.ReadProtectedFile(path, 8<<20)
	if err != nil {
		return protectedArgumentError(err, argument)
	}
	defer clear(data)
	return service.DecodeOperationInput(ctx, "-", bytes.NewReader(data), schema, destination)
}

func protectedArgumentError(err error, argument string) error {
	var applicationErr *app.Error
	if !errors.As(err, &applicationErr) {
		return err
	}
	message := applicationErr.Message
	for _, phrase := range []string{"protected key file", "protected key path", "protected file"} {
		message = strings.ReplaceAll(message, phrase, "protected "+argument+" file")
	}
	details := make(map[string]any, len(applicationErr.Details)+1)
	for key, value := range applicationErr.Details {
		details[key] = value
	}
	details["argument"] = argument
	return app.WithDetails(app.NewError(applicationErr.Code, message, applicationErr.Cause), details)
}

func targetCommand() *urfavecli.Command {
	build := targetLeaf("build", "Build target artifacts", false)
	build.Action = targetBuildAction
	check := targetLeaf("check", "Check target bytes and digests", true)
	check.Description += "\n\nA never-built target is reported as not_applicable with build guidance; --all continues in ledger order."
	check.Action = targetCheckAction
	return group("target", "Build or check canonical forecast targets", build, check)
}

func targetLeaf(name, usage string, readOnly bool) *urfavecli.Command {
	flags := []urfavecli.Flag{fileFlag(false), &urfavecli.StringFlag{Name: "question", Usage: "Question ID"}, &urfavecli.StringFlag{Name: "forecast", Usage: "Forecast ID"}, &urfavecli.BoolFlag{Name: "all", Usage: "Select every forecast"}}
	command := leaf(name, usage, fmt.Sprintf("forecast-ledger target %s --file ledger.yaml --question q-launch --forecast f-001", name), readOnly, flags)
	command.Before = requireTargetSelection
	return command
}

func targetBuildAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	all := command.Bool("all")
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	var result service.TargetOperationResult
	var err error
	if runtime.DryRun {
		result, err = service.PlanTargetBuild(operationContext, command.String("file"), all, questionID, forecastID)
	} else {
		result, err = service.CommitTargetBuild(operationContext, command.String("file"), all, questionID, forecastID)
	}
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	return presentOperationOutcome(command, service.OperationTargetBuild, runtime.DryRun, result, nil, "")
}

func targetCheckAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.InspectTargets(operationContext, command.String("file"), command.Bool("all"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")))
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	humanMessage := ""
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		humanMessage = formatTargetInspection(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationTargetCheck, false, result, nil, humanMessage)
}

func formatTargetInspection(mode presentation.Mode, result service.TargetOperationResult) string {
	var output strings.Builder
	for index, target := range result.Targets {
		if index > 0 {
			output.WriteByte('\n')
		}
		reasons := strings.Join(target.ReasonCodes, ",")
		if mode == presentation.ModePlain {
			fmt.Fprintf(&output, "%s\t%s\t%s\t%s\t%s\t%s\t%s", target.QuestionID, target.ForecastID, target.State, reasons, target.Path, target.SHA256, target.ActualSHA256)
			if target.Guidance != "" {
				fmt.Fprintf(&output, "\t%s", target.Guidance)
			}
			continue
		}
		fmt.Fprintf(&output, "Target: %s / %s\n  State: %s\n  Path: %s\n  Expected SHA-256: %s", target.QuestionID, target.ForecastID, target.State, target.Path, target.SHA256)
		if target.ActualSHA256 != "" {
			fmt.Fprintf(&output, "\n  Actual SHA-256: %s", target.ActualSHA256)
		}
		if reasons != "" {
			fmt.Fprintf(&output, "\n  Reason: %s", reasons)
		}
		if target.Guidance != "" {
			fmt.Fprintf(&output, "\n  Next: %s", target.Guidance)
		}
	}
	return output.String()
}

func timestampCommand() *urfavecli.Command {
	stamp := timestampLeaf("stamp", "Request and retain an RFC 3161 timestamp", false, true)
	stamp.Action = timestampStampAction
	status := timestampLeaf("status", "Show local RFC 3161 evidence status", true, false)
	status.Action = timestampStatusAction
	verify := timestampLeaf("verify", "Verify retained RFC 3161 evidence locally", false, false)
	verify.Action = timestampVerifyAction
	command := group("timestamp", "Manage experimental RFC 3161 request and response evidence", stamp, status, verify)
	command.Description = "Stamp defaults to the built-in FreeTSA profile and retained embedded trust. A named provider or an explicit public HTTPS TSA plus ledger-relative PEM bundle can be selected instead. Status and verification are local and make no timestamp-service network request."
	return command
}

func timestampLeaf(name, usage string, readOnly, stampOptions bool) *urfavecli.Command {
	flags := []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag()}
	if stampOptions {
		flags = append(flags, &urfavecli.BoolFlag{Name: "offline", Usage: "Open no network connection"})
		flags = append(flags,
			&urfavecli.StringFlag{Name: "tsa-provider", Usage: "Built-in timestamp provider: auto or freetsa (default: auto)"},
			&urfavecli.StringFlag{Name: "tsa-url", Usage: "Custom public HTTPS RFC 3161 timestamp authority URL; requires --ca-bundle"},
			&urfavecli.StringFlag{Name: "ca-bundle", TakesFile: true, Usage: "Custom retained ledger-relative PEM CA bundle; requires --tsa-url"},
		)
	}
	command := leaf(name, usage, fmt.Sprintf("forecast-ledger timestamp %s --file ledger.yaml --question q-launch --forecast f-001", name), readOnly, flags)
	return command
}

func timestampStampAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.CommitTimestampStamp(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")), service.TimestampStampOptions{
		DryRun: runtime.DryRun, Offline: command.Bool("offline"), TSAProvider: command.String("tsa-provider"), TSAURL: command.String("tsa-url"), CABundlePath: command.String("ca-bundle"), Effects: commandEffects(command),
	})
	if err != nil && result.FailureCode == "" {
		return withRecovery(err, result.Recovery)
	}
	presenter := presenterFor(command)
	humanMessage := ""
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		humanMessage = formatTimestampArtifact(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationTimestampStamp, runtime.DryRun, result, err, humanMessage)
}

func timestampStatusAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.TimestampStatusFor(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")))
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	message := "RFC 3161 local status: " + string(result.State)
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = formatTimestampArtifact(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationTimestampStatus, false, result, nil, message)
}

func timestampVerifyAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.CommitTimestampVerify(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")), service.TimestampVerifyOptions{DryRun: runtime.DryRun, Effects: commandEffects(command)})
	if err != nil && result.FailureCode == "" {
		return err
	}
	presenter := presenterFor(command)
	humanMessage := ""
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		humanMessage = formatTimestampVerification(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationTimestampVerify, runtime.DryRun, result, err, humanMessage)
}

func verifyCommand() *urfavecli.Command {
	command := leaf("verify", "Run layered verification", "forecast-ledger verify --file ledger.yaml --offline", true, []urfavecli.Flag{
		fileFlag(false),
		&urfavecli.StringFlag{Name: "question", Usage: "Optional question ID"},
		&urfavecli.StringFlag{Name: "forecast", Usage: "Optional forecast ID; requires --question"},
		&urfavecli.BoolFlag{Name: "offline", Usage: "Do not retrieve optional outcome sources; timestamp checks remain local"},
		&urfavecli.BoolFlag{Name: "check-sources", Usage: "Check outcome source reachability and stored digests"},
	})
	command.Before = func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
		if command.String("forecast") != "" && command.String("question") == "" {
			return ctx, app.NewError(app.CodeUsage, "--forecast requires --question", nil)
		}
		return ctx, nil
	}
	command.Action = verificationAction
	command.Description += "\n\nNormal human and plain output includes the complete ordered evidence matrix and safe retained timing values. Offline stored values are not freshly rechecked. Pass requires at least one applicable forecast-evidence layer; an empty or all-not-applicable selection returns no_evidence."
	return command
}

func verificationAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	report, err := service.VerifyLedgerEvidence(operationContext, command.String("file"), service.VerificationOptions{
		Offline: command.Bool("offline"), CheckSources: command.Bool("check-sources"),
		QuestionID: ledger.Slug(command.String("question")), ForecastID: ledger.Slug(command.String("forecast")),
	})
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	humanMessage := ""
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		humanMessage = formatVerificationReport(presenter.Mode(), report)
	}
	return presentOperationOutcome(command, service.OperationVerificationRun, false, report, nil, humanMessage)
}

func publishCommand() *urfavecli.Command {
	build := leaf("build", "Build an evidence package", "forecast-ledger publish build --file ledger.yaml --output evidence", false, []urfavecli.Flag{fileFlag(false), &urfavecli.StringFlag{Name: "output", Required: true, TakesFile: true, Usage: "New package directory"}})
	build.Action = publicationBuildAction
	verify := leaf("verify", "Verify an evidence package", "forecast-ledger publish verify --file package/ledger/ledger.yaml --manifest package/manifest.json", true, []urfavecli.Flag{
		fileFlag(false),
		&urfavecli.StringFlag{Name: "manifest", Required: true, TakesFile: true, Usage: "Package manifest file"},
	})
	verify.Description += "\n\nRFC 3161 target, request, response, and retained CA-bundle checks are always local. Manifest and file integrity remain visible separately."
	verify.Action = publicationVerifyAction
	return group("publish", "Build or verify portable evidence packages", build, verify)
}

func publicationBuildAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.CommitPublicationBuild(operationContext, command.String("file"), command.String("output"), runtime.DryRun)
	if err != nil {
		return err
	}
	return presentOperationOutcome(command, service.OperationPublicationBuild, runtime.DryRun, result, nil, "")
}

func publicationVerifyAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.VerifyPublicationPackage(operationContext, command.String("file"), command.String("manifest"))
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	humanMessage := ""
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		humanMessage = formatPublicationVerification(presenter.Mode(), result)
	}
	return presentOperationOutcome(command, service.OperationPublicationVerify, false, result, nil, humanMessage)
}

func mcpCommand() *urfavecli.Command {
	serve := leaf("serve", "Serve MCP over stdio", "forecast-ledger mcp serve --ledger-root main=/data/ledgers", true, []urfavecli.Flag{
		&urfavecli.StringSliceFlag{Name: "ledger-root", Required: true, Usage: "Allowed ledger root as name=path; repeat for more roots"},
		&urfavecli.StringSliceFlag{Name: "output-root", Usage: "Allowed package output root as name=path; repeat for more roots"},
		&urfavecli.StringSliceFlag{Name: "secret-root", Usage: "Allowed protected secret root as name=path; repeat for more roots"},
		&urfavecli.BoolFlag{Name: "read-only", Usage: "Disable every mutation for the whole server"},
		&urfavecli.BoolFlag{Name: "offline", Usage: "Open no network connection for the whole server"},
		&urfavecli.BoolFlag{Name: "allow-reveal", Usage: "Enable the otherwise absent irreversible forecast_reveal tool"},
		&urfavecli.IntFlag{Name: "max-concurrent", Value: 16, Usage: "Maximum concurrent MCP tool calls (1-256)"},
		&urfavecli.IntFlag{Name: "max-tool-bytes", Value: 8 << 20, Usage: "Maximum decoded tool argument bytes"},
	})
	serve.Description += "\n\nRead-only mode omits mutating tools from discovery; direct calls to omitted names return unknown-tool."
	serve.Description += " Ledger writers fail immediately on lock conflict; clients must serialize or use bounded retry with backoff."
	serve.Action = mcpServeAction
	return group("mcp", "Run the MCP adapter", serve)
}

func mcpServeAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	server, err := mcpadapter.New(mcpadapter.Config{
		LedgerRoots: command.StringSlice("ledger-root"), OutputRoots: command.StringSlice("output-root"), SecretRoots: command.StringSlice("secret-root"),
		Mode:          service.AccessMode{ReadOnly: command.Bool("read-only"), Offline: command.Bool("offline"), AllowReveal: command.Bool("allow-reveal")},
		Timeout:       runtime.Timeout,
		MaxConcurrent: command.Int("max-concurrent"), MaxToolBytes: command.Int("max-tool-bytes"), Stderr: command.Root().ErrWriter,
	})
	if err != nil {
		return err
	}
	return server.ServeStdio(ctx)
}

func versionCommand() *urfavecli.Command {
	return &urfavecli.Command{Name: "version", Usage: "Show build and contract versions", Description: "Example:\n  forecast-ledger version\n  forecast-ledger version --json",
		Flags: []urfavecli.Flag{&urfavecli.BoolFlag{Name: "json", Usage: "Write stable JSON metadata", Local: true}},
		Before: func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
			if command.Bool("json") && (command.Root().Bool("plain") || command.Root().Bool("quiet")) {
				return ctx, app.NewError(app.CodeUsage, "--json, --plain, and --quiet cannot be combined", nil)
			}
			return ctx, nil
		},
		Action: func(_ context.Context, command *urfavecli.Command) error {
			info := buildinfo.Current()
			if command.Bool("json") || command.Root().Bool("json") {
				encoder := json.NewEncoder(command.Root().Writer)
				encoder.SetEscapeHTML(false)
				return encoder.Encode(info)
			}
			presenter := presenterFor(command)
			if presenter.Mode() == presentation.ModeQuiet {
				return nil
			}
			return writeVersionInfo(command.Root().Writer, info, presenter.Mode(), presenter.ColorEnabled())
		}}
}

func writeVersionInfo(writer io.Writer, info buildinfo.Info, mode presentation.Mode, color bool) error {
	placeholder := func(value string) string {
		if strings.TrimSpace(value) == "" || value == "unknown" {
			return "not set"
		}
		return value
	}
	providers := strings.Join(info.Timestamp.Providers, ", ")
	if providers == "" {
		providers = "not set"
	}
	fields := [][2]string{
		{"Binary", placeholder(info.Binary)},
		{"Version", placeholder(info.Version)},
		{"Source revision", placeholder(info.SourceRevision)},
		{"Go", placeholder(info.GoVersion)},
		{"Forecast Ledger schema", placeholder(info.Schema.Version)},
		{"Schema commit", placeholder(info.Schema.Commit)},
		{"Schema SHA-256", placeholder(info.Schema.SHA256)},
		{"MCP protocol", placeholder(info.MCPProtocol)},
		{"Timestamp support", fmt.Sprintf("%s/%s (experimental; %s=%s, retained CA bundle; local verification)", info.Timestamp.Protocol, info.Timestamp.HashAlgorithm, info.Timestamp.DefaultMode, providers)},
		{"Timestamp providers", providers},
	}
	for index, field := range fields {
		if index > 0 {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
		if mode == presentation.ModeHuman && color {
			if _, err := fmt.Fprintf(writer, "\x1b[36m%s:\x1b[0m %s", field[0], field[1]); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(writer, "%s: %s", field[0], field[1]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(writer)
	return err
}

func formatQuestionView(mode presentation.Mode, view service.QuestionView) string {
	fields := [][2]string{
		{"id", string(view.ID)}, {"title", view.Title}, {"type", string(view.Type)}, {"status", string(view.Status)},
		{"resolution_criteria", view.ResolutionCriteria}, {"created_at", string(view.CreatedAt)},
		{"forecast_window", compactPublicJSON(view.ForecastWindow)}, {"expected_resolution_at", string(view.ExpectedResolutionAt)},
	}
	if view.Options != nil {
		fields = append(fields, [2]string{"options", compactPublicJSON(*view.Options)})
	}
	if view.Unit != nil {
		fields = append(fields, [2]string{"unit", compactPublicJSON(view.Unit)})
	}
	if view.PlatformRefs != nil {
		fields = append(fields, [2]string{"platform_refs", compactPublicJSON(*view.PlatformRefs)})
	}
	if view.Tags != nil {
		fields = append(fields, [2]string{"tags", compactPublicJSON(*view.Tags)})
	}
	if view.Notes != nil {
		fields = append(fields, [2]string{"notes", *view.Notes})
	}
	if view.Resolution != nil {
		fields = append(fields, [2]string{"resolution", compactPublicJSON(view.Resolution)})
	}
	var output strings.Builder
	writeDisplayFields(&output, mode, fields)
	for _, forecast := range view.Forecasts {
		if output.Len() > 0 {
			output.WriteByte('\n')
		}
		if mode == presentation.ModePlain {
			fmt.Fprintf(&output, "forecast\t%s\t%s\t%s\t%s", forecast.Summary.ID, forecast.Summary.Visibility, forecast.Summary.IntegrityStatus, forecast.Summary.ValueSummary)
		} else {
			fmt.Fprintf(&output, "Forecast: %s (%s, %s)", forecast.Summary.ID, forecast.Summary.Visibility, forecast.Summary.IntegrityStatus)
			if forecast.Summary.ValueSummary != "" {
				fmt.Fprintf(&output, "\n  Value: %s", forecast.Summary.ValueSummary)
			}
		}
	}
	return output.String()
}

func formatForecastView(mode presentation.Mode, view service.ForecastView) string {
	fields := [][2]string{
		{"id", string(view.Summary.ID)}, {"forecasted_at", string(view.Summary.ForecastedAt)}, {"recorded_at", string(view.Summary.RecordedAt)},
		{"visibility", string(view.Summary.Visibility)}, {"integrity_status", string(view.Summary.IntegrityStatus)},
		{"integrity", compactPublicJSON(view.Integrity)},
	}
	if view.Value != nil {
		fields = append(fields, [2]string{"value", compactPublicJSON(view.Value)})
	}
	if view.Rationale != nil {
		fields = append(fields, [2]string{"rationale", *view.Rationale})
	}
	if view.KeyFactors != nil {
		fields = append(fields, [2]string{"key_factors", compactPublicJSON(*view.KeyFactors)})
	}
	if view.Comment != nil {
		fields = append(fields, [2]string{"comment", *view.Comment})
	}
	if view.PublicNote != nil {
		fields = append(fields, [2]string{"public_note", *view.PublicNote})
	}
	if view.Summary.SupersedesForecastID != nil {
		fields = append(fields, [2]string{"supersedes_forecast_id", string(*view.Summary.SupersedesForecastID)})
	}
	if view.Commitment != nil {
		fields = append(fields, [2]string{"commitment", compactPublicJSON(view.Commitment)})
	}
	var output strings.Builder
	writeDisplayFields(&output, mode, fields)
	return output.String()
}

func formatVerificationReport(mode presentation.Mode, report service.VerificationReport) string {
	var output strings.Builder
	if mode == presentation.ModePlain {
		fmt.Fprintf(&output, "overall\t%s\ndocument\t%s", report.Overall, report.Document.State)
		for _, forecast := range report.Forecasts {
			for _, layer := range forecast.Layers {
				fmt.Fprintf(&output, "\n%s\t%s\t%s\t%s\t%s\t%s", forecast.QuestionID, forecast.ForecastID, layer.Name, layer.State, strings.Join(layer.ReasonCodes, ","), compactPublicJSON(layer.Evidence))
			}
		}
		return output.String()
	}
	fmt.Fprintf(&output, "Overall: %s\nDocument: %s", report.Overall, report.Document.State)
	for _, forecast := range report.Forecasts {
		fmt.Fprintf(&output, "\nForecast: %s / %s", forecast.QuestionID, forecast.ForecastID)
		for _, layer := range forecast.Layers {
			fmt.Fprintf(&output, "\n  %s: %s", layer.Name, layer.State)
			if len(layer.ReasonCodes) > 0 {
				fmt.Fprintf(&output, " (%s)", strings.Join(layer.ReasonCodes, ", "))
			}
			if len(layer.Evidence) > 0 {
				fmt.Fprintf(&output, "\n    Evidence: %s", compactPublicJSON(layer.Evidence))
			}
			if len(layer.Limitations) > 0 {
				fmt.Fprintf(&output, "\n    Limits: %s", strings.Join(layer.Limitations, " "))
			}
		}
	}
	return output.String()
}

func formatTimestampVerification(mode presentation.Mode, result service.TimestampVerifyResult) string {
	if mode == presentation.ModePlain {
		return fmt.Sprintf("state\t%s\nverification\t%s\t%s\ntimestamps\t%s",
			result.State, result.Verification.State, strings.Join(result.Verification.ReasonCodes, ","), compactPublicJSON(result.Entries))
	}
	var output strings.Builder
	fmt.Fprintf(&output, "State: %s\nVerification: %s", result.State, result.Verification.State)
	if len(result.Verification.ReasonCodes) > 0 {
		fmt.Fprintf(&output, " (%s)", strings.Join(result.Verification.ReasonCodes, ", "))
	}
	fmt.Fprintf(&output, "\nTimestamp entries: %s", compactPublicJSON(result.Entries))
	return output.String()
}

func formatTimestampArtifact(mode presentation.Mode, result service.TimestampArtifactResult) string {
	if mode == presentation.ModePlain {
		return fmt.Sprintf("state\t%s\nselection\t%s\t%s\nrequests\t%d\nattempts\t%s\ntimestamps\t%s",
			result.State, result.SelectionMode, result.SelectedProvider, result.RequestSummary.RequestCount, compactPublicJSON(result.Attempts), compactPublicJSON(result.Entries))
	}
	var output strings.Builder
	fmt.Fprintf(&output, "State: %s", result.State)
	if result.SelectionMode != "" {
		fmt.Fprintf(&output, "\nSelection: %s", result.SelectionMode)
	}
	if result.SelectedProvider != "" {
		fmt.Fprintf(&output, "\nSelected provider: %s", result.SelectedProvider)
	}
	fmt.Fprintf(&output, "\nRequests: %d", result.RequestSummary.RequestCount)
	if len(result.Attempts) > 0 {
		fmt.Fprintf(&output, "\nAttempts: %s", compactPublicJSON(result.Attempts))
	}
	fmt.Fprintf(&output, "\nTimestamp entries: %s", compactPublicJSON(result.Entries))
	return output.String()
}

func formatPublicationVerification(mode presentation.Mode, result service.PublicationVerifyResult) string {
	var output strings.Builder
	if mode == presentation.ModePlain {
		fmt.Fprintf(&output, "overall\t%s\nmanifest\t%s\nfiles\t%d\nbytes\t%d", result.Overall, result.ManifestSHA256, result.FileCount, result.TotalBytes)
		for _, file := range result.Files {
			fmt.Fprintf(&output, "\nfile\t%s\t%s\t%d\t%s", file.Role, file.Path, file.Size, file.SHA256)
		}
		for _, forecast := range result.Evidence {
			for _, layer := range forecast.Layers {
				fmt.Fprintf(&output, "\n%s\t%s\t%s\t%s", forecast.QuestionID, forecast.ForecastID, layer.Name, layer.State)
			}
		}
		return output.String()
	}
	fmt.Fprintf(&output, "Overall: %s\nManifest SHA-256: %s\nFiles: %d\nBytes: %d", result.Overall, result.ManifestSHA256, result.FileCount, result.TotalBytes)
	for _, file := range result.Files {
		fmt.Fprintf(&output, "\nFile: %s (%s, %d bytes, %s)", file.Path, file.Role, file.Size, file.SHA256)
	}
	for _, forecast := range result.Evidence {
		fmt.Fprintf(&output, "\nForecast: %s / %s", forecast.QuestionID, forecast.ForecastID)
		for _, layer := range forecast.Layers {
			fmt.Fprintf(&output, "\n  %s: %s", layer.Name, layer.State)
			if len(layer.ReasonCodes) > 0 {
				fmt.Fprintf(&output, " (%s)", strings.Join(layer.ReasonCodes, ", "))
			}
		}
	}
	return output.String()
}

func writeDisplayFields(output *strings.Builder, mode presentation.Mode, fields [][2]string) {
	for index, field := range fields {
		if index > 0 {
			output.WriteByte('\n')
		}
		if mode == presentation.ModePlain {
			fmt.Fprintf(output, "%s\t%s", field[0], field[1])
		} else {
			label := strings.ReplaceAll(field[0], "_", " ")
			fmt.Fprintf(output, "%s: %s", strings.ToUpper(label[:1])+label[1:], field[1])
		}
	}
}

func compactPublicJSON(value any) string {
	redacted, err := presentation.Redact(value)
	if err != nil {
		return ""
	}
	encoded, err := json.Marshal(redacted)
	if err != nil {
		return ""
	}
	return string(encoded)
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
	return &urfavecli.Command{Name: name, Usage: usage, Description: "Example:\n  " + example, Flags: flags, Before: admitSupportedLedger, DisableSliceFlagSeparator: true}
}

func admitSupportedLedger(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
	path := command.String("file")
	if command.Name == "init" || path == "" || path == "-" || strings.HasSuffix(command.FullName(), " publish verify") {
		return ctx, nil
	}
	operationContext, cancel := RuntimeFromCommand(command).Context(ctx)
	defer cancel()
	_, err := service.LoadAndValidateLedger(operationContext, path, nil)
	return ctx, err
}

func fileFlag(allowStdin bool) *urfavecli.StringFlag {
	usage := "Ledger file path"
	if allowStdin {
		usage += "; use - for ledger bytes on stdin (sibling artifacts are unavailable)"
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
