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
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
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

func ledgerCommand() *urfavecli.Command {
	command := leaf("update", "Update ledger and current forecaster metadata", "forecast-ledger ledger update --file ledger.yaml --input metadata-patch.yaml", false, []urfavecli.Flag{fileFlag(false), inputFlag()})
	command.Action = ledgerUpdateAction
	return group("ledger", "Manage ledger metadata", command)
}

func ledgerUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var input service.RootMetadataPatchInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaRootMetadata, &input); err != nil {
		return err
	}
	var result service.RootMetadataFileResult
	var err error
	code, message := "ledger.updated", "Ledger metadata was updated"
	if runtime.DryRun {
		result, err = service.PlanRootMetadataFileUpdate(operationContext, command.String("file"), input)
		code, message = "ledger.update.planned", "Ledger metadata update is valid; no file was changed"
	} else {
		result, err = service.CommitRootMetadataFileUpdate(operationContext, command.String("file"), input)
		if err == nil && !result.Changed {
			code, message = "ledger.unchanged", "Ledger metadata is already up to date"
		}
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
}

func initCommand() *urfavecli.Command {
	command := leaf("init", "Create a new ledger", "forecast-ledger init --file ledger.yaml --ledger-id my-forecasts --timezone Europe/London --forecaster-id me --forecaster-name 'My Name' --input initial-question.yaml", false,
		[]urfavecli.Flag{fileFlag(false),
			&urfavecli.StringFlag{Name: "ledger-id", Required: true, Usage: "Stable ledger ID"},
			&urfavecli.StringFlag{Name: "timezone", Required: true, Usage: "IANA timezone name"},
			&urfavecli.StringFlag{Name: "forecaster-id", Required: true, Usage: "Stable forecaster ID"},
			&urfavecli.StringFlag{Name: "forecaster-name", Required: true, Usage: "Forecaster display name"},
			&urfavecli.StringFlag{Name: "forecaster-kind", Value: "individual", Usage: "Forecaster kind: individual or team"},
			inputFlag(),
			&urfavecli.StringFlag{Name: "key-file", OnlyOnce: true, TakesFile: true, Usage: "New protected key file; required only for a sealed first forecast"}})
	command.Action = initAction
	command.Description += "\n\nOmitted initial times use one operation-clock observation; an explicit ledger created_at is not copied into recorded_at."
	return command
}

func initAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var input service.InitInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaInit, &input); err != nil {
		return err
	}
	effects := service.ProductionEffects()
	operationAt, err := service.CaptureOperationTime(effects.Clock)
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
	keyPath := command.String("key-file")
	visibility := input.Question.InitialForecast.Visibility
	var model *ledger.Ledger
	var sealed service.SealedInitialBuild
	switch visibility {
	case ledger.VisibilityPublic:
		if keyPath != "" {
			return app.NewError(app.CodeUsage, "--key-file is only valid for a sealed initial forecast", nil)
		}
		model, err = service.BuildInitialPublicLedgerAt(root, input.Question, operationAt)
	case ledger.VisibilitySealed:
		if command.String("input") != "-" {
			if err := storage.CheckProtectedFile(command.String("input")); err != nil {
				return protectedArgumentError(err, "--input")
			}
		}
		if strings.TrimSpace(keyPath) == "" {
			return app.NewError(app.CodeUsage, "--key-file is required for a sealed initial forecast", nil)
		}
		if runtime.DryRun {
			model, err = service.PlanInitialSealedLedgerAt(root, input.Question, operationAt)
		} else {
			sealed, err = service.BuildInitialSealedLedgerAt(operationContext, root, input.Question, operationAt, effects)
			model = sealed.Ledger
		}
	default:
		err = app.NewError(app.CodeInvalidData, "initial forecast visibility must be public or sealed", nil)
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
		if visibility == ledger.VisibilitySealed {
			resolvedKey, keyErr := storage.ResolveNewFilePath(keyPath, "key file")
			if keyErr != nil {
				return keyErr
			}
			if resolvedKey == resolvedLedger {
				return app.NewError(app.CodeConflict, "ledger and key destinations must be different", nil)
			}
			effects = append([]service.SideEffect{{Kind: service.EffectKey, Action: service.EffectCreate, Status: service.EffectDeferred, Path: filepath.Base(resolvedKey), Owned: true, Rollback: service.RollbackRetainSecret}}, effects...)
		}
		return presenterFor(command).Success("ledger.init.planned", "Ledger initialization is valid; no files were written", initOutput(model, visibility, effects, service.Recovery{State: service.RecoveryNone}))
	}
	recovery := service.Recovery{State: service.RecoveryNone}
	if visibility == ledger.VisibilitySealed {
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
	if visibility == ledger.VisibilitySealed {
		completedEffects = append([]service.SideEffect{{Kind: service.EffectKey, Action: service.EffectCreate, Status: service.EffectCompleted, Path: filepath.Base(keyPath), Owned: true, Rollback: service.RollbackRetainSecret}}, completedEffects...)
	}
	return presenterFor(command).Success("ledger.initialized", "Ledger was created", initOutput(model, visibility, completedEffects, recovery))
}

func initOutput(model *ledger.Ledger, visibility ledger.ForecastVisibility, effects []service.SideEffect, recovery service.Recovery) map[string]any {
	return map[string]any{
		"ledger_id": model.LedgerID, "schema_version": model.SchemaVersion,
		"question_id": model.Questions[0].ID, "forecast_id": model.Questions[0].Forecasts[0].ID,
		"visibility": visibility, "effects": effects, "recovery": recovery,
	}
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
	add := leaf("add", "Add a platform", "forecast-ledger platform add --file ledger.yaml --platform metaculus --input platform.yaml", false, []urfavecli.Flag{fileFlag(false), platformFlag(), inputFlag()})
	add.Action = platformAddAction
	update := leaf("update", "Update a platform", "forecast-ledger platform update --file ledger.yaml --platform metaculus --input platform-patch.yaml", false, []urfavecli.Flag{fileFlag(false), platformFlag(), inputFlag()})
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
	var input service.PlatformCreateInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaPlatformCreate, &input); err != nil {
		return err
	}
	id := ledger.Slug(command.String("platform"))
	var result service.PlatformFileResult
	var err error
	code, message := "platform.added", "Platform was added"
	if runtime.DryRun {
		result, err = service.PlanPlatformAddFile(operationContext, command.String("file"), id, input)
		code, message = "platform.add.planned", "Platform addition is valid; no file was changed"
	} else {
		result, err = service.CommitPlatformAddFile(operationContext, command.String("file"), id, input)
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
}

func platformUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var input service.PlatformPatchInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaPlatformPatch, &input); err != nil {
		return err
	}
	id := ledger.Slug(command.String("platform"))
	var result service.PlatformFileResult
	var err error
	code, message := "platform.updated", "Platform was updated"
	if runtime.DryRun {
		result, err = service.PlanPlatformUpdateFile(operationContext, command.String("file"), id, input)
		code, message = "platform.update.planned", "Platform update is valid; no file was changed"
	} else {
		result, err = service.CommitPlatformUpdateFile(operationContext, command.String("file"), id, input)
		if err == nil && !result.Changed {
			code, message = "platform.unchanged", "Platform is already up to date"
		}
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
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
	return presenterFor(command).Success("platform.list", message, map[string]any{"ledger_id": ledgerID, "platforms": items})
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
	return presenterFor(command).Success("platform.show", message, map[string]any{"ledger_id": ledgerID, "platform": result})
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
		return presenterFor(command).Success("platform.remove.planned", "Platform removal is valid; no file was changed", result)
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
	return presenterFor(command).Success("platform.removed", "Platform was removed", result)
}

func questionCommand() *urfavecli.Command {
	add := leaf("add", "Add a question with its first forecast", "forecast-ledger question add --file ledger.yaml --question q-launch --type binary --input question.yaml", false, []urfavecli.Flag{fileFlag(false), questionFlag(), &urfavecli.StringFlag{Name: "type", Required: true, Usage: "Question type: binary, multiple_choice, numeric, or date"}, inputFlag(), &urfavecli.StringFlag{Name: "key-file", OnlyOnce: true, TakesFile: true, Usage: "New protected key file; required only for a sealed first forecast"}})
	add.Action = questionAddAction
	update := leaf("update", "Update allowed question fields", "forecast-ledger question update --file ledger.yaml --question q-launch --input question-patch.yaml", false, []urfavecli.Flag{fileFlag(false), questionFlag(), inputFlag()})
	update.Action = questionUpdateAction
	list := leaf("list", "List questions", "forecast-ledger question list --file ledger.yaml", true, []urfavecli.Flag{fileFlag(true)})
	list.Action = questionListAction
	show := leaf("show", "Show a question", "forecast-ledger question show --file ledger.yaml --question q-launch", true, []urfavecli.Flag{fileFlag(true), questionFlag()})
	show.Description += "\n\nNormal human and plain output includes public business fields and redacted forecast summaries."
	show.Action = questionShowAction
	resolve := leaf("resolve", "Resolve a question", "forecast-ledger question resolve --file ledger.yaml --question q-launch --input resolution.yaml --yes", false, []urfavecli.Flag{fileFlag(false), questionFlag(), inputFlag()})
	resolve.Action = questionResolveAction
	annul := leaf("annul", "Annul a question", "forecast-ledger question annul --file ledger.yaml --question q-launch --input annulment.yaml --yes", false, []urfavecli.Flag{fileFlag(false), questionFlag(), inputFlag()})
	annul.Action = questionAnnulAction
	dispute := leaf("dispute", "Dispute a resolution", "forecast-ledger question dispute --file ledger.yaml --question q-launch --input dispute.yaml --yes", false, []urfavecli.Flag{fileFlag(false), questionFlag(), inputFlag()})
	dispute.Action = questionDisputeAction
	return group("question", "Manage forecast questions", add, update, list, show, resolve, annul, dispute)
}

func questionAddAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var input service.QuestionAddInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaQuestionAdd, &input); err != nil {
		return err
	}
	normalized := service.NormalizedQuestionCreate{ID: ledger.Slug(command.String("question")), Type: ledger.QuestionType(command.String("type")), Input: input}
	observedAt := ledger.Timestamp(service.ProductionEffects().Clock.Now().Format(time.RFC3339))
	keyPath := command.String("key-file")
	visibility := input.InitialForecast.Visibility
	var result service.QuestionFileResult
	var err error
	code, message := "question.added", "Question and initial forecast were added"
	switch visibility {
	case ledger.VisibilityPublic:
		if keyPath != "" {
			return app.NewError(app.CodeUsage, "--key-file is only valid for a sealed initial forecast", nil)
		}
		if runtime.DryRun {
			result, err = service.PlanQuestionAddPublicFile(operationContext, command.String("file"), normalized, observedAt)
		} else {
			result, err = service.CommitQuestionAddPublicFile(operationContext, command.String("file"), normalized, observedAt)
		}
	case ledger.VisibilitySealed:
		if command.String("input") != "-" {
			if err := storage.CheckProtectedFile(command.String("input")); err != nil {
				return protectedArgumentError(err, "--input")
			}
		}
		if strings.TrimSpace(keyPath) == "" {
			return app.NewError(app.CodeUsage, "--key-file is required for a sealed initial forecast", nil)
		}
		if runtime.DryRun {
			result, err = service.PlanQuestionAddSealedFile(operationContext, command.String("file"), keyPath, normalized, observedAt)
		} else {
			result, err = service.CommitQuestionAddSealedFile(operationContext, command.String("file"), keyPath, normalized, observedAt, service.ProductionEffects())
		}
	default:
		err = app.NewError(app.CodeInvalidData, "initial forecast visibility must be public or sealed", nil)
	}
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	if runtime.DryRun {
		code, message = "question.add.planned", "Question addition is valid; no file was changed"
	}
	return presenterFor(command).Success(code, message, result)
}

func questionUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var input service.QuestionPatchInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaQuestionPatch, &input); err != nil {
		return err
	}
	id := ledger.Slug(command.String("question"))
	var result service.QuestionFileResult
	var err error
	code, message := "question.updated", "Question was updated"
	if runtime.DryRun {
		result, err = service.PlanQuestionUpdateFile(operationContext, command.String("file"), id, input)
		code, message = "question.update.planned", "Question update is valid; no file was changed"
	} else {
		result, err = service.CommitQuestionUpdateFile(operationContext, command.String("file"), id, input)
		if err == nil && !result.Changed {
			code, message = "question.unchanged", "Question is already up to date"
		}
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
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
	return presenterFor(command).Success("question.list", message, map[string]any{"ledger_id": ledgerID, "questions": items})
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
	return presenter.Success("question.show", message, map[string]any{"ledger_id": ledgerID, "question": result})
}

func questionResolveAction(ctx context.Context, command *urfavecli.Command) error {
	var input service.ResolutionInput
	return questionTerminalAction(ctx, command, service.InputSchemaResolution, &input, "resolve", func(operationContext context.Context, id ledger.Slug, observedAt ledger.Timestamp, dryRun bool) (service.QuestionFileResult, error) {
		if dryRun {
			return service.PlanQuestionResolveFile(operationContext, command.String("file"), id, input, observedAt)
		}
		return service.CommitQuestionResolveFile(operationContext, command.String("file"), id, input, observedAt)
	})
}

func questionAnnulAction(ctx context.Context, command *urfavecli.Command) error {
	var input service.AnnulInput
	return questionTerminalAction(ctx, command, service.InputSchemaAnnul, &input, "annul", func(operationContext context.Context, id ledger.Slug, observedAt ledger.Timestamp, dryRun bool) (service.QuestionFileResult, error) {
		if dryRun {
			return service.PlanQuestionAnnulFile(operationContext, command.String("file"), id, input, observedAt)
		}
		return service.CommitQuestionAnnulFile(operationContext, command.String("file"), id, input, observedAt)
	})
}

func questionDisputeAction(ctx context.Context, command *urfavecli.Command) error {
	var input service.DisputeInput
	return questionTerminalAction(ctx, command, service.InputSchemaDispute, &input, "dispute", func(operationContext context.Context, id ledger.Slug, observedAt ledger.Timestamp, dryRun bool) (service.QuestionFileResult, error) {
		if dryRun {
			return service.PlanQuestionDisputeFile(operationContext, command.String("file"), id, input, observedAt)
		}
		return service.CommitQuestionDisputeFile(operationContext, command.String("file"), id, input, observedAt)
	})
}

func questionTerminalAction(ctx context.Context, command *urfavecli.Command, schema service.InputSchemaName, input any, verb string, execute func(context.Context, ledger.Slug, ledger.Timestamp, bool) (service.QuestionFileResult, error)) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, schema, input); err != nil {
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
	observedAt := ledger.Timestamp(service.ProductionEffects().Clock.Now().Format(time.RFC3339))
	result, err := execute(operationContext, id, observedAt, runtime.DryRun)
	if err != nil {
		return err
	}
	code, message := "question."+verb+"d", "Question was "+verb+"d"
	if verb == "annul" {
		code, message = "question.annulled", "Question was annulled; the question and forecasts were retained"
	}
	if runtime.DryRun {
		code, message = "question."+verb+".planned", "Question lifecycle change is valid; no file was changed"
	}
	return presenterFor(command).Success(code, message, result)
}

func forecastCommand() *urfavecli.Command {
	add := leaf("add", "Add a public forecast", "forecast-ledger forecast add --file ledger.yaml --question q-launch --forecast f-002 --input forecast.yaml", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), inputFlag()})
	add.Action = forecastAddAction
	list := leaf("list", "List forecasts", "forecast-ledger forecast list --file ledger.yaml --question q-launch", true, []urfavecli.Flag{fileFlag(true), questionFlag()})
	list.Action = forecastListAction
	show := leaf("show", "Show a forecast", "forecast-ledger forecast show --file ledger.yaml --question q-launch --forecast f-001", true, []urfavecli.Flag{fileFlag(true), questionFlag(), forecastFlag()})
	show.Description += "\n\nNormal human and plain output includes type-aware public values and safe stored integrity evidence; sealed private fields stay redacted. No network check is performed."
	show.Action = forecastShowAction
	seal := leaf("seal", "Create and append a sealed forecast", "forecast-ledger forecast seal --file ledger.yaml --question q-launch --forecast f-002 --input private.yaml --key-file secret.key", false, []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag(), inputFlag(), secretOutputFlag()})
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
	var input service.ForecastCreateInput
	if err := service.DecodeOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaForecastCreate, &input); err != nil {
		return err
	}
	observedAt := ledger.Timestamp(service.ProductionEffects().Clock.Now().Format(time.RFC3339))
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	var result service.ForecastFileResult
	var err error
	code, message := "forecast.added", "Forecast was added"
	if runtime.DryRun {
		result, err = service.PlanPublicForecastAddFile(operationContext, command.String("file"), questionID, forecastID, input, observedAt)
		code, message = "forecast.add.planned", "Forecast addition is valid; no file was changed"
	} else {
		result, err = service.CommitPublicForecastAddFile(operationContext, command.String("file"), questionID, forecastID, input, observedAt)
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
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
	return presenterFor(command).Success("forecast.list", message, map[string]any{"ledger_id": ledgerID, "question_id": questionID, "forecasts": items})
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
	return presenter.Success("forecast.show", message, map[string]any{"ledger_id": ledgerID, "question_id": questionID, "forecast": result})
}

func forecastSealAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var input service.SealedForecastInput
	if err := decodePrivateOperationInput(operationContext, command.String("input"), command.Root().Reader, service.InputSchemaForecastSeal, &input); err != nil {
		return err
	}
	observedAt := ledger.Timestamp(service.ProductionEffects().Clock.Now().Format(time.RFC3339))
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	var result service.ForecastFileResult
	var err error
	code, message := "forecast.sealed", "Sealed forecast and protected key were created"
	if runtime.DryRun {
		result, err = service.PlanForecastSealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, input, observedAt)
		code, message = "forecast.seal.planned", "Sealed forecast creation is valid; no key or ledger file was changed"
	} else {
		result, err = service.CommitForecastSealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, input, observedAt, service.ProductionEffects())
	}
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	return presenterFor(command).Success(code, message, result)
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
		revealedAt = ledger.Timestamp(service.ProductionEffects().Clock.Now().Format(time.RFC3339))
	}
	var result service.ForecastFileResult
	code, message := "forecast.revealed", "Forecast was authenticated and revealed"
	if runtime.DryRun {
		result, err = service.PlanForecastRevealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, revealedAt)
		code, message = "forecast.reveal.planned", "Forecast reveal is valid; no file was changed"
	} else {
		result, err = service.CommitForecastRevealFile(operationContext, command.String("file"), command.String("key-file"), questionID, forecastID, revealedAt)
		if err == nil && !result.Changed {
			code, message = "forecast.reveal.unchanged", "Forecast was already revealed with this authenticated key"
		}
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
}

func forecastKeyHintUpdateAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	questionID, forecastID := ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast"))
	keyHint := command.String("key-hint")
	var result service.ForecastFileResult
	var err error
	code, message := "forecast.key_hint.updated", "Forecast key hint was updated"
	if runtime.DryRun {
		result, err = service.PlanForecastKeyHintUpdateFile(operationContext, command.String("file"), questionID, forecastID, keyHint)
		code, message = "forecast.key_hint.update.planned", "Key hint update is valid; no file was changed"
	} else {
		result, err = service.CommitForecastKeyHintUpdateFile(operationContext, command.String("file"), questionID, forecastID, keyHint)
		if err == nil && !result.Changed {
			code, message = "forecast.key_hint.unchanged", "Forecast key hint is already up to date"
		}
	}
	if err != nil {
		return err
	}
	return presenterFor(command).Success(code, message, result)
}

func decodePrivateOperationInput(ctx context.Context, path string, stdin io.Reader, schema service.InputSchemaName, destination any) error {
	if path == "-" {
		return service.DecodeOperationInput(ctx, path, stdin, schema, destination)
	}
	data, err := storage.ReadProtectedFile(path, 8<<20)
	if err != nil {
		return protectedArgumentError(err, "--input")
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
	code, message := "target.built", "Forecast target artifacts were built"
	if runtime.DryRun {
		result, err = service.PlanTargetBuild(operationContext, command.String("file"), all, questionID, forecastID)
		code, message = "target.build.planned", "Target build is valid; no files were written"
	} else {
		result, err = service.CommitTargetBuild(operationContext, command.String("file"), all, questionID, forecastID)
	}
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	return presenterFor(command).Success(code, message, result)
}

func targetCheckAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.InspectTargets(operationContext, command.String("file"), command.Bool("all"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")))
	if err != nil {
		return err
	}
	code, message := "target.valid", "Forecast target artifacts match the ledger"
	if result.FailureCode != "" {
		code, message = "target.failed", "Target inspection completed with failures"
	} else {
		for _, target := range result.Targets {
			if string(target.State) == string(service.LayerNotApplicable) {
				code, message = "target.checked", "Target inspection completed; some forecasts have no retained target"
				break
			}
		}
	}
	presenter := presenterFor(command)
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = formatTargetInspection(presenter.Mode(), result)
	}
	if err := presenter.Success(code, message, result); err != nil {
		return err
	}
	if result.FailureCode != "" {
		return presentedApplicationError{app.NewError(result.FailureCode, "target inspection found failures", nil)}
	}
	return nil
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
	stamp := timestampLeaf("stamp", "Submit a blinded target commitment to OpenTimestamps calendars", false, true)
	stamp.Action = timestampStampAction
	upgrade := timestampLeaf("upgrade", "Upgrade a pending OpenTimestamps receipt", false, true)
	upgrade.Action = timestampUpgradeAction
	status := timestampLeaf("status", "Show local OpenTimestamps receipt status", true, false)
	status.Action = timestampStatusAction
	verify := timestampLeaf("verify", "Verify Bitcoin evidence and record the result", false, false)
	verify.Flags = append(verify.Flags,
		&urfavecli.StringFlag{Name: "bitcoin-core", Usage: "Explicit Bitcoin Core RPC URL (advanced)"},
		&urfavecli.StringFlag{Name: "bitcoin-auth-file", TakesFile: true, Usage: "Protected JSON file with Bitcoin Core username and password"},
		&urfavecli.StringFlag{Name: "verified-at", Usage: "Exact RFC 3339 verification time; defaults to now"},
	)
	verify.Action = timestampVerifyAction
	command := group("timestamp", "Manage experimental OpenTimestamps receipts", stamp, upgrade, status, verify)
	profile := ots.Profile()
	command.Description = fmt.Sprintf("Experimental profile %s submits to four fixed calendars and needs %d valid responses. Public Bitcoin verification requires both fixed sources to agree; limits are %d heights, %d requests, and %d concurrent requests.", profile.ID, profile.CalendarMinimum, profile.MaximumUniqueHeights, profile.MaximumHTTPRequests, profile.MaximumConcurrentHTTP)
	return command
}

func timestampLeaf(name, usage string, readOnly, calendarOptions bool) *urfavecli.Command {
	flags := []urfavecli.Flag{fileFlag(false), questionFlag(), forecastFlag()}
	if name != "status" {
		flags = append(flags, &urfavecli.BoolFlag{Name: "offline", Usage: "Open no network connection"})
	}
	if calendarOptions {
		flags = append(flags,
			&urfavecli.StringSliceFlag{Name: "calendar", Usage: "Custom public HTTPS calendar origin; repeat to replace built-in calendars"},
			&urfavecli.IntFlag{Name: "calendar-min-success", Usage: "Required valid custom calendar responses"},
		)
	}
	command := leaf(name, usage, fmt.Sprintf("forecast-ledger timestamp %s --file ledger.yaml --question q-launch --forecast f-001", name), readOnly, flags)
	command.Before = func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
		calendars := command.StringSlice("calendar")
		minimum := command.Int("calendar-min-success")
		if (len(calendars) == 0) != (minimum == 0) {
			return ctx, app.NewError(app.CodeUsage, "--calendar and --calendar-min-success must be supplied together", nil)
		}
		return ctx, nil
	}
	return command
}

func timestampStampAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.CommitTimestampStamp(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")), service.TimestampStampOptions{
		DryRun: runtime.DryRun, Offline: command.Bool("offline"), CustomCalendars: command.StringSlice("calendar"), CalendarMinimum: command.Int("calendar-min-success"), Effects: service.ProductionEffects(),
	})
	if err != nil {
		return withRecovery(err, result.Recovery)
	}
	code, message := "timestamp.pending", "OpenTimestamps receipt was stored as pending"
	if runtime.DryRun {
		code, message = "timestamp.stamp.planned", "Timestamp stamp is valid; no entropy, network request, or file write occurred"
	}
	return presenterFor(command).Success(code, message, result)
}

func timestampUpgradeAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.CommitTimestampUpgrade(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")), service.TimestampUpgradeOptions{
		DryRun: runtime.DryRun, Offline: command.Bool("offline"), CustomCalendars: command.StringSlice("calendar"), CalendarMinimum: command.Int("calendar-min-success"),
	})
	if err != nil {
		return err
	}
	code, message := "timestamp.upgraded", "OpenTimestamps receipt was upgraded"
	if runtime.DryRun {
		code, message = "timestamp.upgrade.planned", "Timestamp upgrade is valid; no network request or file write occurred"
	}
	return presenterFor(command).Success(code, message, result)
}

func timestampStatusAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	result, err := service.TimestampStatusFor(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")))
	if err != nil {
		return err
	}
	return presenterFor(command).Success("timestamp.status", "OpenTimestamps local status: "+string(result.State), result)
}

func timestampVerifyAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	var verifiedAt ledger.Timestamp
	if command.String("verified-at") != "" {
		verifiedAt = ledger.Timestamp(command.String("verified-at"))
		if _, err := service.ParseTimestamp(verifiedAt, "verified_at"); err != nil {
			return err
		}
	}
	var observer ots.BitcoinObserver
	var err error
	if !runtime.DryRun {
		observer, err = service.ProtectedCoreObserver(command.String("bitcoin-core"), command.String("bitcoin-auth-file"))
		if err != nil {
			return err
		}
	}
	result, err := service.CommitTimestampVerify(operationContext, command.String("file"), ledger.Slug(command.String("question")), ledger.Slug(command.String("forecast")), service.TimestampVerifyOptions{DryRun: runtime.DryRun, Offline: command.Bool("offline"), VerifiedAt: verifiedAt, Observer: observer})
	if err != nil {
		return err
	}
	code, message := "timestamp.verified", "OpenTimestamps Bitcoin evidence was verified"
	if runtime.DryRun {
		code, message = "timestamp.verify.planned", "Timestamp verification is valid; network observation and ledger update were deferred"
	}
	return presenterFor(command).Success(code, message, result)
}

func verifyCommand() *urfavecli.Command {
	command := leaf("verify", "Run layered verification", "forecast-ledger verify --file ledger.yaml --offline", true, []urfavecli.Flag{
		fileFlag(false),
		&urfavecli.StringFlag{Name: "question", Usage: "Optional question ID"},
		&urfavecli.StringFlag{Name: "forecast", Usage: "Optional forecast ID; requires --question"},
		&urfavecli.BoolFlag{Name: "offline", Usage: "Run local checks without opening a network connection"},
		&urfavecli.BoolFlag{Name: "check-sources", Usage: "Check outcome source reachability and stored digests"},
		&urfavecli.StringFlag{Name: "bitcoin-core", Usage: "Explicit Bitcoin Core RPC URL (advanced)"},
		&urfavecli.StringFlag{Name: "bitcoin-auth-file", TakesFile: true, Usage: "Protected JSON file with Bitcoin Core username and password"},
	})
	command.Before = func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
		if command.String("forecast") != "" && command.String("question") == "" {
			return ctx, app.NewError(app.CodeUsage, "--forecast requires --question", nil)
		}
		if command.Bool("offline") && (command.String("bitcoin-core") != "" || command.String("bitcoin-auth-file") != "") {
			return ctx, app.NewError(app.CodeUsage, "--offline cannot be combined with Bitcoin Core options", nil)
		}
		return ctx, nil
	}
	command.Action = verificationAction
	command.Description += "\n\nNormal human and plain output includes the complete ordered evidence matrix and safe retained timing values. Offline stored values are not freshly rechecked."
	return command
}

func verificationAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	observer, err := service.ProtectedCoreObserver(command.String("bitcoin-core"), command.String("bitcoin-auth-file"))
	if err != nil {
		return err
	}
	report, err := service.VerifyLedgerEvidence(operationContext, command.String("file"), service.VerificationOptions{
		Offline: command.Bool("offline"), CheckSources: command.Bool("check-sources"), Observer: observer,
		QuestionID: ledger.Slug(command.String("question")), ForecastID: ledger.Slug(command.String("forecast")),
	})
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	message := "Verification completed with status " + string(report.Overall)
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = formatVerificationReport(presenter.Mode(), report)
	}
	if err := presenter.Success("verification."+string(report.Overall), message, report); err != nil {
		return err
	}
	if report.FailureCode != "" {
		return presentedApplicationError{app.NewError(report.FailureCode, "verification completed with status "+string(report.Overall), nil)}
	}
	return nil
}

func publishCommand() *urfavecli.Command {
	build := leaf("build", "Build an evidence package", "forecast-ledger publish build --file ledger.yaml --output evidence", false, []urfavecli.Flag{fileFlag(false), &urfavecli.StringFlag{Name: "output", Required: true, TakesFile: true, Usage: "New package directory"}})
	build.Action = publicationBuildAction
	verify := leaf("verify", "Verify an evidence package", "forecast-ledger publish verify --file package/ledger/ledger.yaml --manifest package/manifest.json", true, []urfavecli.Flag{
		fileFlag(false),
		&urfavecli.StringFlag{Name: "manifest", Required: true, TakesFile: true, Usage: "Package manifest file"},
		&urfavecli.BoolFlag{Name: "online", Usage: "Recheck Bitcoin timing with network sources"},
		&urfavecli.BoolFlag{Name: "offline", Usage: "Verify package bytes without opening a network connection (default)"},
		&urfavecli.StringFlag{Name: "bitcoin-core", Usage: "Explicit Bitcoin Core RPC URL (advanced; requires --online)"},
		&urfavecli.StringFlag{Name: "bitcoin-auth-file", TakesFile: true, Usage: "Protected JSON file with Bitcoin Core username and password"},
	})
	verify.Before = func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
		if command.Bool("online") && command.Bool("offline") {
			return ctx, app.NewError(app.CodeUsage, "--online and --offline cannot be combined", nil)
		}
		if !command.Bool("online") && (command.String("bitcoin-core") != "" || command.String("bitcoin-auth-file") != "") {
			return ctx, app.NewError(app.CodeUsage, "Bitcoin Core options require --online", nil)
		}
		return ctx, nil
	}
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
	code, message := "publication.built", "Evidence package was built"
	if runtime.DryRun {
		code, message = "publication.build.planned", "Evidence package build is valid; no files were written"
	}
	return presenterFor(command).Success(code, message, result)
}

func publicationVerifyAction(ctx context.Context, command *urfavecli.Command) error {
	runtime := RuntimeFromCommand(command)
	operationContext, cancel := runtime.Context(ctx)
	defer cancel()
	observer, err := service.ProtectedCoreObserver(command.String("bitcoin-core"), command.String("bitcoin-auth-file"))
	if err != nil {
		return err
	}
	result, err := service.VerifyPublicationPackage(operationContext, command.String("file"), command.String("manifest"), service.PublicationVerifyOptions{
		Online: command.Bool("online"), Offline: !command.Bool("online"), Observer: observer,
	})
	if err != nil {
		return err
	}
	presenter := presenterFor(command)
	message := "Package verification completed with status " + string(result.Overall)
	if presenter.Mode() != presentation.ModeJSON && presenter.Mode() != presentation.ModeQuiet {
		message = formatPublicationVerification(presenter.Mode(), result)
	}
	if err := presenter.Success("publication.verification."+string(result.Overall), message, result); err != nil {
		return err
	}
	if result.FailureCode != "" {
		return presentedApplicationError{app.NewError(result.FailureCode, "package verification completed with status "+string(result.Overall), nil)}
	}
	return nil
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
	return &urfavecli.Command{Name: "version", Usage: "Show build and contract versions", Description: "Example:\n  forecast-ledger version --json",
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
			_, err := fmt.Fprintf(command.Root().Writer, "%s %s\nsource revision: %s\ngo: %s\nforecast ledger schema: %s (%s, sha256:%s)\nmcp protocol: %s\ntimestamp profile: %s (experimental; %d-of-%d calendars; %d Bitcoin sources; %d heights/%d requests/%d concurrent)\n", info.Binary, info.Version, info.SourceRevision, info.GoVersion, info.Schema.Version, info.Schema.Commit, info.Schema.SHA256, info.MCPProtocol, info.TimestampProfile.ID, info.TimestampProfile.CalendarMinimum, len(info.TimestampProfile.Calendars), len(info.TimestampProfile.BitcoinSources), info.TimestampProfile.MaximumUniqueHeights, info.TimestampProfile.MaximumHTTPRequests, info.TimestampProfile.MaximumConcurrentHTTP)
			return err
		}}
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
	encoded, err := json.Marshal(presentation.Redact(value))
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
	return &urfavecli.Command{Name: name, Usage: usage, Description: "Example:\n  " + example, Flags: flags}
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
func inputFlag() *urfavecli.StringFlag {
	return &urfavecli.StringFlag{Name: "input", Aliases: []string{"i"}, Required: true, OnlyOnce: true, TakesFile: true, Usage: "Closed JSON or YAML input file; use - for stdin",
		Validator: func(value string) error {
			if strings.TrimSpace(value) == "" {
				return errors.New("input path is empty")
			}
			return nil
		}}
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
