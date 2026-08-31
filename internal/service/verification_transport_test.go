package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type outcomeRoundTripper func(*http.Request) (*http.Response, error)

func (fn outcomeRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type sequenceOutcomeResolver struct {
	mu       sync.Mutex
	answers  [][]net.IPAddr
	calls    int
	lastHost string
}

func (r *sequenceOutcomeResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.lastHost = host
	index := r.calls
	r.calls++
	if index >= len(r.answers) {
		index = len(r.answers) - 1
	}
	return append([]net.IPAddr(nil), r.answers[index]...), nil
}

type recordingOutcomeDialer struct {
	addresses []string
	err       error
}

func (d *recordingOutcomeDialer) DialContext(ctx context.Context, _, address string) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.addresses = append(d.addresses, address)
	return nil, d.err
}

func TestOutcomeTransportBindsValidationToDialedAddress(t *testing.T) {
	resolver := &sequenceOutcomeResolver{answers: [][]net.IPAddr{
		{{IP: net.ParseIP("8.8.8.8")}},
		{{IP: net.ParseIP("127.0.0.1")}},
	}}
	dialer := &recordingOutcomeDialer{err: errors.New("stop after address selection")}
	client := boundedOutcomeClientWith(resolver, dialer)
	if err := validatePublicSourceURL(&url.URL{Scheme: "https", Host: "source.example", Path: "/outcome"}); err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	_, err := transport.DialContext(t.Context(), "tcp", "source.example:443")
	if err == nil || resolver.calls != 1 || resolver.lastHost != "source.example" || len(dialer.addresses) != 1 || dialer.addresses[0] != "8.8.8.8:443" {
		t.Fatalf("err=%v calls=%d host=%q dialed=%v", err, resolver.calls, resolver.lastHost, dialer.addresses)
	}
	if transport.Proxy != nil {
		t.Fatal("outcome transport unexpectedly honors environment proxies")
	}
}

func TestOutcomeTransportRejectsMixedPrivateAndReservedAnswersBeforeDial(t *testing.T) {
	for _, test := range []struct {
		name      string
		addresses []net.IPAddr
	}{
		{"mixed", []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("10.0.0.1")}}},
		{"cgnat", []net.IPAddr{{IP: net.ParseIP("100.64.0.1")}}},
		{"mapped loopback", []net.IPAddr{{IP: net.ParseIP("::ffff:127.0.0.1")}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := &sequenceOutcomeResolver{answers: [][]net.IPAddr{test.addresses}}
			dialer := &recordingOutcomeDialer{err: errors.New("must not dial")}
			transport := boundedOutcomeClientWith(resolver, dialer).Transport.(*http.Transport)
			_, err := transport.DialContext(t.Context(), "tcp", "source.example:443")
			if err == nil || !strings.Contains(err.Error(), "not public") || len(dialer.addresses) != 0 {
				t.Fatalf("err=%v dialed=%v", err, dialer.addresses)
			}
		})
	}
}

func TestOutcomeTransportRechecksRedirectDestinationAndCancellation(t *testing.T) {
	resolver := &sequenceOutcomeResolver{answers: [][]net.IPAddr{{{IP: net.ParseIP("127.0.0.1")}}}}
	dialer := &recordingOutcomeDialer{err: errors.New("must not dial")}
	client := boundedOutcomeClientWith(resolver, dialer)
	redirect := &http.Request{URL: &url.URL{Scheme: "https", Host: "redirected.example", Path: "/private"}}
	if err := client.CheckRedirect(redirect, []*http.Request{{URL: &url.URL{Scheme: "https", Host: "source.example"}}}); err == nil {
		t.Fatal("cross-origin redirect was accepted")
	}
	redirect.URL.Host = "source.example"
	if err := client.CheckRedirect(redirect, []*http.Request{{URL: &url.URL{Scheme: "https", Host: "source.example"}}}); err != nil {
		t.Fatalf("same-origin redirect syntax was rejected before connection policy: %v", err)
	}
	transport := client.Transport.(*http.Transport)
	_, err := transport.DialContext(t.Context(), "tcp", "source.example:443")
	if err == nil || len(dialer.addresses) != 0 {
		t.Fatalf("private redirect err=%v dialed=%v", err, dialer.addresses)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = transport.DialContext(ctx, "tcp", "source.example:443")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled dial error=%v", err)
	}
}

func TestOutcomeSourceResponseLimitIsEnforced(t *testing.T) {
	client := &http.Client{Transport: outcomeRoundTripper(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (4<<20)+1))), Request: request}, nil
	})}
	data, finalURL, err := fetchOutcomeSource(t.Context(), client, "https://source.example/outcome")
	if err == nil || data != nil || finalURL != "" || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("data=%d final=%q err=%v", len(data), finalURL, err)
	}
}
