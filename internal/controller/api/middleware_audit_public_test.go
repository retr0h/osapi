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

package api_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace"

	"github.com/osapi-io/osapi/internal/audit"
	"github.com/osapi-io/osapi/internal/audit/mocks"
	"github.com/osapi-io/osapi/internal/controller/api"
	"go.uber.org/mock/gomock"
)

type AuditMiddlewarePublicTestSuite struct {
	suite.Suite
}

func (s *AuditMiddlewarePublicTestSuite) TestAuditMiddleware() {
	tests := []struct {
		name         string
		path         string
		subject      string
		roles        []string
		storeErr     error
		wantWrite    bool
		setupReq     func(req *http.Request) *http.Request
		validateFunc func(entry audit.Entry)
	}{
		{
			name:      "authenticated request is logged",
			path:      "/api/node/hostname",
			subject:   "user@example.com",
			roles:     []string{"admin"},
			wantWrite: true,
			validateFunc: func(entry audit.Entry) {
				s.Equal("user@example.com", entry.User)
				s.Equal("GET", entry.Method)
				s.Equal("/api/node/hostname", entry.Path)
				s.Equal(http.StatusOK, entry.ResponseCode)
				s.Equal([]string{"admin"}, entry.Roles)
			},
		},
		{
			name:    "unauthenticated request is skipped",
			path:    "/api/node/hostname",
			subject: "",
		},
		{
			name:    "health path is excluded",
			path:    "/api/health",
			subject: "user@example.com",
		},
		{
			name:    "health ready path is excluded",
			path:    "/api/health/ready",
			subject: "user@example.com",
		},
		{
			name:    "metrics path is excluded",
			path:    "/metrics",
			subject: "user@example.com",
		},
		{
			name:      "authenticated request with trace context captures trace ID",
			path:      "/api/node/hostname",
			subject:   "user@example.com",
			roles:     []string{"admin"},
			wantWrite: true,
			setupReq: func(req *http.Request) *http.Request {
				traceID, _ := trace.TraceIDFromHex(
					"4bf92f3577b34da6a3ce929d0e0e4736",
				)
				spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     trace.SpanID{1},
					TraceFlags: trace.FlagsSampled,
				})
				ctx := trace.ContextWithSpanContext(
					req.Context(), spanCtx,
				)
				return req.WithContext(ctx)
			},
			validateFunc: func(entry audit.Entry) {
				s.Equal(
					"4bf92f3577b34da6a3ce929d0e0e4736",
					entry.TraceID,
				)
			},
		},
		{
			name:      "store error is handled gracefully",
			path:      "/api/node/hostname",
			subject:   "user@example.com",
			roles:     []string{"admin"},
			storeErr:  fmt.Errorf("write failed"),
			wantWrite: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())
			store := mocks.NewMockStore(ctrl)

			// The middleware writes from a goroutine it does not join. The mock
			// closes this channel from the write, so the test waits on the call
			// itself rather than on a duration.
			written := make(chan struct{})
			var got audit.Entry

			if tt.wantWrite {
				store.EXPECT().
					Write(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, entry audit.Entry) error {
						got = entry
						close(written)
						return tt.storeErr
					})
			} else {
				// Excluded and unauthenticated paths return before the
				// goroutine is started, so any write is a regression.
				store.EXPECT().Write(gomock.Any(), gomock.Any()).Times(0)
			}

			logger := slog.Default()

			e := echo.New()
			e.Use(api.ExportAuditMiddleware(store, logger))
			e.GET(tt.path, func(c echo.Context) error {
				// Simulate scopeMiddleware setting context values.
				if tt.subject != "" {
					c.Set(api.ContextKeySubject, tt.subject)
					c.Set(api.ContextKeyRoles, tt.roles)
				}
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.setupReq != nil {
				req = tt.setupReq(req)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			s.Equal(http.StatusOK, rec.Code)

			if tt.wantWrite {
				<-written

				if tt.validateFunc != nil {
					tt.validateFunc(got)
				}
			}
		})
	}
}

func TestAuditMiddlewarePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuditMiddlewarePublicTestSuite))
}
