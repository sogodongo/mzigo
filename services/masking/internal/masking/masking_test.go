package masking_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mzigo-io/mzigo/services/masking/internal/masking"
)

// Tokenizer tests

func TestTokenizer_DeterministicOutput(t *testing.T) {
	tok := masking.NewTokenizer("test-secret-key", "tok_")
	a := tok.Tokenize("account_id", "user-123")
	b := tok.Tokenize("account_id", "user-123")
	if a != b {
		t.Errorf("tokenizer must be deterministic: got %s and %s", a, b)
	}
}

func TestTokenizer_DifferentValuesProduceDifferentTokens(t *testing.T) {
	tok := masking.NewTokenizer("test-secret-key", "tok_")
	a := tok.Tokenize("account_id", "user-123")
	b := tok.Tokenize("account_id", "user-456")
	if a == b {
		t.Error("different values must produce different tokens")
	}
}

func TestTokenizer_NamespaceScopesToken(t *testing.T) {
	tok := masking.NewTokenizer("test-secret-key", "tok_")
	a := tok.Tokenize("account_id", "user-123")
	b := tok.Tokenize("email", "user-123")
	if a == b {
		t.Error("same value in different namespaces must produce different tokens")
	}
}

func TestTokenizer_PrefixIsApplied(t *testing.T) {
	tok := masking.NewTokenizer("test-secret-key", "tok_")
	result := tok.Tokenize("account_id", "user-123")
	if !strings.HasPrefix(result, "tok_") {
		t.Errorf("token should start with prefix, got: %s", result)
	}
}

// Operations / Applier tests

func newApplier() *masking.Applier {
	tok := masking.NewTokenizer("test-key", "tok_")
	return masking.NewApplier(tok, "*", 4)
}

func TestApplier_Redact(t *testing.T) {
	a := newApplier()
	result, err := a.Apply(masking.FieldTransform{
		FieldPath: "card_number",
		Operation: masking.OperationRedact,
	}, "4242424242424242")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "[REDACTED]" {
		t.Errorf("expected [REDACTED], got %v", result)
	}
}

func TestApplier_Tokenize(t *testing.T) {
	a := newApplier()
	result, err := a.Apply(masking.FieldTransform{
		FieldPath: "account_id",
		Operation: masking.OperationTokenize,
		PIIClass:  "FINANCIAL",
	}, "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "user-123" {
		t.Error("tokenize should not return the original value")
	}
}

func TestApplier_Mask_PreservesLastFour(t *testing.T) {
	a := newApplier()
	result, err := a.Apply(masking.FieldTransform{
		FieldPath: "card_last_four",
		Operation: masking.OperationMask,
	}, "4242424242424242")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}
	if !strings.HasSuffix(str, "4242") {
		t.Errorf("masked value should end with last 4 digits, got: %s", str)
	}
	if strings.HasPrefix(str, "4") {
		t.Errorf("masked value should not start with original digits, got: %s", str)
	}
}

func TestApplier_NullValue_IsNoop(t *testing.T) {
	a := newApplier()
	result, err := a.Apply(masking.FieldTransform{
		FieldPath: "email",
		Operation: masking.OperationRedact,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("masking a null field should return null, got: %v", result)
	}
}

// Engine tests

func TestEngine_AppliesPolicyToField(t *testing.T) {
	engine := masking.NewEngine(newApplier(), masking.NewDetector())

	payload := mustJSON(t, map[string]any{
		"transaction_id": "txn-abc",
		"account_id":     "user-999",
		"amount":         100,
	})

	result, err := engine.Mask(masking.MaskRequest{
		Payload: payload,
		Policies: []masking.FieldPolicy{
			{Path: "account_id", Operation: masking.OperationTokenize, PIIClass: "FINANCIAL"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(result.Payload, &out); err != nil {
		t.Fatalf("failed to parse output payload: %v", err)
	}

	if out["account_id"] == "user-999" {
		t.Error("account_id should have been tokenized, not passed through")
	}
	if out["transaction_id"] != "txn-abc" {
		t.Error("transaction_id should be unchanged")
	}
	if len(result.FieldsTransformed) != 1 || result.FieldsTransformed[0] != "account_id" {
		t.Errorf("expected 1 transformed field, got: %v", result.FieldsTransformed)
	}
}

func TestEngine_AbsentOptionalField_IsSkipped(t *testing.T) {
	engine := masking.NewEngine(newApplier(), masking.NewDetector())

	payload := mustJSON(t, map[string]any{"transaction_id": "txn-abc"})

	result, err := engine.Mask(masking.MaskRequest{
		Payload: payload,
		Policies: []masking.FieldPolicy{
			{Path: "account_id", Operation: masking.OperationRedact},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FieldsTransformed) != 0 {
		t.Errorf("absent field should not appear in transformed list")
	}
}

func TestEngine_NoPolicies_ReturnsPayloadUnchanged(t *testing.T) {
	engine := masking.NewEngine(newApplier(), masking.NewDetector())
	payload := mustJSON(t, map[string]any{"id": "x"})

	result, err := engine.Mask(masking.MaskRequest{Payload: payload, Policies: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Payload) != string(payload) {
		t.Error("payload should be unchanged when no policies present")
	}
}

func TestEngine_PatternDetection_FlagsUnpoliciedPIIField(t *testing.T) {
	engine := masking.NewEngine(newApplier(), masking.NewDetector())

	payload := mustJSON(t, map[string]any{
		"transaction_id": "txn-abc",
		"email":          "user@example.com", // PII by pattern, no contract policy
	})

	result, err := engine.Mask(masking.MaskRequest{
		Payload:  payload,
		Policies: []masking.FieldPolicy{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.PatternDetections) == 0 {
		t.Error("expected pattern detection for 'email' field")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal test fixture: %v", err)
	}
	return b
}
