package rfc3161

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestReleasedProviderCatalogIsExactAndImmutable(t *testing.T) {
	if err := ValidateProviderCatalog(); err != nil {
		t.Fatal(err)
	}
	profiles := Providers()
	if len(profiles) != 1 {
		t.Fatalf("providers = %d", len(profiles))
	}
	profile := profiles[0]
	if profile.ID() != ProviderFreeTSA || profile.Endpoint() != "https://freetsa.org/tsr" || profile.Transport() != "https" || profile.ReviewedAt() != "2026-08-30" || profile.TrustNotAfter() != "2041-03-07T01:52:13Z" {
		t.Fatalf("profile = %#v", profile)
	}
	if profile.BundleSHA256() != "2151b61137ffa86bf664691ba67e7da0b19f98c758e3d228d5d8ebf27e044438" || profile.TrustPath() != "trust/rfc3161/freetsa-2151b61137ffa86bf664691ba67e7da0b19f98c758e3d228d5d8ebf27e044438.pem" {
		t.Fatalf("trust metadata = %s %s", profile.BundleSHA256(), profile.TrustPath())
	}
	mutated := profile.Bundle()
	mutated[0] ^= 0xff
	again, ok := ProviderByID(ProviderFreeTSA)
	if !ok || string(mutated) == string(again.Bundle()) {
		t.Fatal("catalog trust bytes were mutable")
	}
	if _, ok := ProviderByID(ProviderAuto); ok {
		t.Fatal("auto resolved as a concrete provider")
	}
	if byEndpoint, ok := ProviderByEndpoint(profile.Endpoint()); !ok || byEndpoint.ID() != ProviderFreeTSA {
		t.Fatal("exact released endpoint was not identified")
	}
	for _, endpoint := range []string{"http://freetsa.org/tsr", "https://freetsa.org:443/tsr", "https://FreeTSA.org/tsr", "https://freetsa.org/tsr/"} {
		if _, ok := ProviderByEndpoint(endpoint); ok {
			t.Fatalf("lookalike endpoint %q identified as built-in", endpoint)
		}
	}
}

func TestProviderCatalogRejectsInvalidProfiles(t *testing.T) {
	valid, _ := ProviderByID(ProviderFreeTSA)
	tests := map[string]func(*ProviderProfile){
		"duplicate":     func(profile *ProviderProfile) {},
		"HTTP mismatch": func(profile *ProviderProfile) { profile.transport = providerTransportHTTP },
		"query":         func(profile *ProviderProfile) { profile.endpoint += "?secret=value" },
		"credentials":   func(profile *ProviderProfile) { profile.endpoint = "https://user@freetsa.org/tsr" },
		"bundle":        func(profile *ProviderProfile) { profile.bundle[0] ^= 0xff },
		"source":        func(profile *ProviderProfile) { profile.sourceURL = "http://freetsa.org/ca.pem" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			profile := valid
			profile.bundle = valid.Bundle()
			mutate(&profile)
			profiles := []ProviderProfile{profile}
			if name == "duplicate" {
				profiles = append(profiles, profile)
			}
			if err := validateProviderCatalog(profiles); err == nil {
				t.Fatal("invalid catalog succeeded")
			}
		})
	}

	httpBundle := valid.Bundle()
	digest := sha256.Sum256(httpBundle)
	syntheticHTTP := ProviderProfile{id: "synthetic-http", endpoint: "http://tsa.example.test/stamp", transport: providerTransportHTTP, bundle: httpBundle, bundleSHA256: hex.EncodeToString(digest[:]), sourceURL: "https://operator.example.test/ca.pem", reviewedAt: "2026-08-30", requestGuidance: "Synthetic test profile.", trustNotAfter: valid.TrustNotAfter()}
	if err := validateProviderCatalog([]ProviderProfile{syntheticHTTP}); err != nil {
		t.Fatalf("exact compiled HTTP profile = %v", err)
	}
	if strings.Contains(strings.Join([]string{valid.ID(), valid.Endpoint()}, " "), "http://") {
		t.Fatal("released catalog unexpectedly contains HTTP")
	}
}
