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

package health_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	"github.com/osapi-io/osapi/internal/controller/api/health"
	"github.com/osapi-io/osapi/internal/controller/api/health/gen"
)

type HealthReadyGetPublicTestSuite struct {
	suite.Suite

	ctx       context.Context
	appConfig config.Config
	logger    *slog.Logger
}

func (s *HealthReadyGetPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *HealthReadyGetPublicTestSuite) TestGetHealthReady() {
	tests := []struct {
		name         string
		checker      health.Checker
		validateFunc func(resp gen.GetHealthReadyResponseObject)
	}{
		{
			name: "ready when all checks pass",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(resp gen.GetHealthReadyResponseObject) {
				r, ok := resp.(gen.GetHealthReady200JSONResponse)
				s.True(ok)
				s.Equal("ready", r.Status)
			},
		},
		{
			name: "not ready when NATS check fails",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats not connected") },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(resp gen.GetHealthReadyResponseObject) {
				r, ok := resp.(gen.GetHealthReady503JSONResponse)
				s.True(ok)
				s.Equal("not_ready", r.Status)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "nats not connected")
			},
		},
		{
			name: "not ready when KV check fails",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return fmt.Errorf("kv bucket not accessible") },
			},
			validateFunc: func(resp gen.GetHealthReadyResponseObject) {
				r, ok := resp.(gen.GetHealthReady503JSONResponse)
				s.True(ok)
				s.Equal("not_ready", r.Status)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "kv bucket not accessible")
			},
		},
		{
			name: "not ready when both checks fail",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats down") },
				KVCheck:   func() error { return fmt.Errorf("kv down") },
			},
			validateFunc: func(resp gen.GetHealthReadyResponseObject) {
				r, ok := resp.(gen.GetHealthReady503JSONResponse)
				s.True(ok)
				s.Equal("not_ready", r.Status)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "nats down")
				s.Contains(*r.Error, "kv down")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			handler := health.New(slog.Default(), tt.checker, time.Now(), "0.1.0", nil, nil)

			resp, err := handler.GetHealthReady(s.ctx, gen.GetHealthReadyRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *HealthReadyGetPublicTestSuite) TestGetHealthReadyHTTP() {
	tests := []struct {
		name         string
		checker      *health.NATSChecker
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when all checks pass returns ready",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "status")
				s.Contains(rec.Body.String(), "ready")
			},
		},
		{
			name: "when NATS check fails returns not ready",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats not connected") },
				KVCheck:   func() error { return nil },
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusServiceUnavailable, rec.Code)
				s.Contains(rec.Body.String(), "status")
				s.Contains(rec.Body.String(), "not_ready")
				s.Contains(rec.Body.String(), "error")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			healthHandler := health.New(
				s.logger,
				tc.checker,
				time.Now(),
				"0.1.0",
				nil,
				nil,
			)
			strictHandler := gen.NewStrictHandler(healthHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestHealthReadyGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HealthReadyGetPublicTestSuite))
}
