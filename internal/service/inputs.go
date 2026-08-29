package service

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/chaoscondensate/cli/internal/ledger"
)

// Optional distinguishes an omitted patch field from an explicit JSON null.
// A present non-null value is stored in Value.
type Optional[T any] struct {
	Set   bool
	Null  bool
	Value T
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Null = true
		var zero T
		o.Value = zero
		return nil
	}
	o.Null = false
	return json.Unmarshal(data, &o.Value)
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Set || o.Null {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

type InitialForecastInput struct {
	Visibility           ledger.ForecastVisibility `json:"visibility"`
	ID                   ledger.Slug               `json:"id"`
	ForecastedAt         ledger.Timestamp          `json:"forecasted_at"`
	RecordedAt           *ledger.Timestamp         `json:"recorded_at,omitempty"`
	Value                ledger.ForecastValue      `json:"value"`
	Rationale            *string                   `json:"rationale,omitempty"`
	KeyFactors           *[]string                 `json:"key_factors,omitempty"`
	Comment              *string                   `json:"comment,omitempty"`
	PublicNote           *string                   `json:"public_note,omitempty"`
	SupersedesForecastID *ledger.Slug              `json:"supersedes_forecast_id,omitempty"`
}

type InitialQuestionInput struct {
	ID                   ledger.Slug           `json:"id"`
	Title                string                `json:"title"`
	Type                 ledger.QuestionType   `json:"type"`
	ResolutionCriteria   string                `json:"resolution_criteria"`
	CreatedAt            *ledger.Timestamp     `json:"created_at,omitempty"`
	ForecastWindow       ledger.ForecastWindow `json:"forecast_window"`
	ExpectedResolutionAt ledger.Timestamp      `json:"expected_resolution_at"`
	Options              *[]ledger.Option      `json:"options,omitempty"`
	Unit                 *ledger.Unit          `json:"unit,omitempty"`
	PlatformRefs         *[]ledger.PlatformRef `json:"platform_refs,omitempty"`
	Tags                 *[]ledger.Slug        `json:"tags,omitempty"`
	Notes                *string               `json:"notes,omitempty"`
	InitialForecast      *InitialForecastInput `json:"initial_forecast,omitempty"`
}

type InitInput struct {
	Title       *string                         `json:"title,omitempty"`
	Description *string                         `json:"description,omitempty"`
	CreatedAt   *ledger.Timestamp               `json:"created_at,omitempty"`
	Contact     *ledger.Contact                 `json:"contact,omitempty"`
	Profiles    *[]ledger.Profile               `json:"profiles,omitempty"`
	Members     *[]ledger.Member                `json:"members,omitempty"`
	Platforms   map[ledger.Slug]ledger.Platform `json:"platforms,omitempty"`
	Question    *InitialQuestionInput           `json:"question,omitempty"`
}

type ForecasterMetadataPatchInput struct {
	Kind     Optional[ledger.ForecasterKind] `json:"kind"`
	Name     Optional[string]                `json:"name"`
	Contact  Optional[ledger.Contact]        `json:"contact"`
	Profiles Optional[[]ledger.Profile]      `json:"profiles"`
	Members  Optional[[]ledger.Member]       `json:"members"`
}

type RootMetadataPatchInput struct {
	Title           Optional[string]                       `json:"title"`
	Description     Optional[string]                       `json:"description"`
	DefaultTimezone Optional[string]                       `json:"default_timezone"`
	Forecaster      Optional[ForecasterMetadataPatchInput] `json:"forecaster"`
}

type PlatformCreateInput struct {
	Name    string                  `json:"name"`
	Kind    ledger.PlatformKind     `json:"kind"`
	URL     *string                 `json:"url,omitempty"`
	Account *ledger.PlatformAccount `json:"account,omitempty"`
}

type PlatformAccountPatchInput struct {
	Username   Optional[string] `json:"username"`
	UserID     Optional[string] `json:"user_id"`
	ProfileURL Optional[string] `json:"profile_url"`
}

type PlatformPatchInput struct {
	Name    Optional[string]                    `json:"name"`
	Kind    Optional[ledger.PlatformKind]       `json:"kind"`
	URL     Optional[string]                    `json:"url"`
	Account Optional[PlatformAccountPatchInput] `json:"account"`
}

// QuestionAddInput deliberately excludes type. The CLI's required scalar
// --type is normalized into NormalizedQuestionCreate before validation.
type QuestionAddInput struct {
	Title                string                `json:"title"`
	ResolutionCriteria   string                `json:"resolution_criteria"`
	CreatedAt            *ledger.Timestamp     `json:"created_at,omitempty"`
	ForecastWindow       ledger.ForecastWindow `json:"forecast_window"`
	ExpectedResolutionAt ledger.Timestamp      `json:"expected_resolution_at"`
	Options              *[]ledger.Option      `json:"options,omitempty"`
	Unit                 *ledger.Unit          `json:"unit,omitempty"`
	PlatformRefs         *[]ledger.PlatformRef `json:"platform_refs,omitempty"`
	Tags                 *[]ledger.Slug        `json:"tags,omitempty"`
	Notes                *string               `json:"notes,omitempty"`
	InitialForecast      *InitialForecastInput `json:"initial_forecast,omitempty"`
}

type NormalizedQuestionCreate struct {
	ID    ledger.Slug
	Type  ledger.QuestionType
	Input QuestionAddInput
}

func NormalizeInitialQuestion(input InitialQuestionInput) NormalizedQuestionCreate {
	return NormalizedQuestionCreate{
		ID:   input.ID,
		Type: input.Type,
		Input: QuestionAddInput{
			Title: input.Title, ResolutionCriteria: input.ResolutionCriteria,
			CreatedAt: input.CreatedAt, ForecastWindow: input.ForecastWindow,
			ExpectedResolutionAt: input.ExpectedResolutionAt, Options: input.Options,
			Unit: input.Unit, PlatformRefs: input.PlatformRefs, Tags: input.Tags,
			Notes: input.Notes, InitialForecast: input.InitialForecast,
		},
	}
}

type InitialCreationShape string

const (
	CreationLedgerOnly     InitialCreationShape = "ledger_only"
	CreationQuestionOnly   InitialCreationShape = "question_only"
	CreationPublicForecast InitialCreationShape = "public_forecast"
	CreationSealedForecast InitialCreationShape = "sealed_forecast"
)

func ClassifyInitInput(input InitInput) (InitialCreationShape, error) {
	if input.Question == nil {
		return CreationLedgerOnly, nil
	}
	return classifyInitialForecast(input.Question.InitialForecast)
}

func ClassifyQuestionAddInput(input QuestionAddInput) (InitialCreationShape, error) {
	return classifyInitialForecast(input.InitialForecast)
}

func classifyInitialForecast(input *InitialForecastInput) (InitialCreationShape, error) {
	if input == nil {
		return CreationQuestionOnly, nil
	}
	switch input.Visibility {
	case ledger.VisibilityPublic:
		return CreationPublicForecast, nil
	case ledger.VisibilitySealed:
		return CreationSealedForecast, nil
	default:
		return "", invalidField("initial_forecast.visibility", "initial forecast visibility must be public or sealed")
	}
}

type ForecastWindowPatchInput struct {
	ClosesAt Optional[ledger.Timestamp] `json:"closes_at"`
}

type QuestionPatchInput struct {
	Title                Optional[string]                   `json:"title"`
	ResolutionCriteria   Optional[string]                   `json:"resolution_criteria"`
	ForecastWindow       Optional[ForecastWindowPatchInput] `json:"forecast_window"`
	ExpectedResolutionAt Optional[ledger.Timestamp]         `json:"expected_resolution_at"`
	PlatformRefs         Optional[[]ledger.PlatformRef]     `json:"platform_refs"`
	Tags                 Optional[[]ledger.Slug]            `json:"tags"`
	Notes                Optional[string]                   `json:"notes"`
	Status               Optional[ledger.QuestionStatus]    `json:"status"`
}

type ForecastCreateInput struct {
	ForecastedAt         ledger.Timestamp     `json:"forecasted_at"`
	RecordedAt           *ledger.Timestamp    `json:"recorded_at,omitempty"`
	Value                ledger.ForecastValue `json:"value"`
	Rationale            *string              `json:"rationale,omitempty"`
	KeyFactors           *[]string            `json:"key_factors,omitempty"`
	Comment              *string              `json:"comment,omitempty"`
	PublicNote           *string              `json:"public_note,omitempty"`
	SupersedesForecastID *ledger.Slug         `json:"supersedes_forecast_id,omitempty"`
}

type SealedForecastInput struct {
	ForecastedAt         ledger.Timestamp     `json:"forecasted_at"`
	RecordedAt           *ledger.Timestamp    `json:"recorded_at,omitempty"`
	Value                ledger.ForecastValue `json:"value"`
	Rationale            string               `json:"rationale"`
	KeyFactors           []string             `json:"key_factors"`
	Comment              string               `json:"comment"`
	PublicNote           *string              `json:"public_note,omitempty"`
	SupersedesForecastID *ledger.Slug         `json:"supersedes_forecast_id,omitempty"`
}

type KeyHintUpdateInput struct {
	KeyHint string `json:"key_hint"`
}

type ResolutionOutcome struct {
	Boolean *bool
	Text    *string
}

func (o *ResolutionOutcome) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("true")) || bytes.Equal(trimmed, []byte("false")) {
		var value bool
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		o.Boolean = &value
		o.Text = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return fmt.Errorf("outcome must be a boolean or exact string: %w", err)
	}
	o.Text = &value
	o.Boolean = nil
	return nil
}

func (o ResolutionOutcome) MarshalJSON() ([]byte, error) {
	if o.Boolean != nil && o.Text == nil {
		return json.Marshal(*o.Boolean)
	}
	if o.Text != nil && o.Boolean == nil {
		return json.Marshal(*o.Text)
	}
	return nil, fmt.Errorf("outcome must contain exactly one typed value")
}

type EvidenceSourceInput struct {
	Title         string            `json:"title"`
	URL           string            `json:"url"`
	RetrievedAt   ledger.Timestamp  `json:"retrieved_at"`
	Publisher     *string           `json:"publisher,omitempty"`
	PublishedAt   *ledger.Timestamp `json:"published_at,omitempty"`
	ContentSHA256 *ledger.Hex32     `json:"content_sha256,omitempty"`
}

type ResolutionInput struct {
	Outcome        ResolutionOutcome     `json:"outcome"`
	OutcomeKnownAt ledger.Timestamp      `json:"outcome_known_at"`
	RecordedAt     *ledger.Timestamp     `json:"recorded_at,omitempty"`
	Sources        []EvidenceSourceInput `json:"sources"`
	Notes          *string               `json:"notes,omitempty"`
}

type AnnulInput struct {
	Reason     string                `json:"reason"`
	RecordedAt *ledger.Timestamp     `json:"recorded_at,omitempty"`
	Sources    []EvidenceSourceInput `json:"sources,omitempty"`
}

type DisputeInput struct {
	Reason     string                `json:"reason"`
	RecordedAt *ledger.Timestamp     `json:"recorded_at,omitempty"`
	Sources    []EvidenceSourceInput `json:"sources,omitempty"`
}

type PublicationBuildInput struct {
	File   string `json:"file"`
	Output string `json:"output"`
	DryRun bool   `json:"dry_run,omitempty"`
}

type PublicationVerifyInput struct {
	File            string  `json:"file"`
	Manifest        string  `json:"manifest"`
	Online          bool    `json:"online,omitempty"`
	Offline         bool    `json:"offline,omitempty"`
	BitcoinCore     *string `json:"bitcoin_core,omitempty"`
	BitcoinAuthFile *string `json:"bitcoin_auth_file,omitempty"`
}
