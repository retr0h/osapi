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

// Package httplog logs one line per HTTP request through the application's
// slog.Logger.
//
// It replaces github.com/samber/slog-echo, which is unusable on Echo v5: it
// tests for a routing error with err.(*echo.HTTPError), and v5 returns the
// unexported *echo.httpError, so every 404 was reported and logged as a 500.
// Echo's own RequestLogger resolves the status through the error handler
// instead, which is the only thing that knows it.
package httplog

import (
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// New returns middleware logging each request to logger. Field names follow
// what slog-echo emitted, so anything reading these logs keeps working.
func New(
	logger *slog.Logger,
) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:       true,
		LogProtocol:      true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogURI:           true,
		LogURIPath:       true,
		LogRoutePath:     true,
		LogRequestID:     true,
		LogReferer:       true,
		LogUserAgent:     true,
		LogStatus:        true,
		LogResponseSize:  true,
		LogContentLength: true,

		// Without this the middleware reports the status written so far
		// rather than the one the error handler settles on, which is how
		// slog-echo turned every 404 into a 500.
		HandleError: true,

		LogValuesFunc: func(
			c *echo.Context,
			v middleware.RequestLoggerValues,
		) error {
			attrs := []slog.Attr{
				slog.Group("request",
					slog.String("method", v.Method),
					slog.String("host", v.Host),
					slog.String("path", v.URIPath),
					slog.String("route", v.RoutePath),
					slog.String("ip", v.RemoteIP),
					slog.String("referer", v.Referer),
					slog.String("id", v.RequestID),
				),
				slog.Group("response",
					slog.Int("status", v.Status),
					slog.Duration("latency", v.Latency),
					slog.Int64("length", v.ResponseSize),
				),
			}

			level := slog.LevelInfo
			msg := "incoming request"

			switch {
			case v.Error != nil:
				level = slog.LevelError
				msg = v.Error.Error()
			case v.Status >= 500:
				level = slog.LevelError
			case v.Status >= 400:
				level = slog.LevelWarn
			}

			logger.LogAttrs(c.Request().Context(), level, msg, attrs...)

			return nil
		},
	})
}
