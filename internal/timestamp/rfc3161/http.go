package rfc3161

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	tspclient "github.com/notaryproject/tspclient-go"
)

const DefaultHTTPTimeout = 15 * time.Second

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type SubmitResult struct {
	Response     []byte `json:"-"`
	RequestCount int    `json:"request_count"`
	TSAOrigin    string `json:"tsa_origin"`
}

// HTTPClient is a constrained one-request TSA client. A custom Client is only
// an internal test seam; production uses a pinned-address transport built here.
type HTTPClient struct {
	Client   *http.Client
	Resolver Resolver
	Limits   Limits
	Timeout  time.Duration
}

func (c HTTPClient) Submit(ctx context.Context, endpoint string, request []byte) (SubmitResult, error) {
	limits := c.Limits.normalized()
	if len(request) == 0 || len(request) > limits.RequestBytes {
		return SubmitResult{}, failure(ReasonLimit, "timestamp request exceeds the supported size")
	}
	parsed, addresses, err := c.validateEndpoint(ctx, endpoint)
	if err != nil {
		return SubmitResult{}, err
	}
	client := c.Client
	if client == nil {
		client = c.productionClient(parsed, addresses)
	} else {
		clone := *client
		clone.CheckRedirect = sameOriginRedirect(parsed)
		client = &clone
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), bytes.NewReader(request))
	if err != nil {
		return SubmitResult{}, failure(ReasonRequestMalformed, "timestamp HTTP request could not be created")
	}
	req.Header.Set("Content-Type", tspclient.MediaTypeTimestampQuery)
	req.Header.Set("Accept", tspclient.MediaTypeTimestampReply)
	resp, err := client.Do(req)
	if err != nil {
		return SubmitResult{}, failure(ReasonResponseRejected, "timestamp authority request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return SubmitResult{}, failure(ReasonResponseRejected, "timestamp authority returned a non-success HTTP status")
	}
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, tspclient.MediaTypeTimestampReply) {
		return SubmitResult{}, failure(ReasonResponseRejected, "timestamp authority returned an unsupported media type")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(limits.ResponseBytes)+1))
	if err != nil {
		return SubmitResult{}, failure(ReasonResponseRejected, "timestamp authority response could not be read")
	}
	if len(body) == 0 || len(body) > limits.ResponseBytes {
		return SubmitResult{}, failure(ReasonLimit, "timestamp authority response exceeds the supported size")
	}
	if _, err := parseResponse(body, limits); err != nil {
		return SubmitResult{}, err
	}
	return SubmitResult{Response: body, RequestCount: 1, TSAOrigin: normalizedOrigin(parsed)}, nil
}

func (c HTTPClient) validateEndpoint(ctx context.Context, raw string) (*url.URL, []net.IPAddr, error) {
	normalized, err := NormalizeEndpoint(raw)
	if err != nil {
		return nil, nil, err
	}
	parsed, _ := url.Parse(normalized)
	resolver := c.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addresses, err := resolver.LookupIPAddr(ctx, parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return nil, nil, failure(ReasonResponseRejected, "timestamp authority host could not be resolved")
	}
	for _, address := range addresses {
		if !publicIP(address.IP) {
			return nil, nil, failure(ReasonRequestProfile, "timestamp authority resolves to a non-public address")
		}
	}
	return parsed, addresses, nil
}

// NormalizeEndpoint performs the no-network part of TSA endpoint validation
// and returns a stable credential-free URL for path derivation and storage.
func NormalizeEndpoint(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" {
		return "", failure(ReasonRequestProfile, "timestamp authority URL must be public HTTPS without credentials, query, or fragment")
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return "", failure(ReasonRequestProfile, "timestamp authority URL must use the HTTPS default port")
	}
	parsed.Scheme = "https"
	parsed.Host = strings.ToLower(parsed.Hostname())
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), nil
}

func (c HTTPClient) productionClient(endpoint *url.URL, addresses []net.IPAddr) *http.Client {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:               nil,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: timeout,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, candidate := range addresses {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
	return &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: sameOriginRedirect(endpoint)}
}

func sameOriginRedirect(origin *url.URL) func(*http.Request, []*http.Request) error {
	expected := normalizedOrigin(origin)
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || normalizedOrigin(req.URL) != expected {
			return fmt.Errorf("timestamp authority redirect is outside the validated origin")
		}
		return nil
	}
}

func normalizedOrigin(value *url.URL) string {
	host := strings.ToLower(value.Hostname())
	port := value.Port()
	if port == "" {
		port = "443"
	}
	return "https://" + net.JoinHostPort(host, port)
}

func publicIP(value net.IP) bool {
	address, ok := netip.AddrFromSlice(value)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range reservedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}
