package domain

import (
	"time"
)

// SchemaFormat identifies which schema serialization format a contract uses.
// The evolution checker applies format-specific compatibility rules.
type SchemaFormat string

const (
	SchemaFormatAvro       SchemaFormat = "AVRO"
	SchemaFormatJSONSchema SchemaFormat = "JSON_SCHEMA"
	SchemaFormatProtobuf   SchemaFormat = "PROTOBUF"
)

// CompatibilityMode mirrors the Schema Registry compatibility modes.
// BACKWARD is the production default: new schema can read data written by old schema.
type CompatibilityMode string

const (
	CompatibilityBackward           CompatibilityMode = "BACKWARD"
	CompatibilityForward            CompatibilityMode = "FORWARD"
	CompatibilityFull               CompatibilityMode = "FULL"
	CompatibilityBackwardTransitive CompatibilityMode = "BACKWARD_TRANSITIVE"
	CompatibilityNone               CompatibilityMode = "NONE"
)

// ContractStatus tracks where a contract version is in its lifecycle.
type ContractStatus string

const (
	StatusDraft           ContractStatus = "DRAFT"
	StatusPendingApproval ContractStatus = "PENDING_APPROVAL"
	StatusActive          ContractStatus = "ACTIVE"
	StatusDeprecated      ContractStatus = "DEPRECATED"
)

// ChangeClassification is the output of the evolution checker.
// SAFE: no downstream impact. COMPATIBLE: consumers may need awareness.
// BREAKING: requires blast-radius analysis and owner approvals before activation.
type ChangeClassification string

const (
	ClassificationSafe       ChangeClassification = "SAFE"
	ClassificationCompatible ChangeClassification = "COMPATIBLE"
	ClassificationBreaking   ChangeClassification = "BREAKING"
)

type Contract struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Topic       string    `json:"topic"`
	OwnerTeam   string    `json:"owner_team"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ContractVersion struct {
	ID            string            `json:"id"`
	ContractID    string            `json:"contract_id"`
	Version       string            `json:"version"`
	SchemaFormat  SchemaFormat      `json:"schema_format"`
	SchemaBody    []byte            `json:"schema_body"`
	Compatibility CompatibilityMode `json:"compatibility"`
	Status        ContractStatus    `json:"status"`
	RegistryID    *int              `json:"registry_id,omitempty"`
	AuthoredBy    string            `json:"authored_by"`
	CreatedAt     time.Time         `json:"created_at"`
}

type FieldPolicy struct {
	ID           string `json:"id"`
	ContractID   string `json:"contract_id"`
	FieldPath    string `json:"field_path"`
	PolicyType   string `json:"policy_type"`
	PolicyConfig []byte `json:"policy_config"`
}

// ApprovalEvent is an immutable record in the approval chain.
// We never update these; the chain is append-only for auditability.
type ApprovalEvent struct {
	ID                string    `json:"id"`
	ContractVersionID string    `json:"contract_version_id"`
	Actor             string    `json:"actor"`
	Action            string    `json:"action"`
	Comment           string    `json:"comment,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// GateRequest is the payload the CI/CD GitHub Action sends to the gate endpoint.
type GateRequest struct {
	ContractName    string       `json:"contract_name"`
	ProposedSchema  []byte       `json:"proposed_schema"`
	SchemaFormat    SchemaFormat `json:"schema_format"`
	AuthoredBy      string       `json:"authored_by"`
	Environment     string       `json:"environment"`
	PRNumber        int          `json:"pr_number,omitempty"`
	RepositorySlug  string       `json:"repository_slug,omitempty"`
}

// GateResponse is what the GitHub Action receives back.
// The action uses Classification to decide whether to fail the check.
// BreakingChanges and AffectedConsumers are rendered as the PR comment.
type GateResponse struct {
	Classification  ChangeClassification `json:"classification"`
	BreakingChanges []BreakingChange     `json:"breaking_changes,omitempty"`
	AffectedConsumers []AffectedConsumer `json:"affected_consumers,omitempty"`
	RequiresApproval  bool               `json:"requires_approval"`
	ApprovalFromTeams []string           `json:"approval_from_teams,omitempty"`
}

type BreakingChange struct {
	Field       string `json:"field"`
	ChangeType  string `json:"change_type"`
	Description string `json:"description"`
}

type AffectedConsumer struct {
	Name        string   `json:"name"`
	Team        string   `json:"team"`
	ConsumerType string  `json:"consumer_type"`
	FieldsImpacted []string `json:"fields_impacted"`
}
