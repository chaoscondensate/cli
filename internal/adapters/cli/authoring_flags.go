package cli

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	"github.com/chaoscondensate/forecast-ledger/internal/service"
	urfavecli "github.com/urfave/cli/v3"
)

type authoringFieldRoute struct {
	Field     string
	Route     string
	Class     string
	Rationale string
}

type authoringCommandRoute struct {
	Path      string
	Schema    service.InputSchemaName
	Protected bool
	Fields    []authoringFieldRoute
}

// authoringInventory is deliberately executable project memory. Tests check
// that every route is classified and that ordinary authoring leaves do not
// regain a side-loaded public request document.
var authoringInventory = []authoringCommandRoute{
	{Path: "init", Schema: service.InputSchemaInit, Fields: routes(
		"title=--title", "description=--description", "created_at=--created-at", "contact=--contact-email/--contact-website",
		"profiles=--profile", "members=--member/--member-profile", "platforms=--initial-platform", "question=--question-* / --initial-*",
	)},
	{Path: "ledger update", Schema: service.InputSchemaRootMetadata, Fields: routes(
		"title=--title/--clear-title", "description=--description/--clear-description", "default_timezone=--timezone",
		"forecaster.kind=--forecaster-kind", "forecaster.name=--forecaster-name", "forecaster.contact=--contact-*/--clear-contact",
		"forecaster.profiles=--profile/--clear-profiles", "forecaster.members=--member/--member-profile/--clear-members",
	)},
	{Path: "platform add", Schema: service.InputSchemaPlatformCreate, Fields: routes(
		"name=--name", "kind=--kind", "url=--url", "account=--account-username/--account-user-id/--account-profile-url",
	)},
	{Path: "platform update", Schema: service.InputSchemaPlatformPatch, Fields: routes(
		"name=--name", "kind=--kind", "url=--url/--clear-url", "account=--account-*/--clear-account",
	)},
	{Path: "question add", Schema: service.InputSchemaQuestionAdd, Fields: append(routes(
		"title=--title", "resolution_criteria=--resolution-criteria", "created_at=--created-at", "forecast_window=--opens-at",
		"expected_resolution_at=--expected-resolution-at", "options=--option", "unit=--unit-*", "platform_refs=--platform-ref",
		"tags=--tag", "notes=--notes", "initial_forecast.public=--initial-*", "initial_forecast.sealed_private=protected --initial-secret-input",
	), serviceOnlyRoutes("initial_forecast.supersedes_forecast_id=a newly added question has no earlier forecast to supersede")...)},
	{Path: "question update", Schema: service.InputSchemaQuestionPatch, Fields: routes(
		"title=--title", "resolution_criteria=--resolution-criteria", "forecast_window=--opens-at/--clear-forecast-window", "expected_resolution_at=--expected-resolution-at",
		"platform_refs=--platform-ref/--clear-platform-refs", "tags=--tag/--clear-tags", "notes=--notes/--clear-notes", "status=--status",
	)},
	{Path: "question resolve", Schema: service.InputSchemaResolution, Fields: routes(
		"outcome=--outcome/--outcome-boolean", "outcome_known_at=--outcome-known-at", "recorded_at=--recorded-at", "sources=--source", "notes=--notes",
	)},
	{Path: "question annul", Schema: service.InputSchemaAnnul, Fields: routes("reason=--reason", "recorded_at=--recorded-at", "sources=--source")},
	{Path: "question dispute", Schema: service.InputSchemaDispute, Fields: routes("reason=--reason", "recorded_at=--recorded-at", "sources=--source")},
	{Path: "forecast add", Schema: service.InputSchemaForecastCreate, Fields: routes(
		"forecasted_at=--forecasted-at", "recorded_at=--recorded-at", "value=--value-kind and type-specific value flags", "rationale=--rationale",
		"key_factors=--key-factor", "comment=--comment", "public_note=--public-note", "supersedes_forecast_id=--supersedes-forecast",
	)},
	{Path: "forecast seal", Schema: service.InputSchemaForecastSealPrivate, Protected: true, Fields: append(routes(
		"forecasted_at=--forecasted-at", "recorded_at=--recorded-at", "public_note=--public-note", "supersedes_forecast_id=--supersedes-forecast",
	), protectedRoutes("value=protected --secret-input", "rationale=protected --secret-input", "key_factors=protected --secret-input", "comment=protected --secret-input")...)},
	{Path: "forecast key-hint update", Schema: service.InputSchemaKeyHintUpdate, Fields: routes("key_hint=--key-hint")},
}

func routes(values ...string) []authoringFieldRoute {
	result := make([]authoringFieldRoute, 0, len(values))
	for _, value := range values {
		field, route, _ := strings.Cut(value, "=")
		result = append(result, authoringFieldRoute{Field: field, Route: route, Class: "public"})
	}
	return result
}

func protectedRoutes(values ...string) []authoringFieldRoute {
	result := routes(values...)
	for index := range result {
		result[index].Class = "secret"
	}
	return result
}

func serviceOnlyRoutes(values ...string) []authoringFieldRoute {
	result := make([]authoringFieldRoute, 0, len(values))
	for _, value := range values {
		field, rationale, _ := strings.Cut(value, "=")
		result = append(result, authoringFieldRoute{Field: field, Class: "service-only", Rationale: rationale})
	}
	return result
}

func requireDirectFlags(command *urfavecli.Command, names ...string) error {
	var missing []string
	for _, name := range names {
		if !command.IsSet(name) || strings.TrimSpace(command.String(name)) == "" {
			missing = append(missing, "--"+name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return app.NewError(app.CodeUsage, strings.Join(missing, " and ")+" required", nil)
}

func parseCSVValues(flag string, values []string, minimum, maximum int) ([][]string, error) {
	result := make([][]string, 0, len(values))
	for position, value := range values {
		reader := csv.NewReader(strings.NewReader(value))
		reader.FieldsPerRecord = -1
		record, err := reader.Read()
		if err != nil {
			return nil, app.NewError(app.CodeUsage, fmt.Sprintf("--%s value %d is not valid CSV: %v", flag, position+1, err), err)
		}
		if _, err := reader.Read(); err != io.EOF {
			return nil, app.NewError(app.CodeUsage, fmt.Sprintf("--%s value %d must contain one CSV record", flag, position+1), err)
		}
		if len(record) < minimum || len(record) > maximum {
			return nil, app.NewError(app.CodeUsage, fmt.Sprintf("--%s value %d needs %d to %d CSV fields", flag, position+1, minimum, maximum), nil)
		}
		for index := range record {
			record[index] = strings.TrimSpace(record[index])
		}
		result = append(result, record)
	}
	return result, nil
}

func pointer[T any](value T) *T { return &value }

func optionalStringValue(command *urfavecli.Command, name string) *string {
	if !command.IsSet(name) {
		return nil
	}
	return pointer(command.String(name))
}

func optionalTimestampValue(command *urfavecli.Command, name string) *ledger.Timestamp {
	if !command.IsSet(name) {
		return nil
	}
	return pointer(ledger.Timestamp(command.String(name)))
}

func rootAuthoringFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "title", OnlyOnce: true, Usage: "Ledger title"},
		&urfavecli.StringFlag{Name: "description", OnlyOnce: true, Usage: "Ledger description"},
		&urfavecli.StringFlag{Name: "created-at", OnlyOnce: true, Usage: "Explicit RFC 3339 ledger creation time"},
		&urfavecli.StringFlag{Name: "contact-email", OnlyOnce: true, Usage: "Forecaster contact email"},
		&urfavecli.StringFlag{Name: "contact-website", OnlyOnce: true, Usage: "Forecaster contact website URL"},
		&urfavecli.StringSliceFlag{Name: "profile", Usage: "Forecaster profile as service,url[,username]; repeat for more"},
		&urfavecli.StringSliceFlag{Name: "member", Usage: "Team member as id,name[,role]; repeat for more"},
		&urfavecli.StringSliceFlag{Name: "member-profile", Usage: "Member profile as member-id,service,url[,username]; repeat for more"},
	}
}

func rootPatchFlags() []urfavecli.Flag {
	flags := []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "title", OnlyOnce: true, Usage: "Replace ledger title"},
		&urfavecli.BoolFlag{Name: "clear-title", Usage: "Remove ledger title"},
		&urfavecli.StringFlag{Name: "description", OnlyOnce: true, Usage: "Replace ledger description"},
		&urfavecli.BoolFlag{Name: "clear-description", Usage: "Remove ledger description"},
		&urfavecli.StringFlag{Name: "timezone", OnlyOnce: true, Usage: "Replace default IANA timezone"},
		&urfavecli.StringFlag{Name: "forecaster-kind", OnlyOnce: true, Usage: "Replace forecaster kind: individual or team"},
		&urfavecli.StringFlag{Name: "forecaster-name", OnlyOnce: true, Usage: "Replace forecaster display name"},
		&urfavecli.BoolFlag{Name: "clear-contact", Usage: "Remove forecaster contact"},
		&urfavecli.BoolFlag{Name: "clear-profiles", Usage: "Remove forecaster profiles"},
		&urfavecli.BoolFlag{Name: "clear-members", Usage: "Remove team members"},
	}
	return append(flags, rootAuthoringFlags()[3:]...)
}

func initNestedFlags() []urfavecli.Flag {
	flags := []urfavecli.Flag{
		&urfavecli.StringSliceFlag{Name: "initial-platform", Usage: "Initial platform as id,name,kind[,url[,username[,user-id[,profile-url]]]]; repeat"},
		&urfavecli.StringFlag{Name: "question", OnlyOnce: true, Usage: "Optional initial question ID"},
		&urfavecli.StringFlag{Name: "question-type", OnlyOnce: true, Usage: "Initial question type: binary, multiple_choice, numeric, or date"},
		&urfavecli.StringFlag{Name: "question-title", OnlyOnce: true, Usage: "Initial question title"},
		&urfavecli.StringFlag{Name: "question-resolution-criteria", OnlyOnce: true, Usage: "Initial question resolution criteria"},
		&urfavecli.StringFlag{Name: "question-created-at", OnlyOnce: true, Usage: "Initial question RFC 3339 creation time"},
		&urfavecli.StringFlag{Name: "question-opens-at", OnlyOnce: true, Usage: "Initial question forecast-window opening"},
		&urfavecli.StringFlag{Name: "question-expected-resolution-at", OnlyOnce: true, Usage: "Initial question expected resolution time"},
		&urfavecli.StringSliceFlag{Name: "question-option", Usage: "Initial multiple-choice option as id,label; repeat"},
		&urfavecli.StringFlag{Name: "question-unit-name", OnlyOnce: true, Usage: "Initial numeric unit name"},
		&urfavecli.StringFlag{Name: "question-unit-symbol", OnlyOnce: true, Usage: "Initial numeric unit symbol"},
		&urfavecli.StringFlag{Name: "question-unit-ucum-code", OnlyOnce: true, Usage: "Initial numeric UCUM code"},
		&urfavecli.StringSliceFlag{Name: "question-platform-ref", Usage: "Initial platform ref as platform[,question-id[,url]]; repeat"},
		&urfavecli.StringSliceFlag{Name: "question-tag", Usage: "Initial question tag; repeat"},
		&urfavecli.StringFlag{Name: "question-notes", OnlyOnce: true, Usage: "Initial question notes"},
	}
	return append(flags, initialForecastFlags()...)
}

func platformCreateFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "name", OnlyOnce: true, Usage: "Platform display name; required"},
		&urfavecli.StringFlag{Name: "kind", OnlyOnce: true, Usage: "Platform kind: scoring_platform, prediction_market, self_hosted, internal, or informal"},
		&urfavecli.StringFlag{Name: "url", OnlyOnce: true, Usage: "Platform URL"},
		&urfavecli.StringFlag{Name: "account-username", OnlyOnce: true, Usage: "Account username"},
		&urfavecli.StringFlag{Name: "account-user-id", OnlyOnce: true, Usage: "Platform account user ID"},
		&urfavecli.StringFlag{Name: "account-profile-url", OnlyOnce: true, Usage: "Platform account profile URL"},
	}
}

func platformPatchFlags() []urfavecli.Flag {
	flags := platformCreateFlags()
	return append(flags,
		&urfavecli.BoolFlag{Name: "clear-url", Usage: "Remove platform URL"},
		&urfavecli.BoolFlag{Name: "clear-account", Usage: "Remove platform account metadata"},
		&urfavecli.BoolFlag{Name: "clear-account-username", Usage: "Remove account username"},
		&urfavecli.BoolFlag{Name: "clear-account-user-id", Usage: "Remove account user ID"},
		&urfavecli.BoolFlag{Name: "clear-account-profile-url", Usage: "Remove account profile URL"},
	)
}

func questionCreateFlags(includeInitial bool) []urfavecli.Flag {
	flags := []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "title", OnlyOnce: true, Usage: "Question title; required"},
		&urfavecli.StringFlag{Name: "resolution-criteria", OnlyOnce: true, Usage: "Resolution criteria; required"},
		&urfavecli.StringFlag{Name: "created-at", OnlyOnce: true, Usage: "Explicit RFC 3339 question creation time"},
		&urfavecli.StringFlag{Name: "opens-at", OnlyOnce: true, Usage: "Explicit RFC 3339 forecast-window opening"},
		&urfavecli.StringFlag{Name: "expected-resolution-at", OnlyOnce: true, Usage: "Expected resolution time; RFC 3339, ISO date, or English month date; required"},
		&urfavecli.StringSliceFlag{Name: "option", Usage: "Multiple-choice option as id,label; repeat for more"},
		&urfavecli.StringFlag{Name: "unit-name", OnlyOnce: true, Usage: "Numeric question unit name"},
		&urfavecli.StringFlag{Name: "unit-symbol", OnlyOnce: true, Usage: "Numeric question unit symbol"},
		&urfavecli.StringFlag{Name: "unit-ucum-code", OnlyOnce: true, Usage: "Numeric question UCUM code"},
		&urfavecli.StringSliceFlag{Name: "platform-ref", Usage: "Platform reference as platform[,question-id[,url]]; repeat for more"},
		&urfavecli.StringSliceFlag{Name: "tag", Usage: "Question tag; repeat for more"},
		&urfavecli.StringFlag{Name: "notes", OnlyOnce: true, Usage: "Question notes"},
	}
	if includeInitial {
		flags = append(flags, initialForecastFlags()...)
	}
	return flags
}

func questionPatchFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "title", OnlyOnce: true, Usage: "Replace question title"},
		&urfavecli.StringFlag{Name: "resolution-criteria", OnlyOnce: true, Usage: "Replace resolution criteria"},
		&urfavecli.StringFlag{Name: "opens-at", OnlyOnce: true, Usage: "Replace optional forecast-window opening"},
		&urfavecli.BoolFlag{Name: "clear-forecast-window", Usage: "Remove the optional forecast window"},
		&urfavecli.StringFlag{Name: "expected-resolution-at", OnlyOnce: true, Usage: "Replace expected resolution time"},
		&urfavecli.StringSliceFlag{Name: "platform-ref", Usage: "Replace platform refs with platform[,question-id[,url]]; repeat"},
		&urfavecli.BoolFlag{Name: "clear-platform-refs", Usage: "Remove platform references"},
		&urfavecli.StringSliceFlag{Name: "tag", Usage: "Replace tags; repeat for more"},
		&urfavecli.BoolFlag{Name: "clear-tags", Usage: "Remove tags"},
		&urfavecli.StringFlag{Name: "notes", OnlyOnce: true, Usage: "Replace notes, including an explicit empty value"},
		&urfavecli.BoolFlag{Name: "clear-notes", Usage: "Remove notes"},
		&urfavecli.StringFlag{Name: "status", OnlyOnce: true, Usage: "Set status: open, closed, or awaiting_resolution"},
	}
}

func forecastValueFlags(prefix string) []urfavecli.Flag {
	name := func(value string) string {
		if prefix == "" {
			return value
		}
		return prefix + "-" + value
	}
	return []urfavecli.Flag{
		&urfavecli.StringFlag{Name: name("value-kind"), OnlyOnce: true, Usage: "Forecast value kind: binary, multiple_choice, numeric, or date"},
		&urfavecli.IntFlag{Name: name("probability-bp"), OnlyOnce: true, Usage: "Binary probability in basis points (0-10000)"},
		&urfavecli.StringSliceFlag{Name: name("choice-probability"), Usage: "Choice probability as option-id,probability-bp; repeat for every option"},
		&urfavecli.StringFlag{Name: name("point"), OnlyOnce: true, Usage: "Exact numeric decimal or YYYY-MM-DD date point"},
		&urfavecli.StringFlag{Name: name("interval"), OnlyOnce: true, Usage: "Interval as lower,upper,credibility-bp"},
		&urfavecli.StringSliceFlag{Name: name("quantile"), Usage: "Quantile as probability-bp,value; repeat for more"},
	}
}

func forecastCreateFlags() []urfavecli.Flag {
	flags := []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "forecasted-at", OnlyOnce: true, Usage: "Forecast time; defaults to the current time in ledger default_timezone"},
		&urfavecli.StringFlag{Name: "recorded-at", OnlyOnce: true, Usage: "Explicit RFC 3339 record time; defaults to operation time"},
		&urfavecli.StringFlag{Name: "rationale", OnlyOnce: true, Usage: "Public forecast rationale"},
		&urfavecli.StringSliceFlag{Name: "key-factor", Usage: "Public key factor; repeat for more"},
		&urfavecli.StringFlag{Name: "comment", OnlyOnce: true, Usage: "Public forecast comment"},
		&urfavecli.StringFlag{Name: "public-note", OnlyOnce: true, Usage: "Public note"},
		&urfavecli.StringFlag{Name: "supersedes-forecast", OnlyOnce: true, Usage: "Earlier forecast ID superseded by this revision"},
	}
	return append(flags, forecastValueFlags("")...)
}

func initialForecastFlags() []urfavecli.Flag {
	flags := []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "initial-forecast", OnlyOnce: true, Usage: "Initial forecast ID; omit to create a backlog question"},
		&urfavecli.StringFlag{Name: "initial-visibility", OnlyOnce: true, Value: "public", Usage: "Initial visibility: public or sealed (sealed uses --initial-secret-input)"},
		&urfavecli.StringFlag{Name: "initial-forecasted-at", OnlyOnce: true, Usage: "Initial forecast time; defaults to the current time in the ledger timezone"},
		&urfavecli.StringFlag{Name: "initial-recorded-at", OnlyOnce: true, Usage: "Explicit initial record time; defaults to the same operation time"},
		&urfavecli.StringFlag{Name: "initial-rationale", OnlyOnce: true, Usage: "Initial public rationale"},
		&urfavecli.StringSliceFlag{Name: "initial-key-factor", Usage: "Initial public key factor; repeat"},
		&urfavecli.StringFlag{Name: "initial-comment", OnlyOnce: true, Usage: "Initial public comment"},
		&urfavecli.StringFlag{Name: "initial-public-note", OnlyOnce: true, Usage: "Initial public note"},
		&urfavecli.StringFlag{Name: "initial-secret-input", OnlyOnce: true, TakesFile: true, Usage: "Protected JSON or YAML private bundle for a sealed initial forecast; use - for stdin"},
	}
	return append(flags, forecastValueFlags("initial")...)
}

func forecastSealPublicFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "secret-input", OnlyOnce: true, TakesFile: true, Usage: "Protected JSON or YAML private bundle; use - for stdin"},
		&urfavecli.StringFlag{Name: "forecasted-at", OnlyOnce: true, Usage: "Public forecast time; defaults to the current time in the ledger timezone"},
		&urfavecli.StringFlag{Name: "recorded-at", OnlyOnce: true, Usage: "Explicit public RFC 3339 record time"},
		&urfavecli.StringFlag{Name: "public-note", OnlyOnce: true, Usage: "Public note stored outside the sealed bundle"},
		&urfavecli.StringFlag{Name: "supersedes-forecast", OnlyOnce: true, Usage: "Earlier forecast ID superseded by this sealed revision"},
	}
}

func lifecycleFlags(resolution bool) []urfavecli.Flag {
	flags := []urfavecli.Flag{
		&urfavecli.StringFlag{Name: "recorded-at", OnlyOnce: true, Usage: "Explicit RFC 3339 lifecycle record time"},
		&urfavecli.StringSliceFlag{Name: "source", Usage: "Evidence source as title,url,retrieved-at[,publisher[,published-at[,sha256]]]; repeat"},
	}
	if resolution {
		return append([]urfavecli.Flag{
			&urfavecli.StringFlag{Name: "outcome", OnlyOnce: true, Usage: "Exact non-boolean outcome value"},
			&urfavecli.BoolFlag{Name: "outcome-boolean", Usage: "Boolean outcome; explicit true or false"},
			&urfavecli.StringFlag{Name: "outcome-known-at", OnlyOnce: true, Usage: "RFC 3339 time outcome became known"},
			&urfavecli.StringFlag{Name: "notes", OnlyOnce: true, Usage: "Resolution notes"},
		}, flags...)
	}
	return append([]urfavecli.Flag{&urfavecli.StringFlag{Name: "reason", OnlyOnce: true, Usage: "Lifecycle reason"}}, flags...)
}

func buildProfiles(command *urfavecli.Command, profileFlag, memberFlag, memberProfileFlag string) (*[]ledger.Profile, *[]ledger.Member, error) {
	profileRows, err := parseCSVValues(profileFlag, command.StringSlice(profileFlag), 2, 3)
	if err != nil {
		return nil, nil, err
	}
	profiles := make([]ledger.Profile, 0, len(profileRows))
	for _, row := range profileRows {
		profile := ledger.Profile{Service: row[0], URL: row[1]}
		if len(row) == 3 && row[2] != "" {
			profile.Username = pointer(row[2])
		}
		profiles = append(profiles, profile)
	}
	memberRows, err := parseCSVValues(memberFlag, command.StringSlice(memberFlag), 2, 3)
	if err != nil {
		return nil, nil, err
	}
	members := make([]ledger.Member, 0, len(memberRows))
	memberPositions := make(map[ledger.Slug]int, len(memberRows))
	for _, row := range memberRows {
		member := ledger.Member{ID: ledger.Slug(row[0]), Name: row[1]}
		if len(row) == 3 && row[2] != "" {
			member.Role = pointer(row[2])
		}
		memberPositions[member.ID] = len(members)
		members = append(members, member)
	}
	memberProfileRows, err := parseCSVValues(memberProfileFlag, command.StringSlice(memberProfileFlag), 3, 4)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range memberProfileRows {
		position, ok := memberPositions[ledger.Slug(row[0])]
		if !ok {
			return nil, nil, app.NewError(app.CodeUsage, "--member-profile names a member not present in --member: "+row[0], nil)
		}
		profile := ledger.Profile{Service: row[1], URL: row[2]}
		if len(row) == 4 && row[3] != "" {
			profile.Username = pointer(row[3])
		}
		if members[position].Profiles == nil {
			members[position].Profiles = pointer([]ledger.Profile{})
		}
		*members[position].Profiles = append(*members[position].Profiles, profile)
	}
	var profilePointer *[]ledger.Profile
	if command.IsSet(profileFlag) {
		profilePointer = &profiles
	}
	var memberPointer *[]ledger.Member
	if command.IsSet(memberFlag) || command.IsSet(memberProfileFlag) {
		memberPointer = &members
	}
	return profilePointer, memberPointer, nil
}

func buildContact(command *urfavecli.Command) *ledger.Contact {
	if !command.IsSet("contact-email") && !command.IsSet("contact-website") {
		return nil
	}
	return &ledger.Contact{Email: optionalStringValue(command, "contact-email"), Website: optionalStringValue(command, "contact-website")}
}

func buildPlatformAccount(command *urfavecli.Command) *ledger.PlatformAccount {
	if !command.IsSet("account-username") && !command.IsSet("account-user-id") && !command.IsSet("account-profile-url") {
		return nil
	}
	return &ledger.PlatformAccount{
		Username: optionalStringValue(command, "account-username"), UserID: optionalStringValue(command, "account-user-id"),
		ProfileURL: optionalStringValue(command, "account-profile-url"),
	}
}

func buildRootPatchInput(command *urfavecli.Command) (service.RootMetadataPatchInput, error) {
	input := service.RootMetadataPatchInput{}
	var err error
	input.Title, err = patchString(command, "title", "clear-title")
	if err != nil {
		return input, err
	}
	input.Description, err = patchString(command, "description", "clear-description")
	if err != nil {
		return input, err
	}
	if command.IsSet("timezone") {
		input.DefaultTimezone = service.Optional[string]{Set: true, Value: command.String("timezone")}
	}
	var forecaster service.ForecasterMetadataPatchInput
	if command.IsSet("forecaster-kind") {
		forecaster.Kind = service.Optional[ledger.ForecasterKind]{Set: true, Value: ledger.ForecasterKind(command.String("forecaster-kind"))}
	}
	if command.IsSet("forecaster-name") {
		forecaster.Name = service.Optional[string]{Set: true, Value: command.String("forecaster-name")}
	}
	if command.Bool("clear-contact") && (command.IsSet("contact-email") || command.IsSet("contact-website")) {
		return input, app.NewError(app.CodeUsage, "--clear-contact cannot be combined with contact field flags", nil)
	}
	if command.Bool("clear-contact") {
		forecaster.Contact = service.Optional[ledger.Contact]{Set: true, Null: true}
	} else if contact := buildContact(command); contact != nil {
		forecaster.Contact = service.Optional[ledger.Contact]{Set: true, Value: *contact}
	}
	profiles, members, err := buildProfiles(command, "profile", "member", "member-profile")
	if err != nil {
		return input, err
	}
	if command.Bool("clear-profiles") && profiles != nil {
		return input, app.NewError(app.CodeUsage, "--clear-profiles cannot be combined with --profile", nil)
	}
	if command.Bool("clear-profiles") {
		forecaster.Profiles = service.Optional[[]ledger.Profile]{Set: true, Null: true}
	} else if profiles != nil {
		forecaster.Profiles = service.Optional[[]ledger.Profile]{Set: true, Value: *profiles}
	}
	if command.Bool("clear-members") && members != nil {
		return input, app.NewError(app.CodeUsage, "--clear-members cannot be combined with --member or --member-profile", nil)
	}
	if command.Bool("clear-members") {
		forecaster.Members = service.Optional[[]ledger.Member]{Set: true, Null: true}
	} else if members != nil {
		forecaster.Members = service.Optional[[]ledger.Member]{Set: true, Value: *members}
	}
	if forecaster.Kind.Set || forecaster.Name.Set || forecaster.Contact.Set || forecaster.Profiles.Set || forecaster.Members.Set {
		input.Forecaster = service.Optional[service.ForecasterMetadataPatchInput]{Set: true, Value: forecaster}
	}
	if !input.Title.Set && !input.Description.Set && !input.DefaultTimezone.Set && !input.Forecaster.Set {
		return input, app.NewError(app.CodeUsage, "at least one ledger authoring flag is required", nil)
	}
	return input, nil
}

func buildInitialPlatforms(command *urfavecli.Command) (map[ledger.Slug]ledger.Platform, error) {
	if !command.IsSet("initial-platform") {
		return nil, nil
	}
	rows, err := parseCSVValues("initial-platform", command.StringSlice("initial-platform"), 3, 7)
	if err != nil {
		return nil, err
	}
	values := make(map[ledger.Slug]ledger.Platform, len(rows))
	for _, row := range rows {
		id := ledger.Slug(row[0])
		if _, exists := values[id]; exists {
			return nil, app.NewError(app.CodeUsage, "--initial-platform repeats platform ID "+row[0], nil)
		}
		platform := ledger.Platform{Name: row[1], Kind: ledger.PlatformKind(row[2])}
		if len(row) > 3 && row[3] != "" {
			platform.URL = pointer(row[3])
		}
		if len(row) > 4 {
			account := &ledger.PlatformAccount{}
			if row[4] != "" {
				account.Username = pointer(row[4])
			}
			if len(row) > 5 && row[5] != "" {
				account.UserID = pointer(row[5])
			}
			if len(row) > 6 && row[6] != "" {
				account.ProfileURL = pointer(row[6])
			}
			if account.Username != nil || account.UserID != nil || account.ProfileURL != nil {
				platform.Account = account
			}
		}
		values[id] = platform
	}
	return values, nil
}

func buildInitInput(operationContext context.Context, command *urfavecli.Command, stdin io.Reader) (service.InitInput, error) {
	profiles, members, err := buildProfiles(command, "profile", "member", "member-profile")
	if err != nil {
		return service.InitInput{}, err
	}
	platforms, err := buildInitialPlatforms(command)
	if err != nil {
		return service.InitInput{}, err
	}
	input := service.InitInput{
		Title: optionalStringValue(command, "title"), Description: optionalStringValue(command, "description"), CreatedAt: optionalTimestampValue(command, "created-at"),
		Contact: buildContact(command), Profiles: profiles, Members: members, Platforms: platforms,
	}
	questionFlags := []string{"question-type", "question-title", "question-resolution-criteria", "question-created-at", "question-opens-at", "question-expected-resolution-at", "question-option", "question-unit-name", "question-unit-symbol", "question-unit-ucum-code", "question-platform-ref", "question-tag", "question-notes"}
	questionFlags = append(questionFlags, allMappedFlags(initialForecastFlags())...)
	if !command.IsSet("question") {
		for _, name := range questionFlags {
			if command.IsSet(name) {
				return service.InitInput{}, app.NewError(app.CodeUsage, "--"+name+" requires --question", nil)
			}
		}
		return input, nil
	}
	if err := requireDirectFlags(command, "question", "question-type", "question-title", "question-resolution-criteria", "question-expected-resolution-at"); err != nil {
		return service.InitInput{}, err
	}
	optionRows, err := parseCSVValues("question-option", command.StringSlice("question-option"), 2, 2)
	if err != nil {
		return service.InitInput{}, err
	}
	var options *[]ledger.Option
	if command.IsSet("question-option") {
		values := make([]ledger.Option, 0, len(optionRows))
		for _, row := range optionRows {
			values = append(values, ledger.Option{ID: ledger.Slug(row[0]), Label: row[1]})
		}
		options = &values
	}
	refRows, err := parseCSVValues("question-platform-ref", command.StringSlice("question-platform-ref"), 1, 3)
	if err != nil {
		return service.InitInput{}, err
	}
	var refs *[]ledger.PlatformRef
	if command.IsSet("question-platform-ref") {
		values := make([]ledger.PlatformRef, 0, len(refRows))
		for _, row := range refRows {
			value := ledger.PlatformRef{Platform: ledger.Slug(row[0])}
			if len(row) > 1 && row[1] != "" {
				value.QuestionID = pointer(row[1])
			}
			if len(row) > 2 && row[2] != "" {
				value.URL = pointer(row[2])
			}
			values = append(values, value)
		}
		refs = &values
	}
	var unit *ledger.Unit
	if command.IsSet("question-unit-name") || command.IsSet("question-unit-symbol") || command.IsSet("question-unit-ucum-code") {
		if err := requireDirectFlags(command, "question-unit-name"); err != nil {
			return service.InitInput{}, err
		}
		unit = &ledger.Unit{Name: command.String("question-unit-name"), Symbol: optionalStringValue(command, "question-unit-symbol"), UCUMCode: optionalStringValue(command, "question-unit-ucum-code")}
	}
	window := ledger.ForecastWindow{}
	if command.IsSet("question-opens-at") {
		window.OpensAt = ledger.Timestamp(command.String("question-opens-at"))
	}
	question := &service.InitialQuestionInput{
		ID: ledger.Slug(command.String("question")), Title: command.String("question-title"), Type: ledger.QuestionType(command.String("question-type")),
		ResolutionCriteria: command.String("question-resolution-criteria"), CreatedAt: optionalTimestampValue(command, "question-created-at"), ForecastWindow: window,
		ExpectedResolutionAt: ledger.Timestamp(command.String("question-expected-resolution-at")), Options: options, Unit: unit, PlatformRefs: refs,
		Notes: optionalStringValue(command, "question-notes"),
	}
	if command.IsSet("question-tag") {
		values := command.StringSlice("question-tag")
		tags := make([]ledger.Slug, 0, len(values))
		for _, value := range values {
			tags = append(tags, ledger.Slug(value))
		}
		question.Tags = &tags
	}
	question.InitialForecast, err = buildInitialForecast(operationContext, command, stdin)
	if err != nil {
		return service.InitInput{}, err
	}
	input.Question = question
	return input, nil
}

func buildPlatformCreateInput(command *urfavecli.Command) (service.PlatformCreateInput, error) {
	if err := requireDirectFlags(command, "name", "kind"); err != nil {
		return service.PlatformCreateInput{}, err
	}
	return service.PlatformCreateInput{
		Name: command.String("name"), Kind: ledger.PlatformKind(command.String("kind")), URL: optionalStringValue(command, "url"), Account: buildPlatformAccount(command),
	}, nil
}

func patchString(command *urfavecli.Command, setter, clearer string) (service.Optional[string], error) {
	if command.IsSet(setter) && command.Bool(clearer) {
		return service.Optional[string]{}, app.NewError(app.CodeUsage, "--"+setter+" cannot be combined with --"+clearer, nil)
	}
	if command.Bool(clearer) {
		return service.Optional[string]{Set: true, Null: true}, nil
	}
	if command.IsSet(setter) {
		return service.Optional[string]{Set: true, Value: command.String(setter)}, nil
	}
	return service.Optional[string]{}, nil
}

func buildPlatformPatchInput(command *urfavecli.Command) (service.PlatformPatchInput, error) {
	url, err := patchString(command, "url", "clear-url")
	if err != nil {
		return service.PlatformPatchInput{}, err
	}
	input := service.PlatformPatchInput{URL: url}
	if command.IsSet("name") {
		input.Name = service.Optional[string]{Set: true, Value: command.String("name")}
	}
	if command.IsSet("kind") {
		input.Kind = service.Optional[ledger.PlatformKind]{Set: true, Value: ledger.PlatformKind(command.String("kind"))}
	}
	accountFlags := []string{"account-username", "account-user-id", "account-profile-url"}
	if command.Bool("clear-account") {
		for _, name := range accountFlags {
			if command.IsSet(name) || command.Bool("clear-"+name) {
				return service.PlatformPatchInput{}, app.NewError(app.CodeUsage, "--clear-account cannot be combined with account field flags", nil)
			}
		}
		input.Account = service.Optional[service.PlatformAccountPatchInput]{Set: true, Null: true}
	} else {
		var account service.PlatformAccountPatchInput
		account.Username, err = patchString(command, "account-username", "clear-account-username")
		if err != nil {
			return service.PlatformPatchInput{}, err
		}
		account.UserID, err = patchString(command, "account-user-id", "clear-account-user-id")
		if err != nil {
			return service.PlatformPatchInput{}, err
		}
		account.ProfileURL, err = patchString(command, "account-profile-url", "clear-account-profile-url")
		if err != nil {
			return service.PlatformPatchInput{}, err
		}
		if account.Username.Set || account.UserID.Set || account.ProfileURL.Set {
			input.Account = service.Optional[service.PlatformAccountPatchInput]{Set: true, Value: account}
		}
	}
	if !input.Name.Set && !input.Kind.Set && !input.URL.Set && !input.Account.Set {
		return service.PlatformPatchInput{}, app.NewError(app.CodeUsage, "at least one platform authoring flag is required", nil)
	}
	return input, nil
}

func parseOptions(command *urfavecli.Command, name string) (*[]ledger.Option, error) {
	if !command.IsSet(name) {
		return nil, nil
	}
	rows, err := parseCSVValues(name, command.StringSlice(name), 2, 2)
	if err != nil {
		return nil, err
	}
	values := make([]ledger.Option, 0, len(rows))
	for _, row := range rows {
		values = append(values, ledger.Option{ID: ledger.Slug(row[0]), Label: row[1]})
	}
	return &values, nil
}

func parsePlatformRefs(command *urfavecli.Command, name string) (*[]ledger.PlatformRef, error) {
	if !command.IsSet(name) {
		return nil, nil
	}
	rows, err := parseCSVValues(name, command.StringSlice(name), 1, 3)
	if err != nil {
		return nil, err
	}
	values := make([]ledger.PlatformRef, 0, len(rows))
	for _, row := range rows {
		value := ledger.PlatformRef{Platform: ledger.Slug(row[0])}
		if len(row) > 1 && row[1] != "" {
			value.QuestionID = pointer(row[1])
		}
		if len(row) > 2 && row[2] != "" {
			value.URL = pointer(row[2])
		}
		values = append(values, value)
	}
	return &values, nil
}

func buildForecastValue(command *urfavecli.Command, prefix string) (ledger.ForecastValue, error) {
	name := func(value string) string {
		if prefix == "" {
			return value
		}
		return prefix + "-" + value
	}
	kind := ledger.ForecastValueKind(command.String(name("value-kind")))
	if kind == "" {
		return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "--"+name("value-kind")+" required", nil)
	}
	switch kind {
	case ledger.ValueBinary:
		if !command.IsSet(name("probability-bp")) {
			return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "--"+name("probability-bp")+" required for a binary value", nil)
		}
		return ledger.ForecastValue{Binary: &ledger.BinaryValue{Kind: kind, ProbabilityBP: ledger.BasisPoints(command.Int(name("probability-bp")))}}, nil
	case ledger.ValueMultipleChoice:
		rows, err := parseCSVValues(name("choice-probability"), command.StringSlice(name("choice-probability")), 2, 2)
		if err != nil {
			return ledger.ForecastValue{}, err
		}
		if len(rows) == 0 {
			return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "--"+name("choice-probability")+" must be repeated for every option", nil)
		}
		values := make([]ledger.ChoiceProbability, 0, len(rows))
		for _, row := range rows {
			basisPoints, err := strconv.ParseInt(row[1], 10, 32)
			if err != nil {
				return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "--"+name("choice-probability")+" basis points must be an integer", err)
			}
			values = append(values, ledger.ChoiceProbability{OptionID: ledger.Slug(row[0]), ProbabilityBP: ledger.BasisPoints(basisPoints)})
		}
		return ledger.ForecastValue{MultipleChoice: &ledger.MultipleChoiceValue{Kind: kind, Probabilities: values}}, nil
	case ledger.ValueNumeric, ledger.ValueDate:
		pointSet, intervalSet, quantilesSet := command.IsSet(name("point")), command.IsSet(name("interval")), command.IsSet(name("quantile"))
		if !pointSet && !intervalSet && !quantilesSet {
			return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "use --"+name("point")+", --"+name("interval")+", or --"+name("quantile"), nil)
		}
		if kind == ledger.ValueNumeric {
			value := &ledger.NumericValue{Kind: kind}
			if pointSet {
				value.Point = pointer(ledger.Decimal(command.String(name("point"))))
			}
			if intervalSet {
				row, err := parseCSVValues(name("interval"), []string{command.String(name("interval"))}, 3, 3)
				if err != nil {
					return ledger.ForecastValue{}, err
				}
				bp, err := strconv.ParseInt(row[0][2], 10, 32)
				if err != nil {
					return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "interval credibility must be an integer", err)
				}
				value.Interval = &ledger.NumericInterval{Lower: ledger.Decimal(row[0][0]), Upper: ledger.Decimal(row[0][1]), CredibilityBP: ledger.BasisPoints(bp)}
			}
			if quantilesSet {
				rows, err := parseCSVValues(name("quantile"), command.StringSlice(name("quantile")), 2, 2)
				if err != nil {
					return ledger.ForecastValue{}, err
				}
				quantiles := make([]ledger.NumericQuantile, 0, len(rows))
				for _, row := range rows {
					bp, err := strconv.ParseInt(row[0], 10, 32)
					if err != nil {
						return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "quantile probability must be an integer", err)
					}
					quantiles = append(quantiles, ledger.NumericQuantile{ProbabilityBP: ledger.BasisPoints(bp), Value: ledger.Decimal(row[1])})
				}
				value.Quantiles = &quantiles
			}
			return ledger.ForecastValue{Numeric: value}, nil
		}
		value := &ledger.DateValue{Kind: kind}
		if pointSet {
			value.Point = pointer(ledger.Date(command.String(name("point"))))
		}
		if intervalSet {
			row, err := parseCSVValues(name("interval"), []string{command.String(name("interval"))}, 3, 3)
			if err != nil {
				return ledger.ForecastValue{}, err
			}
			bp, err := strconv.ParseInt(row[0][2], 10, 32)
			if err != nil {
				return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "interval credibility must be an integer", err)
			}
			value.Interval = &ledger.DateInterval{Lower: ledger.Date(row[0][0]), Upper: ledger.Date(row[0][1]), CredibilityBP: ledger.BasisPoints(bp)}
		}
		if quantilesSet {
			rows, err := parseCSVValues(name("quantile"), command.StringSlice(name("quantile")), 2, 2)
			if err != nil {
				return ledger.ForecastValue{}, err
			}
			quantiles := make([]ledger.DateQuantile, 0, len(rows))
			for _, row := range rows {
				bp, err := strconv.ParseInt(row[0], 10, 32)
				if err != nil {
					return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "quantile probability must be an integer", err)
				}
				quantiles = append(quantiles, ledger.DateQuantile{ProbabilityBP: ledger.BasisPoints(bp), Value: ledger.Date(row[1])})
			}
			value.Quantiles = &quantiles
		}
		return ledger.ForecastValue{Date: value}, nil
	default:
		return ledger.ForecastValue{}, app.NewError(app.CodeUsage, "unsupported --"+name("value-kind")+" value", nil)
	}
}

func buildInitialForecast(operationContext context.Context, command *urfavecli.Command, stdin io.Reader) (*service.InitialForecastInput, error) {
	if !command.IsSet("initial-forecast") {
		for _, name := range allMappedFlags(initialForecastFlags()) {
			if name != "initial-forecast" && command.IsSet(name) {
				return nil, app.NewError(app.CodeUsage, "--"+name+" requires --initial-forecast", nil)
			}
		}
		return nil, nil
	}
	if err := requireDirectFlags(command, "initial-forecast"); err != nil {
		return nil, err
	}
	visibility := ledger.ForecastVisibility(command.String("initial-visibility"))
	if visibility == ledger.VisibilitySealed {
		for _, name := range []string{"initial-rationale", "initial-key-factor", "initial-comment", "initial-value-kind", "initial-probability-bp", "initial-choice-probability", "initial-point", "initial-interval", "initial-quantile"} {
			if command.IsSet(name) {
				return nil, app.NewError(app.CodeUsage, "--"+name+" cannot be used for a sealed initial forecast; put private values in --initial-secret-input", nil)
			}
		}
		if err := requireDirectFlags(command, "initial-secret-input"); err != nil {
			return nil, err
		}
		var private service.SealedForecastPrivateInput
		if err := decodePrivateOperationInputForArgument(operationContext, command.String("initial-secret-input"), stdin, service.InputSchemaForecastSealPrivate, &private, "--initial-secret-input"); err != nil {
			return nil, err
		}
		return &service.InitialForecastInput{
			Visibility: visibility, ID: ledger.Slug(command.String("initial-forecast")), ForecastedAt: ledger.Timestamp(command.String("initial-forecasted-at")),
			RecordedAt: optionalTimestampValue(command, "initial-recorded-at"), Value: private.Value, Rationale: &private.Rationale,
			KeyFactors: &private.KeyFactors, Comment: &private.Comment, PublicNote: optionalStringValue(command, "initial-public-note"),
		}, nil
	}
	if visibility != ledger.VisibilityPublic {
		return nil, app.NewError(app.CodeUsage, "--initial-visibility must be public or sealed", nil)
	}
	if command.IsSet("initial-secret-input") {
		return nil, app.NewError(app.CodeUsage, "--initial-secret-input is only valid for a sealed initial forecast", nil)
	}
	value, err := buildForecastValue(command, "initial")
	if err != nil {
		return nil, err
	}
	input := &service.InitialForecastInput{
		Visibility: visibility, ID: ledger.Slug(command.String("initial-forecast")), ForecastedAt: ledger.Timestamp(command.String("initial-forecasted-at")),
		RecordedAt: optionalTimestampValue(command, "initial-recorded-at"), Value: value,
		Rationale: optionalStringValue(command, "initial-rationale"), Comment: optionalStringValue(command, "initial-comment"), PublicNote: optionalStringValue(command, "initial-public-note"),
	}
	if command.IsSet("initial-key-factor") {
		values := command.StringSlice("initial-key-factor")
		input.KeyFactors = &values
	}
	return input, nil
}

func buildQuestionAddInput(operationContext context.Context, command *urfavecli.Command, stdin io.Reader) (service.QuestionAddInput, error) {
	if err := requireDirectFlags(command, "title", "resolution-criteria", "expected-resolution-at"); err != nil {
		return service.QuestionAddInput{}, err
	}
	options, err := parseOptions(command, "option")
	if err != nil {
		return service.QuestionAddInput{}, err
	}
	platformRefs, err := parsePlatformRefs(command, "platform-ref")
	if err != nil {
		return service.QuestionAddInput{}, err
	}
	var unit *ledger.Unit
	if command.IsSet("unit-name") || command.IsSet("unit-symbol") || command.IsSet("unit-ucum-code") {
		if err := requireDirectFlags(command, "unit-name"); err != nil {
			return service.QuestionAddInput{}, err
		}
		unit = &ledger.Unit{Name: command.String("unit-name"), Symbol: optionalStringValue(command, "unit-symbol"), UCUMCode: optionalStringValue(command, "unit-ucum-code")}
	}
	window := ledger.ForecastWindow{}
	if command.IsSet("opens-at") {
		window.OpensAt = ledger.Timestamp(command.String("opens-at"))
	}
	input := service.QuestionAddInput{
		Title: command.String("title"), ResolutionCriteria: command.String("resolution-criteria"), CreatedAt: optionalTimestampValue(command, "created-at"),
		ForecastWindow: window, ExpectedResolutionAt: ledger.Timestamp(command.String("expected-resolution-at")), Options: options, Unit: unit,
		PlatformRefs: platformRefs, Notes: optionalStringValue(command, "notes"),
	}
	if command.IsSet("tag") {
		values := command.StringSlice("tag")
		tags := make([]ledger.Slug, 0, len(values))
		for _, value := range values {
			tags = append(tags, ledger.Slug(value))
		}
		input.Tags = &tags
	}
	input.InitialForecast, err = buildInitialForecast(operationContext, command, stdin)
	return input, err
}

func buildQuestionPatchInput(command *urfavecli.Command) (service.QuestionPatchInput, error) {
	input := service.QuestionPatchInput{}
	if command.IsSet("title") {
		input.Title = service.Optional[string]{Set: true, Value: command.String("title")}
	}
	if command.IsSet("resolution-criteria") {
		input.ResolutionCriteria = service.Optional[string]{Set: true, Value: command.String("resolution-criteria")}
	}
	if command.IsSet("opens-at") && command.Bool("clear-forecast-window") {
		return input, app.NewError(app.CodeUsage, "--opens-at cannot be combined with --clear-forecast-window", nil)
	}
	if command.Bool("clear-forecast-window") {
		input.ForecastWindow = service.Optional[service.ForecastWindowPatchInput]{Set: true, Null: true}
	} else if command.IsSet("opens-at") {
		input.ForecastWindow = service.Optional[service.ForecastWindowPatchInput]{Set: true, Value: service.ForecastWindowPatchInput{OpensAt: service.Optional[ledger.Timestamp]{Set: true, Value: ledger.Timestamp(command.String("opens-at"))}}}
	}
	if command.IsSet("expected-resolution-at") {
		input.ExpectedResolutionAt = service.Optional[ledger.Timestamp]{Set: true, Value: ledger.Timestamp(command.String("expected-resolution-at"))}
	}
	if command.IsSet("platform-ref") && command.Bool("clear-platform-refs") {
		return input, app.NewError(app.CodeUsage, "--platform-ref cannot be combined with --clear-platform-refs", nil)
	}
	if command.Bool("clear-platform-refs") {
		input.PlatformRefs = service.Optional[[]ledger.PlatformRef]{Set: true, Null: true}
	} else if command.IsSet("platform-ref") {
		values, err := parsePlatformRefs(command, "platform-ref")
		if err != nil {
			return input, err
		}
		input.PlatformRefs = service.Optional[[]ledger.PlatformRef]{Set: true, Value: *values}
	}
	if command.IsSet("tag") && command.Bool("clear-tags") {
		return input, app.NewError(app.CodeUsage, "--tag cannot be combined with --clear-tags", nil)
	}
	if command.Bool("clear-tags") {
		input.Tags = service.Optional[[]ledger.Slug]{Set: true, Null: true}
	} else if command.IsSet("tag") {
		values := command.StringSlice("tag")
		tags := make([]ledger.Slug, 0, len(values))
		for _, value := range values {
			tags = append(tags, ledger.Slug(value))
		}
		input.Tags = service.Optional[[]ledger.Slug]{Set: true, Value: tags}
	}
	var err error
	input.Notes, err = patchString(command, "notes", "clear-notes")
	if err != nil {
		return input, err
	}
	if command.IsSet("status") {
		input.Status = service.Optional[ledger.QuestionStatus]{Set: true, Value: ledger.QuestionStatus(command.String("status"))}
	}
	if !input.Title.Set && !input.ResolutionCriteria.Set && !input.ForecastWindow.Set && !input.ExpectedResolutionAt.Set && !input.PlatformRefs.Set && !input.Tags.Set && !input.Notes.Set && !input.Status.Set {
		return input, app.NewError(app.CodeUsage, "at least one question authoring flag is required", nil)
	}
	return input, nil
}

func buildForecastCreateInput(command *urfavecli.Command) (service.ForecastCreateInput, error) {
	if err := requireDirectFlags(command, "value-kind"); err != nil {
		return service.ForecastCreateInput{}, err
	}
	value, err := buildForecastValue(command, "")
	if err != nil {
		return service.ForecastCreateInput{}, err
	}
	input := service.ForecastCreateInput{
		ForecastedAt: ledger.Timestamp(command.String("forecasted-at")), RecordedAt: optionalTimestampValue(command, "recorded-at"), Value: value,
		Rationale: optionalStringValue(command, "rationale"), Comment: optionalStringValue(command, "comment"), PublicNote: optionalStringValue(command, "public-note"),
	}
	if command.IsSet("key-factor") {
		values := command.StringSlice("key-factor")
		input.KeyFactors = &values
	}
	if command.IsSet("supersedes-forecast") {
		input.SupersedesForecastID = pointer(ledger.Slug(command.String("supersedes-forecast")))
	}
	return input, nil
}

func buildSealedForecastInput(operationContext context.Context, command *urfavecli.Command, stdin io.Reader) (service.SealedForecastInput, error) {
	if err := requireDirectFlags(command, "secret-input"); err != nil {
		return service.SealedForecastInput{}, err
	}
	var private service.SealedForecastPrivateInput
	if err := decodePrivateOperationInputForArgument(operationContext, command.String("secret-input"), stdin, service.InputSchemaForecastSealPrivate, &private, "--secret-input"); err != nil {
		return service.SealedForecastInput{}, err
	}
	input := service.SealedForecastInput{
		ForecastedAt: ledger.Timestamp(command.String("forecasted-at")), RecordedAt: optionalTimestampValue(command, "recorded-at"),
		Value: private.Value, Rationale: private.Rationale, KeyFactors: private.KeyFactors, Comment: private.Comment,
		PublicNote: optionalStringValue(command, "public-note"),
	}
	if command.IsSet("supersedes-forecast") {
		input.SupersedesForecastID = pointer(ledger.Slug(command.String("supersedes-forecast")))
	}
	return input, nil
}

func parseSources(command *urfavecli.Command) ([]service.EvidenceSourceInput, error) {
	rows, err := parseCSVValues("source", command.StringSlice("source"), 3, 6)
	if err != nil {
		return nil, err
	}
	values := make([]service.EvidenceSourceInput, 0, len(rows))
	for _, row := range rows {
		value := service.EvidenceSourceInput{Title: row[0], URL: row[1], RetrievedAt: ledger.Timestamp(row[2])}
		if len(row) > 3 && row[3] != "" {
			value.Publisher = pointer(row[3])
		}
		if len(row) > 4 && row[4] != "" {
			value.PublishedAt = pointer(ledger.Timestamp(row[4]))
		}
		if len(row) > 5 && row[5] != "" {
			value.ContentSHA256 = pointer(ledger.Hex32(row[5]))
		}
		values = append(values, value)
	}
	return values, nil
}

func buildResolutionInput(command *urfavecli.Command) (service.ResolutionInput, error) {
	if command.IsSet("outcome") == command.IsSet("outcome-boolean") {
		return service.ResolutionInput{}, app.NewError(app.CodeUsage, "use exactly one of --outcome or --outcome-boolean", nil)
	}
	if err := requireDirectFlags(command, "outcome-known-at"); err != nil {
		return service.ResolutionInput{}, err
	}
	sources, err := parseSources(command)
	if err != nil {
		return service.ResolutionInput{}, err
	}
	if len(sources) == 0 {
		return service.ResolutionInput{}, app.NewError(app.CodeUsage, "at least one --source is required", nil)
	}
	input := service.ResolutionInput{OutcomeKnownAt: ledger.Timestamp(command.String("outcome-known-at")), RecordedAt: optionalTimestampValue(command, "recorded-at"), Sources: sources, Notes: optionalStringValue(command, "notes")}
	if command.IsSet("outcome-boolean") {
		input.Outcome.Boolean = pointer(command.Bool("outcome-boolean"))
	} else {
		input.Outcome.Text = pointer(command.String("outcome"))
	}
	return input, nil
}

func buildReasonInput(command *urfavecli.Command) (string, *ledger.Timestamp, []service.EvidenceSourceInput, error) {
	if err := requireDirectFlags(command, "reason"); err != nil {
		return "", nil, nil, err
	}
	sources, err := parseSources(command)
	if err != nil {
		return "", nil, nil, err
	}
	return command.String("reason"), optionalTimestampValue(command, "recorded-at"), sources, nil
}

func allMappedFlags(flags []urfavecli.Flag) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		names := flag.Names()
		if len(names) > 0 {
			result = append(result, names[0])
		}
	}
	return result
}
