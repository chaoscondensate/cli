package ots

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type BlockObservation struct {
	Height     uint64    `json:"height"`
	Hash       string    `json:"hash"`
	HeaderHex  string    `json:"header_hex"`
	MerkleRoot string    `json:"merkle_root"`
	BlockTime  time.Time `json:"block_time"`
	SourceIDs  []string  `json:"source_ids"`
}

type RequestSummary struct {
	UniqueHeights int `json:"unique_heights"`
	HTTPRequests  int `json:"http_requests"`
	MaxHeights    int `json:"max_heights"`
	MaxRequests   int `json:"max_requests"`
	MaxConcurrent int `json:"max_concurrent"`
}

type BitcoinObserver interface {
	Observe(context.Context, uint64) (BlockObservation, error)
	Summary() RequestSummary
}

type ObservationIssueKind string

const (
	ObservationSourceUnavailable ObservationIssueKind = "source_unavailable"
	ObservationInconclusive      ObservationIssueKind = "observation_inconclusive"
	ObservationBudgetExhausted   ObservationIssueKind = "observation_budget_exhausted"
)

// ObservationError separates failure to obtain an accepted Bitcoin observation
// from failure of an OpenTimestamps proof against one. Only the closed kind and
// stable source IDs are safe for public projection; cause is retained solely for
// local classification and tests.
type ObservationError struct {
	kind      ObservationIssueKind
	sourceIDs []string
	cause     error
}

func (err *ObservationError) Error() string {
	if err == nil {
		return "Bitcoin observation did not complete"
	}
	switch err.kind {
	case ObservationSourceUnavailable:
		return "Bitcoin observation source is unavailable"
	case ObservationBudgetExhausted:
		return "Bitcoin observation budget is exhausted"
	default:
		return "Bitcoin observation is inconclusive"
	}
}

func (err *ObservationError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func (err *ObservationError) Kind() ObservationIssueKind {
	if err == nil {
		return ObservationInconclusive
	}
	return err.kind
}

func (err *ObservationError) SourceIDs() []string {
	if err == nil {
		return nil
	}
	return append([]string(nil), err.sourceIDs...)
}

func newObservationError(kind ObservationIssueKind, sourceIDs []string, cause error) *ObservationError {
	ids := append([]string(nil), sourceIDs...)
	sort.Strings(ids)
	ids = compactStrings(ids)
	return &ObservationError{kind: kind, sourceIDs: ids, cause: cause}
}

// NewObservationIssue lets alternate bounded observers report the same closed
// acquisition classification without exposing transport-specific errors.
func NewObservationIssue(kind ObservationIssueKind, sourceIDs ...string) error {
	switch kind {
	case ObservationSourceUnavailable, ObservationInconclusive, ObservationBudgetExhausted:
	default:
		kind = ObservationInconclusive
	}
	return newObservationError(kind, sourceIDs, nil)
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func observationErrorOf(err error) (*ObservationError, bool) {
	var issue *ObservationError
	ok := errors.As(err, &issue)
	return issue, ok
}

func withObservationSources(err error, sourceIDs ...string) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	issue, ok := observationErrorOf(err)
	if !ok {
		return newObservationError(ObservationInconclusive, sourceIDs, err)
	}
	return newObservationError(issue.kind, append(issue.SourceIDs(), sourceIDs...), issue.cause)
}

type PublicBitcoinObserver struct {
	client   *http.Client
	profile  PublicProfile
	mu       sync.Mutex
	cache    map[uint64]observationResult
	inflight map[uint64]*observationCall
	requests int
	sem      chan struct{}
}

type observationResult struct {
	observation BlockObservation
	err         error
}

type observationCall struct {
	done chan struct{}
	observationResult
}

func NewPublicBitcoinObserver(client *http.Client) *PublicBitcoinObserver {
	profile := Profile()
	if client == nil {
		client = boundedHTTPClient()
	}
	return &PublicBitcoinObserver{client: client, profile: profile, cache: make(map[uint64]observationResult), inflight: make(map[uint64]*observationCall), sem: make(chan struct{}, profile.MaximumConcurrentHTTP)}
}

func (observer *PublicBitcoinObserver) Observe(ctx context.Context, height uint64) (BlockObservation, error) {
	observer.mu.Lock()
	if result, ok := observer.cache[height]; ok {
		observer.mu.Unlock()
		return result.observation, result.err
	}
	if call, ok := observer.inflight[height]; ok {
		observer.mu.Unlock()
		select {
		case <-ctx.Done():
			return BlockObservation{}, ctx.Err()
		case <-call.done:
			return call.observation, call.err
		}
	}
	if len(observer.cache)+len(observer.inflight) >= observer.profile.MaximumUniqueHeights {
		observer.mu.Unlock()
		return BlockObservation{}, newObservationError(ObservationBudgetExhausted, nil, errors.New("unique-height budget exhausted"))
	}
	call := &observationCall{done: make(chan struct{})}
	observer.inflight[height] = call
	observer.mu.Unlock()

	observation, err := observer.observe(ctx, height)
	observer.mu.Lock()
	call.observationResult = observationResult{observation: observation, err: err}
	delete(observer.inflight, height)
	observer.cache[height] = call.observationResult
	close(call.done)
	observer.mu.Unlock()
	return observation, err
}

func (observer *PublicBitcoinObserver) observe(ctx context.Context, height uint64) (BlockObservation, error) {
	type answer struct {
		id     string
		hash   string
		header []byte
		err    error
	}
	answers := make(chan answer, len(observer.profile.BitcoinSources))
	for _, source := range observer.profile.BitcoinSources {
		source := source
		go func() {
			hash, header, err := observer.observeSource(ctx, source, height)
			answers <- answer{id: source.ID, hash: hash, header: header, err: err}
		}()
	}
	collected := make([]answer, 0, len(observer.profile.BitcoinSources))
	for range observer.profile.BitcoinSources {
		collected = append(collected, <-answers)
	}
	sort.Slice(collected, func(i, j int) bool { return collected[i].id < collected[j].id })
	var issueKind ObservationIssueKind
	var issueSources []string
	var causes []error
	for _, result := range collected {
		if result.err == nil {
			continue
		}
		if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
			return BlockObservation{}, result.err
		}
		issue, ok := observationErrorOf(result.err)
		kind := ObservationInconclusive
		if ok {
			kind = issue.Kind()
			issueSources = append(issueSources, issue.SourceIDs()...)
		}
		issueSources = append(issueSources, result.id)
		causes = append(causes, result.err)
		if issueKind != ObservationSourceUnavailable {
			switch kind {
			case ObservationSourceUnavailable:
				issueKind = kind
			case ObservationBudgetExhausted:
				if issueKind == "" || issueKind == ObservationInconclusive {
					issueKind = kind
				}
			default:
				if issueKind == "" {
					issueKind = ObservationInconclusive
				}
			}
		}
	}
	if issueKind != "" {
		return BlockObservation{}, newObservationError(issueKind, issueSources, errors.Join(causes...))
	}
	allSourceIDs := make([]string, 0, len(collected))
	for _, result := range collected {
		allSourceIDs = append(allSourceIDs, result.id)
	}
	if len(collected) != 2 || collected[0].hash != collected[1].hash || !bytes.Equal(collected[0].header, collected[1].header) {
		return BlockObservation{}, newObservationError(ObservationInconclusive, allSourceIDs, errors.New("public sources disagree"))
	}
	header := collected[0].header
	if err := verifyBlockHeader(header, collected[0].hash); err != nil {
		return BlockObservation{}, newObservationError(ObservationInconclusive, allSourceIDs, err)
	}
	return BlockObservation{
		Height: height, Hash: collected[0].hash, HeaderHex: hex.EncodeToString(header),
		MerkleRoot: hex.EncodeToString(header[36:68]), BlockTime: time.Unix(int64(binary.LittleEndian.Uint32(header[68:72])), 0).UTC(),
		SourceIDs: allSourceIDs,
	}, nil
}

func (observer *PublicBitcoinObserver) observeSource(ctx context.Context, source BitcoinSource, height uint64) (string, []byte, error) {
	hashData, err := observer.get(ctx, strings.TrimRight(source.Endpoint, "/")+"/block-height/"+strconv.FormatUint(height, 10), 128)
	if err != nil {
		return "", nil, withObservationSources(err, source.ID)
	}
	hash := strings.ToLower(strings.TrimSpace(string(hashData)))
	if decoded, err := hex.DecodeString(hash); err != nil || len(decoded) != 32 {
		return "", nil, newObservationError(ObservationInconclusive, []string{source.ID}, errors.New("invalid block hash"))
	}
	headerData, err := observer.get(ctx, strings.TrimRight(source.Endpoint, "/")+"/block/"+hash+"/header", 256)
	if err != nil {
		return "", nil, withObservationSources(err, source.ID)
	}
	header, err := hex.DecodeString(strings.TrimSpace(string(headerData)))
	if err != nil || len(header) != 80 {
		return "", nil, newObservationError(ObservationInconclusive, []string{source.ID}, errors.New("invalid block header"))
	}
	return hash, header, nil
}

func (observer *PublicBitcoinObserver) get(ctx context.Context, endpoint string, limit int64) ([]byte, error) {
	observer.mu.Lock()
	if observer.requests >= observer.profile.MaximumHTTPRequests {
		observer.mu.Unlock()
		return nil, newObservationError(ObservationBudgetExhausted, nil, errors.New("HTTP request budget exhausted"))
	}
	observer.requests++
	observer.mu.Unlock()
	select {
	case observer.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-observer.sem }()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, newObservationError(ObservationInconclusive, nil, err)
	}
	request.Header.Set("User-Agent", "forecast-ledger/ots-v1")
	response, err := observer.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, newObservationError(ObservationSourceUnavailable, nil, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, newObservationError(ObservationSourceUnavailable, nil, fmt.Errorf("HTTP %d", response.StatusCode))
	}
	data, err := readNetworkBody(response.Body, limit)
	if err != nil {
		return nil, newObservationError(ObservationInconclusive, nil, err)
	}
	return data, nil
}

func (observer *PublicBitcoinObserver) Summary() RequestSummary {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	return RequestSummary{UniqueHeights: len(observer.cache) + len(observer.inflight), HTTPRequests: observer.requests, MaxHeights: observer.profile.MaximumUniqueHeights, MaxRequests: observer.profile.MaximumHTTPRequests, MaxConcurrent: observer.profile.MaximumConcurrentHTTP}
}

func VerifyBitcoinAttestation(evaluated EvaluatedAttestation, observation BlockObservation) error {
	if evaluated.Attestation.Kind != AttestationBitcoin || evaluated.Attestation.Height != observation.Height {
		return errors.New("Bitcoin attestation does not match the observed height")
	}
	header, err := hex.DecodeString(observation.HeaderHex)
	if err != nil || len(header) != 80 {
		return errors.New("observed Bitcoin header is invalid")
	}
	if !bytes.Equal(evaluated.Message, header[36:68]) {
		return errors.New("OpenTimestamps proof result does not match the Bitcoin merkle root")
	}
	return verifyBlockHeader(header, observation.Hash)
}

func verifyBlockHeader(header []byte, expectedHash string) error {
	if len(header) != 80 {
		return errors.New("Bitcoin block header must contain exactly 80 bytes")
	}
	first := sha256.Sum256(header)
	second := sha256.Sum256(first[:])
	reversed := make([]byte, len(second))
	for index := range second {
		reversed[index] = second[len(second)-1-index]
	}
	if !strings.EqualFold(hex.EncodeToString(reversed), expectedHash) {
		return errors.New("Bitcoin block header hash does not match the named block")
	}
	target, negative, overflow := compactTarget(binary.LittleEndian.Uint32(header[72:76]))
	if negative || overflow || target.Sign() <= 0 {
		return errors.New("Bitcoin block header has an invalid proof-of-work target")
	}
	hashNumber := new(big.Int).SetBytes(reversed)
	if hashNumber.Cmp(target) > 0 {
		return errors.New("Bitcoin block header does not satisfy its proof-of-work target")
	}
	return nil
}

func compactTarget(compact uint32) (*big.Int, bool, bool) {
	size := compact >> 24
	word := compact & 0x007fffff
	negative := word != 0 && compact&0x00800000 != 0
	overflow := word != 0 && (size > 34 || (word > 0xff && size > 33) || (word > 0xffff && size > 32))
	result := new(big.Int).SetUint64(uint64(word))
	if size <= 3 {
		result.Rsh(result, uint(8*(3-size)))
	} else {
		result.Lsh(result, uint(8*(size-3)))
	}
	return result, negative, overflow
}

type CoreAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func DecodeCoreAuth(data []byte) (CoreAuth, error) {
	if len(data) == 0 || len(data) > 4096 {
		return CoreAuth{}, errors.New("Bitcoin Core auth file is empty or too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var auth CoreAuth
	if err := decoder.Decode(&auth); err != nil {
		return CoreAuth{}, errors.New("Bitcoin Core auth file must be a closed JSON object")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return CoreAuth{}, errors.New("Bitcoin Core auth file must contain one JSON object")
	}
	if auth.Username == "" || auth.Password == "" {
		return CoreAuth{}, errors.New("Bitcoin Core auth file requires username and password")
	}
	return auth, nil
}

type CoreObserver struct {
	endpoint string
	auth     CoreAuth
	client   *http.Client
	mu       sync.Mutex
	requests int
}

func NewCoreObserver(endpoint string, auth CoreAuth, client *http.Client) (*CoreObserver, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Bitcoin Core URL must be an explicit HTTP(S) endpoint without inline credentials")
	}
	if auth.Username == "" || auth.Password == "" {
		return nil, errors.New("Bitcoin Core auth file requires username and password")
	}
	if client == nil {
		client = boundedHTTPClient()
	}
	return &CoreObserver{endpoint: endpoint, auth: auth, client: client}, nil
}

func (observer *CoreObserver) Observe(ctx context.Context, height uint64) (BlockObservation, error) {
	hashValue, err := observer.rpc(ctx, "getblockhash", []any{height})
	if err != nil {
		return BlockObservation{}, withObservationSources(err, "bitcoin-core")
	}
	var hash string
	if err := json.Unmarshal(hashValue, &hash); err != nil {
		return BlockObservation{}, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, errors.New("invalid block hash"))
	}
	headerValue, err := observer.rpc(ctx, "getblockheader", []any{hash, false})
	if err != nil {
		return BlockObservation{}, withObservationSources(err, "bitcoin-core")
	}
	var headerHex string
	if err := json.Unmarshal(headerValue, &headerHex); err != nil {
		return BlockObservation{}, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, errors.New("invalid block header"))
	}
	header, err := hex.DecodeString(headerHex)
	if err != nil || len(header) != 80 {
		return BlockObservation{}, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, errors.New("malformed block header"))
	}
	if err := verifyBlockHeader(header, hash); err != nil {
		return BlockObservation{}, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, err)
	}
	return BlockObservation{Height: height, Hash: strings.ToLower(hash), HeaderHex: strings.ToLower(headerHex), MerkleRoot: hex.EncodeToString(header[36:68]), BlockTime: time.Unix(int64(binary.LittleEndian.Uint32(header[68:72])), 0).UTC(), SourceIDs: []string{"bitcoin-core"}}, nil
}

func (observer *CoreObserver) rpc(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	observer.mu.Lock()
	if observer.requests >= Profile().MaximumHTTPRequests {
		observer.mu.Unlock()
		return nil, newObservationError(ObservationBudgetExhausted, []string{"bitcoin-core"}, errors.New("request budget exhausted"))
	}
	observer.requests++
	observer.mu.Unlock()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "forecast-ledger", "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, observer.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, err)
	}
	request.SetBasicAuth(observer.auth.Username, observer.auth.Password)
	request.Header.Set("Content-Type", "application/json")
	response, err := observer.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, newObservationError(ObservationSourceUnavailable, []string{"bitcoin-core"}, err)
	}
	defer response.Body.Close()
	data, err := readNetworkBody(response.Body, 1<<20)
	if err != nil {
		return nil, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, newObservationError(ObservationSourceUnavailable, []string{"bitcoin-core"}, fmt.Errorf("HTTP %d", response.StatusCode))
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || len(envelope.Result) == 0 {
		return nil, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, errors.New("invalid JSON-RPC response"))
	}
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		return nil, newObservationError(ObservationInconclusive, []string{"bitcoin-core"}, errors.New("RPC error"))
	}
	return envelope.Result, nil
}

func (observer *CoreObserver) Summary() RequestSummary {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	profile := Profile()
	return RequestSummary{HTTPRequests: observer.requests, MaxHeights: profile.MaximumUniqueHeights, MaxRequests: profile.MaximumHTTPRequests, MaxConcurrent: 1}
}
