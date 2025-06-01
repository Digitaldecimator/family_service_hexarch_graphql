package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/knadh/koanf/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
)

// TracerConfig holds configuration for the tracer
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Insecure       bool
}

// NewTracerConfig creates a new tracer configuration from koanf
func NewTracerConfig(k *koanf.Koanf) TracerConfig {
	return TracerConfig{
		ServiceName:    k.String("telemetry.service_name"),
		ServiceVersion: k.String("telemetry.service_version"),
		Environment:    k.String("telemetry.environment"),
		OTLPEndpoint:   k.String("telemetry.otlp.endpoint"),
		Insecure:       k.Bool("telemetry.otlp.insecure"),
	}
}

// InitTracer initializes the OpenTelemetry tracer provider
func InitTracer(ctx context.Context, config TracerConfig, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	// Create OTLP exporter
	var clientOpts []otlptracegrpc.Option
	clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(config.OTLPEndpoint))

	if config.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	client := otlptracegrpc.NewClient(clientOpts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("Initialized OpenTelemetry tracer",
		zap.String("service", config.ServiceName),
		zap.String("version", config.ServiceVersion),
		zap.String("environment", config.Environment),
		zap.String("endpoint", config.OTLPEndpoint),
	)

	return tp, nil
}

// ShutdownTracer gracefully shuts down the tracer provider
func ShutdownTracer(ctx context.Context, tp *sdktrace.TracerProvider, logger *zap.Logger, k *koanf.Koanf) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(k.Int("telemetry.shutdown_timeout"))*time.Second)
	defer cancel()

	if err := tp.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down tracer provider", zap.Error(err))
	} else {
		logger.Info("Tracer provider shut down successfully")
	}
}
