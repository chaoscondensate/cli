package service

import "sort"

type SelectionKind string

const (
	SelectionLedger           SelectionKind = "ledger"
	SelectionPlatform         SelectionKind = "platform"
	SelectionQuestion         SelectionKind = "question"
	SelectionForecast         SelectionKind = "forecast"
	SelectionQuestionForecast SelectionKind = "question_forecast"
	SelectionTarget           SelectionKind = "target"
)

type InputTransport string

const (
	InputNone              InputTransport = "none"
	InputInline            InputTransport = "inline"
	InputProtectedFile     InputTransport = "protected_file"
	InputInlineOrProtected InputTransport = "inline_or_protected_file"
)

type RequestField struct {
	Name        string   `json:"name"`
	Required    bool     `json:"required"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type OperationDefinition struct {
	Name           OperationName   `json:"operation"`
	CLI            string          `json:"cli"`
	MCPTool        string          `json:"mcp_tool"`
	Selection      SelectionKind   `json:"selection"`
	InputSchema    InputSchemaName `json:"input_schema,omitempty"`
	InputTransport InputTransport  `json:"input_transport"`
	Fields         []RequestField  `json:"fields,omitempty"`
	Policy         OperationPolicy `json:"policy"`
	ResultNotes    string          `json:"result_notes,omitempty"`
}

func OperationDefinitions() []OperationDefinition {
	definitions := []OperationDefinition{
		definition(OperationLedgerInit, "init", "ledger_init", SelectionLedger, InputSchemaInit, InputInlineOrProtected,
			requiredString("ledger_id", "Stable ledger ID"), requiredString("timezone", "IANA timezone name"),
			requiredString("forecaster_id", "Stable forecaster ID"), requiredString("forecaster_name", "Forecaster display name"),
			RequestField{Name: "forecaster_kind", Type: "string", Description: "Forecaster kind", Enum: []string{"individual", "team"}},
			RequestField{Name: "key_file", Type: "string", Description: "New protected key reference for a sealed first forecast"}),
		definition(OperationLedgerUpdate, "ledger update", "ledger_update", SelectionLedger, InputSchemaRootMetadata, InputInline),
		definition(OperationLedgerValidate, "validate", "ledger_validate", SelectionLedger, "", InputNone),
		definition(OperationLedgerStatus, "status", "ledger_status", SelectionLedger, "", InputNone),
		definition(OperationPlatformAdd, "platform add", "platform_add", SelectionPlatform, InputSchemaPlatformCreate, InputInline),
		definition(OperationPlatformUpdate, "platform update", "platform_update", SelectionPlatform, InputSchemaPlatformPatch, InputInline),
		definition(OperationPlatformList, "platform list", "platform_list", SelectionLedger, "", InputNone),
		definition(OperationPlatformShow, "platform show", "platform_show", SelectionPlatform, "", InputNone),
		definition(OperationPlatformRemove, "platform remove", "platform_remove", SelectionPlatform, "", InputNone),
		definition(OperationQuestionAdd, "question add", "question_add", SelectionQuestion, InputSchemaQuestionAdd, InputInlineOrProtected,
			RequestField{Name: "type", Required: true, Type: "string", Description: "Question type", Enum: []string{"binary", "multiple_choice", "numeric", "date"}},
			RequestField{Name: "key_file", Type: "string", Description: "New protected key reference for a sealed first forecast"}),
		definition(OperationQuestionUpdate, "question update", "question_update", SelectionQuestion, InputSchemaQuestionPatch, InputInline),
		definition(OperationQuestionList, "question list", "question_list", SelectionLedger, "", InputNone),
		definition(OperationQuestionShow, "question show", "question_show", SelectionQuestion, "", InputNone),
		definition(OperationQuestionResolve, "question resolve", "question_resolve", SelectionQuestion, InputSchemaResolution, InputInline),
		definition(OperationQuestionAnnul, "question annul", "question_annul", SelectionQuestion, InputSchemaAnnul, InputInline),
		definition(OperationQuestionDispute, "question dispute", "question_dispute", SelectionQuestion, InputSchemaDispute, InputInline),
		definition(OperationForecastAdd, "forecast add", "forecast_add", SelectionForecast, InputSchemaForecastCreate, InputInline),
		definition(OperationForecastList, "forecast list", "forecast_list", SelectionQuestion, "", InputNone),
		definition(OperationForecastShow, "forecast show", "forecast_show", SelectionForecast, "", InputNone),
		definition(OperationForecastSeal, "forecast seal", "forecast_seal", SelectionForecast, InputSchemaForecastSeal, InputProtectedFile,
			requiredString("key_file", "New protected key reference")),
		definition(OperationForecastReveal, "forecast reveal", "forecast_reveal", SelectionForecast, "", InputNone,
			requiredString("key_file", "Protected key reference"),
			RequestField{Name: "revealed_at", Type: "string", Description: "Optional exact RFC 3339 reveal time"}),
		definition(OperationForecastKeyHintUpdate, "forecast key-hint update", "forecast_key_hint_update", SelectionForecast, InputSchemaKeyHintUpdate, InputInline),
		definition(OperationTargetBuild, "target build", "target_build", SelectionTarget, "", InputNone),
		definition(OperationTargetCheck, "target check", "target_check", SelectionTarget, "", InputNone),
		definition(OperationTimestampStamp, "timestamp stamp", "timestamp_stamp", SelectionForecast, "", InputNone,
			requiredString("tsa_url", "Public HTTPS timestamp authority URL"),
			requiredString("ca_bundle", "Retained ledger-relative PEM CA bundle")),
		definition(OperationTimestampStatus, "timestamp status", "timestamp_status", SelectionForecast, "", InputNone),
		definition(OperationTimestampVerify, "timestamp verify", "timestamp_verify", SelectionForecast, "", InputNone),
		definition(OperationVerificationRun, "verify", "verification_run", SelectionLedger, "", InputNone,
			RequestField{Name: "question", Type: "string", Description: "Optional question ID"},
			RequestField{Name: "forecast", Type: "string", Description: "Optional forecast ID; requires question"},
			RequestField{Name: "check_sources", Type: "boolean", Description: "Check outcome source reachability"}),
		definition(OperationPublicationBuild, "publish build", "publication_build", SelectionLedger, "", InputNone,
			requiredString("output", "New package path inside an output root")),
		definition(OperationPublicationVerify, "publish verify", "publication_verify", SelectionLedger, "", InputNone,
			requiredString("manifest", "Manifest path inside the package root")),
	}
	return definitions
}

func SortedOperationDefinitions() []OperationDefinition {
	definitions := OperationDefinitions()
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	return definitions
}

func definition(name OperationName, cli, mcp string, selection SelectionKind, input InputSchemaName, transport InputTransport, fields ...RequestField) OperationDefinition {
	policy, ok := PolicyForOperation(name)
	if !ok {
		panic("operation definition has no policy: " + string(name))
	}
	return OperationDefinition{
		Name: name, CLI: cli, MCPTool: mcp, Selection: selection,
		InputSchema: input, InputTransport: transport, Fields: fields, Policy: policy, ResultNotes: operationResultNotes(name),
	}
}

func operationResultNotes(name OperationName) string {
	switch name {
	case OperationTimestampVerify:
		return "Returns a structured timing report for pending, not-checked, mismatch, and verified outcomes; source unavailability is not proof failure."
	case OperationVerificationRun, OperationPublicationVerify:
		return "Pass requires at least one applicable forecast-evidence layer; an empty or all-not-applicable selection returns no_evidence."
	default:
		return ""
	}
}

func requiredString(name, description string) RequestField {
	return RequestField{Name: name, Required: true, Type: "string", Description: description}
}
