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

type ServicePublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *ServicePublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *ServicePublicTestSuite) TestServiceList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceInfoResult]], error)
	}{
		{
			name: "when listing services returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","services":[{"name":"nginx.service","status":"active","enabled":true,"description":"web server","pid":1234}]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.Require().Len(resp.Data.Results[0].Services, 1)
				suite.Equal("nginx.service", resp.Data.Results[0].Services[0].Name)
				suite.Equal("active", resp.Data.Results[0].Services[0].Status)
				suite.True(resp.Data.Results[0].Services[0].Enabled)
			},
		},
		{
			name: "when broadcast list returns multiple results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","status":"ok","services":[]},{"hostname":"server2","status":"ok","services":[]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
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
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
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
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service list")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceInfoResult]],
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

			resp, err := sut.Service.List(suite.ctx, "_any")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceGetResult]], error)
	}{
		{
			name: "when getting service returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","service":{"name":"nginx.service","status":"active","enabled":true,"pid":1234}}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceGetResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Require().NotNil(resp.Data.Results[0].Service)
				suite.Equal("nginx.service", resp.Data.Results[0].Service.Name)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"service not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceGetResult]],
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
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceGetResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceGetResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceGetResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceGetResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service get")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceGetResult]],
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

			resp, err := sut.Service.Get(suite.ctx, "_any", "nginx.service")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceCreate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.ServiceCreateOpts
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when creating service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"myapp.service","changed":true}]}`,
					),
				)
			},
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("myapp.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when broadcast create returns multiple results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","status":"ok","name":"myapp.service","changed":true},{"hostname":"server2","status":"ok","name":"myapp.service","changed":true}]}`,
					),
				)
			},
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid name"}`))
			},
			opts: client.ServiceCreateOpts{
				Name:   "",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service create")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.ServiceCreateOpts{
				Name:   "myapp.service",
				Object: "myapp-unit",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Create(suite.ctx, "_any", tc.opts)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceUpdate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.ServiceUpdateOpts
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when updating service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"myapp.service","changed":true}]}`,
					),
				)
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Equal("myapp.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when broadcast update returns multiple results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","status":"ok","name":"myapp.service","changed":true},{"hostname":"server2","status":"ok","name":"myapp.service","changed":true}]}`,
					),
				)
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid object"}`))
			},
			opts: client.ServiceUpdateOpts{
				Object: "",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"service not found"}`))
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
			},
		},
		{
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service update")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.ServiceUpdateOpts{
				Object: "myapp-unit-v2",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Update(
				suite.ctx,
				"_any",
				"myapp.service",
				tc.opts,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceDelete() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when deleting service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"myapp.service","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Equal("myapp.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when broadcast delete returns multiple results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","status":"ok","name":"myapp.service","changed":true},{"hostname":"server2","status":"failed","error":"not found"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service delete")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Delete(suite.ctx, "_any", "myapp.service")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceStart() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when starting service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"nginx.service","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service start")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Start(suite.ctx, "_any", "nginx.service")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceStop() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when stopping service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"nginx.service","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service stop")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Stop(suite.ctx, "_any", "nginx.service")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceRestart() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when restarting service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"nginx.service","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service restart")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Restart(suite.ctx, "_any", "nginx.service")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceEnable() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when enabling service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"nginx.service","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service enable")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Enable(suite.ctx, "_any", "nginx.service")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *ServicePublicTestSuite) TestServiceDisable() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.ServiceMutationResult]], error)
	}{
		{
			name: "when disabling service returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"nginx.service","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("nginx.service", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 401 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusUnauthorized, target.StatusCode)
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
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusInternalServerError, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "service disable")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.ServiceMutationResult]],
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

			resp, err := sut.Service.Disable(suite.ctx, "_any", "nginx.service")
			tc.validateFunc(resp, err)
		})
	}
}

func TestServicePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ServicePublicTestSuite))
}
