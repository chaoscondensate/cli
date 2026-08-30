package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/canonical"
	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/storage"
)

const (
	ForecastEnvelopeSchema = "forecast-envelope/v1"
	TargetCanonicalization = "RFC8785"
)

type TargetArtifact struct {
	QuestionID   ledger.Slug         `json:"question_id"`
	ForecastID   ledger.Slug         `json:"forecast_id"`
	RelativePath ledger.RelativePath `json:"path"`
	SHA256       string              `json:"sha256"`
	Size         int                 `json:"size"`
	Bytes        []byte              `json:"-"`
}

type targetCommitment struct {
	Scheme         string            `json:"scheme"`
	CommitmentHash ledger.Digest     `json:"commitment_hash"`
	Encryption     ledger.Encryption `json:"encryption"`
}

type targetForecast struct {
	ID                   ledger.Slug               `json:"id"`
	ForecastedAt         ledger.Timestamp          `json:"forecasted_at"`
	RecordedAt           ledger.Timestamp          `json:"recorded_at"`
	Visibility           ledger.ForecastVisibility `json:"visibility"`
	PublicNote           *string                   `json:"public_note,omitempty"`
	SupersedesForecastID *ledger.Slug              `json:"supersedes_forecast_id,omitempty"`
	Value                *ledger.ForecastValue     `json:"value,omitempty"`
	Rationale            *string                   `json:"rationale,omitempty"`
	KeyFactors           *[]string                 `json:"key_factors,omitempty"`
	Comment              *string                   `json:"comment,omitempty"`
	Commitment           *targetCommitment         `json:"commitment,omitempty"`
}

type forecastEnvelope struct {
	Schema     string         `json:"schema"`
	QuestionID ledger.Slug    `json:"question_id"`
	Forecast   targetForecast `json:"forecast"`
}

func BuildForecastTarget(model *ledger.Ledger, questionID, forecastID ledger.Slug) (TargetArtifact, error) {
	if model == nil {
		return TargetArtifact{}, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	_, question, err := selectQuestion(model, questionID)
	if err != nil {
		return TargetArtifact{}, err
	}
	var forecast *ledger.Forecast
	for index := range question.Forecasts {
		if question.Forecasts[index].ID == forecastID {
			forecast = &question.Forecasts[index]
			break
		}
	}
	if forecast == nil {
		return TargetArtifact{}, app.WithDetails(app.NewError(app.CodeNotFound, "forecast was not found in the selected question", nil), map[string]any{"question_id": questionID, "forecast_id": forecastID})
	}
	envelope, err := buildForecastEnvelope(question, *forecast)
	if err != nil {
		return TargetArtifact{}, err
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return TargetArtifact{}, app.NewError(app.CodeInternal, "forecast target cannot be encoded", err)
	}
	parsed, err := document.ParseJSON(bytes.NewReader(encoded), document.DefaultLimits)
	if err != nil {
		return TargetArtifact{}, app.NewError(app.CodeInternal, "forecast target cannot be normalized", err)
	}
	canonicalBytes, err := canonical.Marshal(parsed.Root.Any())
	if err != nil {
		return TargetArtifact{}, app.NewError(app.CodeInvalidData, "forecast target cannot be canonicalized", err)
	}
	digest := sha256.Sum256(canonicalBytes)
	relative := ledger.RelativePath(storage.DeterministicRelativePath("proofs/targets", string(forecastID)+".json"))
	return TargetArtifact{QuestionID: questionID, ForecastID: forecastID, RelativePath: relative, SHA256: hex.EncodeToString(digest[:]), Size: len(canonicalBytes), Bytes: canonicalBytes}, nil
}

func BuildAllForecastTargets(model *ledger.Ledger) ([]TargetArtifact, error) {
	if model == nil {
		return nil, app.NewError(app.CodeInternal, "ledger is nil", nil)
	}
	result := make([]TargetArtifact, 0)
	for _, question := range model.Questions {
		for _, forecast := range question.Forecasts {
			artifact, err := BuildForecastTarget(model, question.ID, forecast.ID)
			if err != nil {
				return nil, err
			}
			result = append(result, artifact)
		}
	}
	paths := make([]string, len(result))
	for index := range result {
		paths[index] = string(result[index].RelativePath)
	}
	if err := storage.DetectPortablePathCollisions(paths); err != nil {
		return nil, err
	}
	return result, nil
}

func buildForecastEnvelope(question ledger.Question, forecast ledger.Forecast) (forecastEnvelope, error) {
	targetForecast := targetForecast{
		ID: forecast.ID, ForecastedAt: forecast.ForecastedAt, RecordedAt: forecast.RecordedAt,
		Visibility: forecast.Visibility, PublicNote: cloneString(forecast.PublicNote), SupersedesForecastID: cloneSlug(forecast.SupersedesForecastID),
	}
	switch forecast.Visibility {
	case ledger.VisibilityPublic:
		targetForecast.Value = cloneForecastValue(forecast.Value)
		targetForecast.Rationale = cloneString(forecast.Rationale)
		targetForecast.KeyFactors = cloneStrings(forecast.KeyFactors)
		targetForecast.Comment = cloneString(forecast.Comment)
	case ledger.VisibilitySealed:
		if forecast.Commitment == nil || forecast.Commitment.Sealed == nil {
			return forecastEnvelope{}, app.NewError(app.CodeInvalidData, "sealed forecast has no sealed commitment", nil)
		}
		sealed := forecast.Commitment.Sealed
		targetForecast.Commitment = &targetCommitment{Scheme: sealed.Scheme, CommitmentHash: sealed.CommitmentHash, Encryption: sealed.Encryption}
	case ledger.VisibilityRevealed:
		if forecast.Commitment == nil || forecast.Commitment.Revealed == nil {
			return forecastEnvelope{}, app.NewError(app.CodeInvalidData, "revealed forecast has no revealed commitment", nil)
		}
		revealed := forecast.Commitment.Revealed
		targetForecast.Visibility = ledger.VisibilitySealed
		targetForecast.Commitment = &targetCommitment{Scheme: revealed.Scheme, CommitmentHash: revealed.CommitmentHash, Encryption: revealed.Encryption}
	default:
		return forecastEnvelope{}, app.NewError(app.CodeInvalidData, "forecast visibility is not supported for targets", nil)
	}
	return forecastEnvelope{Schema: ForecastEnvelopeSchema, QuestionID: question.ID, Forecast: targetForecast}, nil
}

func TargetMetadataFor(artifact TargetArtifact) ledger.ForecastTarget {
	return ledger.ForecastTarget{
		Scope: ForecastEnvelopeSchema, Canonicalization: TargetCanonicalization, ArtifactPath: artifact.RelativePath,
		Digest: ledger.Digest{Algorithm: "sha-256", Value: ledger.Hex32(artifact.SHA256)},
	}
}
