package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/mzigo-io/mzigo/services/masking/internal/config"
	"github.com/mzigo-io/mzigo/services/masking/internal/handler"
	"github.com/mzigo-io/mzigo/services/masking/internal/masking"
	"github.com/mzigo-io/mzigo/services/masking/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("configuration error: " + err.Error())
	}

	log := zerolog.New(os.Stdout).
		With().Timestamp().
		Str("service", "mzigo-masking").
		Logger()

	tokenizer := masking.NewTokenizer(cfg.Masking.TokenizationKey, cfg.Masking.TokenPrefix)
	applier := masking.NewApplier(tokenizer, cfg.Masking.MaskChar, cfg.Masking.MaskKeepSuffix)
	detector := masking.NewDetector()
	engine := masking.NewEngine(applier, detector)

	tracer := noop.NewTracerProvider().Tracer("mzigo-masking")
	maskHandler := handler.NewMaskHandler(engine, log, tracer)

	srv := server.New(cfg.Server, maskHandler, log)
	errCh := srv.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-errCh:
		log.Error().Err(err).Msg("server error")
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error().Err(err).Msg("shutdown error")
		os.Exit(1)
	}
}
