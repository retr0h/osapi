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

type AuditPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *AuditPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *AuditPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		limit        int
		offset       int
		validateFunc func(*client.Response[client.AuditList], error)
	}{
		{
			name:   "when listing audit entries returns audit list",
			limit:  20,
			offset: 0,
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"items":[],"total_items":0}`))
			}),
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal(0, resp.Data.TotalItems)
				suite.Empty(resp.Data.Items)
			},
		},
		{
			name:   "when server returns 401 returns AuthError",
			limit:  20,
			offset: 0,
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			}),
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP request fails returns error",
			limit:     20,
			offset:    0,
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "list audit logs:")
			},
		},
		{
			name:   "when response body is nil returns UnexpectedStatusError",
			limit:  20,
			offset: 0,
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}),
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
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
			serverURL := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				serverURL = server.URL
			}

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Audit.List(suite.ctx, tc.limit, tc.offset)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AuditPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		id           string
		validateFunc func(*client.Response[client.AuditEntry], error)
	}{
		{
			name: "when valid UUID returns audit entry",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"entry":{"id":"550e8400-e29b-41d4-a716-446655440000","timestamp":"2026-01-01T00:00:00Z","user":"admin","roles":["admin"],"method":"GET","path":"/api/v1/health","response_code":200,"duration_ms":5,"source_ip":"127.0.0.1"}}`,
					),
				)
			}),
			validateFunc: func(resp *client.Response[client.AuditEntry], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", resp.Data.ID)
				suite.Equal("admin", resp.Data.User)
				suite.Equal("GET", resp.Data.Method)
				suite.Equal("/api/v1/health", resp.Data.Path)
			},
		},
		{
			name: "when invalid UUID returns error",
			id:   "not-a-uuid",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"entry":{"id":"550e8400-e29b-41d4-a716-446655440000","timestamp":"2026-01-01T00:00:00Z","user":"admin","roles":["admin"],"method":"GET","path":"/api/v1/health","response_code":200,"duration_ms":5,"source_ip":"127.0.0.1"}}`,
					),
				)
			}),
			validateFunc: func(resp *client.Response[client.AuditEntry], err error) {
				suite.Error(err)
				suite.Nil(resp)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"audit entry not found"}`))
			}),
			validateFunc: func(resp *client.Response[client.AuditEntry], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP request fails returns error",
			id:        "550e8400-e29b-41d4-a716-446655440000",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.AuditEntry], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "get audit log")
			},
		},
		{
			name: "when response body is nil returns UnexpectedStatusError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}),
			validateFunc: func(resp *client.Response[client.AuditEntry], err error) {
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
			serverURL := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				serverURL = server.URL
			}

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Audit.Get(suite.ctx, tc.id)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AuditPublicTestSuite) TestExport() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.AuditList], error)
	}{
		{
			name: "when exporting audit entries returns audit list",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"items":[],"total_items":0}`))
			}),
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal(0, resp.Data.TotalItems)
				suite.Empty(resp.Data.Items)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			}),
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP request fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "export audit logs:")
			},
		},
		{
			name: "when response body is nil returns UnexpectedStatusError",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			}),
			validateFunc: func(resp *client.Response[client.AuditList], err error) {
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
			serverURL := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				serverURL = server.URL
			}

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Audit.Export(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func TestAuditPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuditPublicTestSuite))
}
