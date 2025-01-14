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
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"

	"github.com/retr0h/osapi/internal/config"
)

// New initialize a new Server and configure an Echo server.
func New(
	appConfig config.Config,
	logger *slog.Logger,
) *Server {
	e := echo.New()
	e.HideBanner = true

	// Initialize CORS configuration
	corsConfig := middleware.CORSConfig{}

	allowOrigins := appConfig.API.Server.Security.CORS.AllowOrigins
	if len(allowOrigins) > 0 {
		corsConfig.AllowOrigins = allowOrigins
	}

	e.Use(slogecho.New(logger))
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(corsConfig))
	e.Use(middleware.Recover())

	return &Server{
		Echo:      e,
		logger:    logger,
		appConfig: appConfig,
	}
}

// Start starts the Echo server with the configured port.
func (s *Server) Start() {
	go func() {
		s.logger.Info("starting server")
		listenAddr := fmt.Sprintf(":%d", s.appConfig.API.Server.Port)
		if err := s.Echo.Start(listenAddr); err != nil && err != http.ErrServerClosed {
			s.logger.Error(
				"failed to start server",
				slog.String("error", err.Error()),
			)
		}
	}()
}

// Stop gracefully shuts down the Echo server.
func (s *Server) Stop(ctx context.Context) {
	s.logger.Info("stopping server")

	if err := s.Echo.Shutdown(ctx); err != nil {
		s.logger.Error(
			"server shutdown failed",
			slog.String("error", err.Error()),
		)
	} else {
		s.logger.Info("server stopped gracefully")
	}
}
