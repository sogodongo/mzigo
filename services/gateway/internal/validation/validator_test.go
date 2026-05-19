package validation_test

import (
	"encoding/json"
	"testing"

	"github.com/mzigo-io/mzigo/services/gateway/internal/cache"
	"github.com/mzigo-io/mzigo/services/gateway/internal/validation"
)

func TestValidator_RequiredFieldPresent(t *testing.T) {
	v := validation.New()
	contract := contractWithFields(cache.FieldPolicy{
		Path:     "transaction_id",
		Required: true,
	})

	payload := mustJSON(map[string]any{"transaction_id": "abc-123"})
	result, err := v.Validate(payload, contract)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got violations: %+v", result.Violations)
	}
}

func TestValidator_RequiredFieldAbsent(t *testing.T) {
	v := validation.New()
	contract := contractWithFields(cache.FieldPolicy{
		Path:     "transaction_id",
		Required: true,
	})

	payload := mustJSON(map[string]any{"amount": 100})
	result, err := v.Validate(payload, contract)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for missing required field")
	}
	if result.FirstViolation().Type != validation.ViolationMissingField {
		t.Errorf("expected ViolationMissingField, got %s", result.FirstViolation().Type)
	}
}

func TestValidator_PIIFieldWithoutMaskingOp(t *testing.T) {
	v := validation.New()
	contract := contractWithFields(cache.FieldPolicy{
		Path:     "account_id",
		Required: true,
		PIIClass: "PERSONAL",
		// MaskingOp intentionally absent: this contract is misconfigured
	})

	payload := mustJSON(map[string]any{"account_id": "user-999"})
	result, err := v.Validate(payload, contract)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid: PII field without masking_op must be rejected")
	}
	if result.FirstViolation().Type != validation.ViolationPolicybreach {
		t.Errorf("expected ViolationPolicybreach, got %s", result.FirstViolation().Type)
	}
}

func TestValidator_NestedFieldPath(t *testing.T) {
	v := validation.New()
	contract := contractWithFields(cache.FieldPolicy{
		Path:     "payment.card.last_four",
		Required: true,
	})

	payload := mustJSON(map[string]any{
		"payment": map[string]any{
			"card": map[string]any{
				"last_four": "4242",
			},
		},
	})

	result, err := v.Validate(payload, contract)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid for present nested field, got: %+v", result.Violations)
	}
}

func TestValidator_InvalidJSON(t *testing.T) {
	v := validation.New()
	contract := contractWithFields()

	_, err := v.Validate([]byte(`{not valid json`), contract)
	if err == nil {
		t.Fatal("expected error for invalid JSON payload")
	}
}

// contractWithFields builds a minimal contract for testing without
// requiring the full contract service or database.
func contractWithFields(fields ...cache.FieldPolicy) *cache.Contract {
	return &cache.Contract{
		ID:      "test-contract-id",
		Name:    "test.contract",
		Topic:   "test.topic",
		Version: "1.0.0",
		Schema:  json.RawMessage(`{}`),
		Fields:  fields,
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
