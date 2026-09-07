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

type FileDeployPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *FileDeployPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *FileDeployPublicTestSuite) TestDeploy() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		req          client.FileDeployOpts
		validateFunc func(*client.Response[client.Collection[client.FileDeployResult]], error)
	}{
		{
			name: "when deploying file returns result",
			req: client.FileDeployOpts{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
				Target:      "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"550e8400-e29b-41d4-a716-446655440000","results":[{"hostname":"web-01","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileDeployResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", resp.Data.JobID)
				suite.Require().Len(resp.Data.Results, 1)
				suite.Equal("web-01", resp.Data.Results[0].Hostname)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when all options provided returns results",
			req: client.FileDeployOpts{
				ObjectName:  "app.conf.tmpl",
				Path:        "/etc/app/app.conf",
				ContentType: "template",
				Mode:        "0644",
				Owner:       "root",
				Group:       "root",
				Vars:        map[string]any{"port": 8080},
				Target:      "web-01",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"550e8400-e29b-41d4-a716-446655440001","results":[{"hostname":"web-01","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileDeployResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			req: client.FileDeployOpts{
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"object_name is required"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileDeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			req: client.FileDeployOpts{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
				Target:      "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileDeployResult]], err error) {
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
			req: client.FileDeployOpts{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
				Target:      "_any",
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileDeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "file deploy")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			req: client.FileDeployOpts{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
				Target:      "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileDeployResult]], err error) {
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

			resp, err := sut.FileDeploy.Deploy(suite.ctx, tc.req)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FileDeployPublicTestSuite) TestStatus() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		target       string
		path         string
		validateFunc func(*client.Response[client.Collection[client.FileStatusResult]], error)
	}{
		{
			name:   "when checking file status returns result",
			target: "_any",
			path:   "/etc/nginx/nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"550e8400-e29b-41d4-a716-446655440000","results":[{"hostname":"web-01","path":"/etc/nginx/nginx.conf","status":"in-sync","sha256":"abc123"}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileStatusResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", resp.Data.JobID)
				suite.Require().Len(resp.Data.Results, 1)
				r := resp.Data.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("/etc/nginx/nginx.conf", r.Path)
				suite.Equal("in-sync", r.Status)
				suite.Equal("abc123", r.SHA256)
			},
		},
		{
			name:   "when server returns 400 returns ValidationError",
			target: "_any",
			path:   "",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"path is required"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileStatusResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name:   "when server returns 403 returns AuthError",
			target: "_any",
			path:   "/etc/nginx/nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileStatusResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			target:    "_any",
			path:      "/etc/nginx/nginx.conf",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.Collection[client.FileStatusResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "file status")
			},
		},
		{
			name:   "when server returns 200 with no JSON body returns UnexpectedStatusError",
			target: "_any",
			path:   "/etc/nginx/nginx.conf",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileStatusResult]], err error) {
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

			resp, err := sut.FileDeploy.Status(suite.ctx, tc.target, tc.path)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *FileDeployPublicTestSuite) TestUndeploy() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		req          client.FileUndeployOpts
		validateFunc func(*client.Response[client.Collection[client.FileUndeployResult]], error)
	}{
		{
			name: "when undeploy succeeds",
			req: client.FileUndeployOpts{
				Path:   "/etc/cron.d/backup",
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"550e8400-e29b-41d4-a716-446655440000","results":[{"hostname":"web-01","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileUndeployResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", resp.Data.JobID)
				suite.Require().Len(resp.Data.Results, 1)
				r := resp.Data.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.True(r.Changed)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			req: client.FileUndeployOpts{
				Path:   "/etc/cron.d/backup",
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileUndeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			req: client.FileUndeployOpts{
				Path:   "/etc/cron.d/backup",
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileUndeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusForbidden, target.StatusCode)
			},
		},
		{
			name: "when server returns 500 returns error",
			req: client.FileUndeployOpts{
				Path:   "/etc/cron.d/backup",
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal server error"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileUndeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			req: client.FileUndeployOpts{
				Path:   "/etc/cron.d/backup",
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileUndeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusAccepted, target.StatusCode)
				suite.Equal("nil response body", target.Message)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			req: client.FileUndeployOpts{
				Path:   "/etc/cron.d/backup",
				Target: "_any",
			},
			validateFunc: func(resp *client.Response[client.Collection[client.FileUndeployResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "file undeploy")
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

			resp, err := sut.FileDeploy.Undeploy(suite.ctx, tc.req)
			tc.validateFunc(resp, err)
		})
	}
}

func TestFileDeployPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileDeployPublicTestSuite))
}
