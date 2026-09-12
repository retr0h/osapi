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

package node_test

import (
	"context"
	"encoding/json"
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
	apinode "github.com/osapi-io/osapi/internal/controller/api/node"
	"github.com/osapi-io/osapi/internal/controller/api/node/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/load"
	"github.com/osapi-io/osapi/internal/validation"
)

type NodeLoadGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinode.Node
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NodeLoadGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NodeLoadGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinode.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NodeLoadGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NodeLoadGetPublicTestSuite) TestGetNodeLoad() {
	tests := []struct {
		name         string
		request      gen.GetNodeLoadRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeLoadResponseObject)
	}{
		{
			name:    "success",
			request: gen.GetNodeLoadRequestObject{Hostname: "_any"},
			setupMock: func() {
				loadResult := load.Result{
					Load1:  1.5,
					Load5:  2.0,
					Load15: 1.8,
				}
				data, _ := json.Marshal(loadResult)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name:      "validation error empty hostname",
			request:   gen.GetNodeLoadRequestObject{Hostname: ""},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name:    "job client error",
			request: gen.GetNodeLoadRequestObject{Hostname: "_any"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				_, ok := resp.(gen.GetNodeLoad500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when job skipped",
			request: gen.GetNodeLoadRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.LoadResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all success",
			request: gen.GetNodeLoadRequestObject{Hostname: "_all"},
			setupMock: func() {
				load1 := load.Result{Load1: 1.5, Load5: 2.0, Load15: 1.8}
				load2 := load.Result{Load1: 0.5, Load5: 0.8, Load15: 0.6}
				data1, _ := json.Marshal(load1)
				data2, _ := json.Marshal(load2)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				for _, result := range r.Results {
					s.Require().NotNil(result.Changed)
					s.False(*result.Changed)
				}
			},
		},
		{
			name:    "broadcast all with errors",
			request: gen.GetNodeLoadRequestObject{Hostname: "_all"},
			setupMock: func() {
				load1 := load.Result{Load1: 1.5, Load5: 2.0, Load15: 1.8}
				data1, _ := json.Marshal(load1)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "some error",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, res := range r.Results {
					if res.Error != nil {
						foundError = true
						s.Equal("server2", res.Hostname)
						s.Equal("some error", *res.Error)
					}
				}
				s.True(foundError)
			},
		},
		{
			name:    "broadcast with skipped host",
			request: gen.GetNodeLoadRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "host: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.LoadResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast with failed host",
			request: gen.GetNodeLoadRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				r, ok := resp.(gen.GetNodeLoad200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.LoadResultItemStatusFailed, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all error",
			request: gen.GetNodeLoadRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeLoadResponseObject) {
				_, ok := resp.(gen.GetNodeLoad500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeLoad(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NodeLoadGetPublicTestSuite) TestGetNodeLoadValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when get Ok",
			path: "/api/node/server1/load",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				loadResult := load.Result{Load1: 1.5, Load5: 2.0, Load15: 1.8}
				data, _ := json.Marshal(loadResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeLoadGet, gomock.Any()).
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
				s.Equal(http.StatusOK, rec.Code)
			},
		},
		{
			name: "when empty hostname returns 400",
			path: "/api/node/%20/load",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
			},
		},
		{
			name: "when job client errors",
			path: "/api/node/server1/load",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeLoadGet, gomock.Any()).
					Return("", nil, assert.AnError)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			nodeHandler := apinode.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(nodeHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacLoadTestSigningKey = "test-signing-key-for-load-rbac"

func (s *NodeLoadGetPublicTestSuite) TestGetNodeLoadRBACHTTP() {
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
					rbacLoadTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"job:read"},
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
			name: "when valid token with node:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacLoadTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				loadResult := load.Result{Load1: 1.5, Load5: 2.0, Load15: 1.8}
				data, _ := json.Marshal(loadResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeLoadGet, gomock.Any()).
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
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"job_id"`)
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
							SigningKey: rbacLoadTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apinode.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/load", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNodeLoadGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NodeLoadGetPublicTestSuite))
}
