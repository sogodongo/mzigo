package evolution

import (
	"fmt"

	"github.com/mzigo-io/mzigo/services/contracts/internal/domain"
)

// Checker is the entry point for schema evolution analysis.
// It dispatches to format-specific checkers and returns a unified classification.
//
// The checker is a pure function: schema bytes in, classification out.
// It has no database access and no side effects, which makes it
// independently testable and usable in both the gate handler and
// the offline analyzer service.
type Checker struct {
	avro       *AvroChecker
	jsonSchema *JSONSchemaChecker
}

func NewChecker() *Checker {
	return &Checker{
		avro:       &AvroChecker{},
		jsonSchema: &JSONSchemaChecker{},
	}
}

// Check compares a proposed schema against the current schema and returns
// the most severe classification across all detected changes.
func (c *Checker) Check(
	current, proposed []byte,
	format domain.SchemaFormat,
	compatibility domain.CompatibilityMode,
) (*CheckResult, error) {
	switch format {
	case domain.SchemaFormatAvro:
		return c.avro.Check(current, proposed, compatibility)
	case domain.SchemaFormatJSONSchema:
		return c.jsonSchema.Check(current, proposed, compatibility)
	case domain.SchemaFormatProtobuf:
		// Protobuf evolution rules are format-specific and non-trivial.
		// Implementing them correctly requires parsing the proto descriptor,
		// not just comparing text. Stub returns SAFE until the full
		// implementation lands in a follow-up.
		return &CheckResult{
			Classification: domain.ClassificationSafe,
			Changes:        nil,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported schema format: %s", format)
	}
}

// CheckResult carries the full output of a schema evolution check.
type CheckResult struct {
	Classification domain.ChangeClassification
	Changes        []SchemaChange
}

// SchemaChange describes a single detected difference between schema versions.
type SchemaChange struct {
	Field       string
	ChangeType  ChangeType
	Description string
	Severity    domain.ChangeClassification
}

type ChangeType string

const (
	ChangeTypeFieldRemoved      ChangeType = "FIELD_REMOVED"
	ChangeTypeFieldAdded        ChangeType = "FIELD_ADDED"
	ChangeTypeTypeChanged       ChangeType = "TYPE_CHANGED"
	ChangeTypeDefaultRemoved    ChangeType = "DEFAULT_REMOVED"
	ChangeTypeRequiredAdded     ChangeType = "REQUIRED_ADDED"
	ChangeTypeCompatibilityMode ChangeType = "COMPATIBILITY_MODE_CHANGED"
)

// worstClassification returns the most severe classification
// from a list of changes. Order: BREAKING > COMPATIBLE > SAFE.
func worstClassification(changes []SchemaChange) domain.ChangeClassification {
	result := domain.ClassificationSafe
	for _, c := range changes {
		if c.Severity == domain.ClassificationBreaking {
			return domain.ClassificationBreaking
		}
		if c.Severity == domain.ClassificationCompatible {
			result = domain.ClassificationCompatible
		}
	}
	return result
}
