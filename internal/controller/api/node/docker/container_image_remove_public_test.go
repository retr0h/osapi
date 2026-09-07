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
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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

type ContainerImageRemovePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apicontainer.Container
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ContainerImageRemovePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ContainerImageRemovePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apicontainer.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ContainerImageRemovePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ContainerImageRemovePublicTestSuite) TestDeleteNodeContainerDockerImage() {
	forceTrue := true

	tests := []struct {
		name         string
		request      gen.DeleteNodeContainerDockerImageRequestObject
		setupMock    func()
		validateFunc func(resp gen.DeleteNodeContainerDockerImageResponseObject)
	}{
		{
			name: "success",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "server1",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  boolPtr(true),
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Id)
				s.Equal("nginx:latest", *r.Results[0].Id)
				s.Require().NotNil(r.Results[0].Message)
				s.Equal("image removed", *r.Results[0].Message)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "success with force",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "server1",
				Image:    "nginx:latest",
				Params: gen.DeleteNodeContainerDockerImageParams{
					Force: &forceTrue,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  boolPtr(true),
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "validation error empty image",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "server1",
				Image:    "",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "server error",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "server1",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				_, ok := resp.(gen.DeleteNodeContainerDockerImage500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "job client error",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "server1",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				_, ok := resp.(gen.DeleteNodeContainerDockerImage500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "server1",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.DockerActionResultItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast success",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "_all",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  boolPtr(true),
						},
						"server2": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server2",
							Changed:  boolPtr(true),
						},
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with errors",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "_all",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "server1",
							Changed:  boolPtr(true),
						},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "_all",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerImageRemove,
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
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				r, ok := resp.(gen.DeleteNodeContainerDockerImage202JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 1)
				s.Equal(gen.DockerActionResultItemStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("docker: operation not supported on this OS family", *r.Results[0].Error)
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.DeleteNodeContainerDockerImageRequestObject{
				Hostname: "_all",
				Image:    "nginx:latest",
				Params:   gen.DeleteNodeContainerDockerImageParams{},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"docker",
						job.OperationDockerImageRemove,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeContainerDockerImageResponseObject) {
				_, ok := resp.(gen.DeleteNodeContainerDockerImage500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.DeleteNodeContainerDockerImage(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ContainerImageRemovePublicTestSuite) TestDeleteNodeContainerDockerImageValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/container/docker/image/nginx:latest",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "docker", job.OperationDockerImageRemove, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  boolPtr(true),
					}, nil)
				return mock
			},
			wantCode:     http.StatusAccepted,
			wantContains: []string{`"job_id"`, `"results"`, `"image removed"`},
		},
		{
			name: "when empty image returns 400",
			path: "/api/node/server1/container/docker/image/",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode: http.StatusNotFound,
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/container/docker/image/nginx:latest",
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
				http.MethodDelete,
				tc.path,
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacContainerImageRemoveTestSigningKey = "test-signing-key-for-rbac-container-image-remove"

func (s *ContainerImageRemovePublicTestSuite) TestDeleteNodeContainerDockerImageRBACHTTP() {
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
					rbacContainerImageRemoveTestSigningKey,
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
					rbacContainerImageRemoveTestSigningKey,
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
					Modify(gomock.Any(), "server1", "docker", job.OperationDockerImageRemove, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  boolPtr(true),
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
							SigningKey: rbacContainerImageRemoveTestSigningKey,
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
				http.MethodDelete,
				"/api/node/server1/container/docker/image/nginx:latest",
				nil,
			)
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

func TestContainerImageRemovePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ContainerImageRemovePublicTestSuite))
}
