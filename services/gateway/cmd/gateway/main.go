package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mzigo-io/mzigo/services/gateway/internal/cache"
	"github.com/mzigo-io/mzigo/services/gateway/internal/config"
	"github.com/mzigo-io/mzigo/services/gateway/internal/handler"
	"github.com/mzigo-io/mzigo/services/gateway/internal/kafka"
	"github.com/mzigo-io/mzigo/services/gateway/internal/server"
	"github.com/mzigo-io/mzigo/services/gateway/internal/validation"
)

// main is intentionally thin. Its only job is to wire dependencies together
// and manage the process lifecycle. Business logic belongs in the internal packages.
func main() {
	cfg, err := config.Load()
	if err != nil {
		// Use stdlib here: zerolog isn't configured yet.
		// Fatal before logger is ready is acceptable; this is a startup failure.
		panic("configuration error: " + err.Error())
	}

	log := buildLogger(cfg.Log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.OTel.Enabled {
		shutdown, err := initTracer(ctx, cfg.OTel)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize tracer")
		}
		defer shutdown(ctx)
	}

	tracer := otel.Tracer(cfg.OTel.ServiceName)

	contractCache := cache.New(
		cfg.Contracts.ServiceURL,
		cfg.Contracts.CacheTTL,
		cfg.Contracts.FetchTimeout,
		log,
	)

	log.Info().Msg("warming contract cache")
	if err := contractCache.Warm(ctx); err != nil {
		log.Fatal().Err(err).Msg("contract cache warm failed; refusing to start")
	}

	kafkaProducer, err := kafka.NewProducer(cfg.Kafka.BootstrapServers, cfg.Kafka.ProducerTimeout, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create kafka producer")
	}
	defer kafkaProducer.Close()

	validator := validation.New()
	produceHandler := handler.NewProduceHandler(contractCache, validator, kafkaProducer, log, tracer)

	srv := server.New(cfg.Server, produceHandler, log)
	errCh := srv.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-errCh:
		log.Error().Err(err).Msg("server error")
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
		os.Exit(1)
	}

	log.Info().Msg("shutdown complete")
}

func buildLogger(cfg config.LogConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	if cfg.Format == "console" {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			Level(level).
			With().Timestamp().
			Str("service", "mzigo-gateway").
			Logger()
	}

	return zerolog.New(os.Stdout).
		Level(level).
		With().Timestamp().
		Str("service", "mzigo-gateway").
		Logger()
}

func initTracer(ctx context.Context, cfg config.OTelConfig) (func(context.Context) error, error) {
	conn, err := grpc.DialContext(ctx, cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// Sample at 10% in production to keep trace volume manageable.
		// Adjust via config for high-cardinality debugging sessions.
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)),
	)

	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
