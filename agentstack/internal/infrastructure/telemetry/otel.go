// Package telemetry provides OpenTelemetry integration for observability.
/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/raphaelmansuy/agentstack/internal/config"
)

// NewFromConfig creates a new Telemetry instance from application configuration.
func NewFromConfig(ctx context.Context, cfg config.OTelConfig, env string) (*Telemetry, error) {
	if !cfg.Enabled {
		return &Telemetry{
			tracer: otel.Tracer("agentstack-noop"),
			meter:  otel.Meter("agentstack-noop"),
		}, nil
	}

	tConfig := TelemetryConfig{
		ServiceName:     cfg.ServiceName,
		ServiceVersion:  "1.0.0",
		Environment:     env,
		OTLPEndpoint:    cfg.Endpoint,
		SampleRate:      1.0,
		MetricsInterval: 15 * time.Second,
		Insecure:        cfg.Insecure,
	}

	return New(ctx, tConfig)
}

// TelemetryConfig configures the telemetry system.
type TelemetryConfig struct {
	// ServiceName is the name of the service.
	ServiceName string
	// ServiceVersion is the version of the service.
	ServiceVersion string
	// Environment is the deployment environment (e.g., production, staging).
	Environment string
	// OTLPEndpoint is the OTLP collector endpoint.
	OTLPEndpoint string
	// SampleRate is the trace sampling rate (0.0 to 1.0).
	SampleRate float64
	// MetricsInterval is the interval for exporting metrics.
	MetricsInterval time.Duration
	// Insecure indicates whether to use insecure connection.
	Insecure bool
}

// DefaultTelemetryConfig returns default telemetry configuration.
func DefaultTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		ServiceName:     "agentstack",
		ServiceVersion:  "1.0.0",
		Environment:     "production",
		OTLPEndpoint:    "localhost:4317",
		SampleRate:      0.1, // 10% sampling in production
		MetricsInterval: 15 * time.Second,
		Insecure:        false,
	}
}

// Telemetry manages OpenTelemetry providers.
type Telemetry struct {
	config         TelemetryConfig
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter

	// Metrics
	requestCounter   metric.Int64Counter
	requestDuration  metric.Float64Histogram
	activeRequests   metric.Int64UpDownCounter
	errorCounter     metric.Int64Counter
	dbQueryDuration  metric.Float64Histogram
	cacheHitCounter  metric.Int64Counter
	cacheMissCounter metric.Int64Counter
}

// New creates a new Telemetry instance.
func New(ctx context.Context, config TelemetryConfig) (*Telemetry, error) {
	t := &Telemetry{config: config}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracer
	if err := t.initTracer(ctx, res); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	// Initialize meter
	if err := t.initMeter(ctx, res); err != nil {
		return nil, fmt.Errorf("failed to initialize meter: %w", err)
	}

	// Create metrics
	if err := t.createMetrics(); err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(t.tracerProvider)
	otel.SetMeterProvider(t.meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return t, nil
}

func (t *Telemetry) initTracer(ctx context.Context, res *resource.Resource) error {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(t.config.OTLPEndpoint),
	}
	if t.config.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create sampler based on configuration
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(t.config.SampleRate),
	)

	t.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
	)

	t.tracer = t.tracerProvider.Tracer(t.config.ServiceName)
	return nil
}

func (t *Telemetry) initMeter(ctx context.Context, res *resource.Resource) error {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(t.config.OTLPEndpoint),
	}
	if t.config.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	t.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(t.config.MetricsInterval),
			),
		),
	)

	t.meter = t.meterProvider.Meter(t.config.ServiceName)
	return nil
}

func (t *Telemetry) createMetrics() error {
	var err error

	t.requestCounter, err = t.meter.Int64Counter(
		"http.server.request_count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	t.requestDuration, err = t.meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000),
	)
	if err != nil {
		return err
	}

	t.activeRequests, err = t.meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	t.errorCounter, err = t.meter.Int64Counter(
		"http.server.error_count",
		metric.WithDescription("Total number of HTTP errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	t.dbQueryDuration, err = t.meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration in milliseconds"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000),
	)
	if err != nil {
		return err
	}

	t.cacheHitCounter, err = t.meter.Int64Counter(
		"cache.hit_count",
		metric.WithDescription("Number of cache hits"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return err
	}

	t.cacheMissCounter, err = t.meter.Int64Counter(
		"cache.miss_count",
		metric.WithDescription("Number of cache misses"),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the telemetry system.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer: %w", err)
		}
	}
	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter: %w", err)
		}
	}
	return nil
}

// Tracer returns the tracer.
func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

// Meter returns the meter.
func (t *Telemetry) Meter() metric.Meter {
	return t.meter
}

// RecordHTTPRequest records HTTP request metrics.
func (t *Telemetry) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", path),
		attribute.Int("http.status_code", statusCode),
	}

	if t.requestCounter != nil {
		t.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
	if t.requestDuration != nil {
		t.requestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
	}

	if statusCode >= 400 && t.errorCounter != nil {
		t.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordActiveRequest increments active requests.
func (t *Telemetry) RecordActiveRequest(ctx context.Context, delta int64) {
	if t.activeRequests != nil {
		t.activeRequests.Add(ctx, delta)
	}
}

// RecordDBQuery records database query metrics.
func (t *Telemetry) RecordDBQuery(ctx context.Context, operation, table string, duration time.Duration, err error) {
	if t.dbQueryDuration != nil {
		attrs := []attribute.KeyValue{
			attribute.String("db.operation", operation),
			attribute.String("db.table", table),
			attribute.Bool("db.success", err == nil),
		}

		t.dbQueryDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
	}
}

// RecordCacheHit records a cache hit.
func (t *Telemetry) RecordCacheHit(ctx context.Context, cacheType string) {
	if t.cacheHitCounter != nil {
		t.cacheHitCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("cache.type", cacheType),
		))
	}
}

// RecordCacheMiss records a cache miss.
func (t *Telemetry) RecordCacheMiss(ctx context.Context, cacheType string) {
	if t.cacheMissCounter != nil {
		t.cacheMissCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("cache.type", cacheType),
		))
	}
}

// StartSpan starts a new span.
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// HTTPMiddleware returns HTTP middleware for tracing and metrics.
func (t *Telemetry) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if t.tracer == nil {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()

			// Extract trace context from headers
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start span
			ctx, span := t.tracer.Start(ctx, r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("url.full", r.URL.String()),
					attribute.String("user_agent.original", r.UserAgent()),
					attribute.String("server.address", r.Host),
				),
			)
			defer span.End()

			// Track active requests
			t.RecordActiveRequest(ctx, 1)
			defer t.RecordActiveRequest(ctx, -1)

			// Wrap response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Serve request
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record metrics
			duration := time.Since(start)
			t.RecordHTTPRequest(ctx, r.Method, r.URL.Path, rw.statusCode, duration)

			// Update span with response info
			span.SetAttributes(attribute.Int("http.response.status_code", rw.statusCode))
			if rw.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// DBQueryHook wraps database queries with tracing.
func (t *Telemetry) DBQueryHook(ctx context.Context, operation, query string, args []any) func(error) {
	ctx, span := t.tracer.Start(ctx, "db."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("db.operation", operation),
			attribute.String("db.statement", truncateQuery(query, 1000)),
		),
	)

	start := time.Now()

	return func(err error) {
		duration := time.Since(start)
		t.RecordDBQuery(ctx, operation, "", duration, err)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}
}

// truncateQuery truncates a query to the specified length.
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}

// CacheHook wraps cache operations with tracing.
func (t *Telemetry) CacheHook(ctx context.Context, operation, key string) func(hit bool, err error) {
	ctx, span := t.tracer.Start(ctx, "cache."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("cache.operation", operation),
			attribute.String("cache.key", key),
		),
	)

	return func(hit bool, err error) {
		span.SetAttributes(attribute.Bool("cache.hit", hit))

		if operation == "get" {
			if hit {
				t.RecordCacheHit(ctx, "redis")
			} else {
				t.RecordCacheMiss(ctx, "redis")
			}
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}
}
