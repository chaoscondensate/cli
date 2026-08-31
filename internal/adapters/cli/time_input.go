package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	"github.com/chaoscondensate/forecast-ledger/internal/service"
	urfavecli "github.com/urfave/cli/v3"
)

type dateOnlyPolicy uint8

const (
	dateOnlyRejected dateOnlyPolicy = iota
	dateOnlyStart
	dateOnlyEnd
)

var humanTimeLayouts = []string{
	"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02T15:04",
	"2 Jan 2006 15:04:05", "2 Jan 2006 15:04", "2 January 2006 15:04:05", "2 January 2006 15:04",
	"Jan 2 2006 15:04:05", "Jan 2 2006 15:04", "January 2 2006 15:04:05", "January 2 2006 15:04",
}

func mutationTimezone(ctx context.Context, command *urfavecli.Command) (string, error) {
	loaded, err := service.LoadAndValidateLedger(ctx, command.String("file"), nil)
	if err != nil {
		return "", err
	}
	return loaded.Model.DefaultTimezone, nil
}

func normalizeSetTime(command *urfavecli.Command, name, timezone string, policy dateOnlyPolicy) error {
	_, err := normalizeSetTimeWithMetadata(command, name, timezone, policy)
	return err
}

func normalizeSetTimeWithMetadata(command *urfavecli.Command, name, timezone string, policy dateOnlyPolicy) (*service.TimeNormalization, error) {
	if !command.IsSet(name) {
		return nil, nil
	}
	raw := command.String(name)
	normalized, err := normalizeCLIAuthoredTime(raw, timezone, strings.ReplaceAll(name, "-", "_"), policy)
	if err != nil {
		return nil, err
	}
	if raw == string(normalized) {
		return nil, nil
	}
	for _, flag := range command.Flags {
		stringFlag, ok := flag.(*urfavecli.StringFlag)
		if !ok || !containsFlagName(flag.Names(), name) {
			continue
		}
		onlyOnce := stringFlag.OnlyOnce
		stringFlag.OnlyOnce = false
		err := command.Set(name, string(normalized))
		stringFlag.OnlyOnce = onlyOnce
		if err != nil {
			return nil, err
		}
		return &service.TimeNormalization{Field: strings.ReplaceAll(name, "-", "_"), Raw: raw, Normalized: normalized, Timezone: timezone, DatePolicy: datePolicyName(policy)}, nil
	}
	return nil, app.NewError(app.CodeInternal, "timestamp flag is not registered", nil)
}

func datePolicyName(policy dateOnlyPolicy) string {
	switch policy {
	case dateOnlyStart:
		return "start_of_day"
	case dateOnlyEnd:
		return "end_of_day"
	default:
		return "time_required"
	}
}

func appendTimeNormalization(values []service.TimeNormalization, value *service.TimeNormalization) []service.TimeNormalization {
	if value == nil {
		return values
	}
	return append(values, *value)
}

func containsFlagName(names []string, wanted string) bool {
	for _, name := range names {
		if name == wanted {
			return true
		}
	}
	return false
}

func formatOperationTime(value time.Time, timezone string) (ledger.Timestamp, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", app.NewError(app.CodeInvalidData, "timezone is not a known IANA timezone", err)
	}
	return ledger.Timestamp(value.In(location).Format(time.RFC3339)), nil
}

var humanDateLayouts = []string{"2006-01-02", "2 Jan 2006", "2 January 2006", "Jan 2 2006", "January 2 2006"}

func normalizeCLIAuthoredTime(raw, timezone, field string, policy dateOnlyPolicy) (ledger.Timestamp, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", app.NewError(app.CodeUsage, "--"+strings.ReplaceAll(field, "_", "-")+" must not be empty", nil)
	}
	if !hasExcessTimestampPrecision(value) {
		if exact, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return ledger.Timestamp(exact.Format(time.RFC3339Nano)), nil
		}
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", app.NewError(app.CodeInvalidData, "ledger default_timezone is not a known IANA timezone", err)
	}
	for _, layout := range humanTimeLayouts {
		if parsed, parseErr := time.ParseInLocation(layout, value, location); parseErr == nil {
			if err := rejectAmbiguousWallTime(parsed, value, layout, location); err != nil {
				return "", app.NewError(app.CodeUsage, field+" uses a skipped or repeated local wall time; supply an explicit numeric offset", err)
			}
			return ledger.Timestamp(parsed.Format(time.RFC3339)), nil
		}
	}
	for _, layout := range humanDateLayouts {
		parsed, parseErr := time.ParseInLocation(layout, value, location)
		if parseErr != nil {
			continue
		}
		switch policy {
		case dateOnlyStart:
			return ledger.Timestamp(parsed.Format(time.RFC3339)), nil
		case dateOnlyEnd:
			end := parsed.AddDate(0, 0, 1).Add(-time.Second)
			return ledger.Timestamp(end.Format(time.RFC3339)), nil
		default:
			return "", app.NewError(app.CodeUsage, field+" requires a time; date-only input is not allowed", nil)
		}
	}
	return "", app.NewError(app.CodeUsage, fmt.Sprintf("%s must be RFC 3339, YYYY-MM-DD, or an English month date with an optional 24-hour time", field), nil)
}

func hasExcessTimestampPrecision(value string) bool {
	timeStart := strings.IndexByte(value, 'T')
	if timeStart < 0 {
		return false
	}
	dot := strings.IndexByte(value[timeStart:], '.')
	if dot < 0 {
		return false
	}
	dot += timeStart
	end := len(value)
	for index := dot + 1; index < len(value); index++ {
		if value[index] == 'Z' || value[index] == '+' || value[index] == '-' {
			end = index
			break
		}
	}
	return end-dot-1 > 9
}

func rejectAmbiguousWallTime(parsed time.Time, raw, layout string, location *time.Location) error {
	if parsed.Format(layout) != raw {
		return fmt.Errorf("local time does not exist")
	}
	wall := parsed.Format("2006-01-02 15:04:05")
	for delta := 15 * time.Minute; delta <= 4*time.Hour; delta += 15 * time.Minute {
		for _, direction := range []time.Duration{-1, 1} {
			candidate := parsed.Add(direction * delta).In(location)
			if candidate.Format("2006-01-02 15:04:05") == wall {
				return fmt.Errorf("local time occurs more than once")
			}
		}
	}
	return nil
}
