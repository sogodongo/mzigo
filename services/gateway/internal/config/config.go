package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the complete runtime configuration for the gateway.
// Every field has a production-safe default. Fields with no safe default
// are required and will cause startup to fail with a clear error.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	Contracts ContractsConfig `mapstructure:"contracts"`
	OTel      OTelConfig      `mapstructure:"otel"`
	Log       LogConfig       `mapstructure:"log"`
}

type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	MetricsPort     int           `mapstructure:"metrics_port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type KafkaConfig struct {
	BootstrapServers string        `mapstructure:"bootstrap_servers"`
	SchemaRegistryURL string       `mapstructure:"schema_registry_url"`
	ProducerTimeout  time.Duration `mapstructure:"producer_timeout"`
	// MessageMaxBytes caps the size of a single message we will forward.
	// Default matches Kafka broker default. Operators raising the broker limit
	// must raise this to match or producers will see inconsistent rejections.
	MessageMaxBytes int `mapstructure:"message_max_bytes"`
}

type ContractsConfig struct {
	// ServiceURL is the address of the Contracts service.
	// The gateway fetches contracts from here at startup and on cache invalidation.
	ServiceURL string `mapstructure:"service_url"`

	// CacheTTL is the maximum age of a cached contract before the gateway
	// re-fetches it. This bounds the staleness window after a contract update.
	CacheTTL time.Duration `mapstructure:"cache_ttl"`

	// FetchTimeout bounds how long a cache-fill operation can take.
	// If exceeded, the gateway uses the stale cached version and emits a warning metric.
	FetchTimeout time.Duration `mapstructure:"fetch_timeout"`
}

type OTelConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"service_name"`
	Enabled     bool   `mapstructure:"enabled"`
}

type LogConfig struct {
	// Level: debug, info, warn, error
	Level string `mapstructure:"level"`
	// Format: json (production) or console (local dev)
	Format string `mapstructure:"format"`
}

// Load reads configuration from environment variables and an optional config file.
// Environment variables take precedence over file values.
// Prefix: MZIGO_GATEWAY_ (e.g. MZIGO_GATEWAY_SERVER_PORT=8080)
func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("MZIGO_GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	v.SetConfigName("gateway")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/mzigo/")
	v.AddConfigPath(".")

	// Config file is optional. Missing file is not an error;
	// environment variables alone are sufficient for Kubernetes deployments.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.metrics_port", 9100)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.shutdown_timeout", "30s")

	v.SetDefault("kafka.producer_timeout", "5s")
	v.SetDefault("kafka.message_max_bytes", 1048576) // 1MB

	v.SetDefault("contracts.cache_ttl", "30s")
	v.SetDefault("contracts.fetch_timeout", "3s")

	v.SetDefault("otel.service_name", "mzigo-gateway")
	v.SetDefault("otel.enabled", true)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
}

func validate(cfg *Config) error {
	if cfg.Kafka.BootstrapServers == "" {
		return fmt.Errorf("kafka.bootstrap_servers is required")
	}
	if cfg.Contracts.ServiceURL == "" {
		return fmt.Errorf("contracts.service_url is required")
	}
	if cfg.OTel.Enabled && cfg.OTel.Endpoint == "" {
		return fmt.Errorf("otel.endpoint is required when otel.enabled is true")
	}
	return nil
}
