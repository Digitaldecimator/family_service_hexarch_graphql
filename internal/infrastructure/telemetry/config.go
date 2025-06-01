package telemetry

import (
	"github.com/knadh/koanf/v2"
)

// Config holds all telemetry configuration
type Config struct {
	Enabled          bool         `mapstructure:"enabled"`
	ServiceName      string       `mapstructure:"service_name"`
	Environment      string       `mapstructure:"environment"`
	Version          string       `mapstructure:"version"`
	ShutdownTimeout  int          `mapstructure:"shutdown_timeout"`
	OTLP             OTLPConfig   `mapstructure:"otlp"`
	Tracing          TracingConfig `mapstructure:"tracing"`
	Metrics          MetricsConfig `mapstructure:"metrics"`
	HTTP             HTTPConfig    `mapstructure:"http"`
}

// OTLPConfig holds configuration for OTLP exporter
type OTLPConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Insecure bool   `mapstructure:"insecure"`
	Timeout  int    `mapstructure:"timeout_seconds"`
}

// TracingConfig holds configuration for tracing
type TracingConfig struct {
	Enabled         bool    `mapstructure:"enabled"`
	SamplingRatio   float64 `mapstructure:"sampling_ratio"`
	PropagationKeys []string `mapstructure:"propagation_keys"`
}

// MetricsConfig holds configuration for metrics
type MetricsConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	ReportingFreq int    `mapstructure:"reporting_frequency_seconds"`
	Prometheus    PrometheusConfig `mapstructure:"prometheus"`
}

// PrometheusConfig holds configuration for Prometheus metrics
type PrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Listen  string `mapstructure:"listen"`
	Path    string `mapstructure:"path"`
}

// LoadConfig loads telemetry configuration from koanf
func LoadConfig(k *koanf.Koanf) Config {
	return Config{
		Enabled:         k.Bool("telemetry.enabled"),
		ServiceName:     k.String("telemetry.service_name"),
		Environment:     k.String("telemetry.environment"),
		Version:         k.String("telemetry.version"),
		ShutdownTimeout: k.Int("telemetry.shutdown_timeout"),
		OTLP: OTLPConfig{
			Endpoint: k.String("telemetry.otlp.endpoint"),
			Insecure: k.Bool("telemetry.otlp.insecure"),
			Timeout:  k.Int("telemetry.otlp.timeout_seconds"),
		},
		Tracing: TracingConfig{
			Enabled:         k.Bool("telemetry.tracing.enabled"),
			SamplingRatio:   k.Float64("telemetry.tracing.sampling_ratio"),
			PropagationKeys: k.Strings("telemetry.tracing.propagation_keys"),
		},
		Metrics: MetricsConfig{
			Enabled:       k.Bool("telemetry.metrics.enabled"),
			ReportingFreq: k.Int("telemetry.metrics.reporting_frequency_seconds"),
			Prometheus: PrometheusConfig{
				Enabled: k.Bool("telemetry.exporters.metrics.prometheus.enabled"),
				Listen:  k.String("telemetry.exporters.metrics.prometheus.listen"),
				Path:    k.String("telemetry.exporters.metrics.prometheus.path"),
			},
		},
		HTTP: HTTPConfig{
			TracingEnabled: k.Bool("telemetry.http.tracing_enabled"),
		},
	}
}

// GetTelemetryDefaults returns default values for telemetry configuration
func GetTelemetryDefaults() map[string]interface{} {
	return map[string]interface{}{
		"telemetry.enabled":         true,
		"telemetry.service_name":    "family-service",
		"telemetry.environment":     "development",
		"telemetry.version":         "1.0.0",
		"telemetry.shutdown_timeout": 5,

		// OTLP defaults
		"telemetry.otlp.endpoint":        "localhost:4317",
		"telemetry.otlp.insecure":        true,
		"telemetry.otlp.timeout_seconds": 5,

		// Tracing defaults
		"telemetry.tracing.enabled":          true,
		"telemetry.tracing.sampling_ratio":   1.0, // Sample everything by default
		"telemetry.tracing.propagation_keys": []string{"traceparent", "tracestate", "baggage"},

		// Metrics defaults
		"telemetry.metrics.enabled":                   true,
		"telemetry.metrics.reporting_frequency_seconds": 15,
		"telemetry.exporters.metrics.prometheus.enabled": true,
		"telemetry.exporters.metrics.prometheus.listen": "0.0.0.0:8080",
		"telemetry.exporters.metrics.prometheus.path": "/metrics",

		// HTTP defaults
		"telemetry.http.tracing_enabled": true,
	}
}
