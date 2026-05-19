package evolution

import (
	"encoding/json"
	"fmt"

	"github.com/mzigo-io/mzigo/services/contracts/internal/domain"
)

// AvroChecker applies Avro schema evolution rules.
//
// Avro's compatibility model is well-specified in the Avro spec.
// The rules we implement here are the subset that matters most in
// production streaming environments:
//
// BACKWARD compatible (new schema reads old data):
//   - Adding a field with a default: COMPATIBLE
//   - Removing a field that had a default: COMPATIBLE
//   - Removing a field with no default: BREAKING
//   - Adding a field with no default: BREAKING (old readers don't know the field)
//
// BREAKING in all compatibility modes:
//   - Changing a field's type to an incompatible type
//   - Removing a field entirely (no default on either side)
//   - Renaming a field without adding an alias
type AvroChecker struct{}

type avroSchema struct {
	Type      string       `json:"type"`
	Name      string       `json:"name"`
	Namespace string       `json:"namespace,omitempty"`
	Fields    []avroField  `json:"fields"`
}

type avroField struct {
	Name    string          `json:"name"`
	Type    json.RawMessage `json:"type"`
	Default json.RawMessage `json:"default,omitempty"`
	Doc     string          `json:"doc,omitempty"`
	Aliases []string        `json:"aliases,omitempty"`
}

func (c *AvroChecker) Check(
	currentBytes, proposedBytes []byte,
	compatibility domain.CompatibilityMode,
) (*CheckResult, error) {
	var current, proposed avroSchema

	if err := json.Unmarshal(currentBytes, &current); err != nil {
		return nil, fmt.Errorf("parsing current avro schema: %w", err)
	}
	if err := json.Unmarshal(proposedBytes, &proposed); err != nil {
		return nil, fmt.Errorf("parsing proposed avro schema: %w", err)
	}

	currentFields := indexFields(current.Fields)
	proposedFields := indexFields(proposed.Fields)

	var changes []SchemaChange

	// Detect removed fields
	for name, cf := range currentFields {
		if _, exists := proposedFields[name]; !exists {
			severity := domain.ClassificationCompatible
			// A field removal is BREAKING if the field had no default,
			// because old data written without the field cannot be read
			// by a reader that now requires it. In BACKWARD mode this is
			// the primary concern.
			if cf.Default == nil {
				severity = domain.ClassificationBreaking
			}
			changes = append(changes, SchemaChange{
				Field:       name,
				ChangeType:  ChangeTypeFieldRemoved,
				Description: fmt.Sprintf("field %q removed from schema", name),
				Severity:    severity,
			})
		}
	}

	// Detect added fields and type changes
	for name, pf := range proposedFields {
		cf, existed := currentFields[name]
		if !existed {
			severity := domain.ClassificationCompatible
			// Adding a field without a default is BREAKING in BACKWARD mode:
			// old data does not have this field, so new readers cannot
			// populate it when consuming historical messages.
			if pf.Default == nil && compatibility == domain.CompatibilityBackward {
				severity = domain.ClassificationBreaking
			}
			changes = append(changes, SchemaChange{
				Field:       name,
				ChangeType:  ChangeTypeFieldAdded,
				Description: fmt.Sprintf("field %q added to schema", name),
				Severity:    severity,
			})
			continue
		}

		// Type change detection. We do a simple string comparison here.
		// Full Avro type promotion rules (int -> long -> float -> double) are
		// complex; we flag any type change as BREAKING and let the operator
		// verify. False positives on legal promotions are acceptable;
		// false negatives on illegal type changes are not.
		if string(cf.Type) != string(pf.Type) {
			changes = append(changes, SchemaChange{
				Field:      name,
				ChangeType: ChangeTypeTypeChanged,
				Description: fmt.Sprintf(
					"field %q type changed from %s to %s",
					name, string(cf.Type), string(pf.Type),
				),
				Severity: domain.ClassificationBreaking,
			})
		}

		// Default removal: a field that had a default now lacks one.
		// This is COMPATIBLE but worth flagging for operator awareness.
		if cf.Default != nil && pf.Default == nil {
			changes = append(changes, SchemaChange{
				Field:       name,
				ChangeType:  ChangeTypeDefaultRemoved,
				Description: fmt.Sprintf("field %q default value removed", name),
				Severity:    domain.ClassificationCompatible,
			})
		}
	}

	return &CheckResult{
		Classification: worstClassification(changes),
		Changes:        changes,
	}, nil
}

func indexFields(fields []avroField) map[string]avroField {
	idx := make(map[string]avroField, len(fields))
	for _, f := range fields {
		idx[f.Name] = f
	}
	return idx
}
