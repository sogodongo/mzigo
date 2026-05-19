package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/mzigo-io/mzigo/services/contracts/internal/domain"
	"github.com/mzigo-io/mzigo/services/contracts/internal/evolution"
	"github.com/mzigo-io/mzigo/services/contracts/internal/store"
)

// GateHandler serves POST /v1/gate - the endpoint called by the mzigo-contracts
// GitHub Action on every contract pull request.
//
// The action expects a synchronous response it can use to:
//   1. Pass or fail the CI check
//   2. Post a structured comment on the PR with the blast-radius report
//
// This handler intentionally does not write to the database. The gate is
// a read-only analysis endpoint. Contract version creation is a separate
// operation triggered after the PR merges.
type GateHandler struct {
	store   *store.ContractStore
	checker *evolution.Checker
	log     zerolog.Logger
	tracer  trace.Tracer
	metrics *gateMetrics
}

func NewGateHandler(
	store *store.ContractStore,
	checker *evolution.Checker,
	log zerolog.Logger,
	tracer trace.Tracer,
) *GateHandler {
	return &GateHandler{
		store:   store,
		checker: checker,
		log:     log.With().Str("handler", "gate").Logger(),
		tracer:  tracer,
		metrics: newGateMetrics(),
	}
}

func (h *GateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "contracts.gate")
	defer span.End()

	var req domain.GateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ContractName == "" || len(req.ProposedSchema) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "contract_name and proposed_schema are required"})
		return
	}

	span.SetAttributes(
		attribute.String("contract_name", req.ContractName),
		attribute.String("schema_format", string(req.SchemaFormat)),
		attribute.String("environment", req.Environment),
	)

	contract, err := h.store.GetContractByName(ctx, req.ContractName)
	if err != nil {
		h.log.Error().Err(err).Str("contract", req.ContractName).Msg("failed to fetch contract")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// New contract with no history: any schema is SAFE since there are
	// no existing consumers to break.
	if contract == nil {
		h.metrics.classifications.WithLabelValues("SAFE").Inc()
		writeJSON(w, http.StatusOK, domain.GateResponse{
			Classification:  domain.ClassificationSafe,
			RequiresApproval: false,
		})
		return
	}

	latest, err := h.store.GetLatestVersion(ctx, contract.ID)
	if err != nil {
		h.log.Error().Err(err).Str("contract", req.ContractName).Msg("failed to fetch latest version")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if latest == nil {
		h.metrics.classifications.WithLabelValues("SAFE").Inc()
		writeJSON(w, http.StatusOK, domain.GateResponse{
			Classification:  domain.ClassificationSafe,
			RequiresApproval: false,
		})
		return
	}

	result, err := h.checker.Check(
		latest.SchemaBody,
		req.ProposedSchema,
		req.SchemaFormat,
		latest.Compatibility,
	)
	if err != nil {
		h.log.Error().Err(err).Str("contract", req.ContractName).Msg("evolution check failed")
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "schema analysis failed: " + err.Error()})
		return
	}

	h.metrics.classifications.WithLabelValues(string(result.Classification)).Inc()

	resp := buildGateResponse(result)

	if result.Classification == domain.ClassificationBreaking {
		consumers, err := h.store.GetLineageEdges(ctx, contract.Topic)
		if err != nil {
			h.log.Warn().Err(err).Msg("failed to fetch lineage edges for blast radius")
		} else {
			resp.AffectedConsumers = consumers
			resp.RequiresApproval = len(consumers) > 0
			resp.ApprovalFromTeams = uniqueTeams(consumers)
		}
	}

	h.log.Info().
		Str("contract", req.ContractName).
		Str("classification", string(result.Classification)).
		Int("breaking_changes", len(result.Changes)).
		Int("affected_consumers", len(resp.AffectedConsumers)).
		Msg("gate check complete")

	writeJSON(w, http.StatusOK, resp)
}

func buildGateResponse(result *evolution.CheckResult) domain.GateResponse {
	resp := domain.GateResponse{
		Classification: result.Classification,
	}

	for _, c := range result.Changes {
		resp.BreakingChanges = append(resp.BreakingChanges, domain.BreakingChange{
			Field:       c.Field,
			ChangeType:  string(c.ChangeType),
			Description: c.Description,
		})
	}

	resp.RequiresApproval = result.Classification == domain.ClassificationBreaking

	return resp
}

func uniqueTeams(consumers []domain.AffectedConsumer) []string {
	seen := make(map[string]bool)
	var teams []string
	for _, c := range consumers {
		if c.Team != "" && !seen[c.Team] {
			seen[c.Team] = true
			teams = append(teams, c.Team)
		}
	}
	return teams
}
