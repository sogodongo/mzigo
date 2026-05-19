package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Masking MaskingConfig `mapstructure:"masking"`
	Log     LogConfig     `mapstructure:"log"`
}

type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	MetricsPort     int           `mapstructure:"metrics_port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type MaskingConfig struct {
	// TokenizationKey is the HMAC secret used to produce deterministic tokens.
	// Rotating this key invalidates all existing tokens. Treat as a secret.
	// In Kubernetes this is injected via a Secret volume, not an env var,
	// to keep it out of pod spec logs.
	TokenizationKey string `mapstructure:"tokenization_key"`

	// TokenPrefix is prepended to all generated tokens for easy identification
	// in downstream systems. "tok_" is the default; teams can override per
	// deployment to namespace tokens by environment.
	TokenPrefix string `mapstructure:"token_prefix"`

	// MaskChar is the character used for MASK operations.
	MaskChar string `mapstructure:"mask_char"`

	// MaskKeepSuffix is how many trailing characters to preserve in MASK mode.
	// For card numbers: 4 preserves the last four digits.
	MaskKeepSuffix int `mapstructure:"mask_keep_suffix"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("MZIGO_MASKING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.port", 8084)
	v.SetDefault("server.metrics_port", 9104)
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "5s")
	v.SetDefault("server.shutdown_timeout", "15s")

	v.SetDefault("masking.token_prefix", "tok_")
	v.SetDefault("masking.mask_char", "*")
	v.SetDefault("masking.mask_keep_suffix", 4)

	v.SetConfigName("masking")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/mzigo/")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if cfg.Masking.TokenizationKey == "" {
		return nil, fmt.Errorf("masking.tokenization_key is required")
	}

	return &cfg, nil
}
