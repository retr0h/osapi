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
	"errors"
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
	apiservice "github.com/osapi-io/osapi/internal/controller/api/node/service"
	"github.com/osapi-io/osapi/internal/controller/api/node/service/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type ServiceUpdatePutPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiservice.Service
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *ServiceUpdatePutPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *ServiceUpdatePutPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiservice.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *ServiceUpdatePutPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ServiceUpdatePutPublicTestSuite) TestPutNodeService() {
	tests := []struct {
		name         string
		request      gen.PutNodeServiceRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeServiceResponseObject)
	}{
		{
			name: "success",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1",
				Name:     "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{
					Object: "my-app-unit-object-v2",
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							JobID:    "550e8400-e29b-41d4-a716-446655440000",
							Hostname: "agent1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"my-app.service","changed":true}`),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
				s.Equal("my-app.service", *r.Results[0].Name)
			},
		},
		{
			name: "success with nil response data",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1",
				Name:     "my-app.service",
				Body:     &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: boolPtr(true), Data: nil,
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "validation error empty hostname",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "body validation error empty object",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: ""},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "not found error",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1", Name: "nonexistent.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("", nil, errors.New("service not found"))
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "not found")
			},
		},
		{
			name: "does not exist error",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1", Name: "missing.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("", nil, errors.New("service does not exist"))
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "does not exist")
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1",
						Error: "service: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Skipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "job client error",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "server1", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				_, ok := resp.(gen.PutNodeService500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "broadcast success",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "_all", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Hostname: "server1",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"my-app.service","changed":true}`),
						},
						"server2": {
							Hostname: "server2",
							Changed:  boolPtr(true),
							Data:     json.RawMessage(`{"name":"my-app.service","changed":true}`),
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.JobId)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "broadcast with nil response data",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "_all", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Changed: boolPtr(true), Data: nil},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("", *r.Results[0].Name)
			},
		},
		{
			name: "broadcast with failed host",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "_all", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "agent unreachable",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Failed, r.Results[0].Status)
				s.Contains(*r.Results[0].Error, "unreachable")
			},
		},
		{
			name: "broadcast with skipped host",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "_all", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "service: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				r, ok := resp.(gen.PutNodeService200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.Skipped, r.Results[0].Status)
				s.Contains(*r.Results[0].Error, "not supported")
			},
		},
		{
			name: "broadcast error collecting responses",
			request: gen.PutNodeServiceRequestObject{
				Hostname: "_all", Name: "my-app.service",
				Body: &gen.PutNodeServiceJSONRequestBody{Object: "my-app-unit-object-v2"},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeServiceResponseObject) {
				_, ok := resp.(gen.PutNodeService500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.PutNodeService(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *ServiceUpdatePutPublicTestSuite) TestPutNodeServiceValidationHTTP() {
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
			path: "/api/node/server1/service/my-app.service",
			body: `{"object":"my-app-unit-object-v2"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: boolPtr(true),
						Data: json.RawMessage(`{"name":"my-app.service","changed":true}`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`, `"results"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/service/my-app.service",
			body: `{"object":"my-app-unit-object-v2"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target"},
		},
		{
			name: "when invalid body empty object",
			path: "/api/node/server1/service/my-app.service",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			serviceHandler := apiservice.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(serviceHandler, nil)
			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)
			req := httptest.NewRequest(http.MethodPut, tc.path, strings.NewReader(tc.body))
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

const rbacServiceUpdateTestSigningKey = "test-signing-key-for-rbac-service-update"

func (s *ServiceUpdatePutPublicTestSuite) TestPutNodeServiceRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name:      "when no token returns 401",
			setupAuth: func(_ *http.Request) {},
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
					rbacServiceUpdateTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"docker:write"},
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
			name: "when valid admin token returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacServiceUpdateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "node", job.OperationServiceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						JobID: "550e8400-e29b-41d4-a716-446655440000", Hostname: "agent1", Changed: boolPtr(true),
						Data: json.RawMessage(`{"name":"my-app.service","changed":true}`),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
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
							SigningKey: rbacServiceUpdateTestSigningKey,
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
				http.MethodPut,
				"/api/node/server1/service/my-app.service",
				strings.NewReader(`{"object":"my-app-unit-object-v2"}`),
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

func TestServiceUpdatePutPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ServiceUpdatePutPublicTestSuite))
}
