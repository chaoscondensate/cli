package document

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chaoscondensate/cli/internal/canonical"
	"go.yaml.in/yaml/v3"
)

type PatchKind string

const (
	PatchAdd     PatchKind = "add"
	PatchReplace PatchKind = "replace"
	PatchRemove  PatchKind = "remove"
)

type PatchOperation struct {
	Kind    PatchKind
	Pointer string
	Value   any
}

// ApplyPatch applies structural edits one at a time and reparses between them.
// Reparse keeps source offsets exact while each edit splices only the addressed
// value/member/item and leaves every unrelated source byte untouched.
func ApplyPatch(sourceDocument *Document, operations []PatchOperation) ([]byte, error) {
	if sourceDocument == nil || sourceDocument.Root == nil {
		return nil, fmt.Errorf("document has no source tree")
	}
	current := sourceDocument
	output := bytes.Clone(sourceDocument.Raw)
	for _, operation := range operations {
		patched, err := applyOnePatch(current, operation)
		if err != nil {
			return nil, err
		}
		current, err = parsePatchedBytes(patched, sourceDocument.Format)
		if err != nil {
			return nil, fmt.Errorf("patched document is invalid: %w", err)
		}
		output = patched
	}
	return output, nil
}

func applyOnePatch(sourceDocument *Document, operation PatchOperation) ([]byte, error) {
	if operation.Pointer == "" {
		return nil, fmt.Errorf("%w: root replacement is not supported", ErrUnsupportedPatch)
	}
	if operation.Kind == PatchReplace {
		if node, err := lookupValue(sourceDocument.Root, operation.Pointer); err == nil && node.Kind != ValueArray && node.Kind != ValueObject {
			if scalar, ok := scalarReplacementValue(operation.Value); ok {
				return ReplaceScalars(sourceDocument, []ScalarEdit{{Pointer: operation.Pointer, Value: scalar}})
			}
		}
	}
	var edit byteEdit
	var err error
	switch sourceDocument.Format {
	case FormatJSON:
		edit, err = jsonStructuralEdit(sourceDocument, operation)
	case FormatYAML:
		edit, err = yamlStructuralEdit(sourceDocument, operation)
	default:
		err = fmt.Errorf("%w: unknown document format", ErrUnsupportedPatch)
	}
	if err != nil {
		return nil, err
	}
	if edit.start < 0 || edit.end < edit.start || edit.end > int64(len(sourceDocument.Raw)) {
		return nil, fmt.Errorf("%w: patch source range is invalid", ErrUnsupportedPatch)
	}
	result := make([]byte, 0, len(sourceDocument.Raw)+len(edit.replacement))
	result = append(result, sourceDocument.Raw[:edit.start]...)
	result = append(result, edit.replacement...)
	result = append(result, sourceDocument.Raw[edit.end:]...)
	return result, nil
}

func jsonStructuralEdit(document *Document, operation PatchOperation) (byteEdit, error) {
	if operation.Kind == PatchReplace {
		node, err := lookupValue(document.Root, operation.Pointer)
		if err != nil {
			return byteEdit{}, err
		}
		if node.Source.End == nil {
			return byteEdit{}, fmt.Errorf("%w: JSON source range is incomplete", ErrUnsupportedPatch)
		}
		rawNode := document.Raw[node.Source.Start.Offset:node.Source.End.Offset]
		multiline := bytes.Contains(rawNode, []byte{'\n'}) || bytes.Contains(rawNode, []byte{'\r'})
		indent := jsonLineIndent(document.Raw, node.Source.Start.Offset)
		replacement, err := renderJSONFragment(operation.Value, multiline, indent, preferredNewline(document.Newlines))
		if err != nil {
			return byteEdit{}, fmt.Errorf("encode JSON replacement: %w", err)
		}
		return byteEdit{start: node.Source.Start.Offset, end: node.Source.End.Offset, replacement: replacement}, nil
	}
	parentPointer, token, err := splitPatchPointer(operation.Pointer)
	if err != nil {
		return byteEdit{}, err
	}
	parent, err := lookupValue(document.Root, parentPointer)
	if err != nil {
		return byteEdit{}, err
	}
	switch operation.Kind {
	case PatchAdd:
		return jsonAdd(document, parent, token, operation.Value)
	case PatchRemove:
		return jsonRemove(document, parent, token)
	default:
		return byteEdit{}, fmt.Errorf("%w: unknown patch operation %q", ErrUnsupportedPatch, operation.Kind)
	}
}

func jsonAdd(document *Document, parent *Value, token string, value any) (byteEdit, error) {
	if parent.Source.End == nil {
		return byteEdit{}, fmt.Errorf("%w: JSON parent source range is incomplete", ErrUnsupportedPatch)
	}
	switch parent.Kind {
	case ValueObject:
		for _, member := range parent.Object {
			if member.Key == token {
				return byteEdit{}, fmt.Errorf("%w: source pointer already exists", ErrUnsupportedPatch)
			}
		}
		key, err := canonical.Marshal(token)
		if err != nil {
			return byteEdit{}, err
		}
		closeOffset := parent.Source.End.Offset - 1
		multiline, insertOffset, indent := jsonCollectionInsertion(document.Raw, parent, closeOffset)
		replaceEnd := insertOffset
		if multiline && len(parent.Object) > 0 {
			last := parent.Object[len(parent.Object)-1].Value
			if last.Source.End == nil {
				return byteEdit{}, fmt.Errorf("%w: JSON member source range is incomplete", ErrUnsupportedPatch)
			}
			insertOffset = last.Source.End.Offset
		}
		encoded, err := renderJSONFragment(value, multiline, indent, preferredNewline(document.Newlines))
		if err != nil {
			return byteEdit{}, fmt.Errorf("encode JSON addition: %w", err)
		}
		var replacement []byte
		if len(parent.Object) == 0 {
			if multiline {
				replacement = append(replacement, indent...)
			}
		} else if multiline {
			replacement = append(replacement, ',')
			replacement = append(replacement, preferredNewline(document.Newlines)...)
			replacement = append(replacement, indent...)
		} else {
			replacement = append(replacement, ',', ' ')
		}
		replacement = append(replacement, key...)
		replacement = append(replacement, ':', ' ')
		replacement = append(replacement, encoded...)
		if multiline {
			replacement = append(replacement, preferredNewline(document.Newlines)...)
		}
		return byteEdit{start: insertOffset, end: replaceEnd, replacement: replacement}, nil
	case ValueArray:
		if token != "-" {
			return byteEdit{}, fmt.Errorf("%w: JSON array additions must use /-", ErrUnsupportedPatch)
		}
		closeOffset := parent.Source.End.Offset - 1
		multiline, insertOffset, indent := jsonCollectionInsertion(document.Raw, parent, closeOffset)
		replaceEnd := insertOffset
		if multiline && len(parent.Array) > 0 {
			last := parent.Array[len(parent.Array)-1]
			if last.Source.End == nil {
				return byteEdit{}, fmt.Errorf("%w: JSON item source range is incomplete", ErrUnsupportedPatch)
			}
			insertOffset = last.Source.End.Offset
		}
		encoded, err := renderJSONFragment(value, multiline, indent, preferredNewline(document.Newlines))
		if err != nil {
			return byteEdit{}, fmt.Errorf("encode JSON addition: %w", err)
		}
		var replacement []byte
		if len(parent.Array) > 0 {
			replacement = append(replacement, ',')
			if multiline {
				replacement = append(replacement, preferredNewline(document.Newlines)...)
				replacement = append(replacement, indent...)
			} else {
				replacement = append(replacement, ' ')
			}
		} else if multiline {
			replacement = append(replacement, indent...)
		}
		replacement = append(replacement, encoded...)
		if multiline {
			replacement = append(replacement, preferredNewline(document.Newlines)...)
		}
		return byteEdit{start: insertOffset, end: replaceEnd, replacement: replacement}, nil
	default:
		return byteEdit{}, fmt.Errorf("%w: patch parent is not a collection", ErrUnsupportedPatch)
	}
}

func renderJSONFragment(value any, multiline bool, indent, newline []byte) ([]byte, error) {
	if !multiline || isScalarReplacement(value) {
		return json.Marshal(value)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	encoded = bytes.ReplaceAll(encoded, []byte("\n"), append(append([]byte(nil), newline...), indent...))
	return encoded, nil
}

func jsonCollectionInsertion(raw []byte, parent *Value, closeOffset int64) (bool, int64, []byte) {
	lineStart := sourceLineStart(raw, closeOffset)
	if lineStart > parent.Source.Start.Offset && onlyHorizontalSpace(raw[lineStart:closeOffset]) {
		indent := append([]byte(nil), raw[lineStart:closeOffset]...)
		indent = append(indent, ' ', ' ')
		if parent.Kind == ValueObject && len(parent.Object) > 0 {
			first := parent.Object[0].KeySource.Start.Offset
			indent = append([]byte(nil), raw[sourceLineStart(raw, first):first]...)
		} else if parent.Kind == ValueArray && len(parent.Array) > 0 {
			first := parent.Array[0].Source.Start.Offset
			indent = append([]byte(nil), raw[sourceLineStart(raw, first):first]...)
		}
		return true, lineStart, indent
	}
	return false, closeOffset, nil
}

func jsonLineIndent(raw []byte, offset int64) []byte {
	start := sourceLineStart(raw, offset)
	end := start
	for end < int64(len(raw)) && (raw[end] == ' ' || raw[end] == '\t') {
		end++
	}
	return append([]byte(nil), raw[start:end]...)
}

func jsonRemove(document *Document, parent *Value, token string) (byteEdit, error) {
	switch parent.Kind {
	case ValueObject:
		index := -1
		for position, member := range parent.Object {
			if member.Key == token {
				index = position
				break
			}
		}
		if index < 0 {
			return byteEdit{}, fmt.Errorf("source pointer does not exist")
		}
		member := parent.Object[index]
		if member.Value.Source.End == nil {
			return byteEdit{}, fmt.Errorf("%w: JSON member range is incomplete", ErrUnsupportedPatch)
		}
		start, end := member.KeySource.Start.Offset, member.Value.Source.End.Offset
		if len(parent.Object) == 1 {
			return byteEdit{start: start, end: end}, nil
		}
		if index < len(parent.Object)-1 {
			end = parent.Object[index+1].KeySource.Start.Offset
		} else {
			previous := parent.Object[index-1].Value
			if previous.Source.End == nil {
				return byteEdit{}, fmt.Errorf("%w: JSON member range is incomplete", ErrUnsupportedPatch)
			}
			start = previous.Source.End.Offset
		}
		return byteEdit{start: start, end: end}, nil
	case ValueArray:
		index, err := patchArrayIndex(token, len(parent.Array))
		if err != nil {
			return byteEdit{}, err
		}
		item := parent.Array[index]
		if item.Source.End == nil {
			return byteEdit{}, fmt.Errorf("%w: JSON item range is incomplete", ErrUnsupportedPatch)
		}
		start, end := item.Source.Start.Offset, item.Source.End.Offset
		if len(parent.Array) == 1 {
			return byteEdit{start: start, end: end}, nil
		}
		if index < len(parent.Array)-1 {
			end = parent.Array[index+1].Source.Start.Offset
		} else {
			previous := parent.Array[index-1]
			start = previous.Source.End.Offset
		}
		return byteEdit{start: start, end: end}, nil
	default:
		return byteEdit{}, fmt.Errorf("%w: patch parent is not a collection", ErrUnsupportedPatch)
	}
}

func yamlStructuralEdit(document *Document, operation PatchOperation) (byteEdit, error) {
	if operation.Kind == PatchReplace {
		node, err := lookupYAMLNode(document.YAMLRoot, operation.Pointer)
		if err != nil {
			return byteEdit{}, err
		}
		start := yamlOffset(document.Raw, document.lineIndex, node.Line, node.Column)
		start = yamlStructuralReplacementStart(document.Raw, start)
		end, err := yamlNodeEnd(document, node)
		if err != nil {
			return byteEdit{}, err
		}
		replacement, err := renderYAMLValue(operation.Value, node, document, start)
		if err != nil {
			return byteEdit{}, err
		}
		return byteEdit{start: start, end: end, replacement: replacement}, nil
	}
	parentPointer, token, err := splitPatchPointer(operation.Pointer)
	if err != nil {
		return byteEdit{}, err
	}
	parent, err := lookupYAMLNode(document.YAMLRoot, parentPointer)
	if err != nil {
		return byteEdit{}, err
	}
	switch operation.Kind {
	case PatchAdd:
		return yamlAdd(document, parent, token, operation.Value)
	case PatchRemove:
		return yamlRemove(document, parent, token)
	default:
		return byteEdit{}, fmt.Errorf("%w: unknown patch operation %q", ErrUnsupportedPatch, operation.Kind)
	}
}

func yamlAdd(document *Document, parent *yaml.Node, token string, value any) (byteEdit, error) {
	if parent.Style&yaml.FlowStyle != 0 {
		return yamlFlowAdd(document, parent, token, value)
	}
	end, err := yamlNodeEnd(document, parent)
	if err != nil {
		return byteEdit{}, err
	}
	insert := lineEnd(document.Raw, end)
	insert = skipNewline(document.Raw, insert)
	newline := preferredNewline(document.Newlines)
	needsLeadingNewline := end == int64(len(document.Raw)) && len(document.Raw) > 0 && document.Raw[len(document.Raw)-1] != '\n' && document.Raw[len(document.Raw)-1] != '\r'
	indent := strings.Repeat(" ", yamlCollectionIndent(parent))
	var rendered []byte
	switch parent.Kind {
	case yaml.MappingNode:
		for index := 0; index < len(parent.Content); index += 2 {
			if parent.Content[index].Value == token {
				return byteEdit{}, fmt.Errorf("%w: source pointer already exists", ErrUnsupportedPatch)
			}
		}
		rendered, err = renderYAMLAddedMapping(indent, token, value, newline)
	case yaml.SequenceNode:
		if token != "-" {
			return byteEdit{}, fmt.Errorf("%w: YAML sequence additions must use /-", ErrUnsupportedPatch)
		}
		rendered, err = renderYAMLAddedSequence(indent, value, newline)
	default:
		return byteEdit{}, fmt.Errorf("%w: patch parent is not a collection", ErrUnsupportedPatch)
	}
	if err != nil {
		return byteEdit{}, err
	}
	if needsLeadingNewline {
		rendered = append(append([]byte(nil), newline...), rendered...)
	}
	return byteEdit{start: insert, end: insert, replacement: rendered}, nil
}

func yamlFlowAdd(document *Document, parent *yaml.Node, token string, value any) (byteEdit, error) {
	updated := cloneYAMLNode(parent)
	valueNode, err := encodeYAMLNode(value)
	if err != nil {
		return byteEdit{}, err
	}
	switch updated.Kind {
	case yaml.MappingNode:
		for index := 0; index < len(updated.Content); index += 2 {
			if updated.Content[index].Value == token {
				return byteEdit{}, fmt.Errorf("%w: source pointer already exists", ErrUnsupportedPatch)
			}
		}
		updated.Content = append(updated.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: token},
			valueNode,
		)
	case yaml.SequenceNode:
		if token != "-" {
			return byteEdit{}, fmt.Errorf("%w: YAML sequence additions must use /-", ErrUnsupportedPatch)
		}
		updated.Content = append(updated.Content, valueNode)
	default:
		return byteEdit{}, fmt.Errorf("%w: patch parent is not a collection", ErrUnsupportedPatch)
	}
	start := yamlOffset(document.Raw, document.lineIndex, parent.Line, parent.Column)
	start = yamlStructuralReplacementStart(document.Raw, start)
	end, err := yamlNodeEnd(document, parent)
	if err != nil {
		return byteEdit{}, err
	}
	replacement, err := renderYAMLNodeReplacement(updated, parent, document, start)
	if err != nil {
		return byteEdit{}, err
	}
	return byteEdit{start: start, end: end, replacement: replacement}, nil
}

func yamlRemove(document *Document, parent *yaml.Node, token string) (byteEdit, error) {
	if parent.Style&yaml.FlowStyle != 0 {
		return yamlFlowRemove(document, parent, token)
	}
	var node *yaml.Node
	var start int64
	switch parent.Kind {
	case yaml.MappingNode:
		for index := 0; index < len(parent.Content); index += 2 {
			if parent.Content[index].Value == token {
				node = parent.Content[index+1]
				start = sourceLineStart(document.Raw, yamlOffset(document.Raw, document.lineIndex, parent.Content[index].Line, parent.Content[index].Column))
				break
			}
		}
	case yaml.SequenceNode:
		index, err := patchArrayIndex(token, len(parent.Content))
		if err != nil {
			return byteEdit{}, err
		}
		node = parent.Content[index]
		start = sourceLineStart(document.Raw, yamlOffset(document.Raw, document.lineIndex, node.Line, node.Column))
	default:
		return byteEdit{}, fmt.Errorf("%w: patch parent is not a collection", ErrUnsupportedPatch)
	}
	if node == nil {
		return byteEdit{}, fmt.Errorf("source pointer does not exist")
	}
	end, err := yamlNodeEnd(document, node)
	if err != nil {
		return byteEdit{}, err
	}
	end = skipNewline(document.Raw, lineEnd(document.Raw, end))
	return byteEdit{start: start, end: end}, nil
}

func yamlFlowRemove(document *Document, parent *yaml.Node, token string) (byteEdit, error) {
	var nodes []*yaml.Node
	var index int
	switch parent.Kind {
	case yaml.MappingNode:
		index = -1
		for position := 0; position < len(parent.Content); position += 2 {
			if parent.Content[position].Value == token {
				index = position / 2
				break
			}
		}
		if index < 0 {
			return byteEdit{}, fmt.Errorf("source pointer does not exist")
		}
		for position := 0; position < len(parent.Content); position += 2 {
			nodes = append(nodes, parent.Content[position])
		}
	case yaml.SequenceNode:
		parsed, err := patchArrayIndex(token, len(parent.Content))
		if err != nil {
			return byteEdit{}, err
		}
		index = parsed
		nodes = parent.Content
	default:
		return byteEdit{}, fmt.Errorf("%w: patch parent is not a collection", ErrUnsupportedPatch)
	}
	start := yamlOffset(document.Raw, document.lineIndex, nodes[index].Line, nodes[index].Column)
	var valueNode *yaml.Node
	if parent.Kind == yaml.MappingNode {
		valueNode = parent.Content[index*2+1]
	} else {
		valueNode = nodes[index]
	}
	end, err := yamlNodeEnd(document, valueNode)
	if err != nil {
		return byteEdit{}, err
	}
	if len(nodes) == 1 {
		return byteEdit{start: start, end: end}, nil
	}
	if index < len(nodes)-1 {
		end = yamlOffset(document.Raw, document.lineIndex, nodes[index+1].Line, nodes[index+1].Column)
	} else {
		previous := nodes[index-1]
		if parent.Kind == yaml.MappingNode {
			previous = parent.Content[(index-1)*2+1]
		}
		previousEnd, previousErr := yamlNodeEnd(document, previous)
		if previousErr != nil {
			return byteEdit{}, previousErr
		}
		start = previousEnd
	}
	return byteEdit{start: start, end: end}, nil
}

func yamlNodeEnd(document *Document, node *yaml.Node) (int64, error) {
	start := yamlOffset(document.Raw, document.lineIndex, node.Line, node.Column)
	if node.Style&yaml.FlowStyle != 0 && (node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode) {
		open, close := byte('{'), byte('}')
		if node.Kind == yaml.SequenceNode {
			open, close = '[', ']'
		}
		return flowCollectionEnd(document.Raw, start, open, close)
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return yamlScalarEnd(document.Raw, start, node)
	case yaml.MappingNode, yaml.SequenceNode:
		if len(node.Content) == 0 {
			return yamlScalarEnd(document.Raw, start, node)
		}
		return yamlNodeEnd(document, node.Content[len(node.Content)-1])
	default:
		return 0, fmt.Errorf("%w: unsupported YAML node", ErrUnsupportedPatch)
	}
}

func flowCollectionEnd(raw []byte, start int64, open, close byte) (int64, error) {
	depth := 0
	quote := byte(0)
	escaped := false
	for offset := start; offset < int64(len(raw)); offset++ {
		current := raw[offset]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if quote == '"' && current == '\\' {
				escaped = true
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			continue
		}
		if current == open {
			depth++
		} else if current == close {
			depth--
			if depth == 0 {
				return offset + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("%w: unterminated YAML flow collection", ErrUnsupportedPatch)
}

func renderYAMLValue(value any, existing *yaml.Node, document *Document, start int64) ([]byte, error) {
	node, err := encodeYAMLNode(value)
	if err != nil {
		return nil, err
	}
	return renderYAMLNodeReplacement(node, existing, document, start)
}

func renderYAMLAddedMapping(indent, key string, value any, newline []byte) ([]byte, error) {
	keyBytes, err := canonical.Marshal(key)
	if err != nil {
		return nil, err
	}
	keyText := string(keyBytes)
	if safePlainYAML(key) {
		keyText = key
	}
	return renderYAMLAdded(indent, keyText+":", value, newline)
}

func renderYAMLAddedSequence(indent string, value any, newline []byte) ([]byte, error) {
	return renderYAMLAdded(indent, "-", value, newline)
}

func renderYAMLAdded(indent, prefix string, value any, newline []byte) ([]byte, error) {
	if !isScalarReplacement(value) {
		encoded, err := renderYAMLBlockValue(value)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(encoded), "\n")
		var output strings.Builder
		if prefix == "-" {
			output.WriteString(indent)
			output.WriteString("- ")
			output.WriteString(lines[0])
			output.Write(newline)
			for _, line := range lines[1:] {
				output.WriteString(indent)
				output.WriteString("  ")
				output.WriteString(line)
				output.Write(newline)
			}
			return []byte(output.String()), nil
		}
		output.WriteString(indent)
		output.WriteString(prefix)
		output.Write(newline)
		for _, line := range lines {
			output.WriteString(indent)
			output.WriteString("  ")
			output.WriteString(line)
			output.Write(newline)
		}
		return []byte(output.String()), nil
	}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	encoded = bytes.TrimSuffix(encoded, []byte("\n"))
	lines := strings.Split(string(encoded), "\n")
	var output strings.Builder
	output.WriteString(indent)
	output.WriteString(prefix)
	if len(lines) == 1 {
		output.WriteByte(' ')
		output.WriteString(lines[0])
		output.Write(newline)
		return []byte(output.String()), nil
	}
	output.Write(newline)
	for _, line := range lines {
		output.WriteString(indent)
		output.WriteString("  ")
		output.WriteString(line)
		output.Write(newline)
	}
	return []byte(output.String()), nil
}

func renderYAMLBlockValue(value any) ([]byte, error) {
	node, err := encodeYAMLNode(value)
	if err != nil {
		return nil, err
	}
	return renderYAMLBlockNode(node)
}

func encodeYAMLNode(value any) (*yaml.Node, error) {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return nil, fmt.Errorf("encode YAML value: %w", err)
	}
	normalizeYAMLCollectionStyle(&node)
	return &node, nil
}

func renderYAMLBlockNode(node *yaml.Node) ([]byte, error) {
	normalizeYAMLCollectionStyle(node)
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return nil, fmt.Errorf("encode YAML value: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("finish YAML value: %w", err)
	}
	return bytes.TrimSuffix(output.Bytes(), []byte("\n")), nil
}

func renderYAMLNodeReplacement(node, existing *yaml.Node, document *Document, start int64) ([]byte, error) {
	encoded, err := renderYAMLBlockNode(node)
	if err != nil {
		return nil, err
	}
	newline := string(preferredNewline(document.Newlines))
	lineStart := sourceLineStart(document.Raw, start)
	prefix := string(document.Raw[lineStart:start])
	leading := prefix[:len(prefix)-len(strings.TrimLeft(prefix, " \t"))]
	trimmed := strings.TrimSpace(prefix)
	if (node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode) && len(node.Content) > 0 && strings.HasSuffix(trimmed, ":") {
		indent := leading + "  "
		return []byte(newline + indent + strings.ReplaceAll(string(encoded), "\n", newline+indent)), nil
	}
	indent := ""
	if existing != nil {
		indent = strings.Repeat(" ", max(existing.Column-1, 0))
	}
	return []byte(strings.ReplaceAll(string(encoded), "\n", newline+indent)), nil
}

func normalizeYAMLCollectionStyle(node *yaml.Node) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode {
		if len(node.Content) == 0 {
			node.Style |= yaml.FlowStyle
		} else {
			node.Style &^= yaml.FlowStyle
		}
	}
	for _, child := range node.Content {
		normalizeYAMLCollectionStyle(child)
	}
}

func cloneYAMLNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Content = make([]*yaml.Node, len(node.Content))
	for index, child := range node.Content {
		clone.Content[index] = cloneYAMLNode(child)
	}
	return &clone
}

func yamlStructuralReplacementStart(raw []byte, start int64) int64 {
	lineStart := sourceLineStart(raw, start)
	prefix := raw[lineStart:start]
	trimmed := bytes.TrimRight(prefix, " \t")
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' {
		return lineStart + int64(len(trimmed))
	}
	return start
}

func yamlCollectionIndent(parent *yaml.Node) int {
	if parent.Kind == yaml.MappingNode && len(parent.Content) > 0 {
		return parent.Content[0].Column - 1
	}
	if parent.Kind == yaml.SequenceNode && len(parent.Content) > 0 {
		value := parent.Content[0].Column - 3
		if value >= 0 {
			return value
		}
	}
	if parent.Column > 0 {
		return parent.Column - 1
	}
	return 0
}

func splitPatchPointer(pointer string) (string, string, error) {
	tokens := pointerTokens(pointer)
	if len(tokens) == 0 || tokens[0] == "\x00invalid-pointer" {
		return "", "", fmt.Errorf("%w: invalid patch pointer %q", ErrUnsupportedPatch, pointer)
	}
	lastSlash := strings.LastIndex(pointer, "/")
	parent := pointer[:lastSlash]
	return parent, tokens[len(tokens)-1], nil
}

func patchArrayIndex(token string, length int) (int, error) {
	index, err := strconv.Atoi(token)
	if err != nil || index < 0 || index >= length {
		return 0, fmt.Errorf("source pointer does not exist")
	}
	return index, nil
}

func parsePatchedBytes(raw []byte, format Format) (*Document, error) {
	switch format {
	case FormatJSON:
		return ParseJSON(bytes.NewReader(raw), DefaultLimits)
	case FormatYAML:
		return ParseYAML(bytes.NewReader(raw), DefaultLimits)
	default:
		return nil, fmt.Errorf("unknown document format")
	}
}

func sourceLineStart(raw []byte, offset int64) int64 {
	for offset > 0 && raw[offset-1] != '\n' && raw[offset-1] != '\r' {
		offset--
	}
	return offset
}

func onlyHorizontalSpace(raw []byte) bool {
	for _, value := range raw {
		if value != ' ' && value != '\t' {
			return false
		}
	}
	return true
}

func preferredNewline(info NewlineInfo) []byte {
	if info.CRLF > info.LF {
		return []byte("\r\n")
	}
	return []byte("\n")
}
