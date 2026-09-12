// Copyright (c) 2024 John Dewey

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

package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.opentelemetry.io/otel"

	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/telemetry/httplog"
)

// shutdownGracePeriod bounds how long Stop waits for in-flight requests.
const shutdownGracePeriod = 10 * time.Second

// New initialize a new Server and configure an Echo server.
func New(
	appConfig config.Config,
	logger *slog.Logger,
	opts ...Option,
) *Server {
	e := echo.New()

	allowOrigins := appConfig.Controller.API.Security.CORS.AllowOrigins

	s := &Server{
		Echo:      e,
		logger:    logger.With(slog.String("subsystem", "controller.api")),
		appConfig: appConfig,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Register middleware after options so MeterProvider is available.
	otelConfig := echootel.Config{
		ServerName:     "osapi-api",
		TracerProvider: otel.GetTracerProvider(),
	}
	if s.meterProvider != nil {
		otelConfig.MeterProvider = s.meterProvider
	}

	e.Use(echootel.NewMiddlewareWithConfig(otelConfig))
	e.Use(httplog.New(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	// CORS is registered only when origins are configured.
	//
	// v4 defaulted an empty AllowOrigins to "*", which sends
	// Access-Control-Allow-Origin: * from an authenticated API without anyone
	// choosing it. v5 panics instead, and this package does not panic, so
	// leaving the middleware off is the remaining option. No header means the
	// browser applies same-origin policy, which is what an operator who has
	// not configured CORS should get.
	if len(allowOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: allowOrigins,
		}))
		s.logger.Info(
			"CORS enabled",
			slog.Any("allow_origins", allowOrigins),
		)
	} else {
		// Say so rather than leaving the operator to infer it from a browser
		// error with no server-side counterpart.
		s.logger.Info(
			"CORS not configured, cross-origin browser requests will be refused",
			slog.String(
				"configure",
				"controller.api.security.cors.allow_origins",
			),
		)
	}

	// Register audit middleware if an audit store is configured.
	if s.auditStore != nil {
		e.Use(auditMiddleware(s.auditStore, logger))
	}

	return s
}

// Start starts the Echo server with the configured port.
func (s *Server) Start() {
	// v5 drives the listener from a context rather than a Shutdown method,
	// so Stop cancels this one.
	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel

	go func() {
		s.logger.Info("starting server")

		sc := echo.StartConfig{
			Address:         fmt.Sprintf(":%d", s.appConfig.Controller.API.Port),
			HideBanner:      true,
			GracefulTimeout: shutdownGracePeriod,
		}
		if err := sc.Start(ctx, s.Echo); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			s.logger.Error(
				"failed to start server",
				slog.String("error", err.Error()),
			)
		}
	}()
}

// Stop gracefully shuts down the Echo server.
func (s *Server) Stop(
	_ context.Context,
) {
	s.logger.Info("stopping server")

	if s.stop == nil {
		s.logger.Info("server stopped gracefully")

		return
	}

	// Cancelling is what asks v5 to drain. StartConfig.GracefulTimeout bounds
	// how long it waits, and the goroutine in Start reports any error.
	s.stop()
	s.stop = nil
	s.logger.Info("server stopped gracefully")
}
