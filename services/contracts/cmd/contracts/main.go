package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/mzigo-io/mzigo/services/contracts/internal/evolution"
	"github.com/mzigo-io/mzigo/services/contracts/internal/handler"
	"github.com/mzigo-io/mzigo/services/contracts/internal/store"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "mzigo-contracts").Logger()

	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("server_port", 8081)
	v.SetDefault("metrics_port", 9101)
	v.SetDefault("shutdown_timeout", "30s")

	dbURL := v.GetString("DATABASE_URL")
	if dbURL == "" {
		log.Fatal().Msg("DATABASE_URL is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create database pool")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("database ping failed")
	}

	contractStore := store.New(pool)
	evolutionChecker := evolution.NewChecker()

	// Tracer initialization omitted here for brevity; mirrors the gateway pattern.
	// See services/gateway/cmd/gateway/main.go for the full OTel wiring.
	tracer := noopTracer{}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	apiMux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	apiMux.Handle("/v1/gate", handler.NewGateHandler(contractStore, evolutionChecker, log, tracer))
	apiMux.Handle("/internal/", handler.NewInternalHandler(contractStore, log))

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	apiServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", v.GetInt("server_port")),
		Handler:      apiMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	metricsServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", v.GetInt("metrics_port")),
		Handler:     metricsMux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() {
		log.Info().Int("port", v.GetInt("server_port")).Msg("api server listening")
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-errCh:
		log.Error().Err(err).Msg("server error")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	apiServer.Shutdown(shutdownCtx)
	metricsServer.Shutdown(shutdownCtx)

	log.Info().Msg("shutdown complete")
}
