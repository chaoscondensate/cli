package ots

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const officialIncompleteHex = "004f70656e54696d657374616d7073000050726f6f6600bf89e2e884e89294010805c4f616a8e5310d19d938cfd769864d7f4ccdc2ca8b479b10af83564b097af9f010e754bf93806a7ebaa680ef7bd0114bf408f010b573e8850cfd9e63d1f043fbb6fc250e08f10457cfa5c4f0086fb1ac8d4e4eb0e70083dfe30d2ef90c8e2e2d68747470733a2f2f616c6963652e6274632e63616c656e6461722e6f70656e74696d657374616d70732e6f7267"
const officialTwoCalendarsHex = "004f70656e54696d657374616d7073000050726f6f6600bf89e2e884e892940108efaa174f68e59705757460f4f7d204bd2b535cfd194d9d945418732129404ddbf010839037eef449dec6dac322ca97347c4508fff0106b4023b6edd3a0eeeb09e5d718723b9e08f10457d46515f008eadd66b1688d55740083dfe30d2ef90c8e2e2d68747470733a2f2f616c6963652e6274632e63616c656e6461722e6f70656e74696d657374616d70732e6f7267f010a3ad701ef9f10535a84968b5a99d858008f10457d46516f008647b90ea1b270a970083dfe30d2ef90c8e2c2b68747470733a2f2f626f622e6274632e63616c656e6461722e6f70656e74696d657374616d70732e6f7267"

func TestOfficialPendingReceiptsRoundTrip(t *testing.T) {
	for _, fixture := range []string{officialIncompleteHex, officialTwoCalendarsHex} {
		data := decodeFixture(t, fixture)
		receipt, err := ParseReceipt(data)
		if err != nil {
			t.Fatalf("ParseReceipt: %v", err)
		}
		encoded, err := receipt.Serialize()
		if err != nil {
			t.Fatalf("Serialize: %v", err)
		}
		if !bytes.Equal(encoded, data) {
			t.Fatalf("official receipt did not round-trip\n got %x\nwant %x", encoded, data)
		}
	}
}

func TestNonceBlindingAndReceiptBinding(t *testing.T) {
	target := []byte("canonical forecast target")
	digest := sha256.Sum256(target)
	var nonce [16]byte
	copy(nonce[:], []byte("0123456789abcdef"))
	blinded := Blind(digest, nonce)
	expected := sha256.Sum256(append(append([]byte{}, digest[:]...), nonce[:]...))
	if blinded != expected {
		t.Fatal("blinding is not append nonce then SHA-256")
	}
	branch := Sequence{{Operation: &Operation{Tag: OperationAppend, Argument: []byte("branch")}}, {Attestation: &Attestation{Kind: AttestationPending, Calendar: "https://alice.btc.calendar.opentimestamps.org"}}}
	receipt, err := NewPendingReceipt(digest, nonce, []Sequence{branch})
	if err != nil {
		t.Fatal(err)
	}
	if err := receipt.VerifyBinding(target); err != nil {
		t.Fatal(err)
	}
	if err := receipt.VerifyBinding([]byte("other")); err == nil {
		t.Fatal("receipt accepted different target bytes")
	}
}

func TestSemanticSupersetAllowsPendingUpgrade(t *testing.T) {
	digest := sha256.Sum256([]byte("target"))
	prefix := Sequence{{Operation: &Operation{Tag: OperationAppend, Argument: []byte("x")}}, {Operation: &Operation{Tag: OperationSHA256}}}
	pending := append(cloneSequence(prefix), Step{Attestation: &Attestation{Kind: AttestationPending, Calendar: "https://alice.btc.calendar.opentimestamps.org"}})
	confirmed := append(cloneSequence(prefix), Step{Operation: &Operation{Tag: OperationPrepend, Argument: []byte("merkle")}}, Step{Attestation: &Attestation{Kind: AttestationBitcoin, Height: 800000}})
	original := &Receipt{Digest: digest, Sequences: []Sequence{pending}}
	candidate := &Receipt{Digest: digest, Sequences: []Sequence{confirmed}}
	if !IsSemanticSuperset(candidate, original) {
		t.Fatal("confirmed extension was not treated as a semantic superset")
	}
}

func TestProfileIsPinned(t *testing.T) {
	profile := Profile()
	if profile.ID != PublicProfileID || len(profile.Calendars) != 4 || profile.CalendarMinimum != 2 || len(profile.BitcoinSources) != 2 || profile.MaximumUniqueHeights != 32 || profile.MaximumHTTPRequests != 128 || profile.MaximumConcurrentHTTP != 4 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestCalendarPublicThresholdAndIdentity(t *testing.T) {
	responseBranch := func(identity string) []byte {
		encoded, err := SerializeCalendarResponse([]Sequence{{{Attestation: &Attestation{Kind: AttestationPending, Calendar: identity}}}})
		if err != nil {
			t.Fatal(err)
		}
		return encoded
	}
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		identity := "https://unknown.example"
		switch request.URL.Host {
		case "a.pool.opentimestamps.org":
			identity = "https://alice.btc.calendar.opentimestamps.org"
		case "b.pool.opentimestamps.org":
			identity = "https://bob.btc.calendar.opentimestamps.org"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(responseBranch(identity))), Header: make(http.Header)}, nil
	})
	client := &CalendarClient{HTTPClient: &http.Client{Transport: transport}}
	results, err := client.StampPublic(context.Background(), sha256.Sum256([]byte("commitment")))
	if err != nil || len(results) != 4 {
		t.Fatalf("StampPublic = %v, %v", results, err)
	}
	successes := 0
	for _, result := range results {
		if result.Err == nil {
			successes++
		}
	}
	if successes != 2 {
		t.Fatalf("accepted calendar identities = %d, want 2", successes)
	}

	remapped := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		identity := "https://unknown.example"
		if request.URL.Host == "a.pool.opentimestamps.org" {
			identity = "https://bob.btc.calendar.opentimestamps.org"
		} else if request.URL.Host == "b.pool.opentimestamps.org" {
			identity = "https://alice.btc.calendar.opentimestamps.org"
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(responseBranch(identity))), Header: make(http.Header)}, nil
	})
	if _, err := (&CalendarClient{HTTPClient: &http.Client{Transport: remapped}}).StampPublic(context.Background(), sha256.Sum256([]byte("commitment"))); err == nil {
		t.Fatal("pool-remapped calendar identities reached the threshold")
	}
}

func TestCustomCalendarsRejectPrivateAddress(t *testing.T) {
	resolver := &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
		return nil, &net.DNSError{Err: "blocked in unit test"}
	}}
	if _, err := ValidateCustomCalendars(context.Background(), resolver, []string{"https://localhost"}, 1); err == nil {
		t.Fatal("localhost custom calendar accepted")
	}
}

func TestMalformedAndUnsupportedProofs(t *testing.T) {
	data := decodeFixture(t, officialIncompleteHex)
	for _, mutation := range [][]byte{nil, data[:10], append(append([]byte{}, data...), 0x08)} {
		if _, err := ParseReceipt(mutation); err == nil {
			t.Fatalf("malformed proof accepted: %x", mutation)
		}
	}
	unsupported := append([]byte{}, data...)
	unsupported[len(detachedMagic)+1+1+32] = 0x02
	if _, err := ParseReceipt(unsupported); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported operation error = %v", err)
	}
}

func TestReceiptLimitsRedirectsAndCancellation(t *testing.T) {
	if _, err := ParseReceipt(make([]byte, MaxReceiptBytes+1)); err == nil {
		t.Fatal("oversized receipt accepted")
	}

	deep := make(Sequence, 0, MaxProofDepth+1)
	for range MaxProofDepth {
		deep = append(deep, Step{Operation: &Operation{Tag: OperationSHA256}})
	}
	deep = append(deep, Step{Attestation: &Attestation{Kind: AttestationPending, Calendar: "https://calendar.example"}})
	if _, err := SerializeCalendarResponse([]Sequence{deep}); err == nil || !strings.Contains(err.Error(), "deep") {
		t.Fatalf("excessive proof depth error = %v", err)
	}

	client := boundedHTTPClient()
	request, _ := http.NewRequest(http.MethodGet, "https://calendar.example", nil)
	if err := client.CheckRedirect(request, []*http.Request{request}); err == nil {
		t.Fatal("network redirect accepted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	calendar := &CalendarClient{HTTPClient: &http.Client{Transport: transport}}
	_, err := calendar.StampPublic(ctx, sha256.Sum256([]byte("commitment")))
	if err == nil || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("cancelled calendar request error = %v", err)
	}
}

func TestPublicBitcoinObserverRequiresAgreementAndDeduplicates(t *testing.T) {
	const genesisHash = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	const genesisHeader = "0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c"
	var requests atomic.Int32
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		body := genesisHash
		if strings.HasSuffix(request.URL.Path, "/header") {
			body = genesisHeader
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	observer := NewPublicBitcoinObserver(&http.Client{Transport: transport})
	observation, err := observer.Observe(context.Background(), 1)
	if err != nil || observation.Hash != genesisHash || observation.MerkleRoot != genesisHeader[72:136] {
		t.Fatalf("Observe = %+v, %v", observation, err)
	}
	if strings.Join(observation.SourceIDs, ",") != "blockstream,mempool-space" {
		t.Fatalf("source IDs are not stable: %v", observation.SourceIDs)
	}
	if _, err := observer.Observe(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 4 || observer.Summary().HTTPRequests != 4 || observer.Summary().UniqueHeights != 1 {
		t.Fatalf("dedup summary = %+v, transport requests = %d", observer.Summary(), requests.Load())
	}
	header, _ := hex.DecodeString(genesisHeader)
	evaluated := EvaluatedAttestation{Message: append([]byte(nil), header[36:68]...), Attestation: Attestation{Kind: AttestationBitcoin, Height: 1}}
	if err := VerifyBitcoinAttestation(evaluated, observation); err != nil {
		t.Fatal(err)
	}
}

func TestPublicBitcoinObserverIsDeterministicAcrossResponseOrder(t *testing.T) {
	const genesisHash = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	const genesisHeader = "0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c"
	for _, delayedHost := range []string{"blockstream.info", "mempool.space"} {
		t.Run(delayedHost, func(t *testing.T) {
			transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Host == delayedHost {
					time.Sleep(5 * time.Millisecond)
				}
				body := genesisHash
				if strings.HasSuffix(request.URL.Path, "/header") {
					body = genesisHeader
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			observation, err := NewPublicBitcoinObserver(&http.Client{Transport: transport}).Observe(context.Background(), 1)
			if err != nil || strings.Join(observation.SourceIDs, ",") != "blockstream,mempool-space" {
				t.Fatalf("ordered observation=%#v err=%v", observation, err)
			}
		})
	}
}

func TestPublicBitcoinObserverClassifiesHeaderRejectionAndRequestBudget(t *testing.T) {
	const genesisHash = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	headerTransport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		body := genesisHash
		if strings.HasSuffix(request.URL.Path, "/header") {
			body = strings.Repeat("00", 80)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	_, err := NewPublicBitcoinObserver(&http.Client{Transport: headerTransport}).Observe(context.Background(), 1)
	assertObservationIssue(t, err, ObservationInconclusive, "blockstream,mempool-space")

	budgetObserver := NewPublicBitcoinObserver(&http.Client{Transport: headerTransport})
	budgetObserver.profile.MaximumHTTPRequests = 1
	_, err = budgetObserver.Observe(context.Background(), 1)
	assertObservationIssue(t, err, ObservationBudgetExhausted, "blockstream,mempool-space")
}

func TestPublicBitcoinObserverRejectsDisagreement(t *testing.T) {
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		value := strings.Repeat("0", 64)
		if request.URL.Host == "blockstream.info" {
			value = strings.Repeat("1", 64)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(value)), Header: make(http.Header)}, nil
	})
	observer := NewPublicBitcoinObserver(&http.Client{Transport: transport})
	if _, err := observer.Observe(context.Background(), 1); err == nil {
		t.Fatal("disagreeing sources accepted")
	} else {
		assertObservationIssue(t, err, ObservationInconclusive, "blockstream,mempool-space")
	}
}

func TestPublicBitcoinObserverOutageAndHeightBudget(t *testing.T) {
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("source outage")
	})
	observer := NewPublicBitcoinObserver(&http.Client{Transport: transport})
	for height := uint64(1); height <= uint64(Profile().MaximumUniqueHeights); height++ {
		if _, err := observer.Observe(context.Background(), height); err == nil {
			t.Fatalf("outage accepted at height %d", height)
		} else {
			assertObservationIssue(t, err, ObservationSourceUnavailable, "blockstream,mempool-space")
			if strings.Contains(err.Error(), "source outage") || strings.Contains(err.Error(), "http") {
				t.Fatalf("public observation error leaked cause or endpoint: %q", err)
			}
		}
	}
	if _, err := observer.Observe(context.Background(), uint64(Profile().MaximumUniqueHeights+1)); err == nil {
		t.Fatal("unique-height budget was not enforced")
	} else {
		assertObservationIssue(t, err, ObservationBudgetExhausted, "")
	}
}

func TestPublicBitcoinObserverClassifiesMalformedResponseDeterministically(t *testing.T) {
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		body := "not-a-block-hash"
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	observer := NewPublicBitcoinObserver(&http.Client{Transport: transport})
	_, err := observer.Observe(context.Background(), 10)
	assertObservationIssue(t, err, ObservationInconclusive, "blockstream,mempool-space")
	_, cachedErr := observer.Observe(context.Background(), 10)
	assertObservationIssue(t, cachedErr, ObservationInconclusive, "blockstream,mempool-space")
	if summary := observer.Summary(); summary.UniqueHeights != 1 || summary.HTTPRequests != 2 {
		t.Fatalf("malformed response cache summary = %#v", summary)
	}
}

func TestBitcoinCoreObserverUsesBoundedAuthenticatedRPC(t *testing.T) {
	const genesisHash = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	const genesisHeader = "0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c"
	var requests atomic.Int32
	transport := roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		username, password, ok := request.BasicAuth()
		if !ok || username != "rpc-user" || password != "rpc-password" {
			t.Fatal("Bitcoin Core request omitted protected basic authentication")
		}
		body, _ := io.ReadAll(request.Body)
		result := `{"result":"` + genesisHeader + `","error":null,"id":"forecast-ledger"}`
		if strings.Contains(string(body), "getblockhash") {
			result = `{"result":"` + genesisHash + `","error":null,"id":"forecast-ledger"}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(result)), Header: make(http.Header)}, nil
	})
	observer, err := NewCoreObserver("http://127.0.0.1:8332", CoreAuth{Username: "rpc-user", Password: "rpc-password"}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	observation, err := observer.Observe(context.Background(), 1)
	if err != nil || observation.Hash != genesisHash || requests.Load() != 2 || observer.Summary().HTTPRequests != 2 {
		t.Fatalf("Core observation = %#v, requests=%d, err=%v", observation, requests.Load(), err)
	}
}

func TestBitcoinCoreObserverClassifiesOutageWithoutLeakingCredentials(t *testing.T) {
	const secret = "rpc-secret-marker"
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New(secret)
	})
	observer, err := NewCoreObserver("http://127.0.0.1:8332", CoreAuth{Username: "rpc-user", Password: secret}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	_, err = observer.Observe(context.Background(), 1)
	assertObservationIssue(t, err, ObservationSourceUnavailable, "bitcoin-core")
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("Core observation error leaked secret or endpoint: %q", err)
	}
}

func TestBitcoinCoreObserverClassifiesMalformedObservation(t *testing.T) {
	transport := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"result":"not-a-hash","error":null,"id":"forecast-ledger"}`)), Header: make(http.Header)}, nil
	})
	observer, err := NewCoreObserver("http://127.0.0.1:8332", CoreAuth{Username: "rpc-user", Password: "rpc-password"}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	_, err = observer.Observe(context.Background(), 1)
	assertObservationIssue(t, err, ObservationInconclusive, "bitcoin-core")
}

func assertObservationIssue(t *testing.T, err error, wantKind ObservationIssueKind, wantSources string) {
	t.Helper()
	var issue *ObservationError
	if !errors.As(err, &issue) {
		t.Fatalf("observation error type = %T, %v", err, err)
	}
	if issue.Kind() != wantKind || strings.Join(issue.SourceIDs(), ",") != wantSources {
		t.Fatalf("observation issue kind=%q sources=%v, want kind=%q sources=%q", issue.Kind(), issue.SourceIDs(), wantKind, wantSources)
	}
}

func TestPublicProfileLiveness(t *testing.T) {
	if os.Getenv("FORECAST_LEDGER_OTS_LIVENESS") != "1" {
		t.Skip("set FORECAST_LEDGER_OTS_LIVENESS=1 in the scheduled network gate")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var commitment [32]byte
	if _, err := rand.Read(commitment[:]); err != nil {
		t.Fatal(err)
	}
	results, err := NewCalendarClient().StampPublic(ctx, commitment)
	if err != nil {
		t.Fatalf("public profile did not reach its fixed threshold: results=%#v err=%v", results, err)
	}
}

func FuzzParseReceipt(f *testing.F) {
	f.Add(decodeFixture(f, officialIncompleteHex))
	f.Add(decodeFixture(f, officialTwoCalendarsHex))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxReceiptBytes {
			return
		}
		receipt, err := ParseReceipt(data)
		if err != nil {
			return
		}
		encoded, err := receipt.Serialize()
		if err != nil {
			t.Fatalf("parsed receipt cannot be serialized: %v", err)
		}
		if _, err := ParseReceipt(encoded); err != nil {
			t.Fatalf("serialized receipt cannot be parsed: %v", err)
		}
	})
}

func FuzzReceiptEvaluation(f *testing.F) {
	f.Add(decodeFixture(f, officialIncompleteHex))
	f.Add(decodeFixture(f, officialTwoCalendarsHex))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxReceiptBytes {
			return
		}
		receipt, err := ParseReceipt(data)
		if err != nil {
			return
		}
		if _, err := receipt.Evaluate(); err != nil {
			t.Fatalf("parsed receipt cannot be evaluated: %v", err)
		}
	})
}

func decodeFixture(t testing.TB, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
