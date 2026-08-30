package doccheck

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/service"
)

var markdownLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)[:space:]]+)(?:[[:space:]]+"[^"]*")?\)`)

type page struct {
	path     string
	rel      string
	content  string
	metadata map[string]string
	links    []string
}

func TestMaintainedYAMLInputsMatchOperationContracts(t *testing.T) {
	repositoryRoot := findRepositoryRoot(t)
	tests := []struct {
		path        string
		fence       int
		schema      service.InputSchemaName
		destination func() any
	}{
		{path: "docs/getting-started/create-ledger.md", fence: 0, schema: service.InputSchemaForecastSealPrivate, destination: func() any { return &service.SealedForecastPrivateInput{} }},
		{path: "docs/how-to/seal-and-reveal-forecasts.md", fence: 0, schema: service.InputSchemaForecastSealPrivate, destination: func() any { return &service.SealedForecastPrivateInput{} }},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(test.path)))
			if err != nil {
				t.Fatal(err)
			}
			fences := fencedExamples(string(content), "yaml")
			if test.fence >= len(fences) {
				t.Fatalf("YAML fence %d is missing", test.fence)
			}
			if err := service.DecodeOperationInput(context.Background(), "-", strings.NewReader(fences[test.fence]), test.schema, test.destination()); err != nil {
				t.Fatalf("maintained YAML input does not match %s: %v", test.schema, err)
			}
		})
	}
}

func fencedExamples(content, wantedLanguage string) []string {
	var examples []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	language := ""
	var body strings.Builder
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if language == "" {
			if strings.HasPrefix(trimmed, "```") {
				language = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				body.Reset()
			}
			continue
		}
		if trimmed == "```" {
			if language == wantedLanguage {
				examples = append(examples, body.String())
			}
			language = ""
			continue
		}
		body.WriteString(scanner.Text())
		body.WriteByte('\n')
	}
	return examples
}

func TestMaintainedDocumentation(t *testing.T) {
	t.Parallel()
	repositoryRoot := findRepositoryRoot(t)
	docsRoot := filepath.Join(repositoryRoot, "docs")
	pages := loadPages(t, docsRoot)

	requireSectionIndexes(t, docsRoot)
	for _, current := range pages {
		validateMetadata(t, repositoryRoot, current)
		validateFences(t, current)
		validateDataExamples(t, current)
	}
	validateLinksAndNavigation(t, repositoryRoot, docsRoot, pages)
}

func validateDataExamples(t *testing.T, current *page) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(current.content))
	language := ""
	startLine := 0
	var content strings.Builder
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		trimmed := strings.TrimSpace(scanner.Text())
		if language == "" {
			if trimmed == "```yaml" || trimmed == "```json" {
				language = strings.TrimPrefix(trimmed, "```")
				startLine = lineNumber + 1
				content.Reset()
			}
			continue
		}
		if trimmed == "```" {
			var err error
			if language == "yaml" {
				_, err = document.ParseYAML(bytes.NewBufferString(content.String()), document.DefaultLimits)
			} else {
				_, err = document.ParseJSON(bytes.NewBufferString(content.String()), document.DefaultLimits)
			}
			if err != nil {
				t.Errorf("%s:%d: %s example is not executable input: %v", current.rel, startLine, language, err)
			}
			language = ""
			continue
		}
		content.WriteString(scanner.Text())
		content.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		t.Errorf("%s: scan data examples: %v", current.rel, err)
	}
}

func findRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate documentation test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repository root is invalid: %v", err)
	}
	return root
}

func loadPages(t *testing.T, docsRoot string) map[string]*page {
	t.Helper()
	pages := make(map[string]*page)
	err := filepath.WalkDir(docsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(docsRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pages[rel] = &page{
			path:     path,
			rel:      rel,
			content:  string(content),
			metadata: parseMetadata(string(content)),
			links:    markdownLinks(string(content)),
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk documentation: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("no maintained documentation pages found")
	}
	return pages
}

func requireSectionIndexes(t *testing.T, docsRoot string) {
	t.Helper()
	err := filepath.WalkDir(docsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() || path == docsRoot {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, "index.md")); err != nil {
			rel, relErr := filepath.Rel(docsRoot, path)
			if relErr != nil {
				return relErr
			}
			t.Errorf("documentation directory %s has no index.md", filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("check documentation indexes: %v", err)
	}
}

func parseMetadata(content string) map[string]string {
	const startMarker = "<!-- doc-metadata\n"
	start := strings.Index(content, startMarker)
	if start < 0 {
		return nil
	}
	rest := content[start+len(startMarker):]
	end := strings.Index(rest, "\n-->")
	if end < 0 {
		return nil
	}
	metadata := make(map[string]string)
	for _, line := range strings.Split(rest[:end], "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		metadata[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return metadata
}

func validateMetadata(t *testing.T, repositoryRoot string, current *page) {
	t.Helper()
	required := []string{"coverage", "reviewed", "owner", "generated", "security-critical", "prerequisites", "next"}
	for _, field := range required {
		if strings.TrimSpace(current.metadata[field]) == "" {
			t.Errorf("%s: missing doc-metadata field %q", current.rel, field)
		}
	}
	if reviewed := current.metadata["reviewed"]; reviewed != "" {
		if _, err := time.Parse("2006-01-02", reviewed); err != nil {
			t.Errorf("%s: reviewed must be an ISO date: %v", current.rel, err)
		}
	}
	owners := map[string]bool{
		"project-maintainer": true,
		"documentation":      true,
		"security":           true,
		"interface":          true,
		"release":            true,
	}
	if owner := current.metadata["owner"]; owner != "" && !owners[owner] {
		t.Errorf("%s: unknown documentation owner %q", current.rel, owner)
	}
	for _, field := range []string{"generated", "security-critical"} {
		value := current.metadata[field]
		if value != "" && value != "true" && value != "false" {
			t.Errorf("%s: %s must be true or false", current.rel, field)
		}
	}
	if current.metadata["generated"] == "true" {
		if current.metadata["source"] == "" {
			t.Errorf("%s: generated page has no source metadata", current.rel)
		}
		if !strings.Contains(current.content, "Generated; do not edit by hand") {
			t.Errorf("%s: generated page has no visible generated warning", current.rel)
		}
	}
	for _, field := range []string{"prerequisites", "next"} {
		validateMetadataPaths(t, repositoryRoot, current, field)
	}
}

func validateMetadataPaths(t *testing.T, repositoryRoot string, current *page, field string) {
	t.Helper()
	value := current.metadata[field]
	if value == "" || value == "none" {
		return
	}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			t.Errorf("%s: empty path in metadata field %s", current.rel, field)
			continue
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(current.path), filepath.FromSlash(item)))
		if !within(repositoryRoot, resolved) {
			t.Errorf("%s: %s path escapes the repository: %s", current.rel, field, item)
			continue
		}
		if _, err := os.Stat(resolved); err != nil {
			t.Errorf("%s: %s path does not exist: %s", current.rel, field, item)
		}
	}
}

func validateFences(t *testing.T, current *page) {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(current.content))
	inFence := false
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		trimmed := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}
		if inFence {
			if trimmed == "```" {
				inFence = false
			}
			continue
		}
		language := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		if language == "" {
			t.Errorf("%s:%d: fenced code block has no language", current.rel, lineNumber)
		}
		inFence = true
	}
	if err := scanner.Err(); err != nil {
		t.Errorf("%s: scan code fences: %v", current.rel, err)
	}
	if inFence {
		t.Errorf("%s: unclosed fenced code block", current.rel)
	}
}

func markdownLinks(content string) []string {
	var links []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFence := false
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(trimmed, "```") {
			if inFence && trimmed == "```" {
				inFence = false
			} else if !inFence {
				inFence = true
			}
			continue
		}
		if inFence {
			continue
		}
		for _, match := range markdownLinkPattern.FindAllStringSubmatch(scanner.Text(), -1) {
			links = append(links, strings.Trim(match[1], "<>"))
		}
	}
	return links
}

func validateLinksAndNavigation(t *testing.T, repositoryRoot, docsRoot string, pages map[string]*page) {
	t.Helper()
	graph := make(map[string][]string, len(pages))
	for _, current := range pages {
		for _, target := range current.links {
			pathPart := strings.SplitN(target, "#", 2)[0]
			if pathPart == "" || isExternalLink(pathPart) {
				continue
			}
			decoded, err := url.PathUnescape(pathPart)
			if err != nil {
				t.Errorf("%s: invalid escaped link %q: %v", current.rel, target, err)
				continue
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(current.path), filepath.FromSlash(decoded)))
			if !within(repositoryRoot, resolved) {
				t.Errorf("%s: relative link escapes the repository: %s", current.rel, target)
				continue
			}
			info, err := os.Stat(resolved)
			if err != nil {
				t.Errorf("%s: relative link does not exist: %s", current.rel, target)
				continue
			}
			if info.IsDir() {
				if !within(docsRoot, resolved) {
					continue
				}
				resolved = filepath.Join(resolved, "index.md")
				if _, err := os.Stat(resolved); err != nil {
					t.Errorf("%s: linked documentation directory has no index: %s", current.rel, target)
					continue
				}
			}
			if filepath.Ext(resolved) != ".md" || !within(docsRoot, resolved) {
				continue
			}
			rel, err := filepath.Rel(docsRoot, resolved)
			if err != nil {
				t.Errorf("%s: resolve documentation link %s: %v", current.rel, target, err)
				continue
			}
			graph[current.rel] = append(graph[current.rel], filepath.ToSlash(rel))
		}
	}

	if _, ok := pages["index.md"]; !ok {
		t.Fatal("docs/index.md is missing")
	}
	seen := map[string]bool{"index.md": true}
	queue := []string{"index.md"}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, target := range graph[current] {
			if !seen[target] {
				seen[target] = true
				queue = append(queue, target)
			}
		}
	}
	var orphans []string
	for rel := range pages {
		if !seen[rel] {
			orphans = append(orphans, rel)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("documentation pages are orphaned from docs/index.md: %s", strings.Join(orphans, ", "))
	}
}

func isExternalLink(target string) bool {
	lower := strings.ToLower(target)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:")
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, fmt.Sprintf("..%c", filepath.Separator))
}
