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

type UserPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *UserPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *UserPublicTestSuite) TestUserList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.UserInfoResult]], error)
	}{
		{
			name: "when listing users returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","users":[{"name":"root","uid":0,"gid":0,"home":"/root","shell":"/bin/bash"}]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserInfoResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.Require().Len(resp.Data.Results[0].Users, 1)
				suite.Equal("root", resp.Data.Results[0].Users[0].Name)
				suite.Equal(0, resp.Data.Results[0].Users[0].UID)
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
				resp *client.Response[client.Collection[client.UserInfoResult]],
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
				resp *client.Response[client.Collection[client.UserInfoResult]],
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
				resp *client.Response[client.Collection[client.UserInfoResult]],
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
				resp *client.Response[client.Collection[client.UserInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "user list")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserInfoResult]],
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

			resp, err := sut.User.List(suite.ctx, "_any")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *UserPublicTestSuite) TestUserGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.UserInfoResult]], error)
	}{
		{
			name: "when getting user returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","users":[{"name":"testuser","uid":1000,"gid":1000,"home":"/home/testuser","shell":"/bin/bash","groups":["sudo","docker"]}]}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserInfoResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Require().Len(resp.Data.Results[0].Users, 1)
				suite.Equal("testuser", resp.Data.Results[0].Users[0].Name)
				suite.Equal([]string{"sudo", "docker"}, resp.Data.Results[0].Users[0].Groups)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"user not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserInfoResult]],
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
				resp *client.Response[client.Collection[client.UserInfoResult]],
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
				resp *client.Response[client.Collection[client.UserInfoResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "user get")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserInfoResult]],
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

			resp, err := sut.User.Get(suite.ctx, "_any", "testuser")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *UserPublicTestSuite) TestUserCreate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.UserCreateOpts
		validateFunc func(*client.Response[client.Collection[client.UserMutationResult]], error)
	}{
		{
			name: "when creating user returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"newuser","changed":true}]}`,
					),
				)
			},
			opts: client.UserCreateOpts{
				Name:     "newuser",
				UID:      1001,
				GID:      1001,
				Home:     "/home/newuser",
				Shell:    "/bin/bash",
				Groups:   []string{"sudo"},
				Password: "secret",
				System:   true,
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("newuser", resp.Data.Results[0].Name)
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
			opts: client.UserCreateOpts{Name: ""},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
			opts: client.UserCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
			opts: client.UserCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
			opts:      client.UserCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "user create")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.UserCreateOpts{Name: "test"},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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

			resp, err := sut.User.Create(suite.ctx, "_any", tc.opts)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *UserPublicTestSuite) TestUserUpdate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.UserMutationResult]], error)
	}{
		{
			name: "when updating user returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"testuser","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				_, _ = w.Write([]byte(`{"error":"user not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				resp *client.Response[client.Collection[client.UserMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "user update")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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

			lockVal := true
			sut := client.New(
				serverURL,
				"test-token",
				client.WithLogger(slog.Default()),
			)

			resp, err := sut.User.Update(suite.ctx, "_any", "testuser", client.UserUpdateOpts{
				Shell:  "/bin/zsh",
				Home:   "/home/testuser",
				Groups: []string{"sudo"},
				Lock:   &lockVal,
			})
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *UserPublicTestSuite) TestUserDelete() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.UserMutationResult]], error)
	}{
		{
			name: "when deleting user returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"testuser","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				_, _ = w.Write([]byte(`{"error":"user not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				resp *client.Response[client.Collection[client.UserMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "user delete")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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

			resp, err := sut.User.Delete(suite.ctx, "_any", "testuser")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *UserPublicTestSuite) TestUserChangePassword() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.UserMutationResult]], error)
	}{
		{
			name: "when changing password returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","name":"testuser","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				_, _ = w.Write([]byte(`{"error":"password too short"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				_, _ = w.Write([]byte(`{"error":"user not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				resp *client.Response[client.Collection[client.UserMutationResult]],
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
				resp *client.Response[client.Collection[client.UserMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "user change password")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.UserMutationResult]],
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

			resp, err := sut.User.ChangePassword(suite.ctx, "_any", "testuser", "newpassword")
			tc.validateFunc(resp, err)
		})
	}
}

func TestUserPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(UserPublicTestSuite))
}
