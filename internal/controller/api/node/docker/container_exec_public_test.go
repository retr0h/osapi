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

package container_test

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

	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apicontainer "github.com/osapi-io/osapi/internal/controller/api/node/docker"
	"github.com/osapi-io/osapi/internal/controller/api/node/docker/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type ContainerExecPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apicontainer.Container
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ContainerExecPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ContainerExecPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apicontainer.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ContainerExecPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ContainerExecPublicTestSuite) TestPostNodeContainerDockerExec() {
	tests := []struct {
		name         string
		request      gen.PostNodeContainerDockerExecRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeContainerDockerExecResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls", "-la"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data: json.RawMessage(
							`{"stdout":"file1\nfile2","stderr":"","exit_code":0}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Stdout)
				s.Equal("file1\nfile2", *r.Results[0].Stdout)
				s.Require().NotNil(r.Results[0].ExitCode)
				s.Equal(0, *r.Results[0].ExitCode)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls"},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "validation error empty id",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls"},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "body validation error empty command",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "success with working dir",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command:    []string{"ls", "-la"},
					WorkingDir: ptr.To("/app"),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data: json.RawMessage(
							`{"stdout":"app files","stderr":"","exit_code":0}`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Stdout)
				s.Equal("app files", *r.Results[0].Stdout)
				s.Require().NotNil(r.Results[0].ExitCode)
				s.Equal(0, *r.Results[0].ExitCode)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "success with nil response data",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Stdout)
				s.Equal("", *r.Results[0].Stdout)
				s.Require().NotNil(r.Results[0].Stderr)
				s.Equal("", *r.Results[0].Stderr)
				s.Require().NotNil(r.Results[0].ExitCode)
				s.Equal(0, *r.Results[0].ExitCode)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				_, ok := resp.(gen.PostNodeContainerDockerExec500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "server1",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls", "-la"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.DockerExecResultItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "_all",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls", "-la"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  ptr.To(true),
							Data: json.RawMessage(
								`{"stdout":"file1\n","stderr":"","exit_code":0}`,
							),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Changed:  ptr.To(true),
							Data: json.RawMessage(
								`{"stdout":"file2\n","stderr":"","exit_code":0}`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with errors",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "_all",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls", "-la"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  ptr.To(true),
							Data: json.RawMessage(
								`{"stdout":"file1\n","stderr":"","exit_code":0}`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "_all",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls", "-la"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "docker: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDockerExec202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 1)
				s.Equal(gen.DockerExecResultItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("docker: operation not supported on this OS family", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PostNodeContainerDockerExecRequestObject{
				Hostname: "_all",
				Id:       "abc123",
				Body: &gen.PostNodeContainerDockerExecJSONRequestBody{
					Command: []string{"ls", "-la"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerExec,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerExecResponseObject) {
				_, ok := resp.(gen.PostNodeContainerDockerExec500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeContainerDockerExec(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ContainerExecPublicTestSuite) TestPostNodeContainerDockerExecValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/container/docker/abc123/exec",
			body: `{"command":["ls","-la"]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "docker", job.OperationDockerExec, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     json.RawMessage(`{"stdout":"output","stderr":"","exit_code":0}`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "stdout")
			},
		},
		{
			name: "when missing command",
			path: "/api/node/server1/container/docker/abc123/exec",
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
			name: "when target agent not found",
			path: "/api/node/nonexistent/container/docker/abc123/exec",
			body: `{"command":["ls"]}`,
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

			containerHandler := apicontainer.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(containerHandler, nil)

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

const rbacContainerExecTestSigningKey = "test-signing-key-for-rbac-container-exec"

func (s *ContainerExecPublicTestSuite) TestPostNodeContainerDockerExecRBACHTTP() {
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
					rbacContainerExecTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"docker:read"},
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
			name: "when valid admin token returns 202",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacContainerExecTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "docker", job.OperationDockerExec, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  ptr.To(true),
						Data:     json.RawMessage(`{"stdout":"output","stderr":"","exit_code":0}`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
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
							SigningKey: rbacContainerExecTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apicontainer.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/container/docker/abc123/exec",
				strings.NewReader(`{"command":["ls","-la"]}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestContainerExecPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ContainerExecPublicTestSuite))
}
