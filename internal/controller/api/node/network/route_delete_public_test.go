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

package network_test

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
	apinetwork "github.com/osapi-io/osapi/internal/controller/api/node/network"
	"github.com/osapi-io/osapi/internal/controller/api/node/network/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type NetworkRouteDeletePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkRouteDeletePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkRouteDeletePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkRouteDeletePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkRouteDeletePublicTestSuite) TestDeleteNodeNetworkRoute() {
	trueVal := true

	tests := []struct {
		name         string
		request      gen.DeleteNodeNetworkRouteRequestObject
		setupMock    func()
		validateFunc func(resp gen.DeleteNodeNetworkRouteResponseObject)
	}{
		{
			name: "when success",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Changed: &trueVal,
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.DeleteNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteMutationEntryStatusOk, r.Results[0].Status)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "", InterfaceName: "eth0",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.DeleteNodeNetworkRoute400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when validation error empty interface",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.DeleteNodeNetworkRoute400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job client error",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.DeleteNodeNetworkRoute500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1", Error: "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.DeleteNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Equal(gen.RouteMutationEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast success",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Changed: &trueVal},
					}, nil)
			},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				s.NotNil(resp)
			},
		},
		{
			name: "when broadcast error",
			request: gen.DeleteNodeNetworkRouteRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.DeleteNodeNetworkRoute500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.DeleteNodeNetworkRoute(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkRouteDeletePublicTestSuite) TestDeleteNetworkRouteValidationHTTP() {
	trueVal := true

	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/route/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{Hostname: "server1", Changed: &trueVal}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/route/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "valid_target"},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			networkHandler := apinetwork.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(networkHandler, nil)
			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			rec := httptest.NewRecorder()
			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacRouteDeleteTestSigningKey = "test-signing-key-for-route-delete-rbac"

func (s *NetworkRouteDeletePublicTestSuite) TestDeleteNetworkRouteRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	trueVal := true

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
				token, _ := tokenManager.Generate(
					rbacRouteDeleteTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"network:read"},
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token returns 200",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacRouteDeleteTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteDelete, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{Hostname: "server1", Changed: &trueVal}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{SigningKey: rbacRouteDeleteTestSigningKey},
					},
				},
			}
			server := api.New(appConfig, s.logger)
			handlers := apinetwork.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/node/server1/network/route/eth0",
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

func TestNetworkRouteDeletePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkRouteDeletePublicTestSuite))
}
