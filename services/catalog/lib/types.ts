// Shared types mirror the backend domain model.
// These are maintained manually and kept in sync with the Go/Python service
// types. A future improvement would generate these from Protobuf definitions.

export type SchemaFormat = "AVRO" | "JSON_SCHEMA" | "PROTOBUF";
export type ContractStatus = "DRAFT" | "PENDING_APPROVAL" | "ACTIVE" | "DEPRECATED";
export type CompatibilityMode = "BACKWARD" | "FORWARD" | "FULL" | "BACKWARD_TRANSITIVE" | "NONE";
export type ImpactLevel = "CRITICAL" | "HIGH" | "MEDIUM" | "NONE";
export type ChangeClassification = "SAFE" | "COMPATIBLE" | "BREAKING";

export interface Contract {
  id: string;
  name: string;
  topic: string;
  owner_team: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface ContractVersion {
  id: string;
  contract_id: string;
  version: string;
  schema_format: SchemaFormat;
  schema_body: object;
  compatibility: CompatibilityMode;
  status: ContractStatus;
  authored_by: string;
  created_at: string;
}

export interface FieldPolicy {
  id: string;
  contract_id: string;
  field_path: string;
  policy_type: "MASK" | "REDACT" | "TOKENIZE" | "AUDIT_LOG";
  policy_config: Record<string, unknown>;
}

export interface ApprovalEvent {
  id: string;
  contract_version_id: string;
  actor: string;
  action: "SUBMITTED" | "APPROVED" | "REJECTED" | "SUPERSEDED";
  comment?: string;
  created_at: string;
}

export interface ScoredConsumer {
  name: string;
  node_type: string;
  owner_team: string | null;
  impact: ImpactLevel;
  affected_fields: string[];
  depth: number;
  is_direct: boolean;
}

export interface BlastRadiusReport {
  topic: string;
  changed_fields: string[];
  generated_at: string;
  total_consumers_affected: number;
  worst_impact: ImpactLevel;
  consumers: ScoredConsumer[];
  required_approvals: string[];
  summary: string;
}

export interface BreakingChange {
  field: string;
  change_type: string;
  description: string;
}

export interface GateResponse {
  classification: ChangeClassification;
  breaking_changes: BreakingChange[];
  affected_consumers: ScoredConsumer[];
  requires_approval: boolean;
  approval_from_teams: string[];
}
