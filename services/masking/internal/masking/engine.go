package masking

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FieldPolicy is the masking policy for a single field, as declared in the contract.
type FieldPolicy struct {
	Path      string
	Operation Operation
	PIIClass  string
}

// MaskRequest is the input to the engine.
type MaskRequest struct {
	Payload  []byte
	Policies []FieldPolicy
}

// MaskResult is the output of the engine.
type MaskResult struct {
	Payload           []byte
	FieldsTransformed []string
	// PatternDetections are fields that looked like PII based on name patterns
	// but were not covered by a contract policy. These are warnings, not errors.
	// The message is allowed through, but operators should formalize these fields
	// in the contract.
	PatternDetections []DetectedField
}

// Engine orchestrates PII detection and masking transformation.
//
// Processing order:
// 1. Parse the payload into a mutable document
// 2. Apply contract-declared field policies (authoritative)
// 3. Scan remaining fields for PII patterns (advisory)
// 4. Re-serialize the document
//
// The engine never modifies the original payload bytes. It works on a
// parsed copy and returns a new serialized document.
type Engine struct {
	applier  *Applier
	detector *Detector
}

func NewEngine(applier *Applier, detector *Detector) *Engine {
	return &Engine{
		applier:  applier,
		detector: detector,
	}
}

func (e *Engine) Mask(req MaskRequest) (*MaskResult, error) {
	if len(req.Policies) == 0 {
		return &MaskResult{Payload: req.Payload}, nil
	}

	var doc map[string]any
	if err := json.Unmarshal(req.Payload, &doc); err != nil {
		return nil, fmt.Errorf("parsing payload for masking: %w", err)
	}

	contractPIIPaths := make(map[string]bool, len(req.Policies))
	for _, p := range req.Policies {
		contractPIIPaths[p.Path] = true
	}

	var transformed []string

	for _, policy := range req.Policies {
		current, exists := getNestedField(doc, policy.Path)
		if !exists {
			// Field absent from payload. Not an error: optional fields may be absent.
			continue
		}

		transform := FieldTransform{
			FieldPath: policy.Path,
			Operation: policy.Operation,
			PIIClass:  policy.PIIClass,
		}

		masked, err := e.applier.Apply(transform, current)
		if err != nil {
			// We cannot apply the masking operation. Block the message.
			// A partial mask (some fields transformed, others not) is worse
			// than blocking: it gives a false sense of safety.
			return nil, fmt.Errorf("masking field %q: %w", policy.Path, err)
		}

		if err := setNestedField(doc, policy.Path, masked); err != nil {
			return nil, fmt.Errorf("setting masked value for %q: %w", policy.Path, err)
		}

		transformed = append(transformed, policy.Path)
	}

	// Advisory pattern scan on remaining fields.
	var allPaths []string
	collectPaths(doc, "", &allPaths)
	patternDetections := e.detector.ScanFieldPaths(allPaths, contractPIIPaths)

	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("re-serializing masked payload: %w", err)
	}

	return &MaskResult{
		Payload:           out,
		FieldsTransformed: transformed,
		PatternDetections: patternDetections,
	}, nil
}

// getNestedField resolves a dot-separated path against a document.
func getNestedField(doc map[string]any, path string) (any, bool) {
	parts := strings.SplitN(path, ".", 2)
	val, ok := doc[parts[0]]
	if !ok {
		return nil, false
	}
	if len(parts) == 1 {
		return val, true
	}
	nested, ok := val.(map[string]any)
	if !ok {
		return nil, false
	}
	return getNestedField(nested, parts[1])
}

// setNestedField writes a value at a dot-separated path within a document.
func setNestedField(doc map[string]any, path string, value any) error {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 1 {
		doc[parts[0]] = value
		return nil
	}
	nested, ok := doc[parts[0]].(map[string]any)
	if !ok {
		return fmt.Errorf("field %q is not an object, cannot traverse to %q", parts[0], path)
	}
	return setNestedField(nested, parts[1], value)
}

// collectPaths enumerates all field paths in a document.
func collectPaths(doc map[string]any, prefix string, paths *[]string) {
	for k, v := range doc {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		*paths = append(*paths, path)
		if nested, ok := v.(map[string]any); ok {
			collectPaths(nested, path, paths)
		}
	}
}
