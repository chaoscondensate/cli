package service

import (
	"fmt"
	"io/fs"

	"github.com/chaoscondensate/cli/internal/document"
	"github.com/chaoscondensate/cli/internal/validation"
)

// ValidateLedgerDocument performs the complete prospective validation used by
// every ledger transaction: exact version, embedded schema, domain decoding,
// and semantic/artifact checks.
func ValidateLedgerDocument(parsed *document.Document, artifacts fs.FS) error {
	if err := RequireSupportedSchemaVersion(parsed); err != nil {
		return err
	}
	structural, err := validation.DefaultStructuralValidator()
	if err != nil {
		return fmt.Errorf("load embedded ledger schema: %w", err)
	}
	issues, err := structural.Validate(parsed.Root.Any())
	if err != nil {
		return fmt.Errorf("run ledger schema validation: %w", err)
	}
	if len(issues) > 0 {
		return fmt.Errorf("ledger schema validation failed: %s at %s", issues[0].Code, issues[0].InstanceLocation)
	}
	model, err := validation.DecodeLedger(parsed)
	if err != nil {
		return fmt.Errorf("decode schema-valid ledger: %w", err)
	}
	semantic, err := validation.ValidateSemantics(model, artifacts)
	if err != nil {
		return fmt.Errorf("run ledger semantic validation: %w", err)
	}
	if len(semantic) > 0 {
		return fmt.Errorf("ledger semantic validation failed: %s at %s", semantic[0].Code, semantic[0].Pointer)
	}
	return nil
}
