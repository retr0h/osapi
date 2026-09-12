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

type ContainerListPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apicontainer.Container
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ContainerListPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ContainerListPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apicontainer.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ContainerListPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ContainerListPublicTestSuite) TestGetNodeContainerDocker() {
	stateAll := gen.All

	tests := []struct {
		name         string
		request      gen.GetNodeContainerDockerRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeContainerDockerResponseObject)
	}{
		{
			name: "success",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeContainerDockerParams{
					State: &stateAll,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"id":"abc123","name":"my-nginx","image":"nginx:latest","state":"running"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Containers)
				s.Len(*r.Results[0].Containers, 1)
				c := (*r.Results[0].Containers)[0]
				s.Equal("abc123", *c.Id)
				s.Equal("my-nginx", *c.Name)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "validation error invalid limit",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeContainerDockerParams{
					Limit: ptr.To(0),
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Limit")
			},
		},
		{
			name: "success with limit param",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeContainerDockerParams{
					State: &stateAll,
					Limit: ptr.To(5),
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"id":"abc123","name":"my-nginx","image":"nginx:latest","state":"running","created":"2026-01-01T00:00:00Z"}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Containers)
				s.Len(*r.Results[0].Containers, 1)
				c := (*r.Results[0].Containers)[0]
				s.Equal("abc123", *c.Id)
				s.Require().NotNil(c.Created)
				s.Equal("2026-01-01T00:00:00Z", *c.Created)
			},
		},
		{
			name: "success with nil response data",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeContainerDockerParams{
					State: &stateAll,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     nil,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Containers)
				s.Empty(*r.Results[0].Containers)
			},
		},
		{
			name: "success with empty created field",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params: gen.GetNodeContainerDockerParams{
					State: &stateAll,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data: json.RawMessage(
							`[{"id":"x","name":"n","image":"i","state":"running","created":""}]`,
						),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Containers)
				s.Len(*r.Results[0].Containers, 1)
				c := (*r.Results[0].Containers)[0]
				s.Equal("x", *c.Id)
				s.Nil(c.Created)
			},
		},
		{
			name: "job client error",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				_, ok := resp.(gen.GetNodeContainerDocker500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "server1",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.DockerListItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast success",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "_all",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Data: json.RawMessage(
								`[{"id":"abc123","name":"web","state":"running"}]`,
							),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Data: json.RawMessage(
								`[{"id":"def456","name":"api","state":"running"}]`,
							),
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with errors",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "_all",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Data: json.RawMessage(
								`[{"id":"abc123","name":"web","state":"running"}]`,
							),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "_all",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerList,
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
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				r, ok := resp.(gen.GetNodeContainerDocker200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 1)
				s.Equal(gen.DockerListItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("docker: operation not supported on this OS family", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.GetNodeContainerDockerRequestObject{
				Hostname: "_all",
				Params:   gen.GetNodeContainerDockerParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerList,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeContainerDockerResponseObject) {
				_, ok := resp.(gen.GetNodeContainerDocker500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeContainerDocker(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ContainerListPublicTestSuite) TestGetNodeContainerDockerValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/container/docker",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "docker", job.OperationDockerList, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"results"`)
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/container/docker",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
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

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacContainerListTestSigningKey = "test-signing-key-for-rbac-container-list"

func (s *ContainerListPublicTestSuite) TestGetNodeContainerDockerRBACHTTP() {
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
					rbacContainerListTestSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"docker:write"},
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
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacContainerListTestSigningKey,
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
					Query(gomock.Any(), "server1", "docker", job.OperationDockerList, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID:    "550e8400-e29b-41d4-a716-446655440000",
						Hostname: "agent1",
						Data:     json.RawMessage(`[]`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
				s.Contains(rec.Body.String(), `"results"`)
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
							SigningKey: rbacContainerListTestSigningKey,
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
				http.MethodGet,
				"/api/node/server1/container/docker",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestContainerListPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ContainerListPublicTestSuite))
}
