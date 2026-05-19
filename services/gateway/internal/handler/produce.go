package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/mzigo-io/mzigo/services/gateway/internal/cache"
	"github.com/mzigo-io/mzigo/services/gateway/internal/kafka"
	"github.com/mzigo-io/mzigo/services/gateway/internal/validation"
)

// ProduceRequest is the wire format accepted by POST /v1/produce.
// The contract_id and contract_version are required so the gateway
// validates against the exact version the producer thinks it is using.
// A mismatch between declared version and active version is itself a violation.
type ProduceRequest struct {
	Topic           string          `json:"topic"`
	ContractID      string          `json:"contract_id"`
	ContractVersion string          `json:"contract_version"`
	ProducerID      string          `json:"producer_id"`
	Key             string          `json:"key,omitempty"`
	Payload         json.RawMessage `json:"payload"`
}

// ProduceResponse is returned for both accepted and rejected messages.
// We always return a structured response body, never an empty 4xx.
// Producers need machine-readable rejection reasons to surface in their logs.
type ProduceResponse struct {
	Status     string               `json:"status"` // ACCEPTED | REJECTED
	MessageID  string               `json:"message_id,omitempty"`
	Violations []validation.Violation `json:"violations,omitempty"`
	DurationMs int64                `json:"duration_ms"`
}

type ProduceHandler struct {
	cache     *cache.ContractCache
	validator *validation.Validator
	producer  *kafka.Producer
	log       zerolog.Logger
	tracer    trace.Tracer
	metrics   *produceMetrics
}

func NewProduceHandler(
	cache *cache.ContractCache,
	validator *validation.Validator,
	producer *kafka.Producer,
	log zerolog.Logger,
	tracer trace.Tracer,
) *ProduceHandler {
	return &ProduceHandler{
		cache:     cache,
		validator: validator,
		producer:  producer,
		log:       log.With().Str("handler", "produce").Logger(),
		tracer:    tracer,
		metrics:   newProduceMetrics(),
	}
}

func (h *ProduceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	ctx, span := h.tracer.Start(r.Context(), "gateway.produce")
	defer span.End()

	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", start)
		return
	}

	if req.Topic == "" || req.ContractID == "" || req.ProducerID == "" {
		h.writeError(w, http.StatusBadRequest, "topic, contract_id, and producer_id are required", start)
		return
	}

	span.SetAttributes(
		attribute.String("topic", req.Topic),
		attribute.String("contract_id", req.ContractID),
		attribute.String("contract_version", req.ContractVersion),
		attribute.String("producer_id", req.ProducerID),
	)

	contract, err := h.cache.Get(ctx, req.Topic)
	if err != nil {
		h.log.Error().Err(err).Str("topic", req.Topic).Msg("contract lookup failed")
		h.metrics.contractLookupErrors.Inc()
		// A missing contract is a hard rejection. We do not allow messages to
		// bypass governance because a contract was not found.
		h.writeRejection(w, req.Topic, []validation.Violation{{
			Type:    validation.ViolationUnknownTopic,
			Message: "no active contract found for topic",
		}}, start)
		return
	}

	// Version mismatch between what the producer declared and what is active
	// is treated as a violation, not a system error. The producer is running
	// against a stale contract version. This surfaces in the violations dashboard
	// and triggers producer-side feedback.
	if req.ContractVersion != "" && req.ContractVersion != contract.Version {
		h.log.Warn().
			Str("topic", req.Topic).
			Str("declared", req.ContractVersion).
			Str("active", contract.Version).
			Msg("producer contract version mismatch")
		h.metrics.versionMismatches.WithLabelValues(req.Topic).Inc()
	}

	result, err := h.validator.Validate(req.Payload, contract)
	if err != nil {
		h.log.Error().Err(err).Str("topic", req.Topic).Msg("validation processing error")
		h.metrics.validationErrors.Inc()
		h.writeError(w, http.StatusInternalServerError, "validation failed", start)
		return
	}

	if !result.Valid {
		h.metrics.rejected.WithLabelValues(req.Topic, string(result.FirstViolation().Type)).Inc()
		h.log.Info().
			Str("topic", req.Topic).
			Str("producer_id", req.ProducerID).
			Int("violations", len(result.Violations)).
			Str("first_violation", string(result.FirstViolation().Type)).
			Msg("message rejected")
		h.writeRejection(w, req.Topic, result.Violations, start)
		return
	}

	msgID, err := h.producer.Produce(ctx, req.Topic, req.Key, req.Payload)
	if err != nil {
		h.log.Error().Err(err).Str("topic", req.Topic).Msg("kafka produce failed")
		h.metrics.produceErrors.WithLabelValues(req.Topic).Inc()
		h.writeError(w, http.StatusBadGateway, "failed to produce message", start)
		return
	}

	h.metrics.accepted.WithLabelValues(req.Topic).Inc()
	h.metrics.latency.WithLabelValues(req.Topic).Observe(float64(time.Since(start).Milliseconds()))

	writeJSON(w, http.StatusAccepted, ProduceResponse{
		Status:     "ACCEPTED",
		MessageID:  msgID,
		DurationMs: time.Since(start).Milliseconds(),
	})
}

func (h *ProduceHandler) writeRejection(w http.ResponseWriter, topic string, violations []validation.Violation, start time.Time) {
	writeJSON(w, http.StatusUnprocessableEntity, ProduceResponse{
		Status:     "REJECTED",
		Violations: violations,
		DurationMs: time.Since(start).Milliseconds(),
	})
}

func (h *ProduceHandler) writeError(w http.ResponseWriter, status int, msg string, start time.Time) {
	writeJSON(w, status, map[string]any{
		"error":       msg,
		"duration_ms": time.Since(start).Milliseconds(),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
