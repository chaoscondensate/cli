package document

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"unicode/utf8"
)

var errInputTooLarge = errors.New("document exceeds the configured byte limit")

func normalizedLimits(limits Limits) Limits {
	defaults := DefaultLimits
	if limits.MaxBytes <= 0 {
		limits.MaxBytes = defaults.MaxBytes
	}
	if limits.MaxDepth <= 0 {
		limits.MaxDepth = defaults.MaxDepth
	}
	if limits.MaxNodes <= 0 {
		limits.MaxNodes = defaults.MaxNodes
	}
	if limits.MaxScalarBytes <= 0 {
		limits.MaxScalarBytes = defaults.MaxScalarBytes
	}
	if limits.MaxAliases <= 0 {
		limits.MaxAliases = defaults.MaxAliases
	}
	if limits.MaxExpandedNodes <= 0 {
		limits.MaxExpandedNodes = defaults.MaxExpandedNodes
	}
	return limits
}

func readBounded(r io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errInputTooLarge
	}
	return data, nil
}

func lineStarts(raw []byte) []int64 {
	starts := []int64{0}
	for i, b := range raw {
		if b == '\n' {
			starts = append(starts, int64(i+1))
		}
	}
	return starts
}

func positionAt(raw []byte, starts []int64, offset int64) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(raw)) {
		offset = int64(len(raw))
	}
	line := sort.Search(len(starts), func(i int) bool { return starts[i] > offset })
	if line == 0 {
		line = 1
	}
	lineStart := starts[line-1]
	column := utf8.RuneCount(raw[lineStart:offset]) + 1
	return Position{Offset: offset, Line: line, Column: column}
}

func newlineInfo(raw []byte) NewlineInfo {
	var info NewlineInfo
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '\r':
			if i+1 < len(raw) && raw[i+1] == '\n' {
				info.CRLF++
				i++
			} else {
				info.LoneCR++
			}
		case '\n':
			info.LF++
		}
	}
	kinds := 0
	if info.LF > 0 {
		kinds++
	}
	if info.CRLF > 0 {
		kinds++
	}
	if info.LoneCR > 0 {
		kinds++
	}
	info.Mixed = kinds > 1
	info.FinalNewline = bytes.HasSuffix(raw, []byte("\n")) || bytes.HasSuffix(raw, []byte("\r"))
	return info
}

func source(pointer string, raw []byte, starts []int64, start, end int64) SourceRef {
	endPosition := positionAt(raw, starts, end)
	return SourceRef{Pointer: pointer, Start: positionAt(raw, starts, start), End: &endPosition}
}

func parseFailure(code, message, pointer string, raw []byte, starts []int64, offset int64, cause error) *ParseError {
	return &ParseError{
		Diagnostic: Diagnostic{
			Code:     code,
			Message:  message,
			Location: source(pointer, raw, starts, offset, offset),
		},
		Cause: cause,
	}
}
