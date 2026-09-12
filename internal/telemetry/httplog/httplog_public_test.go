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

package httplog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/telemetry/httplog"
)

type NewPublicTestSuite struct {
	suite.Suite

	buf    *bytes.Buffer
	logger *slog.Logger
}

func (s *NewPublicTestSuite) SetupTest() {
	s.buf = &bytes.Buffer{}
	s.logger = slog.New(slog.NewJSONHandler(s.buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// record returns the single log line the middleware emitted for one request.
func (s *NewPublicTestSuite) record() map[string]any {
	s.T().Helper()

	var entry map[string]any
	s.Require().NoError(json.Unmarshal(s.buf.Bytes(), &entry))

	return entry
}

func (s *NewPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		register     func(e *echo.Echo)
		requestPath  string
		wantHTTP     int
		wantLevel    string
		validateFunc func(entry map[string]any)
	}{
		{
			name: "logs a successful request at info",
			register: func(e *echo.Echo) {
				e.GET("/ok", func(c *echo.Context) error {
					return c.String(http.StatusOK, "ok")
				})
			},
			requestPath: "/ok",
			wantHTTP:    http.StatusOK,
			wantLevel:   "INFO",
			validateFunc: func(entry map[string]any) {
				req, ok := entry["request"].(map[string]any)
				s.Require().True(ok, "request group is present")
				s.Equal(http.MethodGet, req["method"])
				s.Equal("/ok", req["path"])

				resp, ok := entry["response"].(map[string]any)
				s.Require().True(ok, "response group is present")
				s.Equal(float64(http.StatusOK), resp["status"])
				s.Contains(resp, "latency")
				s.Contains(resp, "length")
			},
		},
		{
			// The reason this package exists. slog-echo tested for a routing
			// error with err.(*echo.HTTPError), but Echo v5 returns the
			// unexported *echo.httpError, so every 404 was reported as a 500.
			name:        "reports an unmatched route as 404, not 500",
			register:    func(_ *echo.Echo) {},
			requestPath: "/nope",
			wantHTTP:    http.StatusNotFound,
			wantLevel:   "ERROR",
			validateFunc: func(entry map[string]any) {
				resp := entry["response"].(map[string]any)
				s.Equal(float64(http.StatusNotFound), resp["status"])
			},
		},
		{
			name: "logs a client error at warn",
			register: func(e *echo.Echo) {
				e.GET("/bad", func(c *echo.Context) error {
					return c.String(http.StatusBadRequest, "bad")
				})
			},
			requestPath: "/bad",
			wantHTTP:    http.StatusBadRequest,
			wantLevel:   "WARN",
			validateFunc: func(entry map[string]any) {
				resp := entry["response"].(map[string]any)
				s.Equal(float64(http.StatusBadRequest), resp["status"])
			},
		},
		{
			name: "logs a server error at error",
			register: func(e *echo.Echo) {
				e.GET("/boom", func(c *echo.Context) error {
					return c.String(http.StatusInternalServerError, "boom")
				})
			},
			requestPath: "/boom",
			wantHTTP:    http.StatusInternalServerError,
			wantLevel:   "ERROR",
			validateFunc: func(entry map[string]any) {
				resp := entry["response"].(map[string]any)
				s.Equal(float64(http.StatusInternalServerError), resp["status"])
			},
		},
		{
			name: "logs a handler error with its message",
			register: func(e *echo.Echo) {
				e.GET("/err", func(_ *echo.Context) error {
					return echo.NewHTTPError(http.StatusTeapot, "short and stout")
				})
			},
			requestPath: "/err",
			wantHTTP:    http.StatusTeapot,
			wantLevel:   "ERROR",
			validateFunc: func(entry map[string]any) {
				s.Contains(entry["msg"], "short and stout")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()

			e := echo.New()
			e.Use(httplog.New(s.logger))
			tt.register(e)

			rec := httptest.NewRecorder()
			e.ServeHTTP(
				rec,
				httptest.NewRequest(http.MethodGet, tt.requestPath, nil),
			)

			s.Equal(tt.wantHTTP, rec.Code, "HTTP status")

			entry := s.record()
			s.Equal(tt.wantLevel, entry["level"], "log level")
			tt.validateFunc(entry)
		})
	}
}

func TestNewPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NewPublicTestSuite))
}
