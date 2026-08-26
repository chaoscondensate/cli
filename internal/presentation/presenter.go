// Package presentation renders transport-safe CLI results and errors.
package presentation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"golang.org/x/term"
)

type Mode string

const (
	ModeHuman Mode = "human"
	ModePlain Mode = "plain"
	ModeJSON  Mode = "json"
	ModeQuiet Mode = "quiet"
)

type Options struct {
	JSON      bool
	Plain     bool
	Quiet     bool
	Verbose   bool
	NoColor   bool
	StdoutTTY *bool
	StderrTTY *bool
	LookupEnv func(string) (string, bool)
}

type Presenter struct {
	stdout      io.Writer
	stderr      io.Writer
	mode        Mode
	verbose     bool
	color       bool
	stdoutIsTTY bool
	stderrIsTTY bool
}

type ResultEnvelope struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorEnvelope struct {
	OK      bool           `json:"ok"`
	Code    app.ErrorCode  `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func New(stdout, stderr io.Writer, options Options) *Presenter {
	stdoutTTY := terminalWriter(stdout)
	stderrTTY := terminalWriter(stderr)
	if options.StdoutTTY != nil {
		stdoutTTY = *options.StdoutTTY
	}
	if options.StderrTTY != nil {
		stderrTTY = *options.StderrTTY
	}
	lookup := options.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	_, noColorEnvironment := lookup("NO_COLOR")
	termName, _ := lookup("TERM")
	dumbTerminal := strings.EqualFold(strings.TrimSpace(termName), "dumb")
	mode := ModeHuman
	switch {
	case options.JSON:
		mode = ModeJSON
	case options.Quiet:
		mode = ModeQuiet
	case options.Plain || !stdoutTTY:
		mode = ModePlain
	}
	return &Presenter{
		stdout: stdout, stderr: stderr, mode: mode, verbose: options.Verbose,
		color:       !options.NoColor && !noColorEnvironment && !dumbTerminal && stdoutTTY,
		stdoutIsTTY: stdoutTTY, stderrIsTTY: stderrTTY,
	}
}

func (p *Presenter) Mode() Mode         { return p.mode }
func (p *Presenter) ColorEnabled() bool { return p.color }
func (p *Presenter) StdoutIsTTY() bool  { return p.stdoutIsTTY }
func (p *Presenter) StderrIsTTY() bool  { return p.stderrIsTTY }

func (p *Presenter) Success(code, message string, data any) error {
	if p.mode == ModeQuiet {
		return nil
	}
	redacted := Redact(data)
	if p.mode == ModeJSON {
		return writeJSON(p.stdout, ResultEnvelope{OK: true, Code: code, Message: message, Data: redacted})
	}
	if p.mode == ModeHuman && p.color {
		_, err := fmt.Fprintf(p.stdout, "\x1b[32m%s\x1b[0m\n", message)
		return err
	}
	_, err := fmt.Fprintln(p.stdout, message)
	return err
}

func (p *Presenter) Failure(err error) error {
	code := app.ErrorCodeOf(err)
	message := "internal error"
	var details map[string]any
	var applicationErr *app.Error
	if errors.As(err, &applicationErr) {
		message = applicationErr.Message
		details, _ = Redact(applicationErr.Details).(map[string]any)
	}
	if p.mode == ModeJSON {
		return writeJSON(p.stderr, ErrorEnvelope{OK: false, Code: code, Message: message, Details: details})
	}
	prefix := "forecast-ledger: "
	if p.mode == ModeHuman && p.color && p.stderrIsTTY {
		prefix = "\x1b[31mforecast-ledger:\x1b[0m "
	}
	if _, writeErr := fmt.Fprintln(p.stderr, prefix+message); writeErr != nil {
		return writeErr
	}
	return p.writeIssues(details)
}

func (p *Presenter) writeIssues(details map[string]any) error {
	if details == nil {
		return nil
	}
	issues, ok := details["issues"].([]any)
	if !ok {
		return nil
	}
	for _, rawIssue := range issues {
		issue, ok := rawIssue.(map[string]any)
		if !ok {
			continue
		}
		location := firstString(issue, "instance_location", "pointer")
		if location == "" {
			if source, ok := issue["location"].(map[string]any); ok {
				location = firstString(source, "pointer")
				if start, ok := source["start"].(map[string]any); ok {
					location += fmt.Sprintf(" (line %v, column %v)", start["line"], start["column"])
				}
			}
		}
		if location == "" {
			location = "/"
		}
		if _, err := fmt.Fprintf(p.stderr, "- %s: %s: %s\n", location, firstString(issue, "code"), firstString(issue, "message")); err != nil {
			return err
		}
	}
	return nil
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok {
			return value
		}
	}
	return ""
}

func (p *Presenter) Verbose(message string) error {
	if !p.verbose {
		return nil
	}
	_, err := fmt.Fprintln(p.stderr, message)
	return err
}

func Redact(value any) any {
	return redact(reflect.ValueOf(value), "")
}

func redact(value reflect.Value, key string) any {
	if isSecretKey(key) {
		return "[redacted]"
	}
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		return redact(value.Elem(), key)
	}
	switch value.Kind() {
	case reflect.Struct:
		result := make(map[string]any)
		typeInfo := value.Type()
		for index := 0; index < value.NumField(); index++ {
			field := typeInfo.Field(index)
			if !field.IsExported() {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" {
				name = field.Name
			}
			if name == "-" {
				continue
			}
			result[name] = redact(value.Field(index), name)
		}
		return result
	case reflect.Map:
		result := make(map[string]any, value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			mapKey := fmt.Sprint(iterator.Key().Interface())
			result[mapKey] = redact(iterator.Value(), mapKey)
		}
		return result
	case reflect.Slice, reflect.Array:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return "[redacted bytes]"
		}
		result := make([]any, value.Len())
		for index := 0; index < value.Len(); index++ {
			result[index] = redact(value.Index(index), key)
		}
		return result
	default:
		return value.Interface()
	}
}

func isSecretKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	switch normalized {
	case "key", "revealed_key", "private_key", "secret", "password", "token", "authorization", "api_key", "salt":
		return true
	}
	return strings.HasSuffix(normalized, "_secret") || strings.HasSuffix(normalized, "_token")
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

type descriptorWriter interface{ Fd() uintptr }

func terminalWriter(writer io.Writer) bool {
	file, ok := writer.(descriptorWriter)
	return ok && term.IsTerminal(int(file.Fd()))
}
