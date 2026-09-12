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

type AgentPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *AgentPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *AgentPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.AgentList], error)
	}{
		{
			name: "when requesting agents returns no error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"agents":[],"total":0}`))
			},
			validateFunc: func(resp *client.Response[client.AgentList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal(0, resp.Data.Total)
				suite.Empty(resp.Data.Agents)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(resp *client.Response[client.AgentList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.AgentList], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "list agents")
			},
		},
		{
			name: "when response JSON200 is nil returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.AgentList], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.List(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AgentPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		hostname     string
		validateFunc func(*client.Response[client.Agent], error)
	}{
		{
			name:     "when requesting agent details returns no error",
			hostname: "server1",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"hostname":"server1","status":"Ready"}`))
			},
			validateFunc: func(resp *client.Response[client.Agent], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("server1", resp.Data.Hostname)
				suite.Equal("Ready", resp.Data.Status)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			hostname: "unknown-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"agent not found"}`))
			},
			validateFunc: func(resp *client.Response[client.Agent], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
				suite.Equal("agent not found", target.Message)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			hostname:  "unknown-host",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.Agent], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "get agent")
			},
		},
		{
			name:     "when response JSON200 is nil returns UnexpectedStatusError",
			hostname: "unknown-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.Agent], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.Get(suite.ctx, tc.hostname)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AgentPublicTestSuite) TestDrain() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		hostname     string
		validateFunc func(*client.Response[client.MessageResponse], error)
	}{
		{
			name:     "when draining agent returns success",
			hostname: "server1",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message":"drain initiated for agent server1"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("drain initiated for agent server1", resp.Data.Message)
			},
		},
		{
			name:     "when server returns 409 returns ConflictError",
			hostname: "test-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"agent already draining"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ConflictError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusConflict, target.StatusCode)
				suite.Equal("agent already draining", target.Message)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			hostname: "test-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"agent not found"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			hostname:  "test-host",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "drain agent")
			},
		},
		{
			name:     "when response JSON200 is nil returns UnexpectedStatusError",
			hostname: "test-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.Drain(suite.ctx, tc.hostname)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AgentPublicTestSuite) TestUndrain() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		hostname     string
		validateFunc func(*client.Response[client.MessageResponse], error)
	}{
		{
			name:     "when undraining agent returns success",
			hostname: "server1",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message":"undrain initiated for agent server1"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("undrain initiated for agent server1", resp.Data.Message)
			},
		},
		{
			name:     "when server returns 409 returns ConflictError",
			hostname: "test-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"agent not in draining state"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ConflictError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusConflict, target.StatusCode)
				suite.Equal("agent not in draining state", target.Message)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			hostname: "test-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"agent not found"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			hostname:  "test-host",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "undrain agent")
			},
		},
		{
			name:     "when response JSON200 is nil returns UnexpectedStatusError",
			hostname: "test-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.Undrain(suite.ctx, tc.hostname)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AgentPublicTestSuite) TestListPending() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.PendingAgentList], error)
	}{
		{
			name: "when requesting pending agents returns no error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"agents":[{"machine_id":"m1","hostname":"web-01","fingerprint":"SHA256:abc","requested_at":"2026-01-01T00:00:00Z"}],"total":1}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.PendingAgentList], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal(1, resp.Data.Total)
				suite.Require().Len(resp.Data.Agents, 1)
				suite.Equal("m1", resp.Data.Agents[0].MachineID)
				suite.Equal("web-01", resp.Data.Agents[0].Hostname)
				suite.Equal("SHA256:abc", resp.Data.Agents[0].Fingerprint)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(resp *client.Response[client.PendingAgentList], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.PendingAgentList], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "list pending agents")
			},
		},
		{
			name: "when response JSON200 is nil returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.PendingAgentList], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.ListPending(suite.ctx)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AgentPublicTestSuite) TestAccept() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		hostname     string
		fingerprint  string
		validateFunc func(*client.Response[client.MessageResponse], error)
	}{
		{
			name:        "when accepting agent returns success",
			hostname:    "web-01",
			fingerprint: "",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message":"agent web-01 accepted"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("agent web-01 accepted", resp.Data.Message)
			},
		},
		{
			name:        "when accepting agent with fingerprint returns success",
			hostname:    "web-01",
			fingerprint: "SHA256:abc123",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message":"agent web-01 accepted"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("agent web-01 accepted", resp.Data.Message)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			hostname: "unknown-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"agent not found"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			hostname:  "web-01",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "accept agent")
			},
		},
		{
			name:     "when response JSON200 is nil returns UnexpectedStatusError",
			hostname: "web-01",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.Accept(suite.ctx, tc.hostname, tc.fingerprint)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *AgentPublicTestSuite) TestReject() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		hostname     string
		validateFunc func(*client.Response[client.MessageResponse], error)
	}{
		{
			name:     "when rejecting agent returns success",
			hostname: "web-01",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"message":"agent web-01 rejected"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("agent web-01 rejected", resp.Data.Message)
			},
		},
		{
			name:     "when server returns 404 returns NotFoundError",
			hostname: "unknown-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"agent not found"}`))
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP error returns wrapped error",
			hostname:  "web-01",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "reject agent")
			},
		},
		{
			name:     "when response JSON200 is nil returns UnexpectedStatusError",
			hostname: "web-01",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.MessageResponse], err error) {
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
			url := tc.serverURL
			if tc.handler != nil {
				server := httptest.NewServer(tc.handler)
				defer server.Close()
				url = server.URL
			}

			sut := client.New(
				url,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Agent.Reject(suite.ctx, tc.hostname)
			tc.validateFunc(resp, err)
		})
	}
}

func TestAgentPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentPublicTestSuite))
}
