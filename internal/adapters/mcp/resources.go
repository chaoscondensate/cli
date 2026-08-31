package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/chaoscondensate/forecast-ledger/internal/app"
	"github.com/chaoscondensate/forecast-ledger/internal/ledger"
	"github.com/chaoscondensate/forecast-ledger/internal/presentation"
	"github.com/chaoscondensate/forecast-ledger/internal/publication"
	"github.com/chaoscondensate/forecast-ledger/internal/service"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const resourceTemplate = "forecast-ledger://v1/{kind}/{root}/{+path}{?question,forecast,ledger}"

func (s *Server) registerResources() {
	s.sdk.AddResourceTemplate(&sdk.ResourceTemplate{
		Name: "forecast-ledger-addressed-resource", Title: "Forecast Ledger addressed resource",
		Description: "Redacted ledger, question, forecast, artifact summary, verification report, or package manifest. Paths are confined to configured named roots.",
		MIMEType:    "application/json", URITemplate: resourceTemplate,
	}, s.resourceHandler)
}

func (s *Server) resourceHandler(ctx context.Context, request *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
	parsed, err := url.Parse(request.Params.URI)
	if err != nil || parsed.Scheme != "forecast-ledger" || parsed.Host != "v1" {
		return nil, sdk.ResourceNotFoundError(request.Params.URI)
	}
	parts := strings.Split(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
	if len(parts) < 3 {
		return nil, sdk.ResourceNotFoundError(request.Params.URI)
	}
	kind, rootName := parts[0], parts[1]
	relative, err := url.PathUnescape(strings.Join(parts[2:], "/"))
	if err != nil || relative == "" {
		return nil, sdk.ResourceNotFoundError(request.Params.URI)
	}
	query := parsed.Query()
	var data any
	switch kind {
	case "ledger", "question", "forecast", "target", "timestamp", "report":
		file, resolveErr := s.roots.Resolve(service.RootLedger, rootName+":"+relative, true)
		if resolveErr != nil {
			return nil, resourceError(request.Params.URI, resolveErr)
		}
		data, err = s.readLedgerResource(ctx, kind, file, query)
	case "manifest":
		path, resolveErr := s.roots.Resolve(service.RootOutput, rootName+":"+relative, true)
		if resolveErr != nil {
			return nil, resourceError(request.Params.URI, resolveErr)
		}
		data, err = readManifestResource(path)
	default:
		return nil, sdk.ResourceNotFoundError(request.Params.URI)
	}
	if err != nil {
		return nil, resourceError(request.Params.URI, err)
	}
	encoded, err := marshalResourceData(data)
	if err != nil {
		return nil, err
	}
	return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{URI: request.Params.URI, MIMEType: "application/json", Text: encoded}}}, nil
}

func marshalResourceData(data any) (string, error) {
	safe, err := presentation.Redact(data)
	if err != nil {
		return "", app.NewError(app.CodeInternal, "resource result cannot be sanitized", err)
	}
	encoded, err := json.Marshal(safe)
	if err != nil {
		return "", app.NewError(app.CodeInternal, "resource result cannot be encoded", err)
	}
	return string(encoded), nil
}

func (s *Server) readLedgerResource(ctx context.Context, kind, file string, query url.Values) (any, error) {
	questionID, forecastID := ledger.Slug(query.Get("question")), ledger.Slug(query.Get("forecast"))
	switch kind {
	case "ledger":
		loaded, err := service.LoadAndValidateLedger(ctx, file, nil)
		if err != nil {
			return nil, err
		}
		return service.StatusForLedger(loaded)
	case "question":
		_, result, err := service.LoadQuestionShow(ctx, file, nil, questionID)
		return result, err
	case "forecast":
		_, result, err := service.LoadForecastShow(ctx, file, nil, questionID, forecastID)
		return result, err
	case "target":
		return service.CheckTargets(ctx, file, false, questionID, forecastID)
	case "timestamp":
		return service.TimestampStatusFor(ctx, file, questionID, forecastID)
	case "report":
		return service.VerifyLedgerEvidence(ctx, file, service.VerificationOptions{Offline: true, QuestionID: questionID, ForecastID: forecastID})
	default:
		return nil, app.NewError(app.CodeNotFound, "resource kind is not available", nil)
	}
}

func readManifestResource(path string) (publication.Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return publication.Manifest{}, app.NewError(app.CodeIO, "manifest cannot be opened", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, publication.MaxManifestBytes+1))
	if err != nil {
		return publication.Manifest{}, app.NewError(app.CodeIO, "manifest cannot be read", err)
	}
	if len(data) > publication.MaxManifestBytes {
		return publication.Manifest{}, app.NewError(app.CodeInvalidData, "manifest exceeds the size limit", nil)
	}
	manifest, err := publication.Decode(data)
	if err != nil {
		return publication.Manifest{}, app.NewError(app.CodeVerification, "manifest is invalid", err)
	}
	return manifest, nil
}

func resourceError(uri string, err error) error {
	if app.ErrorCodeOf(err) == app.CodeNotFound {
		return sdk.ResourceNotFoundError(uri)
	}
	return err
}
