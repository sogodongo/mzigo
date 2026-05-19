package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/mzigo-io/mzigo/services/masking/internal/config"
	"github.com/mzigo-io/mzigo/services/masking/internal/handler"
)

type Server struct {
	api     *http.Server
	metrics *http.Server
	log     zerolog.Logger
}

func New(cfg config.ServerConfig, maskHandler *handler.MaskHandler, log zerolog.Logger) *Server {
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	apiMux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	apiMux.Handle("/v1/mask", maskHandler)

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
			Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
			Handler: metricsMux,
		},
		log: log,
	}
}

func (s *Server) Start() <-chan error {
	errCh := make(chan error, 2)
	go func() {
		s.log.Info().Str("addr", s.api.Addr).Msg("api server listening")
		if err := s.api.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("api server: %w", err)
		}
	}()
	go func() {
		if err := s.metrics.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server: %w", err)
		}
	}()
	return errCh
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.api.Shutdown(ctx)
	s.metrics.Shutdown(ctx)
	return nil
}
