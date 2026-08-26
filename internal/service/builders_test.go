package service

import (
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/ledger"
)

func TestBuildLedgerRootCapturesOneClockAndBuildsValidIdentity(t *testing.T) {
	clock := fixedTestClock{value: time.Date(2026, 8, 26, 15, 4, 5, 900, time.FixedZone("BST", 3600))}
	root, err := BuildLedgerRoot(InitRootRequest{
		LedgerID: "research", Timezone: "Europe/London", ForecasterID: "andrey",
		ForecasterName: "Andrey", ForecasterKind: ledger.ForecasterIndividual,
		Input: InitInput{Platforms: map[ledger.Slug]ledger.Platform{"local": {Name: "Local", Kind: ledger.PlatformInformal}}},
	}, clock)
	if err != nil {
		t.Fatal(err)
	}
	if root.SchemaVersion != "1.0.0" || root.CreatedAt != "2026-08-26T15:04:05+01:00" || root.Platforms["local"].Name != "Local" {
		t.Fatalf("root = %#v", root)
	}
	if root.Questions == nil || len(root.Questions) != 0 || root.Publication != nil {
		t.Fatalf("root collections/publication = %#v / %#v", root.Questions, root.Publication)
	}
}

func TestBuildLedgerRootRejectsInvalidTeamIdentityTimeAndPlatform(t *testing.T) {
	members := []ledger.Member{{ID: "same", Name: "One"}, {ID: "same", Name: "Two"}}
	tests := []InitRootRequest{
		{LedgerID: "Bad ID", Timezone: "Europe/London", ForecasterID: "id", ForecasterName: "Name"},
		{LedgerID: "ok", Timezone: "Mars/Olympus", ForecasterID: "id", ForecasterName: "Name"},
		{LedgerID: "ok", Timezone: "UTC", ForecasterID: "id", ForecasterName: "Name", ForecasterKind: ledger.ForecasterTeam},
		{LedgerID: "ok", Timezone: "UTC", ForecasterID: "id", ForecasterName: "Name", ForecasterKind: ledger.ForecasterTeam, Input: InitInput{Members: &members}},
		{LedgerID: "ok", Timezone: "UTC", ForecasterID: "id", ForecasterName: "Name", Input: InitInput{Platforms: map[ledger.Slug]ledger.Platform{"p": {Name: "", Kind: ledger.PlatformInformal}}}},
	}
	for index, request := range tests {
		if _, err := BuildLedgerRoot(request, fixedTestClock{value: time.Now()}); app.ErrorCodeOf(err) != app.CodeInvalidData {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func TestTimestampAndChronologyRequireExactRFC3339(t *testing.T) {
	for _, value := range []ledger.Timestamp{"2026-08-26", "2026-08-26T12:00Z", "2026-02-30T12:00:00Z", "2026-08-26T12:00:00"} {
		if _, err := ParseTimestamp(value, "time"); app.ErrorCodeOf(err) != app.CodeInvalidData {
			t.Fatalf("timestamp %q error = %v", value, err)
		}
	}
	if err := ValidateChronology("2026-08-27T00:00:00Z", "later", "2026-08-26T00:00:00Z", "earlier", true); app.ErrorCodeOf(err) != app.CodeInvalidData {
		t.Fatalf("reverse chronology error = %v", err)
	}
}
