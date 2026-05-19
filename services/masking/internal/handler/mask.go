package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/mzigo-io/mzigo/services/masking/internal/masking"
)

// MaskRequest is the wire format for POST /v1/mask.
// The gateway sends this after contract validation passes and before
// forwarding the message to Kafka.
type MaskRequest struct {
	Topic    string                `json:"topic"`
	Policies []masking.FieldPolicy `json:"policies"`
	Payload  json.RawMessage       `json:"payload"`
}

// MaskResponse carries the transformed payload and audit metadata.
type MaskResponse struct {
	Payload           json.RawMessage `json:"payload"`
	FieldsTransformed []string        `json:"fields_transformed"`
	// PatternWarnings are fields that look like PII but have no contract policy.
	// The gateway logs these and emits a metric. The message is not blocked.
	PatternWarnings []string `json:"pattern_warnings,omitempty"`
	DurationMs      int64    `json:"duration_ms"`
}

type MaskHandler struct {
	engine  *masking.Engine
	log     zerolog.Logger
	tracer  trace.Tracer
	metrics *maskMetrics
}

func NewMaskHandler(engine *masking.Engine, log zerolog.Logger, tracer trace.Tracer) *MaskHandler {
	return &MaskHandler{
		engine:  engine,
		log:     log.With().Str("handler", "mask").Logger(),
		tracer:  tracer,
		metrics: newMaskMetrics(),
	}
}

func (h *MaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	ctx, span := h.tracer.Start(r.Context(), "masking.mask")
	defer span.End()
	_ = ctx

	var req MaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Topic == "" || len(req.Payload) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "topic and payload are required"})
		return
	}

	span.SetAttributes(
		attribute.String("topic", req.Topic),
		attribute.Int("policy_count", len(req.Policies)),
	)

	result, err := h.engine.Mask(masking.MaskRequest{
		Payload:  []byte(req.Payload),
		Policies: req.Policies,
	})
	if err != nil {
		h.log.Error().Err(err).Str("topic", req.Topic).Msg("masking failed")
		h.metrics.failures.WithLabelValues(req.Topic).Inc()
		// Return 422 rather than 500: the message was structurally valid but
		// could not be safely masked. The producer should not retry unchanged.
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "masking failed: " + err.Error(),
		})
		return
	}

	h.metrics.fieldsTransformed.WithLabelValues(req.Topic).Add(float64(len(result.FieldsTransformed)))
	h.metrics.latency.WithLabelValues(req.Topic).Observe(float64(time.Since(start).Milliseconds()))

	if len(result.PatternDetections) > 0 {
		h.metrics.patternWarnings.WithLabelValues(req.Topic).Add(float64(len(result.PatternDetections)))
		paths := make([]string, len(result.PatternDetections))
		for i, d := range result.PatternDetections {
			paths[i] = d.Path
		}
		h.log.Warn().
			Str("topic", req.Topic).
			Strs("fields", paths).
			Msg("PII-like fields detected without contract policy; contract should be updated")
	}

	patternPaths := make([]string, len(result.PatternDetections))
	for i, d := range result.PatternDetections {
		patternPaths[i] = d.Path
	}

	writeJSON(w, http.StatusOK, MaskResponse{
		Payload:           json.RawMessage(result.Payload),
		FieldsTransformed: result.FieldsTransformed,
		PatternWarnings:   patternPaths,
		DurationMs:        time.Since(start).Milliseconds(),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
