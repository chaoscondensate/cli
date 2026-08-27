package document

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.yaml.in/yaml/v3"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Limits bounds parser work for files, stdin, and MCP inputs alike.
type Limits struct {
	MaxBytes         int64
	MaxDepth         int
	MaxNodes         int
	MaxScalarBytes   int
	MaxAliases       int
	MaxExpandedNodes int
}

var DefaultLimits = Limits{
	MaxBytes:         32 << 20,
	MaxDepth:         64,
	MaxNodes:         500_000,
	MaxScalarBytes:   1 << 20,
	MaxAliases:       64,
	MaxExpandedNodes: 1_000_000,
}

type Position struct {
	Offset int64 `json:"offset"`
	Line   int   `json:"line"`
	Column int   `json:"column"`
}

type SourceRef struct {
	Pointer string    `json:"pointer"`
	Start   Position  `json:"start"`
	End     *Position `json:"end,omitempty"`
}

// MarshalJSON keeps a useful pointer even when a source span cannot be
// resolved. Zero-valued positions are internal sentinels and must never be
// exposed as fabricated line 0 / column 0 diagnostics.
func (s SourceRef) MarshalJSON() ([]byte, error) {
	type publicSourceRef struct {
		Pointer string    `json:"pointer"`
		Start   *Position `json:"start,omitempty"`
		End     *Position `json:"end,omitempty"`
	}
	var start *Position
	if s.Start.Line > 0 && s.Start.Column > 0 {
		value := s.Start
		start = &value
	}
	end := s.End
	if end != nil && (end.Line <= 0 || end.Column <= 0) {
		end = nil
	}
	return json.Marshal(publicSourceRef{Pointer: s.Pointer, Start: start, End: end})
}

type RelatedLocation struct {
	Message  string    `json:"message"`
	Location SourceRef `json:"location"`
}

type Diagnostic struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Location SourceRef         `json:"location"`
	Related  []RelatedLocation `json:"related,omitempty"`
}

type ParseError struct {
	Diagnostic Diagnostic
	Cause      error
}

func (e *ParseError) Error() string {
	if e.Diagnostic.Location.Start.Line <= 0 || e.Diagnostic.Location.Start.Column <= 0 {
		return e.Diagnostic.Message
	}
	return fmt.Sprintf("%s at line %d, column %d", e.Diagnostic.Message, e.Diagnostic.Location.Start.Line, e.Diagnostic.Location.Start.Column)
}

func (e *ParseError) Unwrap() error { return e.Cause }

type ValueKind string

const (
	ValueNull   ValueKind = "null"
	ValueBool   ValueKind = "bool"
	ValueInt    ValueKind = "integer"
	ValueString ValueKind = "string"
	ValueArray  ValueKind = "array"
	ValueObject ValueKind = "object"
)

// Value is a format-independent JSON-compatible tree. Numbers are always safe
// integers; floats never enter this representation.
type Value struct {
	Kind   ValueKind
	Bool   bool
	Int    int64
	String string
	// SourceTag retains a safe parser-level scalar tag when an opt-in parser
	// needs schema-directed normalization. It is never part of Any().
	SourceTag string
	Array     []*Value
	Object    []Member
	Source    SourceRef
}

type Member struct {
	Key       string
	KeySource SourceRef
	Value     *Value
}

// Any returns a fresh JSON-compatible representation suitable for schema
// validation. Object order and source presentation remain available on Value.
func (v *Value) Any() any {
	if v == nil {
		return nil
	}
	switch v.Kind {
	case ValueNull:
		return nil
	case ValueBool:
		return v.Bool
	case ValueInt:
		return v.Int
	case ValueString:
		return v.String
	case ValueArray:
		out := make([]any, len(v.Array))
		for index, child := range v.Array {
			out[index] = child.Any()
		}
		return out
	case ValueObject:
		out := make(map[string]any, len(v.Object))
		for _, member := range v.Object {
			out[member.Key] = member.Value.Any()
		}
		return out
	default:
		return nil
	}
}

// OrderedValue retains the source object's member order while allowing both
// JSON and YAML encoders to render a newly inserted business object. This is
// intentionally distinct from Any(), whose maps are suitable for validation
// but lose the semantic field order declared by the originating Go struct.
type OrderedValue struct {
	value *Value
}

func Ordered(value *Value) OrderedValue { return OrderedValue{value: value} }

func (o OrderedValue) MarshalJSON() ([]byte, error) {
	var output bytes.Buffer
	if err := writeOrderedJSON(&output, o.value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func writeOrderedJSON(output *bytes.Buffer, value *Value) error {
	if value == nil || value.Kind == ValueNull {
		output.WriteString("null")
		return nil
	}
	switch value.Kind {
	case ValueBool:
		if value.Bool {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
	case ValueInt:
		encoded, _ := json.Marshal(value.Int)
		output.Write(encoded)
	case ValueString:
		encoded, err := json.Marshal(value.String)
		if err != nil {
			return err
		}
		output.Write(encoded)
	case ValueArray:
		output.WriteByte('[')
		for index, child := range value.Array {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeOrderedJSON(output, child); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case ValueObject:
		output.WriteByte('{')
		for index, member := range value.Object {
			if index > 0 {
				output.WriteByte(',')
			}
			encoded, err := json.Marshal(member.Key)
			if err != nil {
				return err
			}
			output.Write(encoded)
			output.WriteByte(':')
			if err := writeOrderedJSON(output, member.Value); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported ordered value kind %q", value.Kind)
	}
	return nil
}

func (o OrderedValue) MarshalYAML() (any, error) {
	return orderedYAMLNode(o.value), nil
}

func orderedYAMLNode(value *Value) *yaml.Node {
	if value == nil || value.Kind == ValueNull {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}
	switch value.Kind {
	case ValueBool:
		text := "false"
		if value.Bool {
			text = "true"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: text}
	case ValueInt:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", value.Int)}
	case ValueString:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value.String}
	case ValueArray:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, child := range value.Array {
			node.Content = append(node.Content, orderedYAMLNode(child))
		}
		return node
	case ValueObject:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, member := range value.Object {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: member.Key},
				orderedYAMLNode(member.Value),
			)
		}
		return node
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}
}

type NewlineInfo struct {
	LF           int  `json:"lf"`
	CRLF         int  `json:"crlf"`
	LoneCR       int  `json:"lone_cr"`
	Mixed        bool `json:"mixed"`
	FinalNewline bool `json:"final_newline"`
}

// Document retains immutable source bytes and format-specific syntax state so
// later patches do not need to re-encode the entire file.
type Document struct {
	Format    Format
	Raw       []byte
	Root      *Value
	YAMLRoot  *yaml.Node
	Newlines  NewlineInfo
	Locations map[string][]SourceRef
	lineIndex []int64
}

func (d *Document) Position(offset int64) Position {
	return positionAt(d.Raw, d.lineIndex, offset)
}
