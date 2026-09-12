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

type GroupPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *GroupPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *GroupPublicTestSuite) TestGroupList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.GroupInfoResult]], error)
	}{
		{
			name: "when listing groups returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","groups":[{"name":"sudo","gid":27,"members":["testuser"]}]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupInfoResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.Require().Len(resp.Data.Results[0].Groups, 1)
				suite.Equal("sudo", resp.Data.Results[0].Groups[0].Name)
				suite.Equal(27, resp.Data.Results[0].Groups[0].GID)
				suite.Equal([]string{"testuser"}, resp.Data.Results[0].Groups[0].Members)
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
				resp *client.Response[client.Collection[client.GroupInfoResult]],
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
				resp *client.Response[client.Collection[client.GroupInfoResult]],
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
				resp *client.Response[client.Collection[client.GroupInfoResult]],
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
				resp *client.Response[client.Collection[client.GroupInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "group list")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupInfoResult]],
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

			resp, err := sut.Group.List(suite.ctx, "_any")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *GroupPublicTestSuite) TestGroupGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.GroupInfoResult]], error)
	}{
		{
			name: "when getting group returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","groups":[{"name":"docker","gid":999,"members":["user1","user2"]}]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupInfoResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Require().Len(resp.Data.Results[0].Groups, 1)
				suite.Equal("docker", resp.Data.Results[0].Groups[0].Name)
				suite.Equal([]string{"user1", "user2"}, resp.Data.Results[0].Groups[0].Members)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"group not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupInfoResult]],
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
				resp *client.Response[client.Collection[client.GroupInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "group get")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
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

			resp, err := sut.Group.Get(suite.ctx, "_any", "docker")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *GroupPublicTestSuite) TestGroupCreate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.GroupCreateOpts
		validateFunc func(*client.Response[client.Collection[client.GroupMutationResult]], error)
	}{
		{
			name: "when creating group returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"newgroup","changed":true}]}`,
					),
				)
			},
			opts: client.GroupCreateOpts{
				Name:   "newgroup",
				GID:    2000,
				System: true,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("newgroup", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"validation failed"}`))
			},
			opts: client.GroupCreateOpts{Name: ""},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
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
			opts: client.GroupCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
			},
		},
		{
			name: "when server returns 500 returns ServerError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}`))
			},
			opts: client.GroupCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ServerError
				suite.True(errors.As(err, &target))
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			opts:      client.GroupCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "group create")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.GroupCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
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

			resp, err := sut.Group.Create(suite.ctx, "_any", tc.opts)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *GroupPublicTestSuite) TestGroupUpdate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.GroupMutationResult]], error)
	}{
		{
			name: "when updating group returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"docker","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"validation failed"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"group not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "group update")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
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

			resp, err := sut.Group.Update(suite.ctx, "_any", "docker", client.GroupUpdateOpts{
				Members: []string{"user1", "user2"},
			})
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *GroupPublicTestSuite) TestGroupDelete() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.GroupMutationResult]], error)
	}{
		{
			name: "when deleting group returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"testgroup","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"group not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
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
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.AuthError
				suite.True(errors.As(err, &target))
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "group delete")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.GroupMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
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

			resp, err := sut.Group.Delete(suite.ctx, "_any", "testgroup")
			tc.validateFunc(resp, err)
		})
	}
}

func TestGroupPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GroupPublicTestSuite))
}
