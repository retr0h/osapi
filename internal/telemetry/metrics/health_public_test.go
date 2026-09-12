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

package metrics_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/telemetry/metrics"
)

type HealthPublicTestSuite struct {
	suite.Suite
}

func (s *HealthPublicTestSuite) getFreePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port
}

func (s *HealthPublicTestSuite) newStartedServer() (*metrics.Server, int, func()) {
	port := s.getFreePort()
	srv := metrics.New("127.0.0.1", port, slog.Default())
	s.Require().NotNil(srv)

	srv.Start()
	time.Sleep(100 * time.Millisecond)

	stop := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Stop(ctx)
	}

	return srv, port, stop
}

func (s *HealthPublicTestSuite) TestHandleHealth() {
	tests := []struct {
		name         string
		validateFunc func(port int)
	}{
		{
			name: "returns 200 with status ok",
			validateFunc: func(port int) {
				url := fmt.Sprintf("http://127.0.0.1:%d/health", port)

				resp, err := http.Get(url) //nolint:gosec
				s.Require().NoError(err)
				defer func() { _ = resp.Body.Close() }()

				s.Equal(http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				s.Require().NoError(err)
				s.JSONEq(`{"status":"ok"}`, string(body))
			},
		},
		{
			name: "returns Content-Type application/json",
			validateFunc: func(port int) {
				url := fmt.Sprintf("http://127.0.0.1:%d/health", port)

				resp, err := http.Get(url) //nolint:gosec
				s.Require().NoError(err)
				defer func() { _ = resp.Body.Close() }()

				s.Equal("application/json", resp.Header.Get("Content-Type"))
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, port, stop := s.newStartedServer()
			defer stop()

			tc.validateFunc(port)
		})
	}
}

func (s *HealthPublicTestSuite) TestHandleReady() {
	tests := []struct {
		name           string
		readinessFunc  func() error
		wantStatusCode int
		validateFunc   func([]byte, error)
	}{
		{
			name:           "returns 503 when no readiness func configured",
			readinessFunc:  nil,
			wantStatusCode: http.StatusServiceUnavailable,
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Contains(string(body), "not_ready")
			},
		},
		{
			name:           "returns 503 when readiness func returns error",
			readinessFunc:  func() error { return errors.New("dependency unavailable") },
			wantStatusCode: http.StatusServiceUnavailable,
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Contains(string(body), "dependency unavailable")
			},
		},
		{
			name:           "returns 200 when readiness func returns nil",
			readinessFunc:  func() error { return nil },
			wantStatusCode: http.StatusOK,
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Contains(string(body), "ready")
			},
		},
		{
			name:           "returns Content-Type application/json",
			readinessFunc:  func() error { return nil },
			wantStatusCode: http.StatusOK,
			validateFunc: func(body []byte, err error) {
				s.Require().NoError(err)
				s.Contains(string(body), "ready")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			port := s.getFreePort()
			srv := metrics.New("127.0.0.1", port, slog.Default())
			s.Require().NotNil(srv)

			if tc.readinessFunc != nil {
				srv.SetReadinessFunc(tc.readinessFunc)
			}

			srv.Start()
			time.Sleep(100 * time.Millisecond)

			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				srv.Stop(ctx)
			}()

			url := fmt.Sprintf("http://127.0.0.1:%d/health/ready", port)

			resp, err := http.Get(url) //nolint:gosec
			s.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			s.Equal(tc.wantStatusCode, resp.StatusCode)
			s.Equal("application/json", resp.Header.Get("Content-Type"))

			tc.validateFunc(io.ReadAll(resp.Body))
		})
	}
}

func TestHealthPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HealthPublicTestSuite))
}
