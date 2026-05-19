package masking

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Operation identifies which masking transformation to apply to a field.
type Operation string

const (
	OperationRedact   Operation = "REDACT"
	OperationTokenize Operation = "TOKENIZE"
	OperationMask     Operation = "MASK"
)

const redactedSentinel = "[REDACTED]"

// FieldTransform describes a single field transformation to apply.
type FieldTransform struct {
	FieldPath string
	Operation Operation
	// PIIClass is passed to the tokenizer as the namespace so tokens are
	// scoped to a field type, not just to a raw value.
	PIIClass string
}

// Applier applies masking transformations to field values.
// It is constructed once at startup with the tokenizer and config.
type Applier struct {
	tokenizer      *Tokenizer
	maskChar       string
	maskKeepSuffix int
}

func NewApplier(tokenizer *Tokenizer, maskChar string, maskKeepSuffix int) *Applier {
	return &Applier{
		tokenizer:      tokenizer,
		maskChar:       maskChar,
		maskKeepSuffix: maskKeepSuffix,
	}
}

// Apply transforms a field value according to the specified operation.
// Returns the transformed value and any error.
//
// An error here means the operation could not be applied to this value.
// The caller (engine) treats this as a hard failure: the message is blocked.
// We do not silently pass through an unmasked PII value.
func (a *Applier) Apply(transform FieldTransform, value any) (any, error) {
	if value == nil {
		// Null fields are already safe. Masking a null is a no-op.
		return nil, nil
	}

	switch transform.Operation {
	case OperationRedact:
		return a.redact(value)
	case OperationTokenize:
		return a.tokenize(transform, value)
	case OperationMask:
		return a.mask(value)
	default:
		return nil, fmt.Errorf("unknown masking operation: %s", transform.Operation)
	}
}

func (a *Applier) redact(_ any) (any, error) {
	return redactedSentinel, nil
}

func (a *Applier) tokenize(transform FieldTransform, value any) (any, error) {
	str, err := toString(value)
	if err != nil {
		return nil, fmt.Errorf("tokenize: field %q value is not a string: %w", transform.FieldPath, err)
	}
	namespace := transform.PIIClass
	if namespace == "" {
		namespace = transform.FieldPath
	}
	return a.tokenizer.Tokenize(namespace, str), nil
}

func (a *Applier) mask(value any) (any, error) {
	str, err := toString(value)
	if err != nil {
		return nil, fmt.Errorf("mask: value is not a string: %w", err)
	}

	runeCount := utf8.RuneCountInString(str)

	if runeCount <= a.maskKeepSuffix {
		// Value is shorter than or equal to the kept suffix.
		// Masking the whole thing is more appropriate than masking nothing.
		return strings.Repeat(a.maskChar, runeCount), nil
	}

	masked := strings.Repeat(a.maskChar, runeCount-a.maskKeepSuffix)
	suffix := string([]rune(str)[runeCount-a.maskKeepSuffix:])
	return masked + suffix, nil
}

func toString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return "", fmt.Errorf("expected string, got %T", v)
	}
}
