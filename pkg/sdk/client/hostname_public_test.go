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

type HostnamePublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *HostnamePublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *HostnamePublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		target       string
		validateFunc func(*client.Response[client.Collection[client.HostnameResult]], error)
	}{
		{
			name:   "when requesting hostname returns results",
			target: "_any",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"test-host"}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("test-host", resp.Data.Results[0].Hostname)
			},
		},
		{
			name:   "when server returns 403 returns AuthError",
			target: "_any",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameResult]], err error) {
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
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "get hostname")
			},
		},
		{
			name:   "when server returns 200 with no JSON body returns UnexpectedStatusError",
			target: "_any",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameResult]], err error) {
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

			resp, err := sut.Hostname.Get(suite.ctx, tc.target)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *HostnamePublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		target       string
		hostname     string
		validateFunc func(*client.Response[client.Collection[client.HostnameUpdateResult]], error)
	}{
		{
			name:     "when updating hostname returns results",
			target:   "_any",
			hostname: "new-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"new-host","status":"ok","changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameUpdateResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("new-host", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name:     "when server returns 403 returns AuthError",
			target:   "_any",
			hostname: "new-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameUpdateResult]], err error) {
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
			hostname:  "new-host",
			serverURL: "http://127.0.0.1:0",
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameUpdateResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "set hostname")
			},
		},
		{
			name:     "when server returns 202 with no JSON body returns UnexpectedStatusError",
			target:   "_any",
			hostname: "new-host",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.HostnameUpdateResult]], err error) {
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

			resp, err := sut.Hostname.Update(suite.ctx, tc.target, tc.hostname)
			tc.validateFunc(resp, err)
		})
	}
}

func TestHostnamePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnamePublicTestSuite))
}
