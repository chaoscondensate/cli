package service

import (
	"encoding/json"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
)

type QuestionMutation struct {
	Ledger               *ledger.Ledger
	Patches              []document.PatchOperation
	TargetCoveredChanged bool
	AffectedForecastIDs  []ledger.Slug
	PriorStatus          ledger.QuestionStatus
}

type IntegrityCounts struct {
	Unanchored int `json:"unanchored"`
	Pending    int `json:"pending"`
	Verified   int `json:"verified"`
	Failed     int `json:"failed"`
}

type QuestionSummary struct {
	ID                   ledger.Slug           `json:"id"`
	Title                string                `json:"title"`
	Type                 ledger.QuestionType   `json:"type"`
	Status               ledger.QuestionStatus `json:"status"`
	ForecastWindow       ledger.ForecastWindow `json:"forecast_window"`
	ExpectedResolutionAt ledger.Timestamp      `json:"expected_resolution_at"`
	ForecastCount        int                   `json:"forecast_count"`
	Integrity            IntegrityCounts       `json:"integrity"`
}

type QuestionForecastSummary struct {
	Summary    ForecastSummary `json:"summary"`
	PublicNote *string         `json:"public_note,omitempty"`
	Commitment *CommitmentView `json:"commitment,omitempty"`
}

type QuestionView struct {
	ID                   ledger.Slug               `json:"id"`
	Title                string                    `json:"title"`
	Type                 ledger.QuestionType       `json:"type"`
	Status               ledger.QuestionStatus     `json:"status"`
	ResolutionCriteria   string                    `json:"resolution_criteria"`
	CreatedAt            ledger.Timestamp          `json:"created_at"`
	ForecastWindow       ledger.ForecastWindow     `json:"forecast_window"`
	ExpectedResolutionAt ledger.Timestamp          `json:"expected_resolution_at"`
	Options              *[]ledger.Option          `json:"options,omitempty"`
	Unit                 *ledger.Unit              `json:"unit,omitempty"`
	PlatformRefs         *[]ledger.PlatformRef     `json:"platform_refs,omitempty"`
	Tags                 *[]ledger.Slug            `json:"tags,omitempty"`
	Notes                *string                   `json:"notes,omitempty"`
	Resolution           *ledger.Resolution        `json:"resolution,omitempty"`
	Forecasts            []QuestionForecastSummary `json:"forecasts"`
}

func BuildQuestionAddPublic(model *ledger.Ledger, input NormalizedQuestionCreate, observedAt ledger.Timestamp) (QuestionMutation, error) {
	prospective, err := BuildQuestionWithInitialPublicForecast(model, input, observedAt)
	if err != nil {
		return QuestionMutation{}, err
	}
	question := prospective.Questions[len(prospective.Questions)-1]
	value, err := jsonPatchValue(question)
	if err != nil {
		return QuestionMutation{}, err
	}
	return QuestionMutation{Ledger: prospective, Patches: []document.PatchOperation{{Kind: document.PatchAdd, Pointer: "/questions/-", Value: value}}}, nil
}

func BuildQuestionUpdate(model *ledger.Ledger, id ledger.Slug, input QuestionPatchInput) (QuestionMutation, error) {
	position, question, err := selectQuestion(model, id)
	if err != nil {
		return QuestionMutation{}, err
	}
	prospective, err := cloneLedger(model)
	if err != nil {
		return QuestionMutation{}, err
	}
	updated := &prospective.Questions[position]
	var patches []document.PatchOperation
	covered := false
	base := "/questions/" + strconv.Itoa(position)
	if input.Title.Set {
		if input.Title.Null || strings.TrimSpace(input.Title.Value) == "" {
			return QuestionMutation{}, invalidField("title", "title cannot be null or empty")
		}
		if input.Title.Value != question.Title {
			updated.Title = input.Title.Value
			patches = append(patches, replacePatch(base+"/title", input.Title.Value))
			covered = true
		}
	}
	if input.ResolutionCriteria.Set {
		if input.ResolutionCriteria.Null || strings.TrimSpace(input.ResolutionCriteria.Value) == "" {
			return QuestionMutation{}, invalidField("resolution_criteria", "resolution criteria cannot be null or empty")
		}
		if input.ResolutionCriteria.Value != question.ResolutionCriteria {
			updated.ResolutionCriteria = input.ResolutionCriteria.Value
			patches = append(patches, replacePatch(base+"/resolution_criteria", input.ResolutionCriteria.Value))
			covered = true
		}
	}
	if input.ForecastWindow.Set {
		if input.ForecastWindow.Null || !input.ForecastWindow.Value.ClosesAt.Set || input.ForecastWindow.Value.ClosesAt.Null {
			return QuestionMutation{}, invalidField("forecast_window.closes_at", "closing time is required")
		}
		closing := input.ForecastWindow.Value.ClosesAt.Value
		if closing != question.ForecastWindow.ClosesAt {
			updated.ForecastWindow.ClosesAt = closing
			patches = append(patches, replacePatch(base+"/forecast_window/closes_at", closing))
			covered = true
		}
	}
	if input.ExpectedResolutionAt.Set {
		if input.ExpectedResolutionAt.Null {
			return QuestionMutation{}, invalidField("expected_resolution_at", "expected resolution time cannot be null")
		}
		if input.ExpectedResolutionAt.Value != question.ExpectedResolutionAt {
			updated.ExpectedResolutionAt = input.ExpectedResolutionAt.Value
			patches = append(patches, replacePatch(base+"/expected_resolution_at", input.ExpectedResolutionAt.Value))
			covered = true
		}
	}
	if input.PlatformRefs.Set {
		if input.PlatformRefs.Null {
			updated.PlatformRefs = nil
		} else {
			updated.PlatformRefs = clonePlatformRefsSlice(input.PlatformRefs.Value)
		}
		patches = append(patches, optionalFieldPatch(base+"/platform_refs", question.PlatformRefs != nil, updated.PlatformRefs))
	}
	if input.Tags.Set {
		if input.Tags.Null {
			updated.Tags = nil
		} else {
			updated.Tags = cloneSlugsSlice(input.Tags.Value)
		}
		patches = append(patches, optionalFieldPatch(base+"/tags", question.Tags != nil, updated.Tags))
	}
	if input.Notes.Set {
		if input.Notes.Null {
			updated.Notes = nil
		} else {
			value := input.Notes.Value
			updated.Notes = &value
		}
		patches = append(patches, optionalFieldPatch(base+"/notes", question.Notes != nil, updated.Notes))
	}
	if input.Status.Set {
		if input.Status.Null || !isUnresolvedStatus(input.Status.Value) {
			return QuestionMutation{}, invalidField("status", "question update accepts only open, closed, or awaiting_resolution")
		}
		if !isUnresolvedStatus(question.Status) || question.Resolution != nil {
			return QuestionMutation{}, app.NewError(app.CodeConflict, "terminal question status must be changed through resolve, annul, or dispute", nil)
		}
		if input.Status.Value != question.Status {
			updated.Status = input.Status.Value
			patches = append(patches, replacePatch(base+"/status", input.Status.Value))
		}
	}
	opening := updated.CreatedAt
	if updated.ForecastWindow.OpensAt != nil {
		opening = *updated.ForecastWindow.OpensAt
	}
	if err := ValidateChronology(opening, "forecast_window.opens_at", updated.ForecastWindow.ClosesAt, "forecast_window.closes_at", true); err != nil {
		return QuestionMutation{}, err
	}
	if err := ValidateChronology(updated.ForecastWindow.ClosesAt, "forecast_window.closes_at", updated.ExpectedResolutionAt, "expected_resolution_at", true); err != nil {
		return QuestionMutation{}, err
	}
	for _, forecast := range updated.Forecasts {
		if err := ValidateChronology(opening, "forecast_window.opens_at", forecast.ForecastedAt, "forecasted_at", true); err != nil {
			return QuestionMutation{}, app.NewError(app.CodeConflict, "changed forecast window would exclude an existing forecast", err)
		}
		if err := ValidateChronology(forecast.ForecastedAt, "forecasted_at", updated.ForecastWindow.ClosesAt, "forecast_window.closes_at", true); err != nil {
			return QuestionMutation{}, app.NewError(app.CodeConflict, "changed forecast window would exclude an existing forecast", err)
		}
	}
	if covered && questionHasTargetMetadata(question) {
		return QuestionMutation{}, frozenQuestionConflict(id)
	}
	patches = removeNoopQuestionPatches(question, *updated, patches, base)
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return QuestionMutation{}, err
	}
	ids := make([]ledger.Slug, len(question.Forecasts))
	for index := range question.Forecasts {
		ids[index] = question.Forecasts[index].ID
	}
	return QuestionMutation{Ledger: prospective, Patches: patches, TargetCoveredChanged: covered, AffectedForecastIDs: ids, PriorStatus: question.Status}, nil
}

func ListQuestions(model *ledger.Ledger) ([]QuestionSummary, error) {
	if model == nil {
		return nil, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	items := make([]QuestionSummary, len(model.Questions))
	for index, question := range model.Questions {
		items[index] = summarizeQuestion(question)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func ShowQuestion(model *ledger.Ledger, id ledger.Slug) (QuestionView, error) {
	_, question, err := selectQuestion(model, id)
	if err != nil {
		return QuestionView{}, err
	}
	view := QuestionView{
		ID: question.ID, Title: question.Title, Type: question.Type, Status: question.Status,
		ResolutionCriteria: question.ResolutionCriteria, CreatedAt: question.CreatedAt,
		ForecastWindow: question.ForecastWindow, ExpectedResolutionAt: question.ExpectedResolutionAt,
		Options: cloneOptions(question.Options), Unit: cloneUnit(question.Unit), PlatformRefs: clonePlatformRefs(question.PlatformRefs),
		Tags: cloneSlugs(question.Tags), Notes: cloneString(question.Notes), Resolution: cloneResolution(question.Resolution),
		Forecasts: make([]QuestionForecastSummary, len(question.Forecasts)),
	}
	for index, forecast := range question.Forecasts {
		view.Forecasts[index] = QuestionForecastSummary{Summary: summarizeForecast(forecast), PublicNote: cloneString(forecast.PublicNote), Commitment: commitmentView(forecast.Commitment)}
		if view.Forecasts[index].Commitment != nil {
			view.Forecasts[index].Commitment.Encryption.Nonce = ""
			view.Forecasts[index].Commitment.Encryption.Ciphertext = ""
		}
	}
	return view, nil
}

func BuildQuestionResolve(model *ledger.Ledger, id ledger.Slug, input ResolutionInput, observedAt ledger.Timestamp) (QuestionMutation, error) {
	position, question, err := selectQuestion(model, id)
	if err != nil {
		return QuestionMutation{}, err
	}
	if question.Status != ledger.QuestionClosed && question.Status != ledger.QuestionAwaitingResolution && question.Status != ledger.QuestionDisputed {
		return QuestionMutation{}, app.NewError(app.CodeConflict, "question must be closed, awaiting resolution, or disputed before resolution", nil)
	}
	if len(input.Sources) == 0 {
		return QuestionMutation{}, invalidField("sources", "at least one evidence source is required")
	}
	outcome, err := validateResolutionOutcome(question, input.Outcome)
	if err != nil {
		return QuestionMutation{}, err
	}
	recordedAt := observedAt
	if input.RecordedAt != nil {
		recordedAt = *input.RecordedAt
	}
	if err := ValidateChronology(input.OutcomeKnownAt, "outcome_known_at", recordedAt, "recorded_at", true); err != nil {
		return QuestionMutation{}, err
	}
	sources, err := buildResolutionSources(input.Sources)
	if err != nil {
		return QuestionMutation{}, err
	}
	resolution := ledger.Resolution{Resolved: &ledger.ResolvedResolution{Status: ledger.ResolutionResolved, Outcome: outcome, OutcomeKnownAt: input.OutcomeKnownAt, RecordedAt: recordedAt, Sources: sources, Notes: cloneString(input.Notes)}}
	return buildQuestionTerminalMutation(model, position, question, ledger.QuestionResolved, resolution)
}

func BuildQuestionAnnul(model *ledger.Ledger, id ledger.Slug, input AnnulInput, observedAt ledger.Timestamp) (QuestionMutation, error) {
	position, question, err := selectQuestion(model, id)
	if err != nil {
		return QuestionMutation{}, err
	}
	if strings.TrimSpace(input.Reason) == "" {
		return QuestionMutation{}, invalidField("reason", "annulment reason must not be empty")
	}
	recordedAt := observedAt
	if input.RecordedAt != nil {
		recordedAt = *input.RecordedAt
	}
	if _, err := ParseTimestamp(recordedAt, "recorded_at"); err != nil {
		return QuestionMutation{}, err
	}
	sources, err := buildResolutionSources(input.Sources)
	if err != nil {
		return QuestionMutation{}, err
	}
	resolution := ledger.Resolution{NonResolved: &ledger.NonResolvedResolution{Status: ledger.ResolutionAnnulled, Reason: input.Reason, RecordedAt: recordedAt, Sources: optionalSources(sources, input.Sources != nil)}}
	return buildQuestionTerminalMutation(model, position, question, ledger.QuestionAnnulled, resolution)
}

func BuildQuestionDispute(model *ledger.Ledger, id ledger.Slug, input DisputeInput, observedAt ledger.Timestamp) (QuestionMutation, error) {
	position, question, err := selectQuestion(model, id)
	if err != nil {
		return QuestionMutation{}, err
	}
	if question.Status != ledger.QuestionResolved && question.Status != ledger.QuestionAnnulled {
		return QuestionMutation{}, app.NewError(app.CodeConflict, "only a resolved or annulled question can be disputed", nil)
	}
	if strings.TrimSpace(input.Reason) == "" {
		return QuestionMutation{}, invalidField("reason", "dispute reason must not be empty")
	}
	recordedAt := observedAt
	if input.RecordedAt != nil {
		recordedAt = *input.RecordedAt
	}
	if _, err := ParseTimestamp(recordedAt, "recorded_at"); err != nil {
		return QuestionMutation{}, err
	}
	sources, err := buildResolutionSources(input.Sources)
	if err != nil {
		return QuestionMutation{}, err
	}
	resolution := ledger.Resolution{NonResolved: &ledger.NonResolvedResolution{Status: ledger.ResolutionDisputed, Reason: input.Reason, RecordedAt: recordedAt, Sources: optionalSources(sources, input.Sources != nil)}}
	return buildQuestionTerminalMutation(model, position, question, ledger.QuestionDisputed, resolution)
}

func buildQuestionTerminalMutation(model *ledger.Ledger, position int, question ledger.Question, status ledger.QuestionStatus, resolution ledger.Resolution) (QuestionMutation, error) {
	prospective, err := cloneLedger(model)
	if err != nil {
		return QuestionMutation{}, err
	}
	prospective.Questions[position].Status = status
	prospective.Questions[position].Resolution = &resolution
	if err := ValidateProspectiveLedgerModel(prospective); err != nil {
		return QuestionMutation{}, err
	}
	base := "/questions/" + strconv.Itoa(position)
	resolutionValue, err := jsonPatchValue(resolution)
	if err != nil {
		return QuestionMutation{}, err
	}
	resolutionKind := document.PatchAdd
	if question.Resolution != nil {
		resolutionKind = document.PatchReplace
	}
	return QuestionMutation{Ledger: prospective, PriorStatus: question.Status, Patches: []document.PatchOperation{
		replacePatch(base+"/status", status), {Kind: resolutionKind, Pointer: base + "/resolution", Value: resolutionValue},
	}}, nil
}

func summarizeQuestion(question ledger.Question) QuestionSummary {
	result := QuestionSummary{ID: question.ID, Title: question.Title, Type: question.Type, Status: question.Status, ForecastWindow: question.ForecastWindow, ExpectedResolutionAt: question.ExpectedResolutionAt, ForecastCount: len(question.Forecasts)}
	for _, forecast := range question.Forecasts {
		switch integrityStatus(forecast.Integrity) {
		case ledger.IntegrityUnanchored:
			result.Integrity.Unanchored++
		case ledger.IntegrityPending:
			result.Integrity.Pending++
		case ledger.IntegrityVerified:
			result.Integrity.Verified++
		case ledger.IntegrityFailed:
			result.Integrity.Failed++
		}
	}
	return result
}

func validateResolutionOutcome(question ledger.Question, input ResolutionOutcome) (ledger.ResolutionOutcome, error) {
	switch question.Type {
	case ledger.QuestionBinary:
		if input.Boolean == nil || input.Text != nil {
			return ledger.ResolutionOutcome{}, invalidField("outcome", "binary outcome must be true or false")
		}
		value := *input.Boolean
		return ledger.ResolutionOutcome{Binary: &value}, nil
	case ledger.QuestionMultipleChoice:
		if input.Text == nil || input.Boolean != nil {
			return ledger.ResolutionOutcome{}, invalidField("outcome", "multiple-choice outcome must be an option ID")
		}
		for _, option := range *question.Options {
			if string(option.ID) == *input.Text {
				value := *input.Text
				return ledger.ResolutionOutcome{Text: &value}, nil
			}
		}
		return ledger.ResolutionOutcome{}, invalidField("outcome", "multiple-choice outcome is not a current option ID")
	case ledger.QuestionNumeric:
		if input.Text == nil || input.Boolean != nil || !validExactDecimal(ledger.Decimal(*input.Text)) {
			return ledger.ResolutionOutcome{}, invalidField("outcome", "numeric outcome must be an exact decimal string")
		}
	case ledger.QuestionDate:
		if input.Text == nil || input.Boolean != nil || !validFullDate(ledger.Date(*input.Text)) {
			return ledger.ResolutionOutcome{}, invalidField("outcome", "date outcome must be a valid full date string")
		}
	default:
		return ledger.ResolutionOutcome{}, invalidField("outcome", "question type is unsupported")
	}
	value := *input.Text
	return ledger.ResolutionOutcome{Text: &value}, nil
}

func buildResolutionSources(inputs []EvidenceSourceInput) ([]ledger.ResolutionSource, error) {
	result := make([]ledger.ResolutionSource, len(inputs))
	for index, input := range inputs {
		field := "sources." + strconv.Itoa(index)
		if strings.TrimSpace(input.Title) == "" {
			return nil, invalidField(field+".title", "source title must not be empty")
		}
		parsed, err := url.ParseRequestURI(input.URL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, invalidField(field+".url", "source URL must be absolute")
		}
		if _, err := ParseTimestamp(input.RetrievedAt, field+".retrieved_at"); err != nil {
			return nil, err
		}
		if input.PublishedAt != nil {
			if err := ValidateChronology(*input.PublishedAt, field+".published_at", input.RetrievedAt, field+".retrieved_at", true); err != nil {
				return nil, err
			}
		}
		var digest *ledger.Digest
		if input.ContentSHA256 != nil {
			digest = &ledger.Digest{Algorithm: "sha-256", Value: *input.ContentSHA256}
		}
		result[index] = ledger.ResolutionSource{Title: input.Title, Publisher: cloneString(input.Publisher), URL: input.URL, PublishedAt: cloneTimestamp(input.PublishedAt), RetrievedAt: input.RetrievedAt, ContentDigest: digest}
	}
	return result, nil
}

func questionHasTargetMetadata(question ledger.Question) bool {
	for _, forecast := range question.Forecasts {
		if forecast.Integrity.Pending != nil || forecast.Integrity.Verified != nil || forecast.Integrity.Failed != nil && forecast.Integrity.Failed.Target != nil {
			return true
		}
	}
	return false
}

func frozenQuestionConflict(id ledger.Slug) error {
	return app.WithDetails(app.NewError(app.CodeConflict, "target-covered question fields are frozen because forecast evidence exists; annul this question and create a new question with a new ID", nil), map[string]any{"question_id": id, "guidance": "Annul the original question, create a new question, and record the predecessor ID in notes."})
}

func isUnresolvedStatus(status ledger.QuestionStatus) bool {
	return status == ledger.QuestionOpen || status == ledger.QuestionClosed || status == ledger.QuestionAwaitingResolution
}

func replacePatch(pointer string, value any) document.PatchOperation {
	normalized, err := jsonPatchValue(value)
	if err == nil {
		value = normalized
	}
	return document.PatchOperation{Kind: document.PatchReplace, Pointer: pointer, Value: value}
}

func optionalFieldPatch(pointer string, existed bool, value any) document.PatchOperation {
	if value == nil || reflect.ValueOf(value).Kind() == reflect.Pointer && reflect.ValueOf(value).IsNil() {
		return document.PatchOperation{Kind: document.PatchRemove, Pointer: pointer}
	}
	kind := document.PatchAdd
	if existed {
		kind = document.PatchReplace
	}
	normalized, err := jsonPatchValue(value)
	if err == nil {
		value = normalized
	}
	return document.PatchOperation{Kind: kind, Pointer: pointer, Value: value}
}

func removeNoopQuestionPatches(before, after ledger.Question, patches []document.PatchOperation, base string) []document.PatchOperation {
	// Optional patch input may explicitly repeat the current value. Comparing the
	// prospective source values keeps dry-run and commit correctly idempotent.
	result := patches[:0]
	for _, patch := range patches {
		if questionPointerEqual(before, after, strings.TrimPrefix(patch.Pointer, base+"/")) {
			continue
		}
		result = append(result, patch)
	}
	return result
}

func questionPointerEqual(before, after ledger.Question, field string) bool {
	// JSON encoding is acceptable here because both values are already typed and
	// this comparison never becomes persisted output.
	var left, right any
	switch field {
	case "title":
		left, right = before.Title, after.Title
	case "resolution_criteria":
		left, right = before.ResolutionCriteria, after.ResolutionCriteria
	case "forecast_window/closes_at":
		left, right = before.ForecastWindow.ClosesAt, after.ForecastWindow.ClosesAt
	case "expected_resolution_at":
		left, right = before.ExpectedResolutionAt, after.ExpectedResolutionAt
	case "platform_refs":
		left, right = before.PlatformRefs, after.PlatformRefs
	case "tags":
		left, right = before.Tags, after.Tags
	case "notes":
		left, right = before.Notes, after.Notes
	case "status":
		left, right = before.Status, after.Status
	default:
		return false
	}
	leftValue, _ := jsonPatchValue(left)
	rightValue, _ := jsonPatchValue(right)
	return deepEqualJSON(leftValue, rightValue)
}

func deepEqualJSON(left, right any) bool {
	return reflect.DeepEqual(left, right)
}

func clonePlatformRefsSlice(value []ledger.PlatformRef) *[]ledger.PlatformRef {
	copy := append([]ledger.PlatformRef{}, value...)
	return &copy
}
func cloneSlugsSlice(value []ledger.Slug) *[]ledger.Slug {
	copy := append([]ledger.Slug{}, value...)
	return &copy
}
func cloneTimestamp(value *ledger.Timestamp) *ledger.Timestamp {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func optionalSources(value []ledger.ResolutionSource, present bool) *[]ledger.ResolutionSource {
	if !present {
		return nil
	}
	copy := append([]ledger.ResolutionSource{}, value...)
	return &copy
}
func cloneResolution(value *ledger.Resolution) *ledger.Resolution {
	if value == nil {
		return nil
	}
	copy, err := cloneLedgerValue(*value)
	if err != nil {
		return nil
	}
	return &copy
}

func cloneLedgerValue[T any](value T) (T, error) {
	var result T
	encoded, err := json.Marshal(value)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(encoded, &result)
	return result, err
}
