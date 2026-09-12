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

type CronPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *CronPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *CronPublicTestSuite) TestCronList() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.CronEntryResult]], error)
	}{
		{
			name: "when listing cron entries returns results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","name":"backup","schedule":"0 2 * * *","user":"root","object":"/usr/bin/backup.sh"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("backup", resp.Data.Results[0].Name)
				suite.Equal("0 2 * * *", resp.Data.Results[0].Schedule)
				suite.Equal("root", resp.Data.Results[0].User)
				suite.Equal("/usr/bin/backup.sh", resp.Data.Results[0].Object)
			},
		},
		{
			name: "when response has interval-based entries",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"name":"logrotate","interval":"daily","source":"daily","object":"/usr/sbin/logrotate"},{"name":"backup","schedule":"0 2 * * *","source":"cron.d","user":"root","object":"/usr/bin/backup.sh"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)

				suite.Equal("logrotate", resp.Data.Results[0].Name)
				suite.Equal("daily", resp.Data.Results[0].Interval)
				suite.Equal("daily", resp.Data.Results[0].Source)
				suite.Empty(resp.Data.Results[0].Schedule)

				suite.Equal("backup", resp.Data.Results[1].Name)
				suite.Equal("0 2 * * *", resp.Data.Results[1].Schedule)
				suite.Equal("cron.d", resp.Data.Results[1].Source)
				suite.Empty(resp.Data.Results[1].Interval)
			},
		},
		{
			name: "when CronList server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
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
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "cron list")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
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

			resp, err := sut.Cron.List(suite.ctx, "_any")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *CronPublicTestSuite) TestCronGet() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.CronEntryResult]], error)
	}{
		{
			name: "when getting cron entry returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","name":"backup","schedule":"0 2 * * *","user":"root","object":"/usr/bin/backup.sh"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("backup", resp.Data.Results[0].Name)
				suite.Equal("0 2 * * *", resp.Data.Results[0].Schedule)
				suite.Equal("root", resp.Data.Results[0].User)
				suite.Equal("/usr/bin/backup.sh", resp.Data.Results[0].Object)
			},
		},
		{
			name: "when getting interval-based cron entry returns interval field",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","name":"logrotate","interval":"daily","source":"daily","object":"logrotate-script"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("logrotate", resp.Data.Results[0].Name)
				suite.Equal("daily", resp.Data.Results[0].Interval)
				suite.Equal("daily", resp.Data.Results[0].Source)
			},
		},
		{
			name: "when getting cron entry with broadcast returns multiple results",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","name":"backup","schedule":"0 2 * * *"},{"hostname":"server2","name":"backup","schedule":"0 2 * * *"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"cron entry not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
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
				resp *client.Response[client.Collection[client.CronEntryResult]],
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
				resp *client.Response[client.Collection[client.CronEntryResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "cron get")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronEntryResult]],
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

			resp, err := sut.Cron.Get(suite.ctx, "_any", "backup")
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *CronPublicTestSuite) TestCronCreate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.CronCreateOpts
		validateFunc func(*client.Response[client.Collection[client.CronMutationResult]], error)
	}{
		{
			name: "when creating cron entry returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","name":"backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronCreateOpts{
				Name:     "backup",
				Schedule: "0 2 * * *",
				Object:   "/usr/bin/backup.sh",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("agent1", resp.Data.Results[0].Hostname)
				suite.Equal("backup", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when creating cron entry with user returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"agent1","name":"cleanup","changed":true}]}`,
					),
				)
			},
			opts: client.CronCreateOpts{
				Name:     "cleanup",
				Schedule: "*/5 * * * *",
				Object:   "/usr/bin/cleanup.sh",
				User:     "www-data",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.Equal("cleanup", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when creating cron entry with interval returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000003","results":[{"hostname":"agent1","name":"daily-backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronCreateOpts{
				Name:     "daily-backup",
				Interval: "daily",
				Object:   "/usr/bin/backup.sh",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000003", resp.Data.JobID)
				suite.Equal("daily-backup", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when creating cron entry with ContentType and Vars returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000004","results":[{"hostname":"agent1","name":"template-job","changed":true}]}`,
					),
				)
			},
			opts: client.CronCreateOpts{
				Name:        "template-job",
				Schedule:    "0 1 * * *",
				Object:      "/usr/bin/template-job.sh",
				ContentType: "template",
				Vars:        map[string]any{"env": "prod"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000004", resp.Data.JobID)
				suite.Equal("template-job", resp.Data.Results[0].Name)
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
						`{"job_id":"00000000-0000-0000-0000-000000000005","results":[{"hostname":"server1","name":"backup","changed":true},{"hostname":"server2","name":"backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronCreateOpts{
				Name:     "backup",
				Schedule: "0 2 * * *",
				Object:   "/usr/bin/backup.sh",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
				_, _ = w.Write([]byte(`{"error":"invalid schedule"}`))
			},
			opts: client.CronCreateOpts{
				Name:     "bad",
				Schedule: "invalid",
				Object:   "echo",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			opts: client.CronCreateOpts{
				Name:     "backup",
				Schedule: "0 2 * * *",
				Object:   "/usr/bin/backup.sh",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
			opts: client.CronCreateOpts{
				Name:     "backup",
				Schedule: "0 2 * * *",
				Object:   "/usr/bin/backup.sh",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "cron create")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.CronCreateOpts{
				Name:     "backup",
				Schedule: "0 2 * * *",
				Object:   "/usr/bin/backup.sh",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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

			resp, err := sut.Cron.Create(suite.ctx, "_any", tc.opts)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *CronPublicTestSuite) TestCronUpdate() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		opts         client.CronUpdateOpts
		validateFunc func(*client.Response[client.Collection[client.CronMutationResult]], error)
	}{
		{
			name: "when updating cron entry returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","name":"backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronUpdateOpts{
				Schedule: "0 3 * * *",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Equal("backup", resp.Data.Results[0].Name)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when updating cron entry with all fields returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"agent1","name":"backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronUpdateOpts{
				Schedule: "0 4 * * *",
				Object:   "/usr/bin/new-backup.sh",
				User:     "admin",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000002", resp.Data.JobID)
				suite.True(resp.Data.Results[0].Changed)
			},
		},
		{
			name: "when updating cron entry with Object ContentType and Vars returns result",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000003","results":[{"hostname":"agent1","name":"backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronUpdateOpts{
				Object:      "/usr/bin/new-template.sh",
				ContentType: "template",
				Vars:        map[string]any{"region": "us-east"},
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000003", resp.Data.JobID)
				suite.Equal("backup", resp.Data.Results[0].Name)
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
						`{"job_id":"00000000-0000-0000-0000-000000000004","results":[{"hostname":"server1","name":"backup","changed":true},{"hostname":"server2","name":"backup","changed":true}]}`,
					),
				)
			},
			opts: client.CronUpdateOpts{
				Schedule: "0 3 * * *",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"cron entry not found"}`))
			},
			opts: client.CronUpdateOpts{
				Schedule: "0 3 * * *",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
			name: "when server returns 400 returns ValidationError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid schedule"}`))
			},
			opts: client.CronUpdateOpts{
				Schedule: "bad",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
			name: "when server returns 403 returns AuthError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
			},
			opts: client.CronUpdateOpts{
				Schedule: "0 3 * * *",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
			opts: client.CronUpdateOpts{
				Schedule: "0 3 * * *",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "cron update")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			opts: client.CronUpdateOpts{
				Schedule: "0 3 * * *",
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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

			resp, err := sut.Cron.Update(
				suite.ctx,
				"_any",
				"backup",
				tc.opts,
			)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *CronPublicTestSuite) TestCronDelete() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		validateFunc func(*client.Response[client.Collection[client.CronMutationResult]], error)
	}{
		{
			name: "when deleting cron entry returns result collection",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(
					[]byte(
						`{"job_id":"00000000-0000-0000-0000-000000000001","results":[{"hostname":"agent1","name":"backup","changed":true}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Equal("00000000-0000-0000-0000-000000000001", resp.Data.JobID)
				suite.Equal("backup", resp.Data.Results[0].Name)
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
						`{"job_id":"00000000-0000-0000-0000-000000000002","results":[{"hostname":"server1","name":"backup","changed":true},{"hostname":"server2","error":"not found"}]}`,
					),
				)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 2)
			},
		},
		{
			name: "when server returns 404 returns NotFoundError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"cron entry not found"}`))
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
				resp *client.Response[client.Collection[client.CronMutationResult]],
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
				resp *client.Response[client.Collection[client.CronMutationResult]],
				err error,
			) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "cron delete")
			},
		},
		{
			name: "when server returns 200 with no JSON body returns UnexpectedStatusError",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			validateFunc: func(
				resp *client.Response[client.Collection[client.CronMutationResult]],
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

			resp, err := sut.Cron.Delete(suite.ctx, "_any", "backup")
			tc.validateFunc(resp, err)
		})
	}
}

func TestCronPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CronPublicTestSuite))
}
