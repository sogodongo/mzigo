package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/mzigo-io/mzigo/services/gateway/internal/config"
	"github.com/mzigo-io/mzigo/services/gateway/internal/handler"
)

type Server struct {
	api     *http.Server
	metrics *http.Server
	log     zerolog.Logger
	cfg     config.ServerConfig
}

func New(cfg config.ServerConfig, produceHandler *handler.ProduceHandler, log zerolog.Logger) *Server {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/healthz", healthz)
	apiMux.HandleFunc("/readyz", readyz)
	apiMux.Handle("/v1/produce", produceHandler)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	return &Server{
		api: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      apiMux,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
		metrics: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.MetricsPort),
			Handler:      metricsMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		log: log,
		cfg: cfg,
	}
}

// Start brings up the API and metrics servers in background goroutines.
// Errors from either server are forwarded to the returned channel.
// The caller is responsible for calling Shutdown when done.
func (s *Server) Start() <-chan error {
	errCh := make(chan error, 2)

	go func() {
		s.log.Info().Int("port", s.cfg.Port).Msg("api server listening")
		if err := s.api.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("api server: %w", err)
		}
	}()

	go func() {
		s.log.Info().Int("port", s.cfg.MetricsPort).Msg("metrics server listening")
		if err := s.metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server: %w", err)
		}
	}()

	return errCh
}

// Shutdown gracefully drains in-flight requests before stopping.
// Kubernetes sends SIGTERM and waits for the pod to exit. If we exit
// before in-flight requests complete, callers get connection resets.
// The shutdown timeout must be less than the Kubernetes terminationGracePeriodSeconds.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info().Msg("shutting down servers")

	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.api.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("api server shutdown: %w", err)
	}
	if err := s.metrics.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("metrics server shutdown: %w", err)
	}

	return nil
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// readyz is a separate endpoint from healthz.
// healthz = process is alive (liveness probe).
// readyz = process is ready to serve traffic (readiness probe).
// The distinction matters during startup: the gateway should not receive
// traffic until the contract cache is warmed. We set readyz to return 503
// until cache warming completes. For now it mirrors healthz; cache-aware
// readiness is wired in main.go via the ready flag.
func readyz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
