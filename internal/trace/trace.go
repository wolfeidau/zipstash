package trace

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var globalProvider *sdktrace.TracerProvider
var tracer trace.Tracer

func NewProvider(ctx context.Context, name, version string) (*sdktrace.TracerProvider, error) {
	exp, err := newExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	res, err := newResource(ctx, name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracer = tp.Tracer(name)

	// globalProvider = &Provider{tp: tp}
	globalProvider = tp

	return globalProvider, nil
}

func Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name)
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	exporter := os.Getenv("TRACE_EXPORTER")

	switch exporter {
	case "grpc":
		clientOTel := otlptracegrpc.NewClient()
		return otlptrace.New(ctx, clientOTel)
	case "stdout":
		return stdouttrace.New()
	default:
		return tracetest.NewNoopExporter(), nil
	}
}

func newResource(cxt context.Context, name, version string) (*resource.Resource, error) {
	options := []resource.Option{
		resource.WithSchemaURL(semconv.SchemaURL),
	}
	options = append(options, resource.WithHost())
	options = append(options, resource.WithFromEnv())
	options = append(options, resource.WithAttributes(
		semconv.TelemetrySDKNameKey.String("otelconfig"),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKVersionKey.String(version),
	))

	return resource.New(
		cxt,
		options...,
	)
}
