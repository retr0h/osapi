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

package api

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/osapi-io/osapi/internal/audit"
)

// excludedAuditPaths lists path prefixes that should not generate audit entries.
var excludedAuditPaths = []string{
	"/api/health",
	"/metrics",
}

// auditMiddleware returns Echo middleware that records audit entries for
// authenticated requests. Writes are asynchronous to avoid adding latency.
func auditMiddleware(
	store audit.Store,
	logger *slog.Logger,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := c.Request().URL.Path

			for _, prefix := range excludedAuditPaths {
				if strings.HasPrefix(path, prefix) {
					return next(c)
				}
			}

			start := time.Now()

			err := next(c)

			// Only audit authenticated requests.
			subject, _ := c.Get(ContextKeySubject).(string)
			if subject == "" {
				return err
			}

			roles, _ := c.Get(ContextKeyRoles).([]string)

			// v5 returns the bare http.ResponseWriter from Response().
			// UnwrapResponse recovers Echo's wrapper, which is what
			// records the status actually written.
			responseCode := 0
			if resp, uerr := echo.UnwrapResponse(c.Response()); uerr == nil {
				responseCode = resp.Status
			}

			entry := audit.Entry{
				ID:           uuid.New().String(),
				Timestamp:    start,
				User:         subject,
				Roles:        roles,
				Method:       c.Request().Method,
				Path:         path,
				SourceIP:     c.RealIP(),
				ResponseCode: responseCode,
				DurationMs:   time.Since(start).Milliseconds(),
			}

			spanCtx := trace.SpanContextFromContext(
				c.Request().Context(),
			)
			if spanCtx.HasTraceID() {
				entry.TraceID = spanCtx.TraceID().String()
			}

			// Use Background context because this goroutine runs after the HTTP
			// response is sent, at which point the request context is canceled.
			go func() {
				if writeErr := store.Write(context.Background(), entry); writeErr != nil {
					logger.Warn(
						"failed to write audit entry",
						slog.String("error", writeErr.Error()),
						slog.String("entry_id", entry.ID),
					)
				}
			}()

			return err
		}
	}
}
