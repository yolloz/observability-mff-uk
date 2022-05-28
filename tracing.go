package main

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"net/http"
)

func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		propagatedCtx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		spanCtx, span := otel.Tracer(appName).Start(propagatedCtx, "tracingMiddleware")
		tracingId := span.SpanContext().TraceID().String()
		r.Header.Set(hdrTracingId, tracingId)
		log := requestLog("tracingMiddleware", r)
		log.Debug("starting tracing...")
		defer span.End()
		spannedRequest := r.WithContext(spanCtx)
		w.Header().Set(hdrTracingId, tracingId)
		next.ServeHTTP(w, spannedRequest)
		log.Debug("closing tracing...")
	})
}

func tracerProvider(url string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(appName),
		)),
	)
	return tp, nil
}
