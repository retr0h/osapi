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

type TimezonePublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *TimezonePublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *TimezonePublicTestSuite) TestTimezoneGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.TimezoneResult]], error)
	}{
		{
			name: "when getting timezone returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","timezone":"America/New_York","utc_offset":"-05:00"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.Equal("America/New_York", resp.Data.Results[0].Timezone)
				suite.Equal("-05:00", resp.Data.Results[0].UTCOffset)
			},
		},
		{
			name: "when broadcast get returns multiple results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","status":"ok","timezone":"UTC"},{"hostname":"server2","status":"ok","timezone":"America/Chicago"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneResult]],
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
				resp *client.Response[client.Collection[client.TimezoneResult]],
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
				resp *client.Response[client.Collection[client.TimezoneResult]],
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
				resp *client.Response[client.Collection[client.TimezoneResult]],
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
				resp *client.Response[client.Collection[client.TimezoneResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "timezone get")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneResult]],
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

			resp, err := sut.Timezone.Get(suite.ctx, "_any")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *TimezonePublicTestSuite) TestTimezoneUpdate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.TimezoneUpdateOpts
		validateFunc func(*client.Response[client.Collection[client.TimezoneMutationResult]], error)
	}{
		{
			name: "when updating timezone returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","status":"ok","timezone":"America/New_York","changed":true}]}`,
					),
				)
			},
			opts: client.TimezoneUpdateOpts{
				Timezone: "America/New_York",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("ok", resp.Data.Results[0].Status)
				suite.Equal("America/New_York", resp.Data.Results[0].Timezone)
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
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","status":"ok","changed":true},{"hostname":"server2","status":"ok","changed":false}]}`,
					),
				)
			},
			opts: client.TimezoneUpdateOpts{
				Timezone: "UTC",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
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
				_, _ = w.Write([]byte(`{"error":"invalid timezone"}`))
			},
			opts: client.TimezoneUpdateOpts{
				Timezone: "",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
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
			opts: client.TimezoneUpdateOpts{
				Timezone: "UTC",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
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
			opts: client.TimezoneUpdateOpts{
				Timezone: "UTC",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
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
			opts: client.TimezoneUpdateOpts{
				Timezone: "UTC",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
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
			opts: client.TimezoneUpdateOpts{
				Timezone: "UTC",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "timezone update")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.TimezoneUpdateOpts{
				Timezone: "UTC",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.TimezoneMutationResult]],
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

			resp, err := sut.Timezone.Update(suite.ctx, "_any", tc.opts)
			tc.validateFunc(resp, err)
		})
	}
}

func TestTimezonePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TimezonePublicTestSuite))
}
