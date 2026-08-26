package ots

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type CalendarResult struct {
	SourceID string
	Identity string
	Branch   Sequence
	Err      error
}

type CalendarClient struct {
	HTTPClient *http.Client
	Resolver   *net.Resolver
}

func NewCalendarClient() *CalendarClient {
	return &CalendarClient{HTTPClient: boundedHTTPClient(), Resolver: net.DefaultResolver}
}

func (client *CalendarClient) StampPublic(ctx context.Context, commitment [32]byte) ([]CalendarResult, error) {
	profile := Profile()
	results := make([]CalendarResult, len(profile.Calendars))
	var wait sync.WaitGroup
	for index, source := range profile.Calendars {
		index, source := index, source
		wait.Add(1)
		go func() {
			defer wait.Done()
			branch, identity, err := client.stamp(ctx, source.SubmissionEndpoint, commitment)
			if err == nil && !acceptedCalendarIdentity(source, identity) {
				err = fmt.Errorf("calendar %s returned unapproved identity", source.ID)
			}
			results[index] = CalendarResult{SourceID: source.ID, Identity: identity, Branch: branch, Err: err}
		}()
	}
	wait.Wait()
	success := 0
	for _, result := range results {
		if result.Err == nil {
			success++
		}
	}
	if success < profile.CalendarMinimum {
		return results, fmt.Errorf("only %d of %d required calendars returned valid receipts", success, profile.CalendarMinimum)
	}
	return results, nil
}

func (client *CalendarClient) StampCustom(ctx context.Context, endpoints []string, minimum int, commitment [32]byte) ([]CalendarResult, error) {
	validated, err := ValidateCustomCalendars(ctx, client.Resolver, endpoints, minimum)
	if err != nil {
		return nil, err
	}
	results := make([]CalendarResult, len(validated))
	var wait sync.WaitGroup
	for index, endpoint := range validated {
		index, endpoint := index, endpoint
		wait.Add(1)
		go func() {
			defer wait.Done()
			branch, identity, err := client.stamp(ctx, endpoint, commitment)
			results[index] = CalendarResult{SourceID: safeOrigin(endpoint), Identity: identity, Branch: branch, Err: err}
		}()
	}
	wait.Wait()
	success := 0
	for _, result := range results {
		if result.Err == nil {
			success++
		}
	}
	if success < minimum {
		return results, fmt.Errorf("only %d of %d required custom calendars returned valid receipts", success, minimum)
	}
	return results, nil
}

func (client *CalendarClient) stamp(ctx context.Context, endpoint string, commitment [32]byte) (Sequence, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/digest", bytes.NewReader(commitment[:]))
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Accept", "application/vnd.opentimestamps.v1")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "forecast-ledger/ots-v1")
	response, err := client.httpClient().Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("calendar returned HTTP %d", response.StatusCode)
	}
	data, err := readNetworkBody(response.Body, MaxReceiptBytes)
	if err != nil {
		return nil, "", err
	}
	sequences, err := ParseCalendarResponse(data)
	if err != nil || len(sequences) != 1 {
		return nil, "", errors.New("calendar returned an invalid proof branch")
	}
	attestation := sequences[0][len(sequences[0])-1].Attestation
	if attestation == nil || attestation.Kind != AttestationPending {
		return nil, "", errors.New("new calendar receipt is not pending")
	}
	return sequences[0], strings.TrimRight(attestation.Calendar, "/"), nil
}

func (client *CalendarClient) Upgrade(ctx context.Context, calendar string, commitment []byte) ([]Sequence, error) {
	if len(commitment) == 0 || len(commitment) > 4096 {
		return nil, errors.New("invalid calendar commitment")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(calendar, "/")+"/timestamp/"+fmt.Sprintf("%x", commitment), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.opentimestamps.v1")
	request.Header.Set("User-Agent", "forecast-ledger/ots-v1")
	response, err := client.httpClient().Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, errors.New("calendar proof is not ready")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("calendar returned HTTP %d", response.StatusCode)
	}
	data, err := readNetworkBody(response.Body, MaxReceiptBytes)
	if err != nil {
		return nil, err
	}
	return ParseCalendarResponse(data)
}

func ValidateCustomCalendars(ctx context.Context, resolver *net.Resolver, endpoints []string, minimum int) ([]string, error) {
	if len(endpoints) == 0 || minimum < 1 || minimum > len(endpoints) {
		return nil, errors.New("custom calendars require a threshold between one and the number of endpoints")
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	seen := make(map[string]struct{}, len(endpoints))
	result := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
			return nil, errors.New("custom calendars must be distinct public HTTPS origins")
		}
		host := strings.ToLower(parsed.Hostname())
		if host == "localhost" {
			return nil, errors.New("custom calendar host is not public")
		}
		addresses, err := resolver.LookupIPAddr(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, errors.New("custom calendar host cannot be resolved safely")
		}
		for _, address := range addresses {
			ip, ok := netip.AddrFromSlice(address.IP)
			if !ok || !ip.Unmap().IsGlobalUnicast() || ip.Unmap().IsPrivate() || ip.Unmap().IsLoopback() || ip.Unmap().IsLinkLocalUnicast() {
				return nil, errors.New("custom calendar resolves to a non-public address")
			}
		}
		origin := safeOrigin(endpoint)
		if _, exists := seen[origin]; exists {
			return nil, errors.New("custom calendar origins must be distinct")
		}
		seen[origin] = struct{}{}
		result = append(result, origin)
	}
	sort.Strings(result)
	return result, nil
}

func safeOrigin(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "invalid-calendar"
	}
	return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
}

func (client *CalendarClient) httpClient() *http.Client {
	if client != nil && client.HTTPClient != nil {
		return client.HTTPClient
	}
	return boundedHTTPClient()
}

func boundedHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 15 * time.Second
	transport.MaxIdleConnsPerHost = 2
	return &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("network redirects are disabled")
	}}
}

func readNetworkBody(reader io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("network response exceeds its byte limit")
	}
	return data, nil
}
