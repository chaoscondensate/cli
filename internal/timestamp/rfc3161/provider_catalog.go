package rfc3161

import (
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const (
	ProviderAuto    = "auto"
	ProviderFreeTSA = "freetsa"
)

type providerTransport uint8

const (
	providerTransportHTTPS providerTransport = iota + 1
	providerTransportHTTP
)

// ProviderProfile is an immutable catalog value. Its transport authorization
// cannot be created from CLI, MCP, environment, configuration, or URL input.
type ProviderProfile struct {
	id              string
	endpoint        string
	transport       providerTransport
	bundle          []byte
	bundleSHA256    string
	sourceURL       string
	reviewedAt      string
	requestGuidance string
	trustNotAfter   string
}

func (p ProviderProfile) ID() string              { return p.id }
func (p ProviderProfile) Endpoint() string        { return p.endpoint }
func (p ProviderProfile) BundleSHA256() string    { return p.bundleSHA256 }
func (p ProviderProfile) SourceURL() string       { return p.sourceURL }
func (p ProviderProfile) ReviewedAt() string      { return p.reviewedAt }
func (p ProviderProfile) RequestGuidance() string { return p.requestGuidance }
func (p ProviderProfile) TrustNotAfter() string   { return p.trustNotAfter }

func (p ProviderProfile) Bundle() []byte {
	return append([]byte(nil), p.bundle...)
}

func (p ProviderProfile) TrustPath() string {
	return path.Join("trust", "rfc3161", p.id+"-"+p.bundleSHA256+".pem")
}

func (p ProviderProfile) Transport() string {
	switch p.transport {
	case providerTransportHTTPS:
		return "https"
	case providerTransportHTTP:
		return "http"
	default:
		return ""
	}
}

//go:embed providers/freetsa/ca.pem
var providerAssets embed.FS

var providerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

var releasedProviders = mustProviderCatalog()

func mustProviderCatalog() []ProviderProfile {
	bundle, err := providerAssets.ReadFile("providers/freetsa/ca.pem")
	if err != nil {
		panic(err)
	}
	digest := sha256.Sum256(bundle)
	profiles := []ProviderProfile{{
		id:              ProviderFreeTSA,
		endpoint:        "https://freetsa.org/tsr",
		transport:       providerTransportHTTPS,
		bundle:          bundle,
		bundleSHA256:    hex.EncodeToString(digest[:]),
		sourceURL:       "https://freetsa.org/files/cacert.pem",
		reviewedAt:      "2026-08-30",
		requestGuidance: "Anonymous RFC 3161 requests over SHA-256 file hashes; best effort with no numeric public rate limit.",
		trustNotAfter:   "2041-03-07T01:52:13Z",
	}}
	if err := validateProviderCatalog(profiles); err != nil {
		panic(err)
	}
	return profiles
}

// Providers returns the released provider order with independently cloned
// trust bytes. The current release contains only FreeTSA.
func Providers() []ProviderProfile {
	result := make([]ProviderProfile, len(releasedProviders))
	for index, profile := range releasedProviders {
		result[index] = profile
		result[index].bundle = profile.Bundle()
	}
	return result
}

func ProviderByID(id string) (ProviderProfile, bool) {
	for _, profile := range releasedProviders {
		if id == profile.id {
			profile.bundle = profile.Bundle()
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

// ProviderByEndpoint identifies only an exact released catalog endpoint. It
// never promotes a caller-supplied lookalike URL into a built-in provider.
func ProviderByEndpoint(endpoint string) (ProviderProfile, bool) {
	for _, profile := range releasedProviders {
		if endpoint == profile.endpoint {
			profile.bundle = profile.Bundle()
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

func ValidateProviderCatalog() error {
	return validateProviderCatalog(releasedProviders)
}

func validateProviderCatalog(profiles []ProviderProfile) error {
	if len(profiles) == 0 {
		return fmt.Errorf("RFC 3161 provider catalog is empty")
	}
	seen := make(map[string]struct{}, len(profiles))
	for index, profile := range profiles {
		if !providerIDPattern.MatchString(profile.id) || profile.id == ProviderAuto {
			return fmt.Errorf("RFC 3161 provider %d has an invalid ID", index)
		}
		if _, exists := seen[profile.id]; exists {
			return fmt.Errorf("RFC 3161 provider ID %q is duplicated", profile.id)
		}
		seen[profile.id] = struct{}{}
		parsed, err := url.Parse(profile.endpoint)
		if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path == "" {
			return fmt.Errorf("RFC 3161 provider %q has an invalid endpoint", profile.id)
		}
		expectedScheme := profile.Transport()
		if expectedScheme == "" || parsed.Scheme != expectedScheme || parsed.Hostname() != strings.ToLower(parsed.Hostname()) || parsed.Host != parsed.Hostname() || parsed.String() != profile.endpoint {
			return fmt.Errorf("RFC 3161 provider %q endpoint does not match its exact transport profile", profile.id)
		}
		if len(profile.bundle) == 0 || len(profile.bundle) > MaxCABundleBytes {
			return fmt.Errorf("RFC 3161 provider %q trust bundle is outside limits", profile.id)
		}
		digest := sha256.Sum256(profile.bundle)
		if profile.bundleSHA256 != hex.EncodeToString(digest[:]) {
			return fmt.Errorf("RFC 3161 provider %q trust bundle digest is wrong", profile.id)
		}
		if err := ValidateCABundle(profile.bundle, DefaultLimits()); err != nil {
			return fmt.Errorf("RFC 3161 provider %q trust bundle is invalid: %w", profile.id, err)
		}
		if err := validateProviderTrustProfile(profile.bundle, profile.trustNotAfter); err != nil {
			return fmt.Errorf("RFC 3161 provider %q trust profile is invalid: %w", profile.id, err)
		}
		source, err := url.Parse(profile.sourceURL)
		if err != nil || source.Scheme != "https" || source.Host == "" || source.User != nil || source.RawQuery != "" || source.Fragment != "" {
			return fmt.Errorf("RFC 3161 provider %q source URL is invalid", profile.id)
		}
		if profile.reviewedAt == "" || profile.requestGuidance == "" || strings.ContainsAny(profile.endpoint, "\r\n\t") || strings.ContainsAny(profile.requestGuidance, "\r\n") {
			return fmt.Errorf("RFC 3161 provider %q provenance is incomplete", profile.id)
		}
		if profile.TrustPath() != path.Clean(profile.TrustPath()) || strings.HasPrefix(profile.TrustPath(), "/") || strings.Contains(profile.TrustPath(), "..") {
			return fmt.Errorf("RFC 3161 provider %q trust path is not portable", profile.id)
		}
	}
	return nil
}

func validateProviderTrustProfile(bundle []byte, expectedNotAfter string) error {
	rest := bundle
	count := 0
	latest := ""
	for len(strings.TrimSpace(string(rest))) > 0 {
		block, remaining := pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return fmt.Errorf("bundle contains non-certificate PEM data")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil || !certificate.BasicConstraintsValid || !certificate.IsCA || certificate.KeyUsage&x509.KeyUsageCertSign == 0 || weakSignature(certificate.SignatureAlgorithm) || !strongPublicKey(certificate) {
			return fmt.Errorf("bundle contains a certificate outside the CA strength profile")
		}
		value := certificate.NotAfter.UTC().Format("2006-01-02T15:04:05Z")
		if value > latest {
			latest = value
		}
		count++
		rest = remaining
	}
	if count == 0 || expectedNotAfter == "" || latest != expectedNotAfter {
		return fmt.Errorf("bundle expiry metadata does not match the embedded certificates")
	}
	return nil
}
