module github.com/mzigo-io/mzigo/services/gateway

go 1.22

require (
	github.com/confluentinc/confluent-kafka-go/v2 v2.3.0
	github.com/prometheus/client_golang v1.18.0
	github.com/rs/zerolog v1.32.0
	github.com/spf13/viper v1.18.2
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.22.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	google.golang.org/grpc v1.61.0
)
