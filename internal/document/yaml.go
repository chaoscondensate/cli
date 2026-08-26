package document

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

var yamlLinePattern = regexp.MustCompile(`(?:^| )line ([0-9]+)(?::|$)`)

type yamlParser struct {
	raw                   []byte
	starts                []int64
	limits                Limits
	nodes                 int
	aliases               int
	expanded              int
	anchors               map[string]*yaml.Node
	locations             map[string][]SourceRef
	allowTimestampScalars bool
}

func ParseYAML(r io.Reader, limits Limits) (*Document, error) {
	return parseYAML(r, limits, false)
}

// ParseYAMLWithTimestampScalars accepts YAML's !!timestamp scalar tag as a
// string while retaining the tag on Value.SourceTag. Callers must reject tags
// outside fields whose schema explicitly declares an RFC 3339 timestamp.
func ParseYAMLWithTimestampScalars(r io.Reader, limits Limits) (*Document, error) {
	return parseYAML(r, limits, true)
}

func parseYAML(r io.Reader, limits Limits, allowTimestampScalars bool) (*Document, error) {
	limits = normalizedLimits(limits)
	raw, err := readBounded(r, limits.MaxBytes)
	if err != nil {
		if errors.Is(err, errInputTooLarge) {
			return nil, parseFailure("document.too_large", "document exceeds the size limit", "", nil, []int64{0}, limits.MaxBytes, err)
		}
		return nil, fmt.Errorf("read YAML document: %w", err)
	}
	starts := lineStarts(raw)
	if !utf8.Valid(raw) {
		offset := firstInvalidUTF8(raw)
		return nil, parseFailure("document.invalid_utf8", "document is not valid UTF-8", "", raw, starts, int64(offset), nil)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var syntax yaml.Node
	if err := decoder.Decode(&syntax); err != nil {
		return nil, yamlSyntaxFailure(raw, starts, err)
	}
	if syntax.Kind != yaml.DocumentNode || len(syntax.Content) != 1 {
		return nil, parseFailure("document.syntax", "YAML document must contain exactly one root value", "", raw, starts, 0, nil)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return nil, yamlSyntaxFailure(raw, starts, err)
		}
		return nil, parseFailure("document.multiple_documents", "only one YAML document is allowed", "", raw, starts, yamlOffset(raw, starts, extra.Line, extra.Column), nil)
	}

	parser := &yamlParser{
		raw:                   raw,
		starts:                starts,
		limits:                limits,
		anchors:               make(map[string]*yaml.Node),
		locations:             make(map[string][]SourceRef),
		allowTimestampScalars: allowTimestampScalars,
	}
	if err := parser.checkPhysical(syntax.Content[0], "", 1); err != nil {
		return nil, err
	}
	root, err := parser.toValue(syntax.Content[0], "", 1, make(map[*yaml.Node]bool))
	if err != nil {
		return nil, err
	}

	return &Document{
		Format:    FormatYAML,
		Raw:       bytes.Clone(raw),
		Root:      root,
		YAMLRoot:  &syntax,
		Newlines:  newlineInfo(raw),
		Locations: parser.locations,
		lineIndex: starts,
	}, nil
}

func (p *yamlParser) checkPhysical(node *yaml.Node, pointer string, depth int) error {
	if depth > p.limits.MaxDepth {
		return p.failure("document.too_deep", "document nesting exceeds the depth limit", pointer, node, nil)
	}
	p.nodes++
	if p.nodes > p.limits.MaxNodes {
		return p.failure("document.too_many_nodes", "document exceeds the node limit", pointer, node, nil)
	}
	if node.Anchor != "" {
		if first, exists := p.anchors[node.Anchor]; exists {
			err := p.failure("document.duplicate_anchor", "duplicate YAML anchor is not allowed", pointer, node, nil)
			err.Diagnostic.Related = []RelatedLocation{{Message: "first anchor is here", Location: p.nodeSource(pointer, first)}}
			return err
		}
		p.anchors[node.Anchor] = node
	}

	switch node.Kind {
	case yaml.AliasNode:
		p.aliases++
		if p.aliases > p.limits.MaxAliases {
			return p.failure("document.alias_limit", "document exceeds the YAML alias limit", pointer, node, nil)
		}
		if node.Alias == nil {
			return p.failure("document.syntax", "YAML alias has no target", pointer, node, nil)
		}
		return nil
	case yaml.ScalarNode:
		if len(node.Value) > p.limits.MaxScalarBytes {
			return p.failure("document.scalar_too_large", "scalar exceeds the size limit", pointer, node, nil)
		}
		return p.checkScalarTag(node, pointer)
	case yaml.SequenceNode:
		if node.ShortTag() != "!!seq" {
			return p.failure("document.unsupported_tag", "YAML sequence tag is not supported", pointer, node, nil)
		}
		for index, child := range node.Content {
			if err := p.checkPhysical(child, appendPointer(pointer, strconv.Itoa(index)), depth+1); err != nil {
				return err
			}
		}
		return nil
	case yaml.MappingNode:
		if node.ShortTag() != "!!map" {
			return p.failure("document.unsupported_tag", "YAML mapping tag is not supported", pointer, node, nil)
		}
		if len(node.Content)%2 != 0 {
			return p.failure("document.syntax", "YAML mapping has an incomplete entry", pointer, node, nil)
		}
		seen := make(map[string]*yaml.Node)
		for index := 0; index < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			if key.Kind == yaml.ScalarNode && (key.Value == "<<" || key.Tag == "!!merge") {
				return p.failure("document.merge_key_not_allowed", "YAML merge keys are not allowed", appendPointer(pointer, key.Value), key, nil)
			}
			if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
				return p.failure("document.non_string_key", "YAML mapping keys must be strings", pointer, key, nil)
			}
			childPointer := appendPointer(pointer, key.Value)
			if first, exists := seen[key.Value]; exists {
				err := p.failure("document.duplicate_key", "duplicate mapping key is not allowed", childPointer, key, nil)
				err.Diagnostic.Related = []RelatedLocation{{Message: "first key is here", Location: p.nodeSource(childPointer, first)}}
				return err
			}
			seen[key.Value] = key
			if err := p.checkPhysical(key, childPointer, depth+1); err != nil {
				return err
			}
			if err := p.checkPhysical(value, childPointer, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return p.failure("document.syntax", "unsupported YAML node", pointer, node, nil)
	}
}

func (p *yamlParser) checkScalarTag(node *yaml.Node, pointer string) error {
	switch node.ShortTag() {
	case "!!str", "!!bool", "!!null", "!!int":
		return nil
	case "!!timestamp":
		if p.allowTimestampScalars {
			return nil
		}
		return p.failure("document.unsupported_tag", "YAML scalar tag is not supported", pointer, node, nil)
	case "!!float":
		return p.failure("document.float_not_allowed", "floating-point numbers are not allowed", pointer, node, nil)
	default:
		return p.failure("document.unsupported_tag", "YAML scalar tag is not supported", pointer, node, nil)
	}
}

func (p *yamlParser) toValue(node *yaml.Node, pointer string, depth int, active map[*yaml.Node]bool) (*Value, error) {
	p.expanded++
	if p.expanded > p.limits.MaxExpandedNodes {
		return nil, p.failure("document.alias_limit", "expanded YAML aliases exceed the node limit", pointer, node, nil)
	}
	if depth > p.limits.MaxDepth {
		return nil, p.failure("document.too_deep", "expanded YAML nesting exceeds the depth limit", pointer, node, nil)
	}
	if node.Kind == yaml.AliasNode {
		if active[node.Alias] {
			err := p.failure("document.alias_cycle", "YAML alias cycle is not allowed", pointer, node, nil)
			err.Diagnostic.Related = []RelatedLocation{{Message: "alias target is here", Location: p.nodeSource(pointer, node.Alias)}}
			return nil, err
		}
		active[node.Alias] = true
		value, err := p.toValue(node.Alias, pointer, depth, active)
		delete(active, node.Alias)
		if err != nil {
			return nil, err
		}
		value.Source = p.nodeSource(pointer, node)
		p.locations[pointer] = append(p.locations[pointer], value.Source)
		return value, nil
	}

	value := &Value{Source: p.nodeSource(pointer, node)}
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.ShortTag() {
		case "!!str":
			value.Kind = ValueString
			value.String = node.Value
		case "!!timestamp":
			if !p.allowTimestampScalars {
				return nil, p.failure("document.unsupported_tag", "YAML scalar tag is not supported", pointer, node, nil)
			}
			value.Kind = ValueString
			value.String = node.Value
			value.SourceTag = "!!timestamp"
		case "!!bool":
			parsed, err := strconv.ParseBool(strings.ToLower(node.Value))
			if err != nil {
				return nil, p.failure("document.syntax", "invalid YAML boolean", pointer, node, err)
			}
			value.Kind = ValueBool
			value.Bool = parsed
		case "!!null":
			value.Kind = ValueNull
		case "!!int":
			parsed, err := parseYAMLInteger(node.Value)
			if err != nil || parsed < -maxSafeInteger || parsed > maxSafeInteger {
				return nil, p.failure("document.unsafe_integer", "integer is outside the safe range", pointer, node, err)
			}
			value.Kind = ValueInt
			value.Int = parsed
		default:
			return nil, p.failure("document.unsupported_tag", "YAML scalar tag is not supported", pointer, node, nil)
		}
	case yaml.SequenceNode:
		value.Kind = ValueArray
		value.Array = make([]*Value, 0, len(node.Content))
		for index, child := range node.Content {
			converted, err := p.toValue(child, appendPointer(pointer, strconv.Itoa(index)), depth+1, active)
			if err != nil {
				return nil, err
			}
			value.Array = append(value.Array, converted)
		}
	case yaml.MappingNode:
		value.Kind = ValueObject
		value.Object = make([]Member, 0, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key, child := node.Content[index], node.Content[index+1]
			childPointer := appendPointer(pointer, key.Value)
			converted, err := p.toValue(child, childPointer, depth+1, active)
			if err != nil {
				return nil, err
			}
			value.Object = append(value.Object, Member{Key: key.Value, KeySource: p.nodeSource(childPointer, key), Value: converted})
		}
	default:
		return nil, p.failure("document.syntax", "unsupported YAML node", pointer, node, nil)
	}
	p.locations[pointer] = append(p.locations[pointer], value.Source)
	return value, nil
}

func parseYAMLInteger(input string) (int64, error) {
	cleaned := strings.ReplaceAll(input, "_", "")
	sign := ""
	if strings.HasPrefix(cleaned, "+") || strings.HasPrefix(cleaned, "-") {
		sign, cleaned = cleaned[:1], cleaned[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(cleaned, "0b"):
		base, cleaned = 2, cleaned[2:]
	case strings.HasPrefix(cleaned, "0o"):
		base, cleaned = 8, cleaned[2:]
	case strings.HasPrefix(cleaned, "0x"):
		base, cleaned = 16, cleaned[2:]
	case len(cleaned) > 1 && cleaned[0] == '0':
		base, cleaned = 8, cleaned[1:]
	}
	if cleaned == "" {
		cleaned = "0"
	}
	integer := new(big.Int)
	if _, ok := integer.SetString(sign+cleaned, base); !ok {
		return 0, fmt.Errorf("invalid YAML integer")
	}
	if !integer.IsInt64() {
		return 0, fmt.Errorf("YAML integer overflows int64")
	}
	return integer.Int64(), nil
}

func (p *yamlParser) failure(code, message, pointer string, node *yaml.Node, cause error) *ParseError {
	location := p.nodeSource(pointer, node)
	return &ParseError{Diagnostic: Diagnostic{Code: code, Message: message, Location: location}, Cause: cause}
}

func (p *yamlParser) nodeSource(pointer string, node *yaml.Node) SourceRef {
	return SourceRef{
		Pointer: pointer,
		Start:   positionAt(p.raw, p.starts, yamlOffset(p.raw, p.starts, node.Line, node.Column)),
	}
}

func yamlOffset(raw []byte, starts []int64, line, column int) int64 {
	if line < 1 || line > len(starts) {
		return 0
	}
	offset := starts[line-1]
	remaining := column - 1
	for remaining > 0 && offset < int64(len(raw)) && raw[offset] != '\r' && raw[offset] != '\n' {
		_, size := utf8.DecodeRune(raw[offset:])
		offset += int64(size)
		remaining--
	}
	return offset
}

func yamlSyntaxFailure(raw []byte, starts []int64, err error) *ParseError {
	line := 1
	match := yamlLinePattern.FindStringSubmatch(err.Error())
	if len(match) == 2 {
		if parsed, parseErr := strconv.Atoi(match[1]); parseErr == nil {
			line = parsed
		}
	}
	offset := yamlOffset(raw, starts, line, 1)
	return parseFailure("document.syntax", "invalid YAML syntax", "", raw, starts, offset, err)
}
