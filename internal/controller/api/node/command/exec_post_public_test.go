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

package command_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apicommand "github.com/osapi-io/osapi/internal/controller/api/node/command"
	"github.com/osapi-io/osapi/internal/controller/api/node/command/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/command"
	"github.com/osapi-io/osapi/internal/validation"
)

type CommandExecPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apicommand.Command
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *CommandExecPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *CommandExecPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apicommand.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *CommandExecPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func intPtr(
	i int,
) *int {
	return &i
}

func strPtr(
	s string,
) *string {
	return &s
}

func (s *CommandExecPostPublicTestSuite) TestPostNodeCommandExec() {
	tests := []struct {
		name         string
		request      gen.PostNodeCommandExecRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeCommandExecResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				data, _ := json.Marshal(command.Result{
					Stdout:     "file1\nfile2",
					Stderr:     "",
					ExitCode:   0,
					DurationMs: 12,
					Changed:    false,
				})
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data:     json.RawMessage(data),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Stdout)
				s.Equal("file1\nfile2", *r.Results[0].Stdout)
				s.Require().NotNil(r.Results[0].ExitCode)
				s.Equal(0, *r.Results[0].ExitCode)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name: "success with all optional fields",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Cwd:     strPtr("/tmp"),
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				data, _ := json.Marshal(command.Result{
					Stdout:     "file1\nfile2",
					ExitCode:   0,
					DurationMs: 12,
				})
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data:     json.RawMessage(data),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "body validation error empty command",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_any",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				_, ok := resp.(gen.PostNodeCommandExec500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.Skipped, r.Results[0].Status)
			},
		},
		{
			name: "broadcast all success",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				data1, _ := json.Marshal(command.Result{Stdout: "file1", ExitCode: 0})
				data2, _ := json.Marshal(command.Result{Stdout: "file2", ExitCode: 0})
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
							"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				s.NotNil(resp)
			},
		},
		{
			name: "broadcast all with errors",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				data1, _ := json.Marshal(command.Result{Stdout: "file1", ExitCode: 0})
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "command not found",
								Hostname: "server2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec202JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, item := range r.Results {
					if item.Error != nil {
						foundError = true
						s.Equal("server2", item.Hostname)
						s.Equal("command not found", *item.Error)
					}
				}
				s.True(foundError)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusSkipped,
								Error:    "host: operation not supported on this OS family",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.Skipped, r.Results[0].Status)
			},
		},
		{
			name: "broadcast with failed host",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				r, ok := resp.(gen.PostNodeCommandExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.Failed, r.Results[0].Status)
			},
		},
		{
			name: "broadcast all error",
			request: gen.PostNodeCommandExecRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeCommandExecJSONRequestBody{
					Command: "ls",
					Args:    &[]string{"-la"},
					Timeout: intPtr(30),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"command",
						job.OperationCommandExecExecute,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeCommandExecResponseObject) {
				_, ok := resp.(gen.PostNodeCommandExec500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeCommandExec(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *CommandExecPostPublicTestSuite) TestPostCommandExecValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/command/exec",
			body: `{"command":"ls","args":["-la"]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				data, _ := json.Marshal(command.Result{
					Stdout:     "file1\nfile2",
					Stderr:     "",
					ExitCode:   0,
					DurationMs: 42,
					Changed:    true,
				})
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "command", job.OperationCommandExecExecute, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "agent1")
				s.Contains(rec.Body.String(), "changed")
			},
		},
		{
			name: "when missing command",
			path: "/api/node/server1/command/exec",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "Command")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when invalid timeout",
			path: "/api/node/server1/command/exec",
			body: `{"command":"ls","timeout":999}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "Timeout")
				s.Contains(rec.Body.String(), "max")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/command/exec",
			body: `{"command":"ls"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "valid_target")
				s.Contains(rec.Body.String(), "not found")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			commandHandler := apicommand.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(commandHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodPost,
				tc.path,
				strings.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacExecTestSigningKey = "test-signing-key-for-exec-rbac"

func (s *CommandExecPostPublicTestSuite) TestPostCommandExecRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusUnauthorized, rec.Code)
				s.Contains(rec.Body.String(), "Bearer token required")
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacExecTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"network:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusForbidden, rec.Code)
				s.Contains(rec.Body.String(), "Insufficient permissions")
			},
		},
		{
			name: "when valid token with command:execute returns 202",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacExecTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				data, _ := json.Marshal(command.Result{
					Stdout:     "output",
					Stderr:     "",
					ExitCode:   0,
					DurationMs: 10,
					Changed:    true,
				})
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "command", job.OperationCommandExecExecute, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data:     json.RawMessage(data),
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "changed")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacExecTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apicommand.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/command/exec",
				strings.NewReader(`{"command":"ls","args":["-la"]}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestCommandExecPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CommandExecPostPublicTestSuite))
}
