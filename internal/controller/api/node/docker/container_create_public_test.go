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

type ContainerCreatePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apicontainer.Container
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ContainerCreatePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ContainerCreatePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apicontainer.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ContainerCreatePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ContainerCreatePublicTestSuite) TestPostNodeContainerDocker() {
	tests := []struct {
		name         string
		request      gen.PostNodeContainerDockerRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeContainerDockerResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
					Name:  strPtr("my-nginx"),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     json.RawMessage(`{"id":"abc123"}`),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Id)
				s.Equal("abc123", *r.Results[0].Id)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "body validation error empty image",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "",
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "success with hostname and dns",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image:    "nginx:latest",
					Hostname: strPtr("web-01"),
					Dns:      &[]string{"8.8.8.8", "8.8.4.4"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					DoAndReturn(func(
						_ interface{},
						_ string,
						_ string,
						_ job.OperationType,
						data interface{},
					) (string, *job.Response, error) {
						d, ok := data.(*job.DockerCreateData)
						s.True(ok)
						s.Equal("web-01", d.Hostname)
						s.Equal([]string{"8.8.8.8", "8.8.4.4"}, d.DNS)
						return "550e8400-e29b-41d4-a716-446655440000", &job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"id":"abc123"}`),
						}, nil
					})
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Id)
				s.Equal("abc123", *r.Results[0].Id)
			},
		},
		{
			name: "success with explicit auto_start false",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image:     "nginx:latest",
					AutoStart: boolPtr(false),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     json.RawMessage(`{"id":"xyz789"}`),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Id)
				s.Equal("xyz789", *r.Results[0].Id)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "success with nil response data",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Nil(r.Results[0].Id)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				_, ok := resp.(gen.PostNodeContainerDocker500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "server1",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.DockerResponseStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"id":"abc123"}`),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"id":"def456"}`),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with errors",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"id":"abc123"}`),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerCreate,
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
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.PostNodeContainerDocker202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 1)
				s.Equal(gen.DockerResponseStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("docker: operation not supported on this OS family", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PostNodeContainerDockerRequestObject{
				Hostname: "_all",
				Body: &gen.PostNodeContainerDockerJSONRequestBody{
					Image: "nginx:latest",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerCreate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeContainerDockerResponseObject) {
				_, ok := resp.(gen.PostNodeContainerDocker500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeContainerDocker(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ContainerCreatePublicTestSuite) TestPostNodeContainerDockerValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/container/docker",
			body: `{"image":"nginx:latest"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "docker", job.OperationDockerCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     json.RawMessage(`{"id":"abc123"}`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusAccepted,
			wantContains: []string{`"job_id"`, `"results"`, `"agent1"`},
		},
		{
			name: "when missing image",
			path: "/api/node/server1/container/docker",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Image", "required"},
		},
		{
			name: "when hostname exceeds max length",
			path: "/api/node/server1/container/docker",
			body: `{"image":"nginx:latest","hostname":"` + strings.Repeat("a", 254) + `"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Hostname", "max"},
		},
		{
			name: "when dns contains invalid ip",
			path: "/api/node/server1/container/docker",
			body: `{"image":"nginx:latest","dns":["not-an-ip"]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Dns", "ip"},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/container/docker",
			body: `{"image":"nginx:latest"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target", "not found"},
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

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacContainerCreateTestSigningKey = "test-signing-key-for-rbac-container-create"

func (s *ContainerCreatePublicTestSuite) TestPostNodeContainerDockerRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusUnauthorized,
			wantContains: []string{"Bearer token required"},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacContainerCreateTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid admin token returns 202",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacContainerCreateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "docker", job.OperationDockerCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Changed:  boolPtr(true),
						Data:     json.RawMessage(`{"id":"abc123"}`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusAccepted,
			wantContains: []string{`"job_id"`, `"results"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacContainerCreateTestSigningKey,
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
				"/api/node/server1/container/docker",
				strings.NewReader(`{"image":"nginx:latest"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

func TestContainerCreatePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ContainerCreatePublicTestSuite))
}
