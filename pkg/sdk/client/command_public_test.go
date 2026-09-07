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

type CommandPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (suite *CommandPublicTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *CommandPublicTestSuite) TestExec() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		req          client.ExecRequest
		validateFunc func(*client.Response[client.Collection[client.CommandResult]], error)
	}{
		{
			name: "when basic command returns results",
			req: client.ExecRequest{
				Command: "whoami",
				Target:  "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"exec-host","stdout":"root\n","exit_code":0,"changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("exec-host", resp.Data.Results[0].Hostname)
				suite.Equal("root\n", resp.Data.Results[0].Stdout)
				suite.Equal(0, resp.Data.Results[0].ExitCode)
			},
		},
		{
			name: "when all options provided returns results",
			req: client.ExecRequest{
				Command: "ls",
				Args:    []string{"-la", "/tmp"},
				Cwd:     "/tmp",
				Timeout: 10,
				Target:  "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"exec-host","stdout":"root\n","exit_code":0,"changed":true}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			req: client.ExecRequest{
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"command is required"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			req: client.ExecRequest{
				Command: "whoami",
				Target:  "_any",
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "exec command")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			req: client.ExecRequest{
				Command: "whoami",
				Target:  "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
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

			resp, err := sut.Command.Exec(suite.ctx, tc.req)
			tc.validateFunc(resp, err)
		})
	}
}

func (suite *CommandPublicTestSuite) TestShell() {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		serverURL    string
		req          client.ShellRequest
		validateFunc func(*client.Response[client.Collection[client.CommandResult]], error)
	}{
		{
			name: "when basic command returns results",
			req: client.ShellRequest{
				Command: "uname -a",
				Target:  "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"shell-host","exit_code":0,"changed":false}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
				suite.Len(resp.Data.Results, 1)
				suite.Equal("shell-host", resp.Data.Results[0].Hostname)
			},
		},
		{
			name: "when cwd and timeout provided returns results",
			req: client.ShellRequest{
				Command: "ls -la",
				Cwd:     "/var/log",
				Timeout: 15,
				Target:  "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write(
					[]byte(
						`{"results":[{"hostname":"shell-host","exit_code":0,"changed":false}]}`,
					),
				)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.NoError(err)
				suite.NotNil(resp)
			},
		},
		{
			name: "when server returns 400 returns ValidationError",
			req: client.ShellRequest{
				Target: "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"command is required"}`))
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)

				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(http.StatusBadRequest, target.StatusCode)
			},
		},
		{
			name:      "when client HTTP call fails returns error",
			serverURL: "http://127.0.0.1:0",
			req: client.ShellRequest{
				Command: "uname -a",
				Target:  "_any",
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
				suite.Error(err)
				suite.Nil(resp)
				suite.Contains(err.Error(), "shell command")
			},
		},
		{
			name: "when server returns 202 with no JSON body returns UnexpectedStatusError",
			req: client.ShellRequest{
				Command: "uname -a",
				Target:  "_any",
			},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			validateFunc: func(resp *client.Response[client.Collection[client.CommandResult]], err error) {
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

			resp, err := sut.Command.Shell(suite.ctx, tc.req)
			tc.validateFunc(resp, err)
		})
	}
}

func TestCommandPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CommandPublicTestSuite))
}
