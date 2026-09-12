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

type DockerPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *DockerPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *DockerPublicTestSuite) TestCreate() {
	autoStart := true

	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.DockerCreateOpts
		validateFunc func(*client.Response[client.Collection[client.DockerResult]], error)
	}{
		{
			name: "when creating container returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","id":"abc123","name":"my-nginx","image":"nginx:latest","state":"running","created":"2026-01-01T00:00:00Z","changed":true}]}`,
					),
				)
			},
			opts: client.DockerCreateOpts{
				Image: "nginx:latest",
				Name:  "my-nginx",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("abc123", resp.Data.Results[0].ID)
				suite.Equal("my-nginx", resp.Data.Results[0].Name)
				suite.Equal("nginx:latest", resp.Data.Results[0].Image)
				suite.Equal("running", resp.Data.Results[0].State)
				suite.Equal("2026-01-01T00:00:00Z", resp.Data.Results[0].Created)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when creating container with all optional fields returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"web-01","id":"def456","name":"my-app","image":"myapp:v1","state":"running","created":"2026-01-01T00:00:00Z","changed":true}]}`,
					),
				)
			},
			opts: client.DockerCreateOpts{
				Image:     "myapp:v1",
				Name:      "my-app",
				Command:   []string{"serve", "--port", "8080"},
				Env:       []string{"FOO=bar", "BAZ=qux"},
				Ports:     []string{"8080:80", "443:443"},
				Volumes:   []string{"/host/data:/data", "/host/config:/config"},
				AutoStart: &autoStart,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("def456", resp.Data.Results[0].ID)
				suite.Equal("my-app", resp.Data.Results[0].Name)
			},
		},
		{
			name: "when creating container with hostname and dns returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000003","results":[{"hostname":"web-01","id":"ghi789","name":"my-dns","image":"nginx:latest","state":"running","created":"2026-01-01T00:00:00Z","changed":true}]}`,
					),
				)
			},
			opts: client.DockerCreateOpts{
				Image:    "nginx:latest",
				Name:     "my-dns",
				Hostname: "web-01",
				DNS:      []string{"8.8.8.8", "8.8.4.4"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000003", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("ghi789", resp.Data.Results[0].ID)
				suite.Equal("my-dns", resp.Data.Results[0].Name)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			opts: client.DockerCreateOpts{
				Image: "nginx:latest",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			opts: client.DockerCreateOpts{
				Image: "nginx:latest",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker create")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			opts: client.DockerCreateOpts{
				Image: "nginx:latest",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Create(
				suite.ctx,
				"_any",
				tc.opts,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		params       *client.DockerListParams
		validateFunc func(*client.Response[client.Collection[client.DockerListResult]], error)
	}{
		{
			name: "when listing containers returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","containers":[{"id":"abc123","name":"my-nginx","image":"nginx:latest","state":"running","created":"2026-01-01T00:00:00Z"}]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerListResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Len(resp.Data.Results[0].Containers, 1)
				suite.Equal("abc123", resp.Data.Results[0].Containers[0].ID)
				suite.Equal("my-nginx", resp.Data.Results[0].Containers[0].Name)
			},
		},
		{
			name: "when listing containers with state and limit params returns results",
			handler: func(w http.ResponseWriter, r *http.Request) {
				suite.Equal("running", r.URL.Query().Get("state"))
				suite.Equal("5", r.URL.Query().Get("limit"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"web-01","containers":[{"id":"abc123","name":"my-nginx","image":"nginx:latest","state":"running","created":"2026-01-01T00:00:00Z"}]}]}`,
					),
				)
			},
			params: &client.DockerListParams{
				State: "running",
				Limit: 5,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerListResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerListResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerListResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker list")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerListResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.List(suite.ctx, "_any", tc.params)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestInspect() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.DockerDetailResult]], error)
	}{
		{
			name: "when inspecting container returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","id":"abc123","name":"my-nginx","image":"nginx:latest","state":"running","created":"2026-01-01T00:00:00Z","ports":["80/tcp"],"mounts":["/data:/data"],"env":["FOO=bar"],"network_settings":{"ip":"172.17.0.2"},"health":"healthy"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerDetailResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)

				r := resp.Data.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("abc123", r.ID)
				suite.Equal("my-nginx", r.Name)
				suite.Equal("nginx:latest", r.Image)
				suite.Equal("running", r.State)
				suite.Equal("2026-01-01T00:00:00Z", r.Created)
				suite.Equal([]string{"80/tcp"}, r.Ports)
				suite.Equal([]string{"/data:/data"}, r.Mounts)
				suite.Equal([]string{"FOO=bar"}, r.Env)
				suite.Equal(map[string]string{"ip": "172.17.0.2"}, r.NetworkSettings)
				suite.Equal("healthy", r.Health)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"container not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerDetailResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerDetailResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerDetailResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker inspect")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerDetailResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusOK, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Inspect(suite.ctx, "_any", "abc123")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestStart() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.DockerActionResult]], error)
	}{
		{
			name: "when starting container returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","id":"abc123","message":"container started","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("abc123", resp.Data.Results[0].ID)
				suite.Equal("container started", resp.Data.Results[0].Message)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"container not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker start")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Start(suite.ctx, "_any", "abc123")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestStop() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.DockerStopOpts
		validateFunc func(*client.Response[client.Collection[client.DockerActionResult]], error)
	}{
		{
			name: "when stopping container returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","id":"abc123","message":"container stopped","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("abc123", resp.Data.Results[0].ID)
				suite.Equal("container stopped", resp.Data.Results[0].Message)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when stopping container with timeout returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"web-01","id":"abc123","message":"container stopped","changed":true}]}`,
					),
				)
			},
			opts: client.DockerStopOpts{
				Timeout: 30,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("abc123", resp.Data.Results[0].ID)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"container not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker stop")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Stop(
				suite.ctx,
				"_any",
				"abc123",
				tc.opts,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestRemove() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		params       *client.DockerRemoveParams
		validateFunc func(*client.Response[client.Collection[client.DockerActionResult]], error)
	}{
		{
			name: "when removing container returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","id":"abc123","message":"container removed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("abc123", resp.Data.Results[0].ID)
				suite.Equal("container removed", resp.Data.Results[0].Message)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when removing container with force returns result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				suite.Equal("true", r.URL.Query().Get("force"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"web-01","id":"abc123","message":"container removed","changed":true}]}`,
					),
				)
			},
			params: &client.DockerRemoveParams{
				Force: true,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("abc123", resp.Data.Results[0].ID)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"container not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker remove")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Remove(suite.ctx, "_any", "abc123", tc.params)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestExec() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.DockerExecOpts
		validateFunc func(*client.Response[client.Collection[client.DockerExecResult]], error)
	}{
		{
			name: "when executing command returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","stdout":"hello\n","stderr":"","exit_code":0,"changed":true}]}`,
					),
				)
			},
			opts: client.DockerExecOpts{
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerExecResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("hello\n", resp.Data.Results[0].Stdout)
				suite.Empty(resp.Data.Results[0].Stderr)
				suite.Equal(0, resp.Data.Results[0].ExitCode)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when executing command with env and working dir returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"web-01","stdout":"bar\n","stderr":"","exit_code":0,"changed":true}]}`,
					),
				)
			},
			opts: client.DockerExecOpts{
				Command:    []string{"printenv", "FOO"},
				Env:        []string{"FOO=bar", "BAZ=qux"},
				WorkingDir: "/app",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerExecResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("bar\n", resp.Data.Results[0].Stdout)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"container not found"}`))
			},
			opts: client.DockerExecOpts{
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerExecResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusNotFound, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			opts: client.DockerExecOpts{
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerExecResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			opts: client.DockerExecOpts{
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerExecResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker exec")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			opts: client.DockerExecOpts{
				Command: []string{"echo", "hello"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerExecResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Exec(
				suite.ctx,
				"_any",
				"abc123",
				tc.opts,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestPull() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.DockerPullResult]], error)
	}{
		{
			name: "when pulling image returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","image_id":"sha256:abc123","tag":"latest","size":52428800,"changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerPullResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("sha256:abc123", resp.Data.Results[0].ImageID)
				suite.Equal("latest", resp.Data.Results[0].Tag)
				suite.Equal(int64(52428800), resp.Data.Results[0].Size)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerPullResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerPullResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker pull")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerPullResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.Pull(
				suite.ctx,
				"_any",
				client.DockerPullOpts{
					Image: "nginx:latest",
				},
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *DockerPublicTestSuite) TestImageRemove() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		params       *client.DockerImageRemoveParams
		validateFunc func(*client.Response[client.Collection[client.DockerActionResult]], error)
	}{
		{
			name: "when removing image returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"web-01","id":"nginx:latest","message":"image removed","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.Equal("nginx:latest", resp.Data.Results[0].ID)
				suite.Equal("image removed", resp.Data.Results[0].Message)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when removing image with force returns result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				suite.Equal("true", r.URL.Query().Get("force"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"web-01","id":"nginx:latest","message":"image removed","changed":true}]}`,
					),
				)
			},
			params: &client.DockerImageRemoveParams{
				Force: true,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("nginx:latest", resp.Data.Results[0].ID)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "docker image remove")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.DockerActionResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var (
				serverURL string
				cleanup   func()
			)

			if tc.serverURL != "" {
				serverURL = tc.serverURL
				cleanup = func() {}
			} else {
				server := httptest.NewServer(tc.handler)
				serverURL = server.URL
				cleanup = server.Close
			}
			defer cleanup()

			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.Docker.ImageRemove(
				suite.ctx,
				"_any",
				"nginx:latest",
				tc.params,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func TestDockerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DockerPublicTestSuite))
}
