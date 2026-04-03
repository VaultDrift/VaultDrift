package tracing

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration.
type Config struct {
	Enabled      bool    `yaml:"enabled" env:"TRACING_ENABLED"`
	Exporter     string  `yaml:"exporter" env:"TRACING_EXPORTER"`       // "otlp", "stdout", "jaeger"
	OTLPEndpoint string  `yaml:"otlp_endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceName  string  `yaml:"service_name" env:"OTEL_SERVICE_NAME"`
	ServiceVer   string  `yaml:"service_version" env:"OTEL_SERVICE_VERSION"`
	Environment  string  `yaml:"environment" env:"OTEL_ENVIRONMENT"`
	SampleRate   float64 `yaml:"sample_rate" env:"OTEL_TRACES_SAMPLER_ARG"` // 0.0 to 1.0
}

// DefaultConfig returns default tracing configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		Exporter:     "stdout",
		OTLPEndpoint: "http://localhost:4318",
		ServiceName:  "vaultdrift",
		ServiceVer:   "1.0.0",
		Environment:  "development",
		SampleRate:   1.0,
	}
}

// Provider manages the OpenTelemetry tracer provider.
type Provider struct {
	provider *sdktrace.TracerProvider
	config   Config
}

// NewProvider creates a new tracing provider.
func NewProvider(cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{config: cfg}, nil
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "vaultdrift"
	}

	// Create exporter
	exp, err := createExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVer),
			attribute.String("deployment.environment", cfg.Environment),
		),
		resource.WithProcessRuntimeDescription(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure sampler
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SampleRate),
	)

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global provider
	otel.SetTracerProvider(provider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Printf("Tracing initialized: exporter=%s, service=%s, sample_rate=%.2f",
		cfg.Exporter, cfg.ServiceName, cfg.SampleRate)

	return &Provider{
		provider: provider,
		config:   cfg,
	}, nil
}

// createExporter creates the appropriate trace exporter based on config.
func createExporter(cfg Config) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "otlp":
		return createOTLPExporter(cfg)
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "jaeger":
		// Jaeger supports OTLP natively now
		return createOTLPExporter(cfg)
	default:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
}

// createOTLPExporter creates an OTLP trace exporter.
func createOTLPExporter(cfg Config) (sdktrace.SpanExporter, error) {
	ctx := context.Background()

	// Try HTTP first (commonly used), fall back to gRPC
	if os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL") == "grpc" {
		return otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		)
	}

	return otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
	)
}

// Tracer returns a tracer from the global provider.
func (p *Provider) Tracer(name string) trace.Tracer {
	if p.provider == nil {
		return otel.Tracer(name)
	}
	return p.provider.Tracer(name)
}

// Shutdown gracefully shuts down the tracer provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.provider == nil {
		return nil
	}
	return p.provider.Shutdown(ctx)
}

// IsEnabled returns whether tracing is enabled.
func (p *Provider) IsEnabled() bool {
	return p.config.Enabled
}

// Global tracer accessor helpers

// Tracer returns the global tracer.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// StartSpan starts a new span from the global tracer.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("vaultdrift").Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the given span.
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}
