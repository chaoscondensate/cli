package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Timestamp is an RFC 3339 timestamp with seconds and an explicit UTC offset.
// It remains a string so parsing never normalizes the author's original value.
type Timestamp string

// Date is a full-date value in YYYY-MM-DD form.
type Date string

// Decimal is an exact JSON-Schema decimal encoded as a string.
type Decimal string

type Slug string
type RelativePath string
type Hex32 string
type Base64Nonce12 string
type Base64Ciphertext string
type BasisPoints int32
type SchemaVersion string

type Ledger struct {
	Schema          *string            `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	SchemaVersion   SchemaVersion      `json:"schema_version" yaml:"schema_version"`
	LedgerID        Slug               `json:"ledger_id" yaml:"ledger_id"`
	Title           *string            `json:"title,omitempty" yaml:"title,omitempty"`
	Description     *string            `json:"description,omitempty" yaml:"description,omitempty"`
	CreatedAt       Timestamp          `json:"created_at" yaml:"created_at"`
	DefaultTimezone string             `json:"default_timezone" yaml:"default_timezone"`
	Forecaster      Forecaster         `json:"forecaster" yaml:"forecaster"`
	Publication     *LegacyPublication `json:"publication,omitempty" yaml:"publication,omitempty"`
	Platforms       map[Slug]Platform  `json:"platforms" yaml:"platforms"`
	Questions       []Question         `json:"questions" yaml:"questions"`
}

// LegacyPublication represents optional publication metadata retained by the
// upstream v1.1.0 schema. The CLI does not require Git and provides no Git
// automation; this type exists only for schema-compatible document handling.
type LegacyPublication struct {
	History       string       `json:"history" yaml:"history"`
	RepositoryURL string       `json:"repository_url" yaml:"repository_url"`
	DefaultBranch string       `json:"default_branch" yaml:"default_branch"`
	LedgerPath    RelativePath `json:"ledger_path" yaml:"ledger_path"`
}

type ForecasterKind string

const (
	ForecasterIndividual ForecasterKind = "individual"
	ForecasterTeam       ForecasterKind = "team"
)

type Forecaster struct {
	ID       Slug           `json:"id" yaml:"id"`
	Kind     ForecasterKind `json:"kind" yaml:"kind"`
	Name     string         `json:"name" yaml:"name"`
	Contact  *Contact       `json:"contact,omitempty" yaml:"contact,omitempty"`
	Profiles *[]Profile     `json:"profiles,omitempty" yaml:"profiles,omitempty"`
	Members  *[]Member      `json:"members,omitempty" yaml:"members,omitempty"`
}

type Contact struct {
	Email   *string `json:"email,omitempty" yaml:"email,omitempty"`
	Website *string `json:"website,omitempty" yaml:"website,omitempty"`
}

type Profile struct {
	Service  string  `json:"service" yaml:"service"`
	Username *string `json:"username,omitempty" yaml:"username,omitempty"`
	URL      string  `json:"url" yaml:"url"`
}

type Member struct {
	ID       Slug       `json:"id" yaml:"id"`
	Name     string     `json:"name" yaml:"name"`
	Role     *string    `json:"role,omitempty" yaml:"role,omitempty"`
	Profiles *[]Profile `json:"profiles,omitempty" yaml:"profiles,omitempty"`
}

type PlatformKind string

const (
	PlatformScoringMarket PlatformKind = "scoring_platform"
	PlatformPrediction    PlatformKind = "prediction_market"
	PlatformSelfHosted    PlatformKind = "self_hosted"
	PlatformInternal      PlatformKind = "internal"
	PlatformInformal      PlatformKind = "informal"
)

type Platform struct {
	Name    string           `json:"name" yaml:"name"`
	Kind    PlatformKind     `json:"kind" yaml:"kind"`
	URL     *string          `json:"url,omitempty" yaml:"url,omitempty"`
	Account *PlatformAccount `json:"account,omitempty" yaml:"account,omitempty"`
}

type PlatformAccount struct {
	Username   *string `json:"username,omitempty" yaml:"username,omitempty"`
	UserID     *string `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	ProfileURL *string `json:"profile_url,omitempty" yaml:"profile_url,omitempty"`
}

type PlatformRef struct {
	Platform   Slug    `json:"platform" yaml:"platform"`
	QuestionID *string `json:"question_id,omitempty" yaml:"question_id,omitempty"`
	URL        *string `json:"url,omitempty" yaml:"url,omitempty"`
}

type QuestionType string

const (
	QuestionBinary         QuestionType = "binary"
	QuestionMultipleChoice QuestionType = "multiple_choice"
	QuestionNumeric        QuestionType = "numeric"
	QuestionDate           QuestionType = "date"
)

type QuestionStatus string

const (
	QuestionOpen               QuestionStatus = "open"
	QuestionClosed             QuestionStatus = "closed"
	QuestionAwaitingResolution QuestionStatus = "awaiting_resolution"
	QuestionResolved           QuestionStatus = "resolved"
	QuestionAnnulled           QuestionStatus = "annulled"
	QuestionDisputed           QuestionStatus = "disputed"
)

type Question struct {
	ID                   Slug           `json:"id" yaml:"id"`
	Title                string         `json:"title" yaml:"title"`
	Type                 QuestionType   `json:"type" yaml:"type"`
	Status               QuestionStatus `json:"status" yaml:"status"`
	ResolutionCriteria   string         `json:"resolution_criteria" yaml:"resolution_criteria"`
	CreatedAt            Timestamp      `json:"created_at" yaml:"created_at"`
	ForecastWindow       ForecastWindow `json:"forecast_window" yaml:"forecast_window"`
	ExpectedResolutionAt Timestamp      `json:"expected_resolution_at" yaml:"expected_resolution_at"`
	Options              *[]Option      `json:"options,omitempty" yaml:"options,omitempty"`
	Unit                 *Unit          `json:"unit,omitempty" yaml:"unit,omitempty"`
	PlatformRefs         *[]PlatformRef `json:"platform_refs,omitempty" yaml:"platform_refs,omitempty"`
	Tags                 *[]Slug        `json:"tags,omitempty" yaml:"tags,omitempty"`
	Notes                *string        `json:"notes,omitempty" yaml:"notes,omitempty"`
	Forecasts            []Forecast     `json:"forecasts" yaml:"forecasts"`
	Resolution           *Resolution    `json:"resolution,omitempty" yaml:"resolution,omitempty"`
}

type ForecastWindow struct {
	OpensAt  *Timestamp `json:"opens_at,omitempty" yaml:"opens_at,omitempty"`
	ClosesAt Timestamp  `json:"closes_at" yaml:"closes_at"`
}

type Option struct {
	ID    Slug   `json:"id" yaml:"id"`
	Label string `json:"label" yaml:"label"`
}

type Unit struct {
	Name     string  `json:"name" yaml:"name"`
	Symbol   *string `json:"symbol,omitempty" yaml:"symbol,omitempty"`
	UCUMCode *string `json:"ucum_code,omitempty" yaml:"ucum_code,omitempty"`
}

type ForecastVisibility string

const (
	VisibilityPublic   ForecastVisibility = "public"
	VisibilitySealed   ForecastVisibility = "sealed"
	VisibilityRevealed ForecastVisibility = "revealed"
)

type Forecast struct {
	ID                   Slug               `json:"id" yaml:"id"`
	ForecastedAt         Timestamp          `json:"forecasted_at" yaml:"forecasted_at"`
	RecordedAt           Timestamp          `json:"recorded_at" yaml:"recorded_at"`
	Visibility           ForecastVisibility `json:"visibility" yaml:"visibility"`
	Value                *ForecastValue     `json:"value,omitempty" yaml:"value,omitempty"`
	Rationale            *string            `json:"rationale,omitempty" yaml:"rationale,omitempty"`
	KeyFactors           *[]string          `json:"key_factors,omitempty" yaml:"key_factors,omitempty"`
	Comment              *string            `json:"comment,omitempty" yaml:"comment,omitempty"`
	PublicNote           *string            `json:"public_note,omitempty" yaml:"public_note,omitempty"`
	SupersedesForecastID *Slug              `json:"supersedes_forecast_id,omitempty" yaml:"supersedes_forecast_id,omitempty"`
	Commitment           *Commitment        `json:"commitment,omitempty" yaml:"commitment,omitempty"`
	Integrity            Integrity          `json:"integrity" yaml:"integrity"`
}

type ForecastValueKind string

const (
	ValueBinary         ForecastValueKind = "binary"
	ValueMultipleChoice ForecastValueKind = "multiple_choice"
	ValueNumeric        ForecastValueKind = "numeric"
	ValueDate           ForecastValueKind = "date"
)

// ForecastValue is a closed discriminated union. Exactly one field is set
// after unmarshalling a schema-valid document.
type ForecastValue struct {
	Binary         *BinaryValue
	MultipleChoice *MultipleChoiceValue
	Numeric        *NumericValue
	Date           *DateValue
}

type BinaryValue struct {
	Kind          ForecastValueKind `json:"kind" yaml:"kind"`
	ProbabilityBP BasisPoints       `json:"probability_bp" yaml:"probability_bp"`
}

type ChoiceProbability struct {
	OptionID      Slug        `json:"option_id" yaml:"option_id"`
	ProbabilityBP BasisPoints `json:"probability_bp" yaml:"probability_bp"`
}

type MultipleChoiceValue struct {
	Kind          ForecastValueKind   `json:"kind" yaml:"kind"`
	Probabilities []ChoiceProbability `json:"probabilities" yaml:"probabilities"`
}

type NumericInterval struct {
	Lower         Decimal     `json:"lower" yaml:"lower"`
	Upper         Decimal     `json:"upper" yaml:"upper"`
	CredibilityBP BasisPoints `json:"credibility_bp" yaml:"credibility_bp"`
}

type NumericQuantile struct {
	ProbabilityBP BasisPoints `json:"probability_bp" yaml:"probability_bp"`
	Value         Decimal     `json:"value" yaml:"value"`
}

type NumericValue struct {
	Kind      ForecastValueKind  `json:"kind" yaml:"kind"`
	Point     *Decimal           `json:"point,omitempty" yaml:"point,omitempty"`
	Interval  *NumericInterval   `json:"interval,omitempty" yaml:"interval,omitempty"`
	Quantiles *[]NumericQuantile `json:"quantiles,omitempty" yaml:"quantiles,omitempty"`
}

type DateInterval struct {
	Lower         Date        `json:"lower" yaml:"lower"`
	Upper         Date        `json:"upper" yaml:"upper"`
	CredibilityBP BasisPoints `json:"credibility_bp" yaml:"credibility_bp"`
}

type DateQuantile struct {
	ProbabilityBP BasisPoints `json:"probability_bp" yaml:"probability_bp"`
	Value         Date        `json:"value" yaml:"value"`
}

type DateValue struct {
	Kind      ForecastValueKind `json:"kind" yaml:"kind"`
	Point     *Date             `json:"point,omitempty" yaml:"point,omitempty"`
	Interval  *DateInterval     `json:"interval,omitempty" yaml:"interval,omitempty"`
	Quantiles *[]DateQuantile   `json:"quantiles,omitempty" yaml:"quantiles,omitempty"`
}

func (v ForecastValue) MarshalJSON() ([]byte, error) {
	return marshalOne("forecast value", v.Binary, v.MultipleChoice, v.Numeric, v.Date)
}

func (v *ForecastValue) UnmarshalJSON(data []byte) error {
	*v = ForecastValue{}
	var discriminator struct {
		Kind ForecastValueKind `json:"kind"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	switch discriminator.Kind {
	case ValueBinary:
		v.Binary = new(BinaryValue)
		return json.Unmarshal(data, v.Binary)
	case ValueMultipleChoice:
		v.MultipleChoice = new(MultipleChoiceValue)
		return json.Unmarshal(data, v.MultipleChoice)
	case ValueNumeric:
		v.Numeric = new(NumericValue)
		return json.Unmarshal(data, v.Numeric)
	case ValueDate:
		v.Date = new(DateValue)
		return json.Unmarshal(data, v.Date)
	default:
		return fmt.Errorf("unknown forecast value kind %q", discriminator.Kind)
	}
}

type Digest struct {
	Algorithm string `json:"algorithm" yaml:"algorithm"`
	Value     Hex32  `json:"value" yaml:"value"`
}

type ForecastTarget struct {
	Scope            string       `json:"scope" yaml:"scope"`
	Canonicalization string       `json:"canonicalization" yaml:"canonicalization"`
	ArtifactPath     RelativePath `json:"artifact_path" yaml:"artifact_path"`
	Digest           Digest       `json:"digest" yaml:"digest"`
}

type OTSTimestampState string

const (
	OTSPending   OTSTimestampState = "pending"
	OTSConfirmed OTSTimestampState = "confirmed"
)

type OTSTimestamp struct {
	Type               string            `json:"type" yaml:"type"`
	ProofPath          RelativePath      `json:"proof_path" yaml:"proof_path"`
	State              OTSTimestampState `json:"state" yaml:"state"`
	AnchoredBefore     *Timestamp        `json:"anchored_before,omitempty" yaml:"anchored_before,omitempty"`
	BitcoinBlockHeight *int64            `json:"bitcoin_block_height,omitempty" yaml:"bitcoin_block_height,omitempty"`
}

type ExternalAnchorKind string

const (
	AnchorGitCommit             ExternalAnchorKind = "git_commit"
	AnchorURL                   ExternalAnchorKind = "url"
	AnchorNostrEvent            ExternalAnchorKind = "nostr_event"
	AnchorBlockchainTransaction ExternalAnchorKind = "blockchain_transaction"
)

type ExternalAnchor struct {
	Kind       ExternalAnchorKind `json:"kind" yaml:"kind"`
	Value      string             `json:"value" yaml:"value"`
	ObservedAt *Timestamp         `json:"observed_at,omitempty" yaml:"observed_at,omitempty"`
}

type IntegrityStatus string

const (
	IntegrityUnanchored IntegrityStatus = "unanchored"
	IntegrityPending    IntegrityStatus = "pending"
	IntegrityVerified   IntegrityStatus = "verified"
	IntegrityFailed     IntegrityStatus = "failed"
)

type Integrity struct {
	Unanchored *UnanchoredIntegrity
	Pending    *PendingIntegrity
	Verified   *VerifiedIntegrity
	Failed     *FailedIntegrity
}

type UnanchoredIntegrity struct {
	Status IntegrityStatus `json:"status" yaml:"status"`
	Note   *string         `json:"note,omitempty" yaml:"note,omitempty"`
}

type PendingIntegrity struct {
	Status          IntegrityStatus   `json:"status" yaml:"status"`
	Target          ForecastTarget    `json:"target" yaml:"target"`
	Timestamps      []OTSTimestamp    `json:"timestamps" yaml:"timestamps"`
	ExternalAnchors *[]ExternalAnchor `json:"external_anchors,omitempty" yaml:"external_anchors,omitempty"`
}

type VerifiedIntegrity struct {
	Status          IntegrityStatus   `json:"status" yaml:"status"`
	Target          ForecastTarget    `json:"target" yaml:"target"`
	Timestamps      []OTSTimestamp    `json:"timestamps" yaml:"timestamps"`
	VerifiedAt      Timestamp         `json:"verified_at" yaml:"verified_at"`
	ExternalAnchors *[]ExternalAnchor `json:"external_anchors,omitempty" yaml:"external_anchors,omitempty"`
}

type FailedIntegrity struct {
	Status        IntegrityStatus `json:"status" yaml:"status"`
	FailureReason string          `json:"failure_reason" yaml:"failure_reason"`
	Target        *ForecastTarget `json:"target,omitempty" yaml:"target,omitempty"`
	Timestamps    *[]OTSTimestamp `json:"timestamps,omitempty" yaml:"timestamps,omitempty"`
}

func (v Integrity) MarshalJSON() ([]byte, error) {
	return marshalOne("integrity", v.Unanchored, v.Pending, v.Verified, v.Failed)
}

func (v *Integrity) UnmarshalJSON(data []byte) error {
	*v = Integrity{}
	var discriminator struct {
		Status IntegrityStatus `json:"status"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	switch discriminator.Status {
	case IntegrityUnanchored:
		v.Unanchored = new(UnanchoredIntegrity)
		return json.Unmarshal(data, v.Unanchored)
	case IntegrityPending:
		v.Pending = new(PendingIntegrity)
		return json.Unmarshal(data, v.Pending)
	case IntegrityVerified:
		v.Verified = new(VerifiedIntegrity)
		return json.Unmarshal(data, v.Verified)
	case IntegrityFailed:
		v.Failed = new(FailedIntegrity)
		return json.Unmarshal(data, v.Failed)
	default:
		return fmt.Errorf("unknown integrity status %q", discriminator.Status)
	}
}

type Encryption struct {
	Algorithm  string           `json:"algorithm" yaml:"algorithm"`
	Nonce      Base64Nonce12    `json:"nonce" yaml:"nonce"`
	Ciphertext Base64Ciphertext `json:"ciphertext" yaml:"ciphertext"`
}

type Commitment struct {
	Sealed   *SealedCommitment
	Revealed *RevealedCommitment
}

type SealedCommitment struct {
	Scheme         string     `json:"scheme" yaml:"scheme"`
	CommitmentHash Digest     `json:"commitment_hash" yaml:"commitment_hash"`
	Encryption     Encryption `json:"encryption" yaml:"encryption"`
	KeyHint        string     `json:"key_hint" yaml:"key_hint"`
}

type RevealedCommitment struct {
	Scheme         string     `json:"scheme" yaml:"scheme"`
	CommitmentHash Digest     `json:"commitment_hash" yaml:"commitment_hash"`
	Encryption     Encryption `json:"encryption" yaml:"encryption"`
	KeyHint        string     `json:"key_hint" yaml:"key_hint"`
	RevealedAt     Timestamp  `json:"revealed_at" yaml:"revealed_at"`
	RevealedKey    Hex32      `json:"revealed_key" yaml:"revealed_key"`
}

func (v Commitment) MarshalJSON() ([]byte, error) {
	return marshalOne("commitment", v.Sealed, v.Revealed)
}

func (v *Commitment) UnmarshalJSON(data []byte) error {
	*v = Commitment{}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	_, hasAt := fields["revealed_at"]
	_, hasKey := fields["revealed_key"]
	if hasAt || hasKey {
		v.Revealed = new(RevealedCommitment)
		return json.Unmarshal(data, v.Revealed)
	}
	v.Sealed = new(SealedCommitment)
	return json.Unmarshal(data, v.Sealed)
}

type ResolutionStatus string

const (
	ResolutionResolved ResolutionStatus = "resolved"
	ResolutionAnnulled ResolutionStatus = "annulled"
	ResolutionDisputed ResolutionStatus = "disputed"
)

type Resolution struct {
	Resolved    *ResolvedResolution
	NonResolved *NonResolvedResolution
}

// ResolutionOutcome is the exact JSON scalar domain permitted by the schema.
// Text is interpreted as an option ID, decimal, or date using the question type.
type ResolutionOutcome struct {
	Binary *bool
	Text   *string
}

func (v ResolutionOutcome) MarshalJSON() ([]byte, error) {
	return marshalOne("resolution outcome", v.Binary, v.Text)
}

func (v *ResolutionOutcome) UnmarshalJSON(data []byte) error {
	*v = ResolutionOutcome{}
	if bytes.Equal(data, []byte("true")) || bytes.Equal(data, []byte("false")) {
		v.Binary = new(bool)
		return json.Unmarshal(data, v.Binary)
	}
	v.Text = new(string)
	if err := json.Unmarshal(data, v.Text); err != nil {
		return fmt.Errorf("resolution outcome must be a boolean or string: %w", err)
	}
	return nil
}

type ResolvedResolution struct {
	Status         ResolutionStatus   `json:"status" yaml:"status"`
	Outcome        ResolutionOutcome  `json:"outcome" yaml:"outcome"`
	OutcomeKnownAt Timestamp          `json:"outcome_known_at" yaml:"outcome_known_at"`
	RecordedAt     Timestamp          `json:"recorded_at" yaml:"recorded_at"`
	Sources        []ResolutionSource `json:"sources" yaml:"sources"`
	Notes          *string            `json:"notes,omitempty" yaml:"notes,omitempty"`
}

type NonResolvedResolution struct {
	Status     ResolutionStatus    `json:"status" yaml:"status"`
	Reason     string              `json:"reason" yaml:"reason"`
	RecordedAt Timestamp           `json:"recorded_at" yaml:"recorded_at"`
	Sources    *[]ResolutionSource `json:"sources,omitempty" yaml:"sources,omitempty"`
}

type ResolutionSource struct {
	Title         string     `json:"title" yaml:"title"`
	Publisher     *string    `json:"publisher,omitempty" yaml:"publisher,omitempty"`
	URL           string     `json:"url" yaml:"url"`
	PublishedAt   *Timestamp `json:"published_at,omitempty" yaml:"published_at,omitempty"`
	RetrievedAt   Timestamp  `json:"retrieved_at" yaml:"retrieved_at"`
	ContentDigest *Digest    `json:"content_digest,omitempty" yaml:"content_digest,omitempty"`
}

func (v Resolution) MarshalJSON() ([]byte, error) {
	return marshalOne("resolution", v.Resolved, v.NonResolved)
}

func (v *Resolution) UnmarshalJSON(data []byte) error {
	*v = Resolution{}
	var discriminator struct {
		Status ResolutionStatus `json:"status"`
	}
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return err
	}
	switch discriminator.Status {
	case ResolutionResolved:
		v.Resolved = new(ResolvedResolution)
		return json.Unmarshal(data, v.Resolved)
	case ResolutionAnnulled, ResolutionDisputed:
		v.NonResolved = new(NonResolvedResolution)
		return json.Unmarshal(data, v.NonResolved)
	default:
		return fmt.Errorf("unknown resolution status %q", discriminator.Status)
	}
}

func marshalOne(name string, variants ...any) ([]byte, error) {
	var selected any
	for _, variant := range variants {
		if isNil(variant) {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("%s has more than one variant", name)
		}
		selected = variant
	}
	if selected == nil {
		return nil, fmt.Errorf("%s has no variant", name)
	}
	return json.Marshal(selected)
}

func isNil(value any) bool {
	switch value := value.(type) {
	case *BinaryValue:
		return value == nil
	case *MultipleChoiceValue:
		return value == nil
	case *NumericValue:
		return value == nil
	case *DateValue:
		return value == nil
	case *UnanchoredIntegrity:
		return value == nil
	case *PendingIntegrity:
		return value == nil
	case *VerifiedIntegrity:
		return value == nil
	case *FailedIntegrity:
		return value == nil
	case *SealedCommitment:
		return value == nil
	case *RevealedCommitment:
		return value == nil
	case *ResolvedResolution:
		return value == nil
	case *NonResolvedResolution:
		return value == nil
	case *bool:
		return value == nil
	case *string:
		return value == nil
	default:
		return value == nil
	}
}
