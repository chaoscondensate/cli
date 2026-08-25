package canonical

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

const maxSafeInteger = int64(9_007_199_254_740_991)

var ErrUnsupportedValue = errors.New("value is outside the supported RFC 8785 profile")

// Marshal returns RFC 8785/JCS bytes for the project's I-JSON subset. The
// profile deliberately rejects floats; exact decimals are represented as
// strings and probabilities as safe integers.
func Marshal(value any) ([]byte, error) {
	return appendValue(nil, value)
}

func appendValue(output []byte, value any) ([]byte, error) {
	switch value := value.(type) {
	case nil:
		return append(output, "null"...), nil
	case bool:
		return strconv.AppendBool(output, value), nil
	case int:
		return appendInteger(output, int64(value))
	case int8:
		return appendInteger(output, int64(value))
	case int16:
		return appendInteger(output, int64(value))
	case int32:
		return appendInteger(output, int64(value))
	case int64:
		return appendInteger(output, value)
	case uint:
		if uint64(value) > uint64(maxSafeInteger) {
			return nil, fmt.Errorf("%w: integer is outside the safe range", ErrUnsupportedValue)
		}
		return strconv.AppendUint(output, uint64(value), 10), nil
	case uint8:
		return strconv.AppendUint(output, uint64(value), 10), nil
	case uint16:
		return strconv.AppendUint(output, uint64(value), 10), nil
	case uint32:
		return strconv.AppendUint(output, uint64(value), 10), nil
	case uint64:
		if value > uint64(maxSafeInteger) {
			return nil, fmt.Errorf("%w: integer is outside the safe range", ErrUnsupportedValue)
		}
		return strconv.AppendUint(output, value, 10), nil
	case string:
		if !validIJSONString(value) {
			return nil, fmt.Errorf("%w: string is not valid I-JSON", ErrUnsupportedValue)
		}
		quoted, err := jsontext.AppendQuote(output, value)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnsupportedValue, err)
		}
		return quoted, nil
	case []any:
		output = append(output, '[')
		for index, item := range value {
			if index > 0 {
				output = append(output, ',')
			}
			var err error
			output, err = appendValue(output, item)
			if err != nil {
				return nil, err
			}
		}
		return append(output, ']'), nil
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			if !validIJSONString(key) {
				return nil, fmt.Errorf("%w: object key is not valid I-JSON", ErrUnsupportedValue)
			}
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return utf16Less(keys[i], keys[j])
		})
		output = append(output, '{')
		for index, key := range keys {
			if index > 0 {
				output = append(output, ',')
			}
			var err error
			output, err = jsontext.AppendQuote(output, key)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrUnsupportedValue, err)
			}
			output = append(output, ':')
			output, err = appendValue(output, value[key])
			if err != nil {
				return nil, err
			}
		}
		return append(output, '}'), nil
	case float32, float64:
		return nil, fmt.Errorf("%w: floating-point values are forbidden", ErrUnsupportedValue)
	default:
		return nil, fmt.Errorf("%w: unsupported type %T", ErrUnsupportedValue, value)
	}
}

func appendInteger(output []byte, value int64) ([]byte, error) {
	if value < -maxSafeInteger || value > maxSafeInteger {
		return nil, fmt.Errorf("%w: integer is outside the safe range", ErrUnsupportedValue)
	}
	return strconv.AppendInt(output, value, 10), nil
}

func validIJSONString(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if character >= 0xD800 && character <= 0xDFFF {
			return false
		}
	}
	return true
}

func utf16Less(left, right string) bool {
	leftUnits := utf16.Encode([]rune(left))
	rightUnits := utf16.Encode([]rune(right))
	limit := len(leftUnits)
	if len(rightUnits) < limit {
		limit = len(rightUnits)
	}
	for index := 0; index < limit; index++ {
		if leftUnits[index] != rightUnits[index] {
			return leftUnits[index] < rightUnits[index]
		}
	}
	return len(leftUnits) < len(rightUnits)
}
