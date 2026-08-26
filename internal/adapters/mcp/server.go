package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/buildinfo"
	"github.com/chaoscondensate/cli/internal/service"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const ProtocolRevision = buildinfo.MCPProtocolVersion

type Config struct {
	LedgerRoots   []string
	OutputRoots   []string
	SecretRoots   []string
	Mode          service.AccessMode
	Timeout       time.Duration
	MaxConcurrent int
	MaxToolBytes  int
	Stderr        io.Writer
	Effects       service.Effects
}

type Server struct {
	config       Config
	roots        *RootSet
	effects      service.Effects
	sdk          *sdk.Server
	sem          chan struct{}
	maxToolBytes int
}

func New(config Config) (*Server, error) {
	if config.Mode.AllowReveal && config.Mode.ReadOnly {
		return nil, app.NewError(app.CodeUsage, "--allow-reveal cannot be combined with --read-only", nil)
	}
	if config.Mode.AllowReveal && len(config.SecretRoots) == 0 {
		return nil, app.NewError(app.CodeUsage, "--allow-reveal requires at least one --secret-root", nil)
	}
	roots, err := NewRootSet(config.LedgerRoots, config.OutputRoots, config.SecretRoots)
	if err != nil {
		return nil, err
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 16
	}
	if config.MaxToolBytes == 0 {
		config.MaxToolBytes = 8 << 20
	}
	if config.MaxConcurrent < 1 || config.MaxConcurrent > 256 || config.MaxToolBytes < 1024 || config.MaxToolBytes > 64<<20 {
		return nil, app.NewError(app.CodeUsage, "MCP resource limits are outside the supported range", nil)
	}
	if config.Stderr == nil {
		config.Stderr = io.Discard
	}
	effects := config.Effects
	if err := effects.Validate(); err != nil {
		effects = service.ProductionEffects()
	}
	info := buildinfo.Current()
	profile := ots.Profile()
	mode := "online"
	if config.Mode.Offline {
		mode = "offline"
	}
	access := "read-write"
	if config.Mode.ReadOnly {
		access = "read-only"
	}
	instructions := fmt.Sprintf("Forecast Ledger MCP. schema=%s schema_commit=%s schema_sha256=%s network_profile=%s mode=%s access=%s timestamps=experimental protocol=%s", info.Schema.Version, info.Schema.Commit, info.Schema.SHA256, profile.ID, mode, access, info.MCPProtocol)
	logger := slog.New(slog.NewTextHandler(config.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	server := sdk.NewServer(&sdk.Implementation{Name: info.Binary, Version: info.Version}, &sdk.ServerOptions{Instructions: instructions, Logger: logger, PageSize: 100})
	result := &Server{config: config, roots: roots, effects: effects, sdk: server, sem: make(chan struct{}, config.MaxConcurrent), maxToolBytes: config.MaxToolBytes}
	if err := result.registerTools(); err != nil {
		return nil, err
	}
	result.registerResources()
	return result, nil
}

func (s *Server) SDK() *sdk.Server { return s.sdk }

func (s *Server) Run(ctx context.Context, transport sdk.Transport) error {
	if s == nil || s.sdk == nil {
		return app.NewError(app.CodeInternal, "MCP server is not initialized", nil)
	}
	return s.sdk.Run(ctx, transport)
}

func (s *Server) ServeStdio(ctx context.Context) error {
	return s.Run(ctx, &sdk.StdioTransport{})
}

type toolContract struct {
	Allowed  []string
	Required []string
}

func contracts() map[service.OperationName]toolContract {
	file := []string{"file"}
	return map[service.OperationName]toolContract{
		service.OperationLedgerInit:            {Allowed: []string{"file", "ledger_id", "timezone", "forecaster_id", "forecaster_name", "forecaster_kind", "input", "input_file", "key_file", "dry_run"}, Required: []string{"file", "ledger_id", "timezone", "forecaster_id", "forecaster_name"}},
		service.OperationLedgerUpdate:          {Allowed: append(file, "input", "dry_run"), Required: []string{"file", "input"}},
		service.OperationLedgerValidate:        {Allowed: file, Required: file},
		service.OperationLedgerStatus:          {Allowed: file, Required: file},
		service.OperationPlatformAdd:           {Allowed: append(file, "platform", "input", "dry_run"), Required: []string{"file", "platform", "input"}},
		service.OperationPlatformUpdate:        {Allowed: append(file, "platform", "input", "dry_run"), Required: []string{"file", "platform", "input"}},
		service.OperationPlatformList:          {Allowed: file, Required: file},
		service.OperationPlatformShow:          {Allowed: append(file, "platform"), Required: []string{"file", "platform"}},
		service.OperationPlatformRemove:        {Allowed: append(file, "platform", "confirm", "dry_run"), Required: []string{"file", "platform"}},
		service.OperationQuestionAdd:           {Allowed: append(file, "question", "type", "input", "input_file", "key_file", "dry_run"), Required: []string{"file", "question", "type"}},
		service.OperationQuestionUpdate:        {Allowed: append(file, "question", "input", "dry_run"), Required: []string{"file", "question", "input"}},
		service.OperationQuestionList:          {Allowed: file, Required: file},
		service.OperationQuestionShow:          {Allowed: append(file, "question"), Required: []string{"file", "question"}},
		service.OperationQuestionResolve:       {Allowed: append(file, "question", "input", "confirm", "dry_run"), Required: []string{"file", "question", "input"}},
		service.OperationQuestionAnnul:         {Allowed: append(file, "question", "input", "confirm", "dry_run"), Required: []string{"file", "question", "input"}},
		service.OperationQuestionDispute:       {Allowed: append(file, "question", "input", "confirm", "dry_run"), Required: []string{"file", "question", "input"}},
		service.OperationForecastAdd:           {Allowed: append(file, "question", "forecast", "input", "dry_run"), Required: []string{"file", "question", "forecast", "input"}},
		service.OperationForecastList:          {Allowed: append(file, "question"), Required: []string{"file", "question"}},
		service.OperationForecastShow:          {Allowed: append(file, "question", "forecast"), Required: []string{"file", "question", "forecast"}},
		service.OperationForecastSeal:          {Allowed: append(file, "question", "forecast", "input_file", "key_file", "dry_run"), Required: []string{"file", "question", "forecast", "input_file", "key_file"}},
		service.OperationForecastReveal:        {Allowed: append(file, "question", "forecast", "key_file", "revealed_at", "confirm", "dry_run"), Required: []string{"file", "question", "forecast", "key_file", "confirm"}},
		service.OperationForecastKeyHintUpdate: {Allowed: append(file, "question", "forecast", "key_hint", "dry_run"), Required: []string{"file", "question", "forecast", "key_hint"}},
		service.OperationTargetBuild:           {Allowed: append(file, "question", "forecast", "all", "dry_run"), Required: file},
		service.OperationTargetCheck:           {Allowed: append(file, "question", "forecast", "all"), Required: file},
		service.OperationTimestampStamp:        {Allowed: append(file, "question", "forecast", "dry_run"), Required: []string{"file", "question", "forecast"}},
		service.OperationTimestampUpgrade:      {Allowed: append(file, "question", "forecast", "dry_run"), Required: []string{"file", "question", "forecast"}},
		service.OperationTimestampStatus:       {Allowed: append(file, "question", "forecast"), Required: []string{"file", "question", "forecast"}},
		service.OperationTimestampVerify:       {Allowed: append(file, "question", "forecast", "verified_at", "dry_run"), Required: []string{"file", "question", "forecast"}},
		service.OperationVerificationRun:       {Allowed: append(file, "question", "forecast", "check_sources"), Required: file},
		service.OperationPublicationBuild:      {Allowed: append(file, "output", "dry_run"), Required: []string{"file", "output"}},
		service.OperationPublicationVerify:     {Allowed: append(file, "manifest", "online"), Required: []string{"file", "manifest"}},
	}
}

func (s *Server) registerTools() error {
	contracts := contracts()
	for _, definition := range service.SortedOperationDefinitions() {
		contract, ok := contracts[definition.Name]
		if !ok {
			return app.NewError(app.CodeInternal, "MCP operation has no closed tool contract", fmt.Errorf("%s", definition.Name))
		}
		if definition.Name == service.OperationForecastReveal && !s.config.Mode.AllowReveal {
			continue
		}
		if (definition.Name == service.OperationForecastSeal || definition.Name == service.OperationForecastReveal) && !s.roots.Has(service.RootSecret) {
			continue
		}
		if definition.Name == service.OperationPublicationBuild && !s.roots.Has(service.RootOutput) {
			continue
		}
		if definition.Name == service.OperationPublicationVerify && !s.roots.Has(service.RootOutput) {
			continue
		}
		allowed := make(map[string]bool, len(contract.Allowed))
		for _, name := range contract.Allowed {
			allowed[name] = true
		}
		schema, err := toolSchema(definition, contract)
		if err != nil {
			return err
		}
		description := fmt.Sprintf("%s. file paths use root-name:relative/path. access=%s network=%s", definition.CLI, accessLabel(s.config.Mode), definition.Policy.Network)
		s.sdk.AddTool(&sdk.Tool{Name: definition.MCPTool, Description: description, InputSchema: schema}, s.toolHandler(definition, allowed, contract.Required))
	}
	return nil
}

func toolSchema(definition service.OperationDefinition, contract toolContract) (map[string]any, error) {
	properties := map[string]any{}
	for _, name := range contract.Allowed {
		properties[name] = scalarProperty(name)
	}
	if definition.InputSchema != "" {
		var inputSchema map[string]any
		data, err := service.InputSchema(definition.InputSchema)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &inputSchema); err != nil {
			return nil, err
		}
		if _, present := properties["input"]; present {
			properties["input"] = publicInlineSchema(definition.Name, inputSchema)
		}
	}
	return map[string]any{"type": "object", "properties": properties, "required": contract.Required, "additionalProperties": false}, nil
}

func publicInlineSchema(operation service.OperationName, schema map[string]any) any {
	var restriction map[string]any
	switch operation {
	case service.OperationLedgerInit:
		restriction = map[string]any{"properties": map[string]any{"question": map[string]any{"properties": map[string]any{"initial_forecast": map[string]any{"properties": map[string]any{"visibility": map[string]any{"const": "public"}}}}}}}
	case service.OperationQuestionAdd:
		restriction = map[string]any{"properties": map[string]any{"initial_forecast": map[string]any{"properties": map[string]any{"visibility": map[string]any{"const": "public"}}}}}
	default:
		return schema
	}
	return map[string]any{"allOf": []any{schema, restriction}}
}

func scalarProperty(name string) map[string]any {
	switch name {
	case "dry_run", "confirm", "all", "online", "check_sources":
		return map[string]any{"type": "boolean"}
	case "type":
		return map[string]any{"type": "string", "enum": []string{"binary", "multiple_choice", "numeric", "date"}}
	case "forecaster_kind":
		return map[string]any{"type": "string", "enum": []string{"individual", "team"}, "default": "individual"}
	case "input":
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string", "minLength": 1}
	}
}

func accessLabel(mode service.AccessMode) string {
	if mode.ReadOnly {
		return "read-only"
	}
	return "read-write"
}
