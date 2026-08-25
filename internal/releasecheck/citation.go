// Package releasecheck validates metadata that must agree with a release tag.
package releasecheck

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	canonicalRepository = "https://github.com/chaoscondensate/cli"
	approvedLicense     = "Apache-2.0"
	approvedGivenName   = "Andrey"
	approvedFamilyName  = "Korchak"
	approvedAlias       = "57uff3r"
)

type citation struct {
	CFFVersion     string   `yaml:"cff-version"`
	Message        string   `yaml:"message"`
	Title          string   `yaml:"title"`
	Type           string   `yaml:"type"`
	Authors        []author `yaml:"authors"`
	Version        string   `yaml:"version"`
	DateReleased   string   `yaml:"date-released"`
	License        string   `yaml:"license"`
	RepositoryCode string   `yaml:"repository-code"`
	URL            string   `yaml:"url"`
	Abstract       string   `yaml:"abstract"`
	Keywords       []string `yaml:"keywords"`
}

type author struct {
	FamilyName string `yaml:"family-names"`
	GivenName  string `yaml:"given-names"`
	Alias      string `yaml:"alias"`
}

// CheckCitation validates the supported CFF profile and release-specific
// fields. expectedVersion does not include a leading "v".
func CheckCitation(path, expectedVersion, expectedDate string) error {
	if strings.TrimSpace(expectedVersion) == "" {
		return errors.New("expected release version is required")
	}
	if _, err := time.Parse(time.DateOnly, expectedDate); err != nil {
		return fmt.Errorf("expected release date must use YYYY-MM-DD: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open citation file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var got citation
	if err := decoder.Decode(&got); err != nil {
		return fmt.Errorf("decode citation file: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("citation file must contain one YAML document")
		}
		return fmt.Errorf("decode citation file trailer: %w", err)
	}

	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"cff-version", got.CFFVersion, "1.2.0"},
		{"title", got.Title, "Forecast Ledger CLI"},
		{"type", got.Type, "software"},
		{"version", got.Version, expectedVersion},
		{"date-released", got.DateReleased, expectedDate},
		{"license", got.License, approvedLicense},
		{"repository-code", got.RepositoryCode, canonicalRepository},
		{"url", got.URL, canonicalRepository},
	}
	for _, check := range checks {
		if check.got != check.want {
			return fmt.Errorf("citation %s is %q; want %q", check.field, check.got, check.want)
		}
	}
	if strings.TrimSpace(got.Message) == "" || strings.TrimSpace(got.Abstract) == "" {
		return errors.New("citation message and abstract are required")
	}
	if len(got.Keywords) == 0 {
		return errors.New("citation must include at least one keyword")
	}
	if len(got.Authors) != 1 {
		return fmt.Errorf("citation must name exactly one approved author; got %d", len(got.Authors))
	}
	wantAuthor := author{
		FamilyName: approvedFamilyName,
		GivenName:  approvedGivenName,
		Alias:      approvedAlias,
	}
	if got.Authors[0] != wantAuthor {
		return fmt.Errorf("citation author is %#v; want %#v", got.Authors[0], wantAuthor)
	}
	if _, err := time.Parse(time.DateOnly, got.DateReleased); err != nil {
		return fmt.Errorf("citation date-released must use YYYY-MM-DD: %w", err)
	}
	return nil
}
