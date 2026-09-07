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

package client_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
)

type HealthPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *HealthPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *HealthPublicTestSuite) runner(
	handler http.HandlerFunc,
	serverURL string,
) string {
	if handler != nil {
		server := httptest.NewServer(handler)
		suite.T().Cleanup(server.Close)
		return server.URL
	}

	return serverURL
}

func (suite *HealthPublicTestSuite) TestLiveness() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.HealthStatus], error)
	}{
		{
			name: "when checking liveness returns health status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}),
			validateFunc: func(resp *client.Response[client.HealthStatus], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("ok", resp.Data.Status)
			},
		},
		{
			name:      "when client HTTP request fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.HealthStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "health liveness")
			},
		},
		{
			name: "when response body is nil returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}),
			validateFunc: func(resp *client.Response[client.HealthStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Contains(target.Message, "nil response body")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			sut := client.New(
				suite.runner(tc.handler, tc.serverURL),
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Health.Liveness(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *HealthPublicTestSuite) TestReady() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.ReadyStatus], error)
	}{
		{
			name: "when checking readiness returns ready status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ready"}`))
			}),
			validateFunc: func(resp *client.Response[client.ReadyStatus], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("ready", resp.Data.Status)
				suite.False(resp.Data.ServiceUnavailable)
			},
		},
		{
			name:      "when client HTTP request fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.ReadyStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "health ready")
			},
		},
		{
			name: "when 200 response body is nil returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}),
			validateFunc: func(resp *client.Response[client.ReadyStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Contains(target.Message, "nil response body")
			},
		},
		{
			name: "when unexpected status returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
			}),
			validateFunc: func(resp *client.Response[client.ReadyStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
				suite.Contains(target.Message, "unexpected status")
			},
		},
		{
			name: "when server returns 503 returns ready status with ServiceUnavailable",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"not_ready","error":"nats down"}`))
			}),
			validateFunc: func(resp *client.Response[client.ReadyStatus], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("not_ready", resp.Data.Status)
				suite.Equal("nats down", resp.Data.Error)
				suite.True(resp.Data.ServiceUnavailable)
			},
		},
		{
			name: "when 503 response body is nil returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusServiceUnavailable)
			}),
			validateFunc: func(resp *client.Response[client.ReadyStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusServiceUnavailable, target.StatusCode)
				suite.Contains(target.Message, "nil response body")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			sut := client.New(
				suite.runner(tc.handler, tc.serverURL),
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Health.Ready(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *HealthPublicTestSuite) TestStatus() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.SystemStatus], error)
	}{
		{
			name: "when checking status returns system status",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.0","uptime":"1h"}`))
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("ok", resp.Data.Status)
				suite.Equal("1.0.0", resp.Data.Version)
				suite.Equal("1h", resp.Data.Uptime)
				suite.False(resp.Data.ServiceUnavailable)
			},
		},
		{
			name:      "when client HTTP request fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "health status")
			},
		},
		{
			name: "when 200 response body is nil returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Contains(target.Message, "nil response body")
			},
		},
		{
			name: "when unexpected status returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTeapot)
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusTeapot, target.StatusCode)
				suite.Contains(target.Message, "unexpected status")
			},
		},
		{
			name: "when server returns 503 returns status with ServiceUnavailable",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"degraded","version":"1.0.0","uptime":"1h"}`))
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("degraded", resp.Data.Status)
				suite.True(resp.Data.ServiceUnavailable)
			},
		},
		{
			name: "when 503 response body is nil returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusServiceUnavailable)
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusServiceUnavailable, target.StatusCode)
				suite.Contains(target.Message, "nil response body")
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			}),
			validateFunc: func(resp *client.Response[client.SystemStatus], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			sut := client.New(
				suite.runner(tc.handler, tc.serverURL),
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Health.Status(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func TestHealthPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HealthPublicTestSuite))
}
