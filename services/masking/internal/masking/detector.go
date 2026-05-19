package masking

import (
	"regexp"
	"strings"
)

// PIIClass identifies the category of PII in a field.
// The class influences which masking operation is appropriate.
type PIIClass string

const (
	PIIClassPersonal   PIIClass = "PERSONAL"   // name, email, phone
	PIIClassFinancial  PIIClass = "FINANCIAL"   // card numbers, account IDs
	PIIClassHealth     PIIClass = "HEALTH"      // medical identifiers
	PIIClassIdentifier PIIClass = "IDENTIFIER"  // SSN, national ID, passport
)

// DetectedField is a field identified as potentially containing PII.
type DetectedField struct {
	Path     string
	PIIClass PIIClass
	// FromPattern indicates this detection came from heuristic pattern matching
	// rather than an explicit contract policy declaration. Pattern-based detections
	// are flagged in metrics so teams know to formalize them in their contracts.
	FromPattern bool
}

// Detector finds PII fields in a message payload.
//
// Two detection modes:
//
// 1. Contract-declared: the contract explicitly marks a field as PII with a
//    masking operation. This is the authoritative mode. The detector respects
//    whatever the contract says.
//
// 2. Pattern-based: heuristic detection on field names that look like PII
//    even if the contract did not declare them. This is a safety net for
//    contracts that were written before PII classification was enforced.
//    Pattern detections produce a metric and a log warning so operators know
//    a contract needs updating. They do NOT automatically apply masking unless
//    the gateway is configured with strict mode.
type Detector struct {
	patterns []piiPattern
}

type piiPattern struct {
	re      *regexp.Regexp
	class   PIIClass
	// keywords matched as substrings in field path (faster than regex for simple cases)
	keywords []string
}

func NewDetector() *Detector {
	return &Detector{
		patterns: []piiPattern{
			{
				keywords: []string{"email", "e_mail"},
				class:    PIIClassPersonal,
			},
			{
				keywords: []string{"phone", "mobile", "cell_number"},
				class:    PIIClassPersonal,
			},
			{
				keywords: []string{"first_name", "last_name", "full_name", "display_name"},
				class:    PIIClassPersonal,
			},
			{
				keywords: []string{"card_number", "card_num", "pan", "credit_card"},
				class:    PIIClassFinancial,
			},
			{
				keywords: []string{"account_id", "account_number", "iban", "routing_number"},
				class:    PIIClassFinancial,
			},
			{
				keywords: []string{"ssn", "social_security", "national_id", "passport"},
				class:    PIIClassIdentifier,
			},
			{
				keywords: []string{"diagnosis", "condition", "medication", "patient_id"},
				class:    PIIClassHealth,
			},
			{
				// IP addresses and device IDs are personal data under GDPR.
				keywords: []string{"ip_address", "device_id", "user_agent"},
				class:    PIIClassPersonal,
			},
		},
	}
}

// DetectFromPath heuristically classifies a field path as PII.
// Returns nil if no PII pattern matches.
func (d *Detector) DetectFromPath(fieldPath string) *DetectedField {
	lower := strings.ToLower(fieldPath)

	for _, p := range d.patterns {
		for _, kw := range p.keywords {
			if strings.Contains(lower, kw) {
				return &DetectedField{
					Path:        fieldPath,
					PIIClass:    p.class,
					FromPattern: true,
				}
			}
		}
	}

	return nil
}

// ScanFieldPaths scans all field paths in a message against PII patterns.
// Returns only fields that matched and were not already covered by a
// contract policy (those are handled separately by the engine).
func (d *Detector) ScanFieldPaths(paths []string, contractPIIPaths map[string]bool) []DetectedField {
	var detected []DetectedField
	for _, path := range paths {
		if contractPIIPaths[path] {
			continue
		}
		if f := d.DetectFromPath(path); f != nil {
			detected = append(detected, *f)
		}
	}
	return detected
}
