package document

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxSafeInteger = int64(9_007_199_254_740_991)

type jsonParser struct {
	raw       []byte
	starts    []int64
	limits    Limits
	decoder   *jsontext.Decoder
	nodes     int
	locations map[string][]SourceRef
}

func ParseJSON(r io.Reader, limits Limits) (*Document, error) {
	limits = normalizedLimits(limits)
	raw, err := readBounded(r, limits.MaxBytes)
	if err != nil {
		if errors.Is(err, errInputTooLarge) {
			return nil, parseFailure("document.too_large", "document exceeds the size limit", "", nil, []int64{0}, limits.MaxBytes, err)
		}
		return nil, fmt.Errorf("read JSON document: %w", err)
	}
	starts := lineStarts(raw)
	if !utf8.Valid(raw) {
		offset := firstInvalidUTF8(raw)
		return nil, parseFailure("document.invalid_utf8", "document is not valid UTF-8", "", raw, starts, int64(offset), nil)
	}

	parser := &jsonParser{
		raw:    raw,
		starts: starts,
		limits: limits,
		decoder: jsontext.NewDecoder(bytes.NewReader(raw),
			jsontext.AllowDuplicateNames(true),
			jsontext.AllowInvalidUTF8(false),
		),
		locations: make(map[string][]SourceRef),
	}
	root, err := parser.parseValue("", 1)
	if err != nil {
		return nil, err
	}
	if _, err := parser.decoder.ReadToken(); err != io.EOF {
		if err == nil {
			return nil, parser.failure("document.syntax", "only one top-level JSON value is allowed", "", parser.decoder.InputOffset(), nil)
		}
		return nil, parser.syntaxFailure(err)
	}

	return &Document{
		Format:    FormatJSON,
		Raw:       bytes.Clone(raw),
		Root:      root,
		Newlines:  newlineInfo(raw),
		Locations: parser.locations,
		lineIndex: starts,
	}, nil
}

func (p *jsonParser) parseValue(pointer string, depth int) (*Value, error) {
	if depth > p.limits.MaxDepth {
		return nil, p.failure("document.too_deep", "document nesting exceeds the depth limit", pointer, p.decoder.InputOffset(), nil)
	}
	before := p.decoder.InputOffset()
	token, err := p.decoder.ReadToken()
	if err != nil {
		return nil, p.syntaxFailure(err)
	}
	end := p.decoder.InputOffset()
	start := tokenStart(p.raw, before, end)
	if err := p.addNode(pointer, start); err != nil {
		return nil, err
	}

	node := &Value{Source: source(pointer, p.raw, p.starts, start, end)}
	switch token.Kind() {
	case jsontext.KindNull:
		node.Kind = ValueNull
	case jsontext.KindFalse, jsontext.KindTrue:
		node.Kind = ValueBool
		node.Bool = token.Bool()
	case jsontext.KindString:
		if end-start > int64(p.limits.MaxScalarBytes) {
			return nil, p.failure("document.scalar_too_large", "string exceeds the scalar size limit", pointer, start, nil)
		}
		node.Kind = ValueString
		node.String = token.String()
	case jsontext.KindNumber:
		number := token.String()
		if strings.ContainsAny(number, ".eE") {
			return nil, p.failure("document.float_not_allowed", "floating-point numbers are not allowed", pointer, start, nil)
		}
		integer, err := strconv.ParseInt(number, 10, 64)
		if err != nil || integer < -maxSafeInteger || integer > maxSafeInteger {
			return nil, p.failure("document.unsafe_integer", "integer is outside the safe range", pointer, start, err)
		}
		node.Kind = ValueInt
		node.Int = integer
	case jsontext.KindBeginObject:
		node.Kind = ValueObject
		members, closeEnd, err := p.parseObject(pointer, depth)
		if err != nil {
			return nil, err
		}
		node.Object = members
		node.Source = source(pointer, p.raw, p.starts, start, closeEnd)
	case jsontext.KindBeginArray:
		node.Kind = ValueArray
		values, closeEnd, err := p.parseArray(pointer, depth)
		if err != nil {
			return nil, err
		}
		node.Array = values
		node.Source = source(pointer, p.raw, p.starts, start, closeEnd)
	default:
		return nil, p.failure("document.syntax", "expected a JSON value", pointer, start, nil)
	}
	p.locations[pointer] = append(p.locations[pointer], node.Source)
	return node, nil
}

func (p *jsonParser) parseObject(pointer string, depth int) ([]Member, int64, error) {
	seen := make(map[string]SourceRef)
	var members []Member
	for p.decoder.PeekKind() != jsontext.KindEndObject {
		before := p.decoder.InputOffset()
		keyToken, err := p.decoder.ReadToken()
		if err != nil {
			return nil, 0, p.syntaxFailure(err)
		}
		end := p.decoder.InputOffset()
		start := tokenStart(p.raw, before, end)
		if keyToken.Kind() != jsontext.KindString {
			return nil, 0, p.failure("document.syntax", "object key must be a string", pointer, start, nil)
		}
		key := keyToken.String()
		childPointer := appendPointer(pointer, key)
		keySource := source(childPointer, p.raw, p.starts, start, end)
		if err := p.addNode(childPointer, start); err != nil {
			return nil, 0, err
		}
		if end-start > int64(p.limits.MaxScalarBytes) {
			return nil, 0, p.failure("document.scalar_too_large", "object key exceeds the scalar size limit", childPointer, start, nil)
		}
		if first, exists := seen[key]; exists {
			err := p.failure("document.duplicate_key", "duplicate object key is not allowed", childPointer, start, nil)
			err.Diagnostic.Related = []RelatedLocation{{Message: "first key is here", Location: first}}
			return nil, 0, err
		}
		seen[key] = keySource
		value, err := p.parseValue(childPointer, depth+1)
		if err != nil {
			return nil, 0, err
		}
		members = append(members, Member{Key: key, KeySource: keySource, Value: value})
	}
	_, err := p.decoder.ReadToken()
	if err != nil {
		return nil, 0, p.syntaxFailure(err)
	}
	return members, p.decoder.InputOffset(), nil
}

func (p *jsonParser) parseArray(pointer string, depth int) ([]*Value, int64, error) {
	var values []*Value
	for index := 0; p.decoder.PeekKind() != jsontext.KindEndArray; index++ {
		value, err := p.parseValue(appendPointer(pointer, strconv.Itoa(index)), depth+1)
		if err != nil {
			return nil, 0, err
		}
		values = append(values, value)
	}
	_, err := p.decoder.ReadToken()
	if err != nil {
		return nil, 0, p.syntaxFailure(err)
	}
	return values, p.decoder.InputOffset(), nil
}

func (p *jsonParser) addNode(pointer string, offset int64) error {
	p.nodes++
	if p.nodes > p.limits.MaxNodes {
		return p.failure("document.too_many_nodes", "document exceeds the node limit", pointer, offset, nil)
	}
	return nil
}

func (p *jsonParser) failure(code, message, pointer string, offset int64, cause error) *ParseError {
	return parseFailure(code, message, pointer, p.raw, p.starts, offset, cause)
}

func (p *jsonParser) syntaxFailure(err error) *ParseError {
	var syntaxErr *jsontext.SyntacticError
	if errors.As(err, &syntaxErr) {
		return p.failure("document.syntax", "invalid JSON syntax", string(syntaxErr.JSONPointer), syntaxErr.ByteOffset, err)
	}
	return p.failure("document.syntax", "invalid JSON syntax", "", p.decoder.InputOffset(), err)
}

func tokenStart(raw []byte, before, end int64) int64 {
	for before < end {
		switch raw[before] {
		case ' ', '\t', '\r', '\n', ',', ':':
			before++
		default:
			return before
		}
	}
	return before
}

func appendPointer(parent, token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return parent + "/" + token
}

func firstInvalidUTF8(raw []byte) int {
	for offset := 0; offset < len(raw); {
		_, size := utf8.DecodeRune(raw[offset:])
		if size == 1 && raw[offset] >= utf8.RuneSelf {
			return offset
		}
		offset += size
	}
	return len(raw)
}
