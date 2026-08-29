package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chaoscondensate/cli/internal/app"
	"github.com/chaoscondensate/cli/internal/forecastcrypto"
	"github.com/chaoscondensate/cli/internal/ledger"
	"github.com/chaoscondensate/cli/internal/timestamp/ots"
)

type LayerState string

const (
	LayerPass          LayerState = "pass"
	LayerFail          LayerState = "fail"
	LayerPending       LayerState = "pending"
	LayerNotApplicable LayerState = "not_applicable"
	LayerNotChecked    LayerState = "not_checked"
)

type VerificationLayer struct {
	Name        string         `json:"name"`
	State       LayerState     `json:"state"`
	ReasonCodes []string       `json:"reason_codes,omitempty"`
	Evidence    map[string]any `json:"evidence,omitempty"`
	Limitations []string       `json:"limitations,omitempty"`
}

type ForecastVerification struct {
	QuestionID ledger.Slug         `json:"question_id"`
	ForecastID ledger.Slug         `json:"forecast_id"`
	Layers     []VerificationLayer `json:"layers"`
}

type VerificationOverall string

const (
	VerificationPass       VerificationOverall = "pass"
	VerificationFail       VerificationOverall = "fail"
	VerificationPending    VerificationOverall = "pending"
	VerificationIncomplete VerificationOverall = "incomplete"
)

type VerificationReport struct {
	LedgerID       ledger.Slug            `json:"ledger_id"`
	Overall        VerificationOverall    `json:"overall"`
	Document       VerificationLayer      `json:"document"`
	Forecasts      []ForecastVerification `json:"forecasts"`
	Limitations    []string               `json:"limitations"`
	NetworkProfile NetworkProfile         `json:"network_profile"`
	RequestSummary ots.RequestSummary     `json:"request_summary,omitempty"`
	FailureCode    app.ErrorCode          `json:"-"`
}

type VerificationOptions struct {
	Offline      bool
	CheckSources bool
	Observer     ots.BitcoinObserver
	HTTPClient   *http.Client
	QuestionID   ledger.Slug
	ForecastID   ledger.Slug
}

var verificationLimitations = []string{
	"Forecast Ledger v1 does not prove authorship.",
	"It does not prove that the ledger or forecast set is complete.",
	"It does not prove forecast truth or calibration.",
	"Forecast and outcome times are self-reported except for the conservative existence bound supplied by verified OpenTimestamps evidence.",
	"Outcome-source checks do not establish authority or substantive truth.",
	"Filesystem, archive, hosting, source-control, and external-anchor times are not cryptographic existence evidence.",
	"Calendar services learn request timing and blinded commitments.",
	"Public Bitcoin services learn requested block heights and approximate timestamp periods.",
}

func VerifyLedgerEvidence(ctx context.Context, path string, options VerificationOptions) (VerificationReport, error) {
	loaded, err := LoadAndValidateLedger(ctx, path, nil)
	if err != nil {
		return VerificationReport{}, err
	}
	report := VerificationReport{
		LedgerID:    loaded.Model.LedgerID,
		Document:    VerificationLayer{Name: "document", State: LayerPass, ReasonCodes: []string{"document.valid"}, Evidence: map[string]any{"schema_version": loaded.Model.SchemaVersion, "format": loaded.Document.Format}},
		Forecasts:   []ForecastVerification{},
		Limitations: append([]string(nil), verificationLimitations...),
	}
	observer := options.Observer
	if observer == nil && !options.Offline {
		observer = ots.NewPublicBitcoinObserver(nil)
	}
	report.NetworkProfile = networkProfileForObserver(observer, options.Offline)
	selectedQuestions, err := selectVerificationQuestions(loaded.Model, options.QuestionID, options.ForecastID)
	if err != nil {
		return report, err
	}
	for _, question := range selectedQuestions {
		for _, forecast := range question.Forecasts {
			if options.ForecastID != "" && forecast.ID != options.ForecastID {
				continue
			}
			item := ForecastVerification{QuestionID: question.ID, ForecastID: forecast.ID}
			content := verifyContentLayer(ctx, loaded, question, forecast)
			item.Layers = append(item.Layers, content)
			item.Layers = append(item.Layers, verifyTimingLayer(ctx, loaded, question, forecast, content, options.Offline, observer))
			item.Layers = append(item.Layers, verifyRevealLayer(question, forecast, content))
			item.Layers = append(item.Layers, verifyOutcomeLayer(ctx, question, options))
			report.Forecasts = append(report.Forecasts, item)
		}
	}
	if observer != nil {
		report.RequestSummary = observer.Summary()
	}
	report.Overall, report.FailureCode = aggregateVerification(report)
	return report, nil
}

func selectVerificationQuestions(model *ledger.Ledger, questionID, forecastID ledger.Slug) ([]ledger.Question, error) {
	if forecastID != "" && questionID == "" {
		return nil, app.NewError(app.CodeUsage, "--forecast requires --question", nil)
	}
	if questionID == "" {
		return append([]ledger.Question(nil), model.Questions...), nil
	}
	_, question, err := selectQuestion(model, questionID)
	if err != nil {
		return nil, err
	}
	if forecastID != "" {
		found := false
		for _, forecast := range question.Forecasts {
			if forecast.ID == forecastID {
				found = true
				break
			}
		}
		if !found {
			return nil, app.NewError(app.CodeNotFound, "forecast was not found in the selected question", nil)
		}
	}
	return []ledger.Question{question}, nil
}

func verifyContentLayer(ctx context.Context, loaded *LoadedLedger, question ledger.Question, forecast ledger.Forecast) VerificationLayer {
	layer := VerificationLayer{Name: "content_binding"}
	artifact, err := BuildForecastTarget(loaded.Model, question.ID, forecast.ID)
	if err != nil {
		return failedLayer(layer.Name, "content.target_build_failed", err)
	}
	root := filepath.Dir(loaded.Path)
	hasTarget := regularFileExists(filepath.Join(root, filepath.FromSlash(string(artifact.RelativePath)))) || recordedForecastTarget(loaded.Model, question.ID, forecast.ID) != nil
	if !hasTarget {
		layer.State, layer.ReasonCodes = LayerNotApplicable, []string{"content.no_retained_target"}
		return layer
	}
	if _, err := CheckTargets(ctx, loaded.Path, false, question.ID, forecast.ID); err != nil {
		return failedLayer(layer.Name, "content.target_mismatch", err)
	}
	layer.State = LayerPass
	layer.ReasonCodes = []string{"content.target_matches"}
	layer.Evidence = map[string]any{"path": artifact.RelativePath, "sha256": artifact.SHA256, "scope": ForecastEnvelopeSchema, "canonicalization": TargetCanonicalization}
	return layer
}

func verifyTimingLayer(ctx context.Context, loaded *LoadedLedger, question ledger.Question, forecast ledger.Forecast, content VerificationLayer, offline bool, observer ots.BitcoinObserver) VerificationLayer {
	layer := VerificationLayer{Name: "existence_timing", Limitations: []string{"Bitcoin evidence establishes a conservative existence bound, not authorship or exact creation time."}}
	if forecast.Integrity.Unanchored != nil {
		layer.State, layer.ReasonCodes = LayerNotApplicable, []string{"timing.unanchored"}
		return layer
	}
	if forecast.Integrity.Failed != nil {
		return failedLayer(layer.Name, "timing.imported_failed", nil)
	}
	if content.State != LayerPass {
		layer.State, layer.ReasonCodes = LayerNotChecked, []string{"timing.blocked_by_content"}
		return layer
	}
	receiptPath := filepath.Join(filepath.Dir(loaded.Path), filepath.FromSlash(string(ReceiptRelativePath(forecast.ID))))
	safeReceiptPath := ReceiptRelativePath(forecast.ID)
	data, err := readBoundedFile(receiptPath, maxReceiptBytes)
	if err != nil {
		return failedLayer(layer.Name, "timing.receipt_missing", err)
	}
	receipt, err := ots.ParseReceipt(data)
	if err != nil {
		return failedLayer(layer.Name, "timing.receipt_invalid", err)
	}
	artifact, _ := BuildForecastTarget(loaded.Model, question.ID, forecast.ID)
	if err := receipt.VerifyBinding(artifact.Bytes); err != nil {
		return failedLayer(layer.Name, "timing.receipt_binding_mismatch", err)
	}
	evaluated, err := receipt.Evaluate()
	if err != nil {
		return failedLayer(layer.Name, "timing.proof_invalid", err)
	}
	bitcoin := make([]ots.EvaluatedAttestation, 0)
	for _, item := range evaluated {
		if item.Attestation.Kind == ots.AttestationBitcoin {
			bitcoin = append(bitcoin, item)
		}
	}
	if err := verifiedTimestampMatchesReceipt(forecast, evaluated); err != nil {
		return failedLayer(layer.Name, "timing.stored_metadata_mismatch", err)
	}
	if len(bitcoin) == 0 {
		layer.State, layer.ReasonCodes = LayerPending, []string{"timing.calendar_pending"}
		return layer
	}
	if offline {
		if forecast.Integrity.Verified != nil {
			layer.Evidence = storedTimingEvidence(forecast, safeReceiptPath, artifact.RelativePath)
			if question.Resolution != nil && question.Resolution.Resolved != nil {
				known, _ := ParseTimestamp(question.Resolution.Resolved.OutcomeKnownAt, "outcome_known_at")
				for _, timestamp := range forecast.Integrity.Verified.Timestamps {
					if timestamp.Type != "opentimestamps" || timestamp.State != ledger.OTSConfirmed || timestamp.AnchoredBefore == nil {
						continue
					}
					bound, parseErr := ParseTimestamp(*timestamp.AnchoredBefore, "anchored_before")
					if parseErr != nil || !bound.Before(known) {
						evidence := storedTimingEvidence(forecast, safeReceiptPath, artifact.RelativePath)
						return failedLayerWithEvidence(layer.Name, "timing.not_before_outcome", evidence)
					}
				}
			}
			layer.State, layer.ReasonCodes = LayerPass, []string{"timing.stored_verification_consistent"}
			layer.Limitations = append(layer.Limitations, "The v1 ledger does not retain the prior Bitcoin source identity; block evidence was not rechecked offline.")
			return layer
		}
		layer.State, layer.ReasonCodes = LayerNotChecked, []string{"timing.offline"}
		return layer
	}
	sort.Slice(bitcoin, func(i, j int) bool { return bitcoin[i].Attestation.Height < bitcoin[j].Attestation.Height })
	for _, item := range bitcoin {
		observation, observeErr := observer.Observe(ctx, item.Attestation.Height)
		if observeErr != nil {
			layer.State, layer.ReasonCodes = LayerNotChecked, []string{"timing.source_unavailable"}
			layer.Evidence = map[string]any{"height": item.Attestation.Height}
			return layer
		}
		if err := ots.VerifyBitcoinAttestation(item, observation); err != nil {
			return failedLayer(layer.Name, "timing.bitcoin_mismatch", err)
		}
		bound := ledger.Timestamp(observation.BlockTime.Format(time.RFC3339))
		layer.State, layer.ReasonCodes = LayerPass, []string{"timing.bitcoin_verified"}
		layer.Evidence = map[string]any{"receipt_path": safeReceiptPath, "target_path": artifact.RelativePath, "target_binding": "pass", "proof_valid": true, "bitcoin_block_height": item.Attestation.Height, "block_hash": observation.Hash, "anchored_before": bound, "source_ids": observation.SourceIDs, "evidence_source": "fresh_observation", "freshly_checked": true}
		if forecast.Integrity.Verified != nil {
			layer.Evidence["verified_at"] = forecast.Integrity.Verified.VerifiedAt
		}
		if question.Resolution != nil && question.Resolution.Resolved != nil {
			known, _ := ParseTimestamp(question.Resolution.Resolved.OutcomeKnownAt, "outcome_known_at")
			if !observation.BlockTime.Before(known) {
				return failedLayerWithEvidence(layer.Name, "timing.not_before_outcome", layer.Evidence)
			}
		}
		return layer
	}
	layer.State, layer.ReasonCodes = LayerNotChecked, []string{"timing.no_supported_attestation"}
	return layer
}

func storedTimingEvidence(forecast ledger.Forecast, receiptPath, targetPath ledger.RelativePath) map[string]any {
	evidence := map[string]any{
		"receipt_path": receiptPath, "target_path": targetPath, "target_binding": "pass", "proof_valid": true,
		"evidence_source": "stored_verification", "freshly_checked": false, "prior_source_retained": false,
	}
	if forecast.Integrity.Verified == nil {
		return evidence
	}
	evidence["verified_at"] = forecast.Integrity.Verified.VerifiedAt
	confirmed := make([]map[string]any, 0)
	for _, timestamp := range forecast.Integrity.Verified.Timestamps {
		if timestamp.Type != "opentimestamps" || timestamp.State != ledger.OTSConfirmed {
			continue
		}
		item := map[string]any{"proof_path": timestamp.ProofPath, "state": timestamp.State}
		if timestamp.BitcoinBlockHeight != nil {
			item["bitcoin_block_height"] = *timestamp.BitcoinBlockHeight
		}
		if timestamp.AnchoredBefore != nil {
			item["anchored_before"] = *timestamp.AnchoredBefore
		}
		confirmed = append(confirmed, item)
	}
	evidence["timestamps"] = confirmed
	if len(confirmed) == 1 {
		for key, value := range confirmed[0] {
			if key == "proof_path" || key == "state" {
				continue
			}
			evidence[key] = value
		}
	}
	return evidence
}

func verifyRevealLayer(question ledger.Question, forecast ledger.Forecast, content VerificationLayer) VerificationLayer {
	layer := VerificationLayer{Name: "reveal"}
	if forecast.Visibility != ledger.VisibilityRevealed {
		layer.State, layer.ReasonCodes = LayerNotApplicable, []string{"reveal.not_revealed"}
		return layer
	}
	if content.State != LayerPass {
		layer.State, layer.ReasonCodes = LayerNotChecked, []string{"reveal.blocked_by_content"}
		return layer
	}
	if forecast.Commitment == nil || forecast.Commitment.Revealed == nil {
		return failedLayer(layer.Name, "reveal.commitment_missing", nil)
	}
	revealed := forecast.Commitment.Revealed
	key, err := hex.DecodeString(string(revealed.RevealedKey))
	if err != nil {
		return failedLayer(layer.Name, "reveal.key_invalid", err)
	}
	keyFile, err := forecastcrypto.EncodeKeyFile(question.ID, forecast.ID, key)
	clear(key)
	if err != nil {
		return failedLayer(layer.Name, "reveal.key_invalid", err)
	}
	opened, err := forecastcrypto.Open(keyFile, question.ID, forecast.ID, ledger.SealedCommitment{Scheme: revealed.Scheme, CommitmentHash: revealed.CommitmentHash, Encryption: revealed.Encryption, KeyHint: revealed.KeyHint})
	clear(keyFile)
	if err != nil {
		return failedLayer(layer.Name, "reveal.authentication_failed", err)
	}
	if err := validateRevealedBundle(question, forecast, opened.Bundle); err != nil {
		return failedLayer(layer.Name, "reveal.mirror_mismatch", err)
	}
	layer.State, layer.ReasonCodes = LayerPass, []string{"reveal.authenticated"}
	layer.Evidence = map[string]any{"scheme": forecast.Commitment.Revealed.Scheme, "commitment_sha256": forecast.Commitment.Revealed.CommitmentHash.Value}
	return layer
}

func verifyOutcomeLayer(ctx context.Context, question ledger.Question, options VerificationOptions) VerificationLayer {
	layer := VerificationLayer{Name: "outcome_evidence", Limitations: []string{"Source availability and digest checks do not establish authority or substantive truth."}}
	if question.Resolution == nil {
		layer.State, layer.ReasonCodes = LayerNotApplicable, []string{"outcome.unresolved"}
		return layer
	}
	var sources []ledger.ResolutionSource
	if question.Resolution.Resolved != nil {
		sources = question.Resolution.Resolved.Sources
	} else if question.Resolution.NonResolved != nil && question.Resolution.NonResolved.Sources != nil {
		sources = *question.Resolution.NonResolved.Sources
	}
	layer.State, layer.ReasonCodes = LayerPass, []string{"outcome.metadata_valid"}
	layer.Evidence = map[string]any{"source_count": len(sources), "status": question.Status}
	if !options.CheckSources {
		return layer
	}
	if options.Offline {
		layer.State, layer.ReasonCodes = LayerNotChecked, []string{"outcome.offline"}
		return layer
	}
	client := options.HTTPClient
	if client == nil {
		client = boundedOutcomeClient()
	}
	for index, source := range sources {
		data, finalURL, err := fetchOutcomeSource(ctx, client, source.URL)
		if err != nil {
			layer.State, layer.ReasonCodes = LayerNotChecked, []string{"outcome.source_unavailable"}
			layer.Evidence = map[string]any{"source_index": index}
			return layer
		}
		if source.ContentDigest != nil {
			digest := sha256.Sum256(data)
			if source.ContentDigest.Algorithm != "sha-256" || !strings.EqualFold(hex.EncodeToString(digest[:]), string(source.ContentDigest.Value)) {
				return failedLayer(layer.Name, "outcome.digest_mismatch", nil)
			}
		}
		layer.Evidence[fmt.Sprintf("source_%d_final_url", index)] = finalURL
	}
	layer.ReasonCodes = []string{"outcome.sources_reachable"}
	return layer
}

func aggregateVerification(report VerificationReport) (VerificationOverall, app.ErrorCode) {
	hasPending, hasNotChecked, sourceUnavailable := false, false, false
	for _, forecast := range report.Forecasts {
		for _, layer := range forecast.Layers {
			if layer.State == LayerFail {
				return VerificationFail, app.CodeVerification
			}
			if layer.State == LayerPending {
				hasPending = true
			}
			if layer.State == LayerNotChecked {
				hasNotChecked = true
				for _, reason := range layer.ReasonCodes {
					if strings.Contains(reason, "source_unavailable") {
						sourceUnavailable = true
					}
				}
			}
		}
	}
	if sourceUnavailable {
		return VerificationIncomplete, app.CodeNetwork
	}
	if hasPending {
		return VerificationPending, app.CodePending
	}
	if hasNotChecked {
		return VerificationIncomplete, app.CodeIncomplete
	}
	return VerificationPass, ""
}

func failedLayer(name, reason string, err error) VerificationLayer {
	layer := VerificationLayer{Name: name, State: LayerFail, ReasonCodes: []string{reason}}
	if err != nil {
		layer.Evidence = map[string]any{"error": safeVerificationError(err)}
	}
	return layer
}

func failedLayerWithEvidence(name, reason string, evidence map[string]any) VerificationLayer {
	return VerificationLayer{Name: name, State: LayerFail, ReasonCodes: []string{reason}, Evidence: evidence}
}

func safeVerificationError(err error) string {
	if err == nil {
		return ""
	}
	return strings.SplitN(err.Error(), ":", 2)[0]
}

func boundedOutcomeClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return validatePublicSourceURL(request.Context(), request.URL)
	}}
}

func fetchOutcomeSource(ctx context.Context, client *http.Client, source string) ([]byte, string, error) {
	parsed, err := url.Parse(source)
	if err != nil || validatePublicSourceURL(ctx, parsed) != nil {
		return nil, "", fmt.Errorf("outcome source URL is not a safe public HTTPS URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("User-Agent", "forecast-ledger/outcome-check-v1")
	response, err := client.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("outcome source returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil || len(data) > 4<<20 {
		return nil, "", fmt.Errorf("outcome source response is invalid or too large")
	}
	return data, response.Request.URL.String(), nil
}

func validatePublicSourceURL(ctx context.Context, parsed *url.URL) error {
	if parsed == nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("source URL must be public HTTPS")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return fmt.Errorf("source host cannot be resolved")
	}
	for _, address := range addresses {
		ip, ok := netip.AddrFromSlice(address.IP)
		if !ok || !ip.Unmap().IsGlobalUnicast() || ip.Unmap().IsPrivate() || ip.Unmap().IsLoopback() || ip.Unmap().IsLinkLocalUnicast() {
			return fmt.Errorf("source host is not public")
		}
	}
	return nil
}
