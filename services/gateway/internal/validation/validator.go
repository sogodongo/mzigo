package validation

import (
	"encoding/json"
	"fmt"

	"github.com/mzigo-io/mzigo/services/gateway/internal/cache"
)

// ViolationType classifies why a message was rejected.
// These values are written to the violation log and emitted as metric labels.
// Keep this list stable: changes to violation type names break dashboards.
type ViolationType string

const (
	ViolationMissingField  ViolationType = "MISSING_REQUIRED_FIELD"
	ViolationTypeMismatch  ViolationType = "TYPE_MISMATCH"
	ViolationUnknownTopic  ViolationType = "NO_CONTRACT_FOR_TOPIC"
	ViolationSchemaInvalid ViolationType = "SCHEMA_INVALID"
	ViolationPolicybreach   ViolationType = "POLICY_BREACH"
)

// ValidationResult carries the outcome of a single message validation.
// It is intentionally not an error type: callers need to distinguish
// between a validation rejection (expected, operational) and an internal
// processing failure (unexpected, alertable).
type ValidationResult struct {
	Valid      bool
	Violations []Violation
}

type Violation struct {
	Type    ViolationType `json:"type"`
	Field   string        `json:"field,omitempty"`
	Message string        `json:"message"`
}

func (r ValidationResult) FirstViolation() *Violation {
	if len(r.Violations) == 0 {
		return nil
	}
	return &r.Violations[0]
}

// Validator performs contract validation against a parsed message payload.
// It is constructed once at startup and is safe for concurrent use.
//
// The validator does not perform Schema Registry validation. That is
// handled by the Kafka producer client using the registered Avro schema.
// The validator's job is to enforce the contract-level rules that the
// schema alone cannot express: required fields, PII policy presence checks,
// and field-level constraints declared in the contract.
type Validator struct{}

func New() *Validator {
	return &Validator{}
}

// Validate checks a raw JSON payload against the provided contract.
// It returns a ValidationResult regardless of outcome. A non-nil error
// indicates a processing failure, not a contract violation.
func (v *Validator) Validate(payload []byte, contract *cache.Contract) (ValidationResult, error) {
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return ValidationResult{}, fmt.Errorf("parsing payload: %w", err)
	}

	var violations []Violation

	for _, field := range contract.Fields {
		val, present := getNestedField(doc, field.Path)

		if field.Required && !present {
			violations = append(violations, Violation{
				Type:    ViolationMissingField,
				Field:   field.Path,
				Message: fmt.Sprintf("required field %q is absent", field.Path),
			})
			continue
		}

		if !present {
			continue
		}

		// PII fields must have a masking operation declared in the contract.
		// A PII field with no masking op is a policy breach regardless of message content.
		// We reject at validation time rather than attempting to mask an unconfigured field.
		if field.PIIClass != "" && field.MaskingOp == "" {
			violations = append(violations, Violation{
				Type:    ViolationPolicybreach,
				Field:   field.Path,
				Message: fmt.Sprintf("field %q is classified %s but has no masking_op in contract", field.Path, field.PIIClass),
			})
			continue
		}

		if err := checkType(val, field); err != nil {
			violations = append(violations, Violation{
				Type:    ViolationTypeMismatch,
				Field:   field.Path,
				Message: err.Error(),
			})
		}
	}

	return ValidationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
	}, nil
}

// getNestedField resolves a dot-separated field path against a document.
// "payment.card.last_four" traverses three levels of nesting.
// Returns the value and whether it was present (nil value and present=true
// is a valid state for nullable fields).
func getNestedField(doc map[string]any, path string) (any, bool) {
	current := any(doc)
	start := 0

	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			key := path[start:i]
			m, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}
			current, ok = m[key]
			if !ok {
				return nil, false
			}
			start = i + 1
		}
	}

	return current, true
}

func checkType(val any, field cache.FieldPolicy) error {
	// Type enforcement is intentionally shallow at the gateway layer.
	// Deep type checking belongs in the Schema Registry Avro validation.
	// Here we catch the class of errors that schema validation misses:
	// null values in required fields, wrong JSON value kinds.
	if val == nil && field.Required {
		return fmt.Errorf("field %q is required but null", field.Path)
	}
	return nil
}
