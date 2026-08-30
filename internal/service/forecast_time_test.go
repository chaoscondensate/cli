package service

import (
	"testing"

	"github.com/chaoscondensate/cli/internal/ledger"
)

func TestDefaultForecastTimesUsesOneObservedValue(t *testing.T) {
	t.Parallel()
	observed := ledger.Timestamp("2026-08-30T18:41:30+01:00")
	forecasted, recorded := DefaultForecastTimes("", nil, observed)
	if forecasted != observed || recorded != observed {
		t.Fatalf("defaults = %q, %q; want %q twice", forecasted, recorded, observed)
	}
}

func TestDefaultForecastTimesPreservesExplicitValues(t *testing.T) {
	t.Parallel()
	observed := ledger.Timestamp("2026-08-30T18:41:30+01:00")
	explicitForecasted := ledger.Timestamp("2026-08-29T10:00:00Z")
	explicitRecorded := ledger.Timestamp("2026-08-29T10:01:00Z")
	forecasted, recorded := DefaultForecastTimes(explicitForecasted, &explicitRecorded, observed)
	if forecasted != explicitForecasted || recorded != explicitRecorded {
		t.Fatalf("explicit values changed to %q, %q", forecasted, recorded)
	}
}
