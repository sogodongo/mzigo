package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/mzigo-io/mzigo/services/contracts/internal/store"
)

// InternalHandler serves the gateway-facing endpoints under /internal/v1/.
// These endpoints are not exposed outside the cluster. In Kubernetes, the
// NetworkPolicy restricts access to the gateway service identity only.
//
// Endpoint inventory:
//   GET /internal/v1/contracts/active          - all active contracts (cache warm)
//   GET /internal/v1/contracts/by-topic/{topic} - single contract by topic
type InternalHandler struct {
	store *store.ContractStore
	log   zerolog.Logger
}

func NewInternalHandler(store *store.ContractStore, log zerolog.Logger) *InternalHandler {
	return &InternalHandler{
		store: store,
		log:   log.With().Str("handler", "internal").Logger(),
	}
}

func (h *InternalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/internal/v1/contracts/active" && r.Method == http.MethodGet:
		h.listActive(w, r)
	case strings.HasPrefix(path, "/internal/v1/contracts/by-topic/") && r.Method == http.MethodGet:
		topic := strings.TrimPrefix(path, "/internal/v1/contracts/by-topic/")
		h.getByTopic(w, r, topic)
	default:
		http.NotFound(w, r)
	}
}

func (h *InternalHandler) listActive(w http.ResponseWriter, r *http.Request) {
	versions, err := h.store.ListActiveVersions(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("failed to list active versions")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, versions)
}

func (h *InternalHandler) getByTopic(w http.ResponseWriter, r *http.Request, topic string) {
	if topic == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "topic is required"})
		return
	}

	version, err := h.store.GetActiveVersion(r.Context(), topic)
	if err != nil {
		h.log.Error().Err(err).Str("topic", topic).Msg("failed to get active version")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if version == nil {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, version)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
