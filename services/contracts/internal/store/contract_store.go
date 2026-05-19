package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzigo-io/mzigo/services/contracts/internal/domain"
)

// ContractStore is the only layer that touches Postgres.
// We use pgx directly rather than an ORM. The queries are simple enough
// that an ORM adds indirection without reducing complexity.
// All methods accept a context so callers control timeout and cancellation.
type ContractStore struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *ContractStore {
	return &ContractStore{pool: pool}
}

// GetActiveVersion returns the currently ACTIVE contract version for a topic.
// This is the hot path for gateway cache fills. The index on
// (contract_id, status) covers this query.
func (s *ContractStore) GetActiveVersion(ctx context.Context, topic string) (*domain.ContractVersion, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT
			cv.id, cv.contract_id, cv.version, cv.schema_format,
			cv.schema_body, cv.compatibility, cv.status,
			cv.registry_id, cv.authored_by, cv.created_at
		FROM contract_versions cv
		JOIN contracts c ON c.id = cv.contract_id
		WHERE c.topic = $1
		  AND cv.status = 'ACTIVE'
		ORDER BY cv.created_at DESC
		LIMIT 1
	`, topic)

	return scanVersion(row)
}

// ListActiveVersions returns all ACTIVE contract versions.
// Used to warm the gateway cache on startup.
func (s *ContractStore) ListActiveVersions(ctx context.Context) ([]*domain.ContractVersion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			cv.id, cv.contract_id, cv.version, cv.schema_format,
			cv.schema_body, cv.compatibility, cv.status,
			cv.registry_id, cv.authored_by, cv.created_at
		FROM contract_versions cv
		WHERE cv.status = 'ACTIVE'
		ORDER BY cv.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying active versions: %w", err)
	}
	defer rows.Close()

	var versions []*domain.ContractVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

// GetContractByName fetches the contract record by its unique name.
func (s *ContractStore) GetContractByName(ctx context.Context, name string) (*domain.Contract, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, topic, owner_team, description, created_at, updated_at
		FROM contracts
		WHERE name = $1
	`, name)

	var c domain.Contract
	err := row.Scan(&c.ID, &c.Name, &c.Topic, &c.OwnerTeam, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning contract: %w", err)
	}

	return &c, nil
}

// GetLatestVersion returns the most recently created version for a contract,
// regardless of status. Used by the evolution checker to diff against.
func (s *ContractStore) GetLatestVersion(ctx context.Context, contractID string) (*domain.ContractVersion, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT
			id, contract_id, version, schema_format,
			schema_body, compatibility, status,
			registry_id, authored_by, created_at
		FROM contract_versions
		WHERE contract_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, contractID)

	return scanVersion(row)
}

// CreateVersion inserts a new contract version in DRAFT status.
// The caller is responsible for transitioning status via SubmitForApproval
// or ActivateVersion.
func (s *ContractStore) CreateVersion(ctx context.Context, v *domain.ContractVersion) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO contract_versions
			(id, contract_id, version, schema_format, schema_body,
			 compatibility, status, authored_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		v.ID, v.ContractID, v.Version, v.SchemaFormat,
		v.SchemaBody, v.Compatibility, v.Status, v.AuthoredBy,
	)
	if err != nil {
		return fmt.Errorf("inserting contract version: %w", err)
	}
	return nil
}

// ActivateVersion transitions a version to ACTIVE and deprecates any
// previously active version for the same contract in a single transaction.
// We do this atomically to prevent a window where a topic has no active contract.
func (s *ContractStore) ActivateVersion(ctx context.Context, contractID, versionID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE contract_versions
		SET status = 'DEPRECATED'
		WHERE contract_id = $1
		  AND status = 'ACTIVE'
		  AND id != $2
	`, contractID, versionID)
	if err != nil {
		return fmt.Errorf("deprecating previous active version: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE contract_versions
		SET status = 'ACTIVE'
		WHERE id = $1
	`, versionID)
	if err != nil {
		return fmt.Errorf("activating version: %w", err)
	}

	return tx.Commit(ctx)
}

// GetFieldPolicies returns all field-level policies for a contract.
func (s *ContractStore) GetFieldPolicies(ctx context.Context, contractID string) ([]*domain.FieldPolicy, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, contract_id, field_path, policy_type, policy_config
		FROM field_policies
		WHERE contract_id = $1
	`, contractID)
	if err != nil {
		return nil, fmt.Errorf("querying field policies: %w", err)
	}
	defer rows.Close()

	var policies []*domain.FieldPolicy
	for rows.Next() {
		var p domain.FieldPolicy
		if err := rows.Scan(&p.ID, &p.ContractID, &p.FieldPath, &p.PolicyType, &p.PolicyConfig); err != nil {
			return nil, fmt.Errorf("scanning field policy: %w", err)
		}
		policies = append(policies, &p)
	}

	return policies, rows.Err()
}

// GetLineageEdges returns known downstream consumers of a topic.
// Used by the gate handler to populate blast-radius information.
func (s *ContractStore) GetLineageEdges(ctx context.Context, topic string) ([]domain.AffectedConsumer, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT consumer_name, consumer_type, consumer_team
		FROM lineage_edges
		WHERE source_topic = $1
	`, topic)
	if err != nil {
		return nil, fmt.Errorf("querying lineage edges: %w", err)
	}
	defer rows.Close()

	var consumers []domain.AffectedConsumer
	for rows.Next() {
		var c domain.AffectedConsumer
		if err := rows.Scan(&c.Name, &c.ConsumerType, &c.Team); err != nil {
			return nil, fmt.Errorf("scanning lineage edge: %w", err)
		}
		consumers = append(consumers, c)
	}

	return consumers, rows.Err()
}

// AppendApprovalEvent writes an immutable approval event to the audit chain.
func (s *ContractStore) AppendApprovalEvent(ctx context.Context, event *domain.ApprovalEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO approval_events
			(id, contract_version_id, actor, action, comment)
		VALUES ($1, $2, $3, $4, $5)
	`, event.ID, event.ContractVersionID, event.Actor, event.Action, event.Comment)
	if err != nil {
		return fmt.Errorf("inserting approval event: %w", err)
	}
	return nil
}

// scanVersion is a shared helper for scanning contract_version rows.
// It handles the nullable registry_id column.
func scanVersion(row pgx.Row) (*domain.ContractVersion, error) {
	var v domain.ContractVersion
	err := row.Scan(
		&v.ID, &v.ContractID, &v.Version, &v.SchemaFormat,
		&v.SchemaBody, &v.Compatibility, &v.Status,
		&v.RegistryID, &v.AuthoredBy, &v.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning contract version: %w", err)
	}
	return &v, nil
}
