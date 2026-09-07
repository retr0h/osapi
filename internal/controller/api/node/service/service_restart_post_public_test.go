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

package service_test

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
	apiservice "github.com/osapi-io/osapi/internal/controller/api/node/service"
	"github.com/osapi-io/osapi/internal/controller/api/node/service/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type ServiceRestartPostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiservice.Service
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ServiceRestartPostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ServiceRestartPostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiservice.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ServiceRestartPostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ServiceRestartPostPublicTestSuite) TestPostNodeServiceRestart() {
	tests := []struct {
		name         string
		request      gen.PostNodeServiceRestartRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeServiceRestartResponseObject)
	}{
		{
			name: "success",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "server1",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: ptr.To(true),
						Data: json.RawMessage(`{"name":"nginx.service","changed":true}`),
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.True(*r.Results[0].Changed)
				s.Equal("nginx.service", *r.Results[0].Name)
			},
		},
		{
			name: "success with nil response data",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "server1",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: ptr.To(true), Data: nil,
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name:      "validation error empty hostname",
			request:   gen.PostNodeServiceRestartRequestObject{Hostname: "", Name: "nginx.service"},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				_, ok := resp.(gen.PostNodeServiceRestart400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "server1",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1", Error: "service: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Skipped, r.Results[0].Status)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "server1",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceRestart, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				_, ok := resp.(gen.PostNodeServiceRestart500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "_all",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"nginx.service","changed":true}`),
						},
						"server2": {
							Hostname: "server2",
							Changed:  ptr.To(true),
							Data:     json.RawMessage(`{"name":"nginx.service","changed":true}`),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with nil response data",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "_all",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Changed: ptr.To(true), Data: nil},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "broadcast with failed host",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "_all",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Failed, r.Results[0].Status)
				s.Contains(*r.Results[0].Error, "unreachable")
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "_all",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "service: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				r, ok := resp.(gen.PostNodeServiceRestart200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Skipped, r.Results[0].Status)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PostNodeServiceRestartRequestObject{
				Hostname: "_all",
				Name:     "nginx.service",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceRestart, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeServiceRestartResponseObject) {
				_, ok := resp.(gen.PostNodeServiceRestart500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.PostNodeServiceRestart(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ServiceRestartPostPublicTestSuite) TestPostNodeServiceRestartValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/service/nginx.service/restart",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: ptr.To(true),
						Data: json.RawMessage(`{"name":"nginx.service","changed":true}`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "job_id")
				s.Contains(rec.Body.String(), "results")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/service/nginx.service/restart",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "valid_target")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			serviceHandler := apiservice.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(serviceHandler, nil)
			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)
			req := httptest.NewRequest(http.MethodPost, tc.path, nil)
			rec := httptest.NewRecorder()
			a.Echo.ServeHTTP(rec, req)
			tc.validateFunc(rec)
		})
	}
}

const rbacServiceRestartTestSigningKey = "test-signing-key-for-rbac-service-restart"

func (s *ServiceRestartPostPublicTestSuite) TestPostNodeServiceRestartRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:         "when no token returns 401",
			setupAuth:    func(_ *http.Request) {},
			setupJobMock: func() *jobmocks.MockJobClient { return jobmocks.NewMockJobClient(s.mockCtrl) },
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusUnauthorized, rec.Code)
				s.Contains(rec.Body.String(), "Bearer token required")
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacServiceRestartTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"docker:write"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient { return jobmocks.NewMockJobClient(s.mockCtrl) },
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusForbidden, rec.Code)
				s.Contains(rec.Body.String(), "Insufficient permissions")
			},
		},
		{
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacServiceRestartTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationServiceRestart, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: ptr.To(true),
						Data: json.RawMessage(`{"name":"nginx.service","changed":true}`),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
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
							SigningKey: rbacServiceRestartTestSigningKey,
						},
					},
				},
			}
			server := api.New(appConfig, s.logger)
			handlers := apiservice.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/node/server1/service/nginx.service/restart",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)
			tc.validateFunc(rec)
		})
	}
}

func TestServiceRestartPostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ServiceRestartPostPublicTestSuite))
}
