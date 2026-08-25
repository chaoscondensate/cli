package document

import (
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
	Array  []*Value
	Object []Member
	Source SourceRef
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
