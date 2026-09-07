// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package metrics provides a lightweight HTTP server for per-component
// Prometheus metrics with isolated registries.
package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogecho "github.com/samber/slog-echo"
	"go.opentelemetry.io/otel/attribute"
	prometheusExporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// prometheusNewFn is injectable for testing the exporter creation error path.
var prometheusNewFn = prometheusExporter.New

// New creates a new metrics server on the given port with an isolated
// Prometheus registry and OTEL MeterProvider.
func New(
	host string,
	port int,
	logger *slog.Logger,
) *Server {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(
		collectors.ProcessCollectorOpts{},
	))

	exporter, err := prometheusNewFn(
		prometheusExporter.WithRegisterer(reg),
		prometheusExporter.WithNamespace("osapi"),
	)
	if err != nil {
		logger.Error(
			"failed to create prometheus exporter",
			slog.String("error", err.Error()),
		)

		return nil
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))

	srv := &Server{
		addr:          fmt.Sprintf("%s:%d", host, port),
		logger:        logger.With(slog.String("subsystem", "metrics")),
		registry:      reg,
		meterProvider: mp,
	}

	meter := mp.Meter("osapi")
	_, _ = meter.Float64ObservableGauge(
		"component_up",
		metric.WithDescription("Whether the component is ready (1) or not (0)."),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			if srv.readinessFunc == nil {
				o.Observe(0)
				return nil
			}
			if srv.readinessFunc() != nil {
				o.Observe(0)
				return nil
			}
			o.Observe(1)
			return nil
		}),
	)

	e := echo.New()
	e.HideBanner = true
	e.Use(slogecho.New(logger))
	e.Use(middleware.Recover())

	e.GET("/metrics", echo.WrapHandler(
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}),
	))
	e.GET("/health", srv.handleHealth)
	e.GET("/health/ready", srv.handleReady)

	srv.echo = e

	return srv
}

// SetReadinessFunc sets a function called by /health/ready to determine
// whether this component is ready to serve traffic. If fn returns an error
// the endpoint responds 503; if nil it responds 200.
func (s *Server) SetReadinessFunc(
	fn func() error,
) {
	s.readinessFunc = fn
}

// SubsystemStatus holds a subsystem name and a function that returns its
// current status string (e.g., "ok", "disabled", "error").
type SubsystemStatus struct {
	Name     string
	StatusFn func() string
}

// RegisterSubsystems registers an osapi_subsystem_up gauge that reports
// 1 when each subsystem status function returns "ok" and 0 otherwise.
// All subsystems are emitted in a single callback with a "subsystem" attribute.
func (s *Server) RegisterSubsystems(
	subsystems []SubsystemStatus,
) {
	meter := s.meterProvider.Meter("osapi")
	_, _ = meter.Float64ObservableGauge(
		"subsystem_up",
		metric.WithDescription("Whether a subsystem is healthy (1) or not (0)."),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			for _, sub := range subsystems {
				val := float64(0)
				if sub.StatusFn() == "ok" {
					val = 1
				}
				o.Observe(val, metric.WithAttributes(
					attribute.String("subsystem", sub.Name),
				))
			}
			return nil
		}),
	)
}

// RegisterHeartbeatAge registers an osapi_heartbeat_age_seconds gauge that
// reports the time since the last successful heartbeat write. The timeFn
// should return the timestamp of the last heartbeat, or zero time if none.
func (s *Server) RegisterHeartbeatAge(
	timeFn func() time.Time,
) {
	meter := s.meterProvider.Meter("osapi")
	_, _ = meter.Float64ObservableGauge(
		"heartbeat_age_seconds",
		metric.WithDescription("Seconds since last successful heartbeat write."),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			t := timeFn()
			if t.IsZero() {
				o.Observe(0)
				return nil
			}
			o.Observe(time.Since(t).Seconds())
			return nil
		}),
	)
}

// MeterProvider returns the isolated OTEL MeterProvider for this server.
// Components use this to create instruments that appear on this server's
// /metrics endpoint.
func (s *Server) MeterProvider() *sdkmetric.MeterProvider {
	return s.meterProvider
}

// Start starts the HTTP server in a background goroutine.
func (s *Server) Start() {
	go func() {
		s.logger.Info(
			"metrics server started",
			slog.String("addr", s.addr),
		)

		if err := s.echo.Start(s.addr); err != nil &&
			err != http.ErrServerClosed {
			s.logger.Error(
				"metrics server error",
				slog.String("error", err.Error()),
			)
		}
	}()
}

// Stop gracefully shuts down the HTTP server and meter provider.
func (s *Server) Stop(
	ctx context.Context,
) {
	if err := s.meterProvider.Shutdown(ctx); err != nil {
		s.logger.Error(
			"meter provider shutdown error",
			slog.String("error", err.Error()),
		)
	}

	if err := s.echo.Shutdown(ctx); err != nil {
		s.logger.Error(
			"metrics server shutdown error",
			slog.String("error", err.Error()),
		)
	}

	s.logger.Info("metrics server stopped")
}
