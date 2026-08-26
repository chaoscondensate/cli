package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

var inputValidators sync.Map

// DecodeOperationInput parses, validates, and decodes one closed operation
// input. Source may be a JSON/YAML path or "-" for the supplied stdin reader.
func DecodeOperationInput(ctx context.Context, source string, stdin io.Reader, schemaName InputSchemaName, destination any) error {
	return decodeOperationInput(ctx, source, stdin, schemaName, destination, document.DefaultLimits)
}

func decodeOperationInput(ctx context.Context, source string, stdin io.Reader, schemaName InputSchemaName, destination any, limits document.Limits) error {
	if err := validateDecodeDestination(destination); err != nil {
		return app.NewError(app.CodeInternal, "operation input destination is not valid", err)
	}
	if ctx != nil && ctx.Err() != nil {
		return app.NewError(app.CodeInterrupted, "operation input was interrupted", ctx.Err())
	}
	if strings.TrimSpace(source) == "" {
		return app.NewError(app.CodeUsage, "--input is required", nil)
	}

	parsed, err := parseOperationInputSource(source, stdin, limits)
	if err != nil {
		var applicationErr *app.Error
		if errors.As(err, &applicationErr) {
			return applicationErr
		}
		return app.NewError(app.CodeInvalidData, "operation input cannot be parsed", err)
	}
	if ctx != nil && ctx.Err() != nil {
		return app.NewError(app.CodeInterrupted, "operation input was interrupted", ctx.Err())
	}

	validator, err := compiledInputValidator(schemaName)
	if err != nil {
		return app.NewError(app.CodeInternal, "operation input schema cannot be loaded", err)
	}
	if err := validator.Validate(parsed.Root.Any()); err != nil {
		return app.WithDetails(
			app.NewError(app.CodeInvalidData, "operation input does not match its closed schema", nil),
			map[string]any{"schema": schemaName, "issue": safeInputSchemaIssue(err)},
		)
	}

	encoded, err := json.Marshal(parsed.Root.Any())
	if err != nil {
		return app.NewError(app.CodeInternal, "validated operation input cannot be encoded", err)
	}
	if err := json.Unmarshal(encoded, destination); err != nil {
		return app.NewError(app.CodeInternal, "validated operation input cannot be decoded", err)
	}
	return nil
}

func parseOperationInputSource(source string, stdin io.Reader, limits document.Limits) (*document.Document, error) {
	if source == "-" {
		if stdin == nil {
			return nil, app.NewError(app.CodeUsage, "stdin is not available for --input -", nil)
		}
		return parseUnknownInputFormat(stdin, limits)
	}
	file, err := os.Open(source)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, app.NewError(app.CodeNotFound, "operation input file does not exist", err)
		}
		return nil, app.NewError(app.CodeIO, "operation input file cannot be opened", err)
	}
	defer file.Close()

	switch strings.ToLower(filepath.Ext(source)) {
	case ".json":
		return document.ParseJSON(file, limits)
	case ".yaml", ".yml":
		return document.ParseYAML(file, limits)
	default:
		return nil, app.NewError(app.CodeUsage, "operation input filename must end in .json, .yaml, or .yml", nil)
	}
}

func parseUnknownInputFormat(reader io.Reader, limits document.Limits) (*document.Document, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, limits.MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read operation input: %w", err)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return document.ParseJSON(bytes.NewReader(raw), limits)
	}
	return document.ParseYAML(bytes.NewReader(raw), limits)
}

func compiledInputValidator(name InputSchemaName) (*jsonschema.Schema, error) {
	if cached, ok := inputValidators.Load(name); ok {
		return cached.(*jsonschema.Schema), nil
	}
	schemaBytes, err := InputSchema(name)
	if err != nil {
		return nil, err
	}
	documentValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return nil, fmt.Errorf("decode operation input schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource("operation-input-schema.json", documentValue); err != nil {
		return nil, fmt.Errorf("register operation input schema: %w", err)
	}
	compiled, err := compiler.Compile("operation-input-schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile operation input schema: %w", err)
	}
	actual, _ := inputValidators.LoadOrStore(name, compiled)
	return actual.(*jsonschema.Schema), nil
}

func validateDecodeDestination(destination any) error {
	if destination == nil {
		return errors.New("destination is nil")
	}
	value := reflect.ValueOf(destination)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.New("destination must be a non-nil pointer")
	}
	return nil
}

func safeInputSchemaIssue(err error) string {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return "schema.validation"
	}
	if len(validationErr.InstanceLocation) == 0 {
		return "schema.root"
	}
	return "schema." + strings.Join(validationErr.InstanceLocation, ".")
}
