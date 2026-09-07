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
	"github.com/osapi-io/osapi/internal/provider/node/host"
	"github.com/osapi-io/osapi/internal/validation"
)

type NodeOSGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinode.Node
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NodeOSGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NodeOSGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinode.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NodeOSGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NodeOSGetPublicTestSuite) TestGetNodeOS() {
	tests := []struct {
		name         string
		request      gen.GetNodeOSRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeOSResponseObject)
	}{
		{
			name:    "success",
			request: gen.GetNodeOSRequestObject{Hostname: "_any"},
			setupMock: func() {
				osResult := host.Result{
					Distribution: "Ubuntu",
					Version:      "22.04",
				}
				data, _ := json.Marshal(osResult)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name:      "validation error empty hostname",
			request:   gen.GetNodeOSRequestObject{Hostname: ""},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name:    "job client error",
			request: gen.GetNodeOSRequestObject{Hostname: "_any"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				_, ok := resp.(gen.GetNodeOS500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when job skipped",
			request: gen.GetNodeOSRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.OSInfoResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all success",
			request: gen.GetNodeOSRequestObject{Hostname: "_all"},
			setupMock: func() {
				os1 := host.Result{Distribution: "Ubuntu", Version: "22.04"}
				os2 := host.Result{Distribution: "CentOS", Version: "8.3"}
				data1, _ := json.Marshal(os1)
				data2, _ := json.Marshal(os2)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS200JSONResponse)
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
			request: gen.GetNodeOSRequestObject{Hostname: "_all"},
			setupMock: func() {
				os1 := host.Result{Distribution: "Ubuntu", Version: "22.04"}
				data1, _ := json.Marshal(os1)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "some error",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS200JSONResponse)
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
			request: gen.GetNodeOSRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "host: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.OSInfoResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast with failed host",
			request: gen.GetNodeOSRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				r, ok := resp.(gen.GetNodeOS200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.OSInfoResultItemStatusFailed, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all error",
			request: gen.GetNodeOSRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeOSResponseObject) {
				_, ok := resp.(gen.GetNodeOS500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeOS(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NodeOSGetPublicTestSuite) TestGetNodeOSValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when get Ok",
			path: "/api/node/server1/os",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				osResult := host.Result{Distribution: "Ubuntu", Version: "22.04"}
				data, _ := json.Marshal(osResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeOSGet, gomock.Any()).
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
			wantCode: http.StatusOK,
		},
		{
			name: "when empty hostname returns 400",
			path: "/api/node/%20/os",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{"error"},
		},
		{
			name: "when job client errors",
			path: "/api/node/server1/os",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeOSGet, gomock.Any()).
					Return("", nil, assert.AnError)
				return mock
			},
			wantCode: http.StatusInternalServerError,
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

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacOSTestSigningKey = "test-signing-key-for-os-rbac"

func (s *NodeOSGetPublicTestSuite) TestGetNodeOSRBACHTTP() {
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
					rbacOSTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with node:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacOSTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				osResult := host.Result{Distribution: "Ubuntu", Version: "22.04"}
				data, _ := json.Marshal(osResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeOSGet, gomock.Any()).
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
			wantCode:     http.StatusOK,
			wantContains: []string{`"job_id"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacOSTestSigningKey,
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

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/os", nil)
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

func TestNodeOSGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NodeOSGetPublicTestSuite))
}
