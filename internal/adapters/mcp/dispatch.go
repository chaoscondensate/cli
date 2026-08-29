package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/service"
	"github.com/chaoscondensate/cli/internal/storage"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type toolInput struct {
	File           string          `json:"file,omitempty"`
	Platform       string          `json:"platform,omitempty"`
	Question       string          `json:"question,omitempty"`
	Forecast       string          `json:"forecast,omitempty"`
	Type           string          `json:"type,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
	InputFile      string          `json:"input_file,omitempty"`
	KeyFile        string          `json:"key_file,omitempty"`
	KeyHint        string          `json:"key_hint,omitempty"`
	LedgerID       string          `json:"ledger_id,omitempty"`
	Timezone       string          `json:"timezone,omitempty"`
	ForecasterID   string          `json:"forecaster_id,omitempty"`
	ForecasterName string          `json:"forecaster_name,omitempty"`
	ForecasterKind string          `json:"forecaster_kind,omitempty"`
	Output         string          `json:"output,omitempty"`
	Manifest       string          `json:"manifest,omitempty"`
	RevealedAt     string          `json:"revealed_at,omitempty"`
	VerifiedAt     string          `json:"verified_at,omitempty"`
	All            bool            `json:"all,omitempty"`
	DryRun         bool            `json:"dry_run,omitempty"`
	Confirm        bool            `json:"confirm,omitempty"`
	Online         bool            `json:"online,omitempty"`
	CheckSources   bool            `json:"check_sources,omitempty"`
}

type toolEnvelope struct {
	Operation service.OperationName `json:"operation"`
	Code      string                `json:"code"`
	Message   string                `json:"message"`
	Data      any                   `json:"data,omitempty"`
	Error     *app.Error            `json:"error,omitempty"`
}

func (s *Server) toolHandler(def service.OperationDefinition, allowed map[string]bool, required []string) sdk.ToolHandler {
	return func(ctx context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		select {
		case s.sem <- struct{}{}:
			defer func() { <-s.sem }()
		default:
			return errorToolResult(def.Name, app.NewError(app.CodeConflict, "MCP concurrent request limit is reached", nil)), nil
		}
		input, err := decodeToolArguments(request.Params.Arguments, s.maxToolBytes, allowed, required)
		if err != nil {
			return errorToolResult(def.Name, err), nil
		}
		data, code, message, err := s.dispatch(ctx, def, input)
		if err != nil {
			return errorToolResult(def.Name, err), nil
		}
		envelope := toolEnvelope{Operation: def.Name, Code: code, Message: message, Data: data}
		if failure := resultFailureCode(data); failure != "" {
			envelope.Code = string(failure)
			return marshalToolResult(envelope, true), nil
		}
		return marshalToolResult(envelope, false), nil
	}
}

func decodeToolArguments(arguments json.RawMessage, maxBytes int, allowed map[string]bool, required []string) (toolInput, error) {
	if len(arguments) > maxBytes {
		return toolInput{}, app.NewError(app.CodeUsage, "tool arguments exceed the size limit", nil)
	}
	var raw map[string]json.RawMessage
	if len(arguments) == 0 {
		raw = map[string]json.RawMessage{}
	} else {
		if _, err := document.ParseJSON(bytes.NewReader(arguments), document.DefaultLimits); err != nil {
			return toolInput{}, app.NewError(app.CodeUsage, "tool arguments are not valid bounded JSON", err)
		}
		if err := json.Unmarshal(arguments, &raw); err != nil {
			return toolInput{}, app.NewError(app.CodeUsage, "tool arguments are not valid JSON", err)
		}
	}
	for name := range raw {
		if !allowed[name] {
			return toolInput{}, app.WithDetails(app.NewError(app.CodeUsage, "tool arguments contain an unknown field", nil), map[string]any{"field": name})
		}
	}
	for _, name := range required {
		value, exists := raw[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return toolInput{}, app.WithDetails(app.NewError(app.CodeUsage, "tool argument is required", nil), map[string]any{"field": name})
		}
	}
	var input toolInput
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return toolInput{}, app.NewError(app.CodeUsage, "tool arguments have invalid types", err)
	}
	return input, nil
}

func (s *Server) dispatch(parent context.Context, def service.OperationDefinition, input toolInput) (any, string, string, error) {
	options := service.RequestOptions{DryRun: input.DryRun, Confirmed: input.Confirm || input.DryRun, Timeout: s.config.Timeout, Mode: s.config.Mode, Roots: s.roots.Public()}
	execution, err := service.PrepareExecution(parent, def.Name, options)
	if err != nil {
		return nil, "", "", err
	}
	defer execution.Close()
	ctx := execution.Context()
	fileMustExist := def.Name != service.OperationLedgerInit
	file := ""
	if input.File != "" {
		file, err = s.roots.Resolve(service.RootLedger, input.File, fileMustExist)
		if err != nil {
			return nil, "", "", err
		}
	}
	now := ledger.Timestamp(s.effects.Clock.Now().UTC().Format(time.RFC3339))

	switch def.Name {
	case service.OperationLedgerInit:
		return s.dispatchInit(ctx, file, input, now)
	case service.OperationLedgerUpdate:
		var value service.RootMetadataPatchInput
		if err := decodeInline(ctx, input.Input, service.InputSchemaRootMetadata, &value); err != nil {
			return nil, "", "", err
		}
		if input.DryRun {
			result, err := service.PlanRootMetadataFileUpdate(ctx, file, value)
			return result, "ledger.update.planned", "Ledger metadata update is valid; no file was changed", err
		}
		result, err := service.CommitRootMetadataFileUpdate(ctx, file, value)
		return result, "ledger.updated", "Ledger metadata was updated", err
	case service.OperationLedgerValidate:
		loaded, err := service.LoadAndValidateLedger(ctx, file, nil)
		if err != nil {
			return nil, "", "", err
		}
		return map[string]any{"ledger_id": loaded.Model.LedgerID, "schema_version": loaded.Model.SchemaVersion}, "ledger.valid", "Ledger is valid", nil
	case service.OperationLedgerStatus:
		loaded, err := service.LoadAndValidateLedger(ctx, file, nil)
		if err != nil {
			return nil, "", "", err
		}
		result, err := service.StatusForLedger(loaded)
		return result, "ledger.status", "Ledger status was read", err
	case service.OperationPlatformAdd, service.OperationPlatformUpdate:
		return dispatchPlatformMutation(ctx, def.Name, file, ledger.Slug(input.Platform), input.Input, input.DryRun)
	case service.OperationPlatformList:
		id, items, err := service.LoadPlatformList(ctx, file, nil)
		return map[string]any{"ledger_id": id, "platforms": items}, "platform.list", "Platforms were read", err
	case service.OperationPlatformShow:
		id, result, err := service.LoadPlatformShow(ctx, file, nil, ledger.Slug(input.Platform))
		return map[string]any{"ledger_id": id, "platform": result}, "platform.show", "Platform was read", err
	case service.OperationPlatformRemove:
		if input.DryRun {
			result, err := service.PlanPlatformRemoveFile(ctx, file, ledger.Slug(input.Platform))
			return result, "platform.remove.planned", "Platform removal is valid; no file was changed", err
		}
		result, err := service.CommitPlatformRemoveFile(ctx, file, ledger.Slug(input.Platform))
		return result, "platform.removed", "Platform was removed", err
	case service.OperationQuestionAdd:
		return s.dispatchQuestionAdd(ctx, file, input, now)
	case service.OperationQuestionUpdate:
		var value service.QuestionPatchInput
		if err := decodeInline(ctx, input.Input, service.InputSchemaQuestionPatch, &value); err != nil {
			return nil, "", "", err
		}
		if input.DryRun {
			result, err := service.PlanQuestionUpdateFile(ctx, file, ledger.Slug(input.Question), value)
			return result, "question.update.planned", "Question update is valid; no file was changed", err
		}
		result, err := service.CommitQuestionUpdateFile(ctx, file, ledger.Slug(input.Question), value)
		return result, "question.updated", "Question was updated", err
	case service.OperationQuestionList:
		id, items, err := service.LoadQuestionList(ctx, file, nil)
		return map[string]any{"ledger_id": id, "questions": items}, "question.list", "Questions were read", err
	case service.OperationQuestionShow:
		id, result, err := service.LoadQuestionShow(ctx, file, nil, ledger.Slug(input.Question))
		return map[string]any{"ledger_id": id, "question": result}, "question.show", "Question was read", err
	case service.OperationQuestionResolve, service.OperationQuestionAnnul, service.OperationQuestionDispute:
		return dispatchQuestionTerminal(ctx, def.Name, file, ledger.Slug(input.Question), input.Input, now, input.DryRun)
	case service.OperationForecastAdd:
		var value service.ForecastCreateInput
		if err := decodeInline(ctx, input.Input, service.InputSchemaForecastCreate, &value); err != nil {
			return nil, "", "", err
		}
		if input.DryRun {
			result, err := service.PlanPublicForecastAddFile(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), value, now)
			return result, "forecast.add.planned", "Forecast addition is valid; no file was changed", err
		}
		result, err := service.CommitPublicForecastAddFile(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), value, now)
		return result, "forecast.added", "Forecast was added", err
	case service.OperationForecastList:
		id, items, err := service.LoadForecastList(ctx, file, nil, ledger.Slug(input.Question))
		return map[string]any{"ledger_id": id, "question_id": input.Question, "forecasts": items}, "forecast.list", "Forecasts were read", err
	case service.OperationForecastShow:
		id, result, err := service.LoadForecastShow(ctx, file, nil, ledger.Slug(input.Question), ledger.Slug(input.Forecast))
		return map[string]any{"ledger_id": id, "question_id": input.Question, "forecast": result}, "forecast.show", "Forecast was read", err
	case service.OperationForecastSeal:
		return s.dispatchForecastSeal(ctx, file, input, now)
	case service.OperationForecastReveal:
		return s.dispatchForecastReveal(ctx, file, input, now)
	case service.OperationForecastKeyHintUpdate:
		if input.DryRun {
			result, err := service.PlanForecastKeyHintUpdateFile(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), input.KeyHint)
			return result, "forecast.key_hint.update.planned", "Key hint update is valid; no file was changed", err
		}
		result, err := service.CommitForecastKeyHintUpdateFile(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), input.KeyHint)
		return result, "forecast.key_hint.updated", "Forecast key hint was updated", err
	case service.OperationTargetBuild:
		if input.DryRun {
			result, err := service.PlanTargetBuild(ctx, file, input.All, ledger.Slug(input.Question), ledger.Slug(input.Forecast))
			return result, "target.build.planned", "Target build is valid; no files were written", err
		}
		result, err := service.CommitTargetBuild(ctx, file, input.All, ledger.Slug(input.Question), ledger.Slug(input.Forecast))
		return result, "target.built", "Target artifacts were built", err
	case service.OperationTargetCheck:
		result, err := service.InspectTargets(ctx, file, input.All, ledger.Slug(input.Question), ledger.Slug(input.Forecast))
		if err != nil {
			return nil, "", "", err
		}
		if result.FailureCode != "" {
			return nil, "", "", app.WithDetails(app.NewError(result.FailureCode, "one or more forecast targets could not be verified", nil), map[string]any{"ledger_id": result.LedgerID, "targets": result.Targets})
		}
		code, message := "target.valid", "Target artifacts match the ledger"
		for _, target := range result.Targets {
			if string(target.State) == string(service.LayerNotApplicable) {
				code, message = "target.checked", "Target inspection completed; some forecasts have no retained target"
				break
			}
		}
		return result, code, message, nil
	case service.OperationTimestampStamp:
		result, err := service.CommitTimestampStamp(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), service.TimestampStampOptions{DryRun: input.DryRun, Offline: s.config.Mode.Offline, Effects: s.effects})
		return result, "timestamp.pending", "OpenTimestamps receipt was stored as pending", err
	case service.OperationTimestampUpgrade:
		result, err := service.CommitTimestampUpgrade(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), service.TimestampUpgradeOptions{DryRun: input.DryRun, Offline: s.config.Mode.Offline})
		return result, "timestamp.upgraded", "OpenTimestamps receipt was upgraded", err
	case service.OperationTimestampStatus:
		result, err := service.TimestampStatusFor(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast))
		return result, "timestamp.status", "OpenTimestamps local status was read", err
	case service.OperationTimestampVerify:
		verifiedAt, err := optionalTimestamp(input.VerifiedAt, "verified_at")
		if err != nil {
			return nil, "", "", err
		}
		result, err := service.CommitTimestampVerify(ctx, file, ledger.Slug(input.Question), ledger.Slug(input.Forecast), service.TimestampVerifyOptions{DryRun: input.DryRun, Offline: s.config.Mode.Offline, VerifiedAt: verifiedAt, Observer: s.bitcoinObserver})
		code, message := "timestamp.verification."+string(result.Verification.State), "Timestamp verification completed with status "+string(result.Verification.State)
		if result.Verification.State == service.LayerPass {
			code, message = "timestamp.verified", "OpenTimestamps Bitcoin evidence was verified"
		}
		if input.DryRun {
			code, message = "timestamp.verify.planned", "Timestamp verification is valid; network observation and ledger update were deferred"
		}
		return result, code, message, err
	case service.OperationVerificationRun:
		result, err := service.VerifyLedgerEvidence(ctx, file, service.VerificationOptions{Offline: s.config.Mode.Offline, CheckSources: input.CheckSources, QuestionID: ledger.Slug(input.Question), ForecastID: ledger.Slug(input.Forecast)})
		return result, "verification." + string(result.Overall), "Verification completed with status " + string(result.Overall), err
	case service.OperationPublicationBuild:
		output, err := s.roots.Resolve(service.RootOutput, input.Output, false)
		if err != nil {
			return nil, "", "", err
		}
		result, err := service.CommitPublicationBuild(ctx, file, output, input.DryRun)
		return result, "publication.built", "Evidence package was built", err
	case service.OperationPublicationVerify:
		manifest, err := s.roots.Resolve(service.RootOutput, input.Manifest, true)
		if err != nil {
			return nil, "", "", err
		}
		if input.Online && s.config.Mode.Offline {
			return nil, "", "", app.NewError(app.CodeNetworkDisabled, "online package verification is disabled by server offline mode", nil)
		}
		result, err := service.VerifyPublicationPackage(ctx, file, manifest, service.PublicationVerifyOptions{Online: input.Online, Offline: !input.Online})
		return result, "publication.verification." + string(result.Overall), "Package verification completed with status " + string(result.Overall), err
	default:
		return nil, "", "", app.NewError(app.CodeUnavailable, "operation is not registered", nil)
	}
}

func (s *Server) dispatchInit(ctx context.Context, file string, input toolInput, operationAt ledger.Timestamp) (any, string, string, error) {
	if input.InputFile == "" && inlineVisibility(input.Input, true) == ledger.VisibilitySealed {
		return nil, "", "", app.NewError(app.CodeUsage, "sealed initialization must use a protected input_file reference", nil)
	}
	var value service.InitInput
	if len(input.Input) > 0 || input.InputFile != "" {
		if err := s.decodePublicOrProtected(ctx, input, service.InputSchemaInit, &value); err != nil {
			return nil, "", "", err
		}
	}
	root, err := service.BuildLedgerRootAt(service.InitRootRequest{LedgerID: ledger.Slug(input.LedgerID), Timezone: input.Timezone, ForecasterID: ledger.Slug(input.ForecasterID), ForecasterName: input.ForecasterName, ForecasterKind: ledger.ForecasterKind(input.ForecasterKind), Input: value}, operationAt)
	if err != nil {
		return nil, "", "", err
	}
	shape, err := service.ClassifyInitInput(value)
	if err != nil {
		return nil, "", "", err
	}
	var model *ledger.Ledger
	var sealed service.SealedInitialBuild
	keyPath := ""
	if input.KeyFile != "" {
		keyPath, err = s.roots.Resolve(service.RootSecret, input.KeyFile, false)
		if err != nil {
			return nil, "", "", err
		}
	}
	if shape != service.CreationSealedForecast && keyPath != "" {
		return nil, "", "", app.NewError(app.CodeUsage, "key_file is only valid for a sealed initial forecast", nil)
	}
	switch shape {
	case service.CreationLedgerOnly:
		model = root
	case service.CreationQuestionOnly:
		model, err = service.BuildInitialQuestionLedgerAt(root, *value.Question, operationAt)
	case service.CreationPublicForecast:
		model, err = service.BuildInitialPublicLedgerAt(root, *value.Question, operationAt)
	case service.CreationSealedForecast:
		if input.InputFile == "" || keyPath == "" {
			return nil, "", "", app.NewError(app.CodeUsage, "sealed initialization requires input_file and key_file in a secret root", nil)
		}
		if input.DryRun {
			model, err = service.PlanInitialSealedLedgerAt(root, *value.Question, operationAt)
		} else {
			sealed, err = service.BuildInitialSealedLedgerAt(ctx, root, *value.Question, operationAt, s.effects)
			model = sealed.Ledger
		}
	}
	if err != nil {
		return nil, "", "", err
	}
	if input.DryRun {
		if _, err := service.EncodeNewLedger(model, file); err != nil {
			return nil, "", "", err
		}
		effects := []service.SideEffect{{Kind: service.EffectLedger, Action: service.EffectCreate, Status: service.EffectDeferred, Path: filepath.Base(file), Owned: true, Rollback: service.RollbackCreatedPublic}}
		if shape == service.CreationSealedForecast {
			effects = append([]service.SideEffect{{Kind: service.EffectKey, Action: service.EffectCreate, Status: service.EffectDeferred, Path: filepath.Base(keyPath), Owned: true, Rollback: service.RollbackRetainSecret}}, effects...)
		}
		return service.NewInitResult(model, effects, service.Recovery{State: service.RecoveryNone}), "ledger.init.planned", "Ledger initialization is valid; no files were written", nil
	}
	recovery := service.Recovery{State: service.RecoveryNone}
	if shape == service.CreationSealedForecast {
		commit, err := service.CommitInitialSealedFiles(ctx, file, keyPath, sealed, service.InitialCommitOptions{})
		recovery = commit.Recovery
		if err != nil {
			return service.NewInitResult(model, nil, recovery), "", "", err
		}
	} else {
		if _, err = service.CommitNewLedger(file, model); err != nil {
			return nil, "", "", err
		}
	}
	effects := []service.SideEffect{{Kind: service.EffectLedger, Action: service.EffectCreate, Status: service.EffectCompleted, Path: filepath.Base(file), Owned: true, Rollback: service.RollbackCreatedPublic}}
	if shape == service.CreationSealedForecast {
		effects = append([]service.SideEffect{{Kind: service.EffectKey, Action: service.EffectCreate, Status: service.EffectCompleted, Path: filepath.Base(keyPath), Owned: true, Rollback: service.RollbackRetainSecret}}, effects...)
	}
	return service.NewInitResult(model, effects, recovery), "ledger.initialized", "Ledger was created", nil
}

func dispatchPlatformMutation(ctx context.Context, operation service.OperationName, file string, id ledger.Slug, raw json.RawMessage, dryRun bool) (any, string, string, error) {
	if operation == service.OperationPlatformAdd {
		var value service.PlatformCreateInput
		if err := decodeInline(ctx, raw, service.InputSchemaPlatformCreate, &value); err != nil {
			return nil, "", "", err
		}
		if dryRun {
			result, err := service.PlanPlatformAddFile(ctx, file, id, value)
			return result, "platform.add.planned", "Platform addition is valid; no file was changed", err
		}
		result, err := service.CommitPlatformAddFile(ctx, file, id, value)
		return result, "platform.added", "Platform was added", err
	}
	var value service.PlatformPatchInput
	if err := decodeInline(ctx, raw, service.InputSchemaPlatformPatch, &value); err != nil {
		return nil, "", "", err
	}
	if dryRun {
		result, err := service.PlanPlatformUpdateFile(ctx, file, id, value)
		return result, "platform.update.planned", "Platform update is valid; no file was changed", err
	}
	result, err := service.CommitPlatformUpdateFile(ctx, file, id, value)
	return result, "platform.updated", "Platform was updated", err
}

func (s *Server) dispatchQuestionAdd(ctx context.Context, file string, input toolInput, now ledger.Timestamp) (any, string, string, error) {
	if input.InputFile == "" && inlineVisibility(input.Input, false) == ledger.VisibilitySealed {
		return nil, "", "", app.NewError(app.CodeUsage, "a sealed first forecast must use a protected input_file reference", nil)
	}
	var value service.QuestionAddInput
	if err := s.decodePublicOrProtected(ctx, input, service.InputSchemaQuestionAdd, &value); err != nil {
		return nil, "", "", err
	}
	normalized := service.NormalizedQuestionCreate{ID: ledger.Slug(input.Question), Type: ledger.QuestionType(input.Type), Input: value}
	shape, err := service.ClassifyQuestionAddInput(value)
	if err != nil {
		return nil, "", "", err
	}
	if shape != service.CreationSealedForecast && input.KeyFile != "" {
		return nil, "", "", app.NewError(app.CodeUsage, "key_file is only valid for a sealed initial forecast", nil)
	}
	if shape == service.CreationQuestionOnly {
		if input.DryRun {
			result, err := service.PlanQuestionAddEmptyFile(ctx, file, normalized, now)
			return result, "question.add.planned", "Question addition is valid; no file was changed", err
		}
		result, err := service.CommitQuestionAddEmptyFile(ctx, file, normalized, now)
		return result, "question.added", "Question was added", err
	}
	if shape == service.CreationPublicForecast {
		if input.DryRun {
			result, err := service.PlanQuestionAddPublicFile(ctx, file, normalized, now)
			return result, "question.add.planned", "Question addition is valid; no file was changed", err
		}
		result, err := service.CommitQuestionAddPublicFile(ctx, file, normalized, now)
		return result, "question.added", "Question and first forecast were added", err
	}
	if shape != service.CreationSealedForecast || input.InputFile == "" || input.KeyFile == "" {
		return nil, "", "", app.NewError(app.CodeUsage, "a sealed first forecast requires input_file and key_file in a secret root", nil)
	}
	keyPath, err := s.roots.Resolve(service.RootSecret, input.KeyFile, false)
	if err != nil {
		return nil, "", "", err
	}
	if input.DryRun {
		result, err := service.PlanQuestionAddSealedFile(ctx, file, keyPath, normalized, now)
		return result, "question.add.planned", "Question addition is valid; no file was changed", err
	}
	result, err := service.CommitQuestionAddSealedFile(ctx, file, keyPath, normalized, now, s.effects)
	return result, "question.added", "Question and first forecast were added", err
}

func dispatchQuestionTerminal(ctx context.Context, operation service.OperationName, file string, id ledger.Slug, raw json.RawMessage, now ledger.Timestamp, dryRun bool) (any, string, string, error) {
	switch operation {
	case service.OperationQuestionResolve:
		var value service.ResolutionInput
		if err := decodeInline(ctx, raw, service.InputSchemaResolution, &value); err != nil {
			return nil, "", "", err
		}
		if dryRun {
			result, err := service.PlanQuestionResolveFile(ctx, file, id, value, now)
			return result, "question.resolve.planned", "Question resolution is valid; no file was changed", err
		}
		result, err := service.CommitQuestionResolveFile(ctx, file, id, value, now)
		return result, "question.resolved", "Question was resolved", err
	case service.OperationQuestionAnnul:
		var value service.AnnulInput
		if err := decodeInline(ctx, raw, service.InputSchemaAnnul, &value); err != nil {
			return nil, "", "", err
		}
		if dryRun {
			result, err := service.PlanQuestionAnnulFile(ctx, file, id, value, now)
			return result, "question.annul.planned", "Question annulment is valid; no file was changed", err
		}
		result, err := service.CommitQuestionAnnulFile(ctx, file, id, value, now)
		return result, "question.annulled", "Question was annulled", err
	default:
		var value service.DisputeInput
		if err := decodeInline(ctx, raw, service.InputSchemaDispute, &value); err != nil {
			return nil, "", "", err
		}
		if dryRun {
			result, err := service.PlanQuestionDisputeFile(ctx, file, id, value, now)
			return result, "question.dispute.planned", "Question dispute is valid; no file was changed", err
		}
		result, err := service.CommitQuestionDisputeFile(ctx, file, id, value, now)
		return result, "question.disputed", "Question was disputed", err
	}
}

func (s *Server) dispatchForecastSeal(ctx context.Context, file string, input toolInput, now ledger.Timestamp) (any, string, string, error) {
	if input.InputFile == "" || input.KeyFile == "" || len(input.Input) > 0 {
		return nil, "", "", app.NewError(app.CodeUsage, "forecast_seal requires protected input_file and new key_file references", nil)
	}
	var value service.SealedForecastInput
	if err := s.decodeProtected(ctx, input.InputFile, service.InputSchemaForecastSeal, &value); err != nil {
		return nil, "", "", err
	}
	keyPath, err := s.roots.Resolve(service.RootSecret, input.KeyFile, false)
	if err != nil {
		return nil, "", "", err
	}
	if input.DryRun {
		result, err := service.PlanForecastSealFile(ctx, file, keyPath, ledger.Slug(input.Question), ledger.Slug(input.Forecast), value, now)
		return result, "forecast.seal.planned", "Sealed forecast creation is valid; no file was changed", err
	}
	result, err := service.CommitForecastSealFile(ctx, file, keyPath, ledger.Slug(input.Question), ledger.Slug(input.Forecast), value, now, s.effects)
	return result, "forecast.sealed", "Sealed forecast and protected key were created", err
}

func (s *Server) dispatchForecastReveal(ctx context.Context, file string, input toolInput, now ledger.Timestamp) (any, string, string, error) {
	keyPath, err := s.roots.Resolve(service.RootSecret, input.KeyFile, true)
	if err != nil {
		return nil, "", "", err
	}
	revealedAt := now
	if input.RevealedAt != "" {
		revealedAt, err = optionalTimestamp(input.RevealedAt, "revealed_at")
		if err != nil {
			return nil, "", "", err
		}
	}
	if input.DryRun {
		result, err := service.PlanForecastRevealFile(ctx, file, keyPath, ledger.Slug(input.Question), ledger.Slug(input.Forecast), revealedAt)
		return result, "forecast.reveal.planned", "Forecast reveal is valid; no file was changed", err
	}
	result, err := service.CommitForecastRevealFile(ctx, file, keyPath, ledger.Slug(input.Question), ledger.Slug(input.Forecast), revealedAt)
	return result, "forecast.revealed", "Forecast was authenticated and revealed", err
}

func (s *Server) decodePublicOrProtected(ctx context.Context, input toolInput, schema service.InputSchemaName, destination any) error {
	if input.InputFile != "" {
		if len(input.Input) > 0 {
			return app.NewError(app.CodeUsage, "input and input_file cannot be combined", nil)
		}
		return s.decodeProtected(ctx, input.InputFile, schema, destination)
	}
	return decodeInline(ctx, input.Input, schema, destination)
}

func (s *Server) decodeProtected(ctx context.Context, reference string, schema service.InputSchemaName, destination any) error {
	path, err := s.roots.Resolve(service.RootSecret, reference, true)
	if err != nil {
		return err
	}
	data, err := storage.ReadProtectedFile(path, 8<<20)
	if err != nil {
		return err
	}
	defer clear(data)
	return service.DecodeOperationInput(ctx, "-", bytes.NewReader(data), schema, destination)
}

func decodeInline(ctx context.Context, raw json.RawMessage, schema service.InputSchemaName, destination any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return app.NewError(app.CodeUsage, "input is required", nil)
	}
	return service.DecodeOperationInput(ctx, "-", bytes.NewReader(raw), schema, destination)
}

func optionalTimestamp(value, field string) (ledger.Timestamp, error) {
	if value == "" {
		return "", nil
	}
	result := ledger.Timestamp(value)
	if _, err := service.ParseTimestamp(result, field); err != nil {
		return "", err
	}
	return result, nil
}

func inlineVisibility(raw json.RawMessage, init bool) ledger.ForecastVisibility {
	var envelope struct {
		Question *struct {
			InitialForecast struct {
				Visibility ledger.ForecastVisibility `json:"visibility"`
			} `json:"initial_forecast"`
		} `json:"question"`
		InitialForecast struct {
			Visibility ledger.ForecastVisibility `json:"visibility"`
		} `json:"initial_forecast"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &envelope) != nil {
		return ""
	}
	if init && envelope.Question != nil {
		return envelope.Question.InitialForecast.Visibility
	}
	return envelope.InitialForecast.Visibility
}

func resultFailureCode(data any) app.ErrorCode {
	switch value := data.(type) {
	case service.VerificationReport:
		return value.FailureCode
	case service.PublicationVerifyResult:
		return value.FailureCode
	case service.TimestampVerifyResult:
		return value.FailureCode
	default:
		return ""
	}
}

func errorToolResult(operation service.OperationName, err error) *sdk.CallToolResult {
	var applicationErr *app.Error
	if !errors.As(err, &applicationErr) {
		applicationErr = app.NewError(app.CodeInternal, "internal operation error", err)
	}
	safe := *applicationErr
	safe.Cause = nil
	return marshalToolResult(toolEnvelope{Operation: operation, Code: string(safe.Code), Message: safe.Message, Error: &safe}, true)
}

func marshalToolResult(value toolEnvelope, isError bool) *sdk.CallToolResult {
	data, err := json.Marshal(value)
	if err != nil {
		data = []byte(`{"operation":"unknown","code":"internal","message":"tool result could not be encoded"}`)
		isError = true
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: string(data)}}, StructuredContent: value, IsError: isError}
}
