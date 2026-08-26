package service

import (
	"encoding/json"
	"fmt"
	"sort"

	ledgerschema "github.com/chaoscondensate/cli/internal/schema"
)

// InputSchemaName identifies one closed operation input contract. Version 1 is
// tied to the exact embedded Forecast Ledger 1.0.0 definitions.
type InputSchemaName string

const (
	InputSchemaInit              InputSchemaName = "init"
	InputSchemaRootMetadata      InputSchemaName = "root-metadata-patch"
	InputSchemaPlatformCreate    InputSchemaName = "platform-create"
	InputSchemaPlatformPatch     InputSchemaName = "platform-patch"
	InputSchemaQuestionAdd       InputSchemaName = "question-add"
	InputSchemaQuestionPatch     InputSchemaName = "question-patch"
	InputSchemaForecastCreate    InputSchemaName = "forecast-create"
	InputSchemaForecastSeal      InputSchemaName = "forecast-seal"
	InputSchemaKeyHintUpdate     InputSchemaName = "key-hint-update"
	InputSchemaResolution        InputSchemaName = "resolution"
	InputSchemaAnnul             InputSchemaName = "annul"
	InputSchemaDispute           InputSchemaName = "dispute"
	InputSchemaPublicationBuild  InputSchemaName = "publication-build"
	InputSchemaPublicationVerify InputSchemaName = "publication-verify"
)

const inputSchemaVersion = "1"

var inputSchemaNames = []InputSchemaName{
	InputSchemaInit, InputSchemaRootMetadata, InputSchemaPlatformCreate,
	InputSchemaPlatformPatch, InputSchemaQuestionAdd, InputSchemaQuestionPatch,
	InputSchemaForecastCreate, InputSchemaForecastSeal, InputSchemaKeyHintUpdate,
	InputSchemaResolution, InputSchemaAnnul, InputSchemaDispute,
	InputSchemaPublicationBuild, InputSchemaPublicationVerify,
}

func InputSchemaNames() []InputSchemaName {
	result := append([]InputSchemaName(nil), inputSchemaNames...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// InputSchema returns a standalone Draft 2020-12 schema for one operation.
// The returned bytes are deterministic and contain no remote references.
func InputSchema(name InputSchemaName) ([]byte, error) {
	definitions, err := inputDefinitions()
	if err != nil {
		return nil, err
	}
	definition, ok := operationDefinitions()[name]
	if !ok {
		return nil, fmt.Errorf("unknown input schema %q", name)
	}
	definitions[string(name)] = definition
	definitions = reachableDefinitions(definitions, string(name))
	document := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     fmt.Sprintf("https://chaoscondensate.com/schemas/forecast-ledger-cli/input/%s/v%s", name, inputSchemaVersion),
		"$ref":    "#/$defs/" + string(name),
		"$defs":   definitions,
	}
	return json.MarshalIndent(document, "", "  ")
}

func reachableDefinitions(all map[string]any, root string) map[string]any {
	result := make(map[string]any)
	queue := []string{root}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if _, visited := result[name]; visited {
			continue
		}
		definition, ok := all[name]
		if !ok {
			continue
		}
		result[name] = definition
		collectDefinitionReferences(definition, func(reference string) {
			if _, visited := result[reference]; !visited {
				queue = append(queue, reference)
			}
		})
	}
	return result
}

func collectDefinitionReferences(value any, add func(string)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "$ref" {
				if reference, ok := child.(string); ok {
					const prefix = "#/$defs/"
					if len(reference) > len(prefix) && reference[:len(prefix)] == prefix {
						add(reference[len(prefix):])
					}
				}
				continue
			}
			collectDefinitionReferences(child, add)
		}
	case []any:
		for _, child := range typed {
			collectDefinitionReferences(child, add)
		}
	}
}

func inputDefinitions() (map[string]any, error) {
	var contract struct {
		Definitions map[string]any `json:"$defs"`
	}
	if err := json.Unmarshal(ledgerschema.Contract(), &contract); err != nil {
		return nil, fmt.Errorf("decode embedded ledger definitions: %w", err)
	}
	if len(contract.Definitions) == 0 {
		return nil, fmt.Errorf("embedded ledger schema has no definitions")
	}
	result := make(map[string]any, len(contract.Definitions)+16)
	for name, definition := range contract.Definitions {
		result[name] = definition
	}
	for name, definition := range commonInputDefinitions() {
		result[name] = definition
	}
	return result, nil
}

func operationDefinitions() map[InputSchemaName]any {
	return map[InputSchemaName]any{
		InputSchemaInit: closedObject(
			[]string{"question"},
			map[string]any{
				"title":       stringValue(0),
				"description": stringValue(0),
				"created_at":  ref("timestamp"),
				"contact":     ref("contact"),
				"profiles":    arrayOf(ref("profile"), 0),
				"members":     arrayOf(ref("member"), 2),
				"platforms": map[string]any{
					"type": "object", "propertyNames": ref("slug"),
					"additionalProperties": ref("platform"),
				},
				"question": ref("initialQuestionInput"),
			}, nil),
		InputSchemaRootMetadata: closedObject(nil, map[string]any{
			"title":            nullable(stringValue(0)),
			"description":      nullable(stringValue(0)),
			"default_timezone": stringValue(1),
			"forecaster":       ref("forecasterPatchInput"),
		}, map[string]any{"minProperties": 1}),
		InputSchemaPlatformCreate: ref("platform"),
		InputSchemaPlatformPatch: closedObject(nil, map[string]any{
			"name":    nullable(stringValue(1)),
			"kind":    nullable(map[string]any{"enum": []string{"scoring_platform", "prediction_market", "self_hosted", "internal", "informal"}}),
			"url":     nullable(map[string]any{"type": "string", "format": "uri"}),
			"account": nullable(ref("platformAccountPatchInput")),
		}, map[string]any{"minProperties": 1}),
		InputSchemaQuestionAdd: ref("questionAddInput"),
		InputSchemaQuestionPatch: closedObject(nil, map[string]any{
			"title":                  stringValue(1),
			"resolution_criteria":    stringValue(1),
			"forecast_window":        ref("forecastWindowPatchInput"),
			"expected_resolution_at": ref("timestamp"),
			"platform_refs":          nullable(arrayOf(ref("platformRef"), 0)),
			"tags":                   nullable(arrayOf(ref("slug"), 0)),
			"notes":                  nullable(stringValue(0)),
			"status":                 map[string]any{"enum": []string{"open", "closed", "awaiting_resolution"}},
		}, map[string]any{"minProperties": 1}),
		InputSchemaForecastCreate: ref("publicForecastInput"),
		InputSchemaForecastSeal:   ref("sealedForecastInput"),
		InputSchemaKeyHintUpdate: closedObject([]string{"key_hint"}, map[string]any{
			"key_hint": map[string]any{"type": "string", "pattern": `^[a-z][a-z0-9+.-]*:[A-Za-z0-9._~+-]+$`},
		}, nil),
		InputSchemaResolution: closedObject(
			[]string{"outcome", "outcome_known_at", "sources"},
			map[string]any{
				"outcome":          map[string]any{"oneOf": []any{map[string]any{"type": "boolean"}, stringValue(1)}},
				"outcome_known_at": ref("timestamp"),
				"recorded_at":      ref("timestamp"),
				"sources":          arrayOf(ref("evidenceSourceInput"), 1),
				"notes":            stringValue(0),
			}, nil),
		InputSchemaAnnul:   reasonInputDefinition(),
		InputSchemaDispute: reasonInputDefinition(),
		InputSchemaPublicationBuild: closedObject([]string{"file", "output"}, map[string]any{
			"file": stringValue(1), "output": stringValue(1), "dry_run": map[string]any{"type": "boolean"},
		}, nil),
		InputSchemaPublicationVerify: closedObject([]string{"file", "manifest"}, map[string]any{
			"file": stringValue(1), "manifest": stringValue(1),
			"online": map[string]any{"type": "boolean"}, "offline": map[string]any{"type": "boolean"},
			"bitcoin_core":      map[string]any{"type": "string", "format": "uri"},
			"bitcoin_auth_file": stringValue(1),
		}, map[string]any{
			"allOf": []any{
				map[string]any{"not": map[string]any{"required": []string{"online", "offline"}, "properties": map[string]any{"online": map[string]any{"const": true}, "offline": map[string]any{"const": true}}}},
				map[string]any{"if": map[string]any{"required": []string{"bitcoin_core"}}, "then": map[string]any{"required": []string{"bitcoin_auth_file"}}},
				map[string]any{"if": map[string]any{"required": []string{"bitcoin_auth_file"}}, "then": map[string]any{"required": []string{"bitcoin_core"}}},
			},
		}),
	}
}

func commonInputDefinitions() map[string]any {
	forecastFields := map[string]any{
		"id":                     ref("slug"),
		"visibility":             map[string]any{"enum": []string{"public", "sealed"}},
		"forecasted_at":          ref("timestamp"),
		"recorded_at":            ref("timestamp"),
		"value":                  ref("forecastValue"),
		"rationale":              stringValue(0),
		"key_factors":            arrayOf(stringValue(1), 0),
		"comment":                stringValue(0),
		"public_note":            stringValue(0),
		"supersedes_forecast_id": ref("slug"),
	}
	questionFields := map[string]any{
		"title":                  stringValue(1),
		"resolution_criteria":    stringValue(1),
		"created_at":             ref("timestamp"),
		"forecast_window":        ref("forecastWindow"),
		"expected_resolution_at": ref("timestamp"),
		"options":                arrayOf(ref("option"), 2),
		"unit":                   ref("unit"),
		"platform_refs":          arrayOf(ref("platformRef"), 0),
		"tags":                   arrayOf(ref("slug"), 0),
		"notes":                  stringValue(0),
		"initial_forecast":       ref("initialForecastInput"),
	}
	initialQuestionFields := cloneProperties(questionFields)
	initialQuestionFields["id"] = ref("slug")
	initialQuestionFields["type"] = map[string]any{"enum": []string{"binary", "multiple_choice", "numeric", "date"}}

	return map[string]any{
		"initialForecastInput": closedObject(
			[]string{"id", "visibility", "forecasted_at", "value"}, forecastFields,
			map[string]any{"allOf": []any{
				map[string]any{
					"if":   map[string]any{"properties": map[string]any{"visibility": map[string]any{"const": "sealed"}}, "required": []string{"visibility"}},
					"then": map[string]any{"required": []string{"rationale", "key_factors", "comment"}},
				},
			}}),
		"initialQuestionInput": questionDefinition(initialQuestionFields, []string{
			"id", "title", "type", "resolution_criteria", "forecast_window", "expected_resolution_at", "initial_forecast",
		}, true),
		"questionAddInput": questionDefinition(questionFields, []string{
			"title", "resolution_criteria", "forecast_window", "expected_resolution_at", "initial_forecast",
		}, false),
		"publicForecastInput": closedObject([]string{"forecasted_at", "value"}, map[string]any{
			"forecasted_at": ref("timestamp"), "recorded_at": ref("timestamp"), "value": ref("forecastValue"),
			"rationale": stringValue(0), "key_factors": arrayOf(stringValue(1), 0), "comment": stringValue(0),
			"public_note": stringValue(0), "supersedes_forecast_id": ref("slug"),
		}, nil),
		"sealedForecastInput": closedObject(
			[]string{"forecasted_at", "value", "rationale", "key_factors", "comment"},
			map[string]any{
				"forecasted_at": ref("timestamp"), "recorded_at": ref("timestamp"), "value": ref("forecastValue"),
				"rationale": stringValue(0), "key_factors": arrayOf(stringValue(1), 0), "comment": stringValue(0),
				"public_note": stringValue(0), "supersedes_forecast_id": ref("slug"),
			}, nil),
		"forecasterPatchInput": closedObject(nil, map[string]any{
			"kind": map[string]any{"enum": []string{"individual", "team"}}, "name": stringValue(1),
			"contact": nullable(ref("contact")), "profiles": nullable(arrayOf(ref("profile"), 0)),
			"members": nullable(arrayOf(ref("member"), 2)),
		}, map[string]any{"minProperties": 1}),
		"platformAccountPatchInput": closedObject(nil, map[string]any{
			"username": nullable(stringValue(1)), "user_id": nullable(stringValue(1)),
			"profile_url": nullable(map[string]any{"type": "string", "format": "uri"}),
		}, map[string]any{"minProperties": 1}),
		"forecastWindowPatchInput": closedObject([]string{"closes_at"}, map[string]any{"closes_at": ref("timestamp")}, nil),
		"evidenceSourceInput": closedObject([]string{"title", "url", "retrieved_at"}, map[string]any{
			"title": stringValue(1), "url": map[string]any{"type": "string", "format": "uri"},
			"retrieved_at": ref("timestamp"), "publisher": stringValue(1), "published_at": ref("timestamp"),
			"content_sha256": ref("hex32"),
		}, nil),
	}
}

func questionDefinition(properties map[string]any, required []string, hasType bool) map[string]any {
	definition := closedObject(required, properties, nil)
	if !hasType {
		return definition
	}
	definition["allOf"] = []any{
		map[string]any{"if": typeCondition("multiple_choice"), "then": map[string]any{"required": []string{"options"}, "not": map[string]any{"required": []string{"unit"}}}},
		map[string]any{"if": typeCondition("numeric"), "then": map[string]any{"required": []string{"unit"}, "not": map[string]any{"required": []string{"options"}}}},
		map[string]any{"if": map[string]any{"properties": map[string]any{"type": map[string]any{"enum": []string{"binary", "date"}}}, "required": []string{"type"}}, "then": map[string]any{"not": map[string]any{"anyOf": []any{map[string]any{"required": []string{"options"}}, map[string]any{"required": []string{"unit"}}}}}},
	}
	return definition
}

func typeCondition(value string) map[string]any {
	return map[string]any{"properties": map[string]any{"type": map[string]any{"const": value}}, "required": []string{"type"}}
}

func reasonInputDefinition() map[string]any {
	return closedObject([]string{"reason"}, map[string]any{
		"reason": stringValue(1), "recorded_at": ref("timestamp"), "sources": arrayOf(ref("evidenceSourceInput"), 0),
	}, nil)
}

func closedObject(required []string, properties map[string]any, extra map[string]any) map[string]any {
	result := map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		result["required"] = required
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
}

func ref(name string) map[string]any { return map[string]any{"$ref": "#/$defs/" + name} }

func nullable(value any) map[string]any {
	return map[string]any{"anyOf": []any{value, map[string]any{"type": "null"}}}
}

func stringValue(minimum int) map[string]any {
	return map[string]any{"type": "string", "minLength": minimum}
}

func arrayOf(items any, minimum int) map[string]any {
	result := map[string]any{"type": "array", "items": items}
	if minimum > 0 {
		result["minItems"] = minimum
	}
	return result
}

func cloneProperties(properties map[string]any) map[string]any {
	result := make(map[string]any, len(properties))
	for key, value := range properties {
		result[key] = value
	}
	return result
}
