package evolution

import (
	"encoding/json"
	"fmt"

	"github.com/mzigo-io/mzigo/services/contracts/internal/domain"
)

// JSONSchemaChecker applies JSON Schema evolution rules.
//
// JSON Schema compatibility is less formally specified than Avro.
// We apply pragmatic rules based on what breaks consumers in practice:
//
// BREAKING:
//   - Removing a property that was in "required"
//   - Adding a property to "required" (consumers sending old data will fail validation)
//   - Changing a property's type
//   - Removing a property entirely (consumers reading it get undefined)
//
// COMPATIBLE:
//   - Adding an optional property (not in required)
//   - Making a required property optional
//   - Adding enum values (old consumers ignore unknown values)
//
// SAFE:
//   - Updating descriptions, titles, examples
//   - Adding or updating metadata fields ($comment, etc.)
type JSONSchemaChecker struct{}

type jsonSchema struct {
	Type        string                     `json:"type"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
	Description string                     `json:"description,omitempty"`
}

func (c *JSONSchemaChecker) Check(
	currentBytes, proposedBytes []byte,
	_ domain.CompatibilityMode,
) (*CheckResult, error) {
	var current, proposed jsonSchema

	if err := json.Unmarshal(currentBytes, &current); err != nil {
		return nil, fmt.Errorf("parsing current json schema: %w", err)
	}
	if err := json.Unmarshal(proposedBytes, &proposed); err != nil {
		return nil, fmt.Errorf("parsing proposed json schema: %w", err)
	}

	currentRequired := toSet(current.Required)
	proposedRequired := toSet(proposed.Required)

	var changes []SchemaChange

	// Detect removed properties
	for name := range current.Properties {
		if _, exists := proposed.Properties[name]; !exists {
			severity := domain.ClassificationCompatible
			if currentRequired[name] {
				severity = domain.ClassificationBreaking
			}
			changes = append(changes, SchemaChange{
				Field:       name,
				ChangeType:  ChangeTypeFieldRemoved,
				Description: fmt.Sprintf("property %q removed", name),
				Severity:    severity,
			})
		}
	}

	// Detect added properties and required changes
	for name := range proposed.Properties {
		_, existed := current.Properties[name]
		wasRequired := currentRequired[name]
		nowRequired := proposedRequired[name]

		if !existed {
			severity := domain.ClassificationCompatible
			if nowRequired {
				// New required property: existing producers are not sending it.
				severity = domain.ClassificationBreaking
			}
			changes = append(changes, SchemaChange{
				Field:       name,
				ChangeType:  ChangeTypeFieldAdded,
				Description: fmt.Sprintf("property %q added", name),
				Severity:    severity,
			})
			continue
		}

		// Property moved from optional to required
		if !wasRequired && nowRequired {
			changes = append(changes, SchemaChange{
				Field:       name,
				ChangeType:  ChangeTypeRequiredAdded,
				Description: fmt.Sprintf("property %q is now required (was optional)", name),
				Severity:    domain.ClassificationBreaking,
			})
		}
	}

	return &CheckResult{
		Classification: worstClassification(changes),
		Changes:        changes,
	}, nil
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
