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
	apinetwork "github.com/osapi-io/osapi/internal/controller/api/node/network"
	"github.com/osapi-io/osapi/internal/controller/api/node/network/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/route"
	"github.com/osapi-io/osapi/internal/validation"
)

type NetworkRouteGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkRouteGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkRouteGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkRouteGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkRouteGetPublicTestSuite) TestGetNodeNetworkRouteByInterface() {
	entry := route.Entry{
		Interface: "eth0",
		Routes:    []route.Route{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
	}
	entryData, _ := json.Marshal(entry)

	tests := []struct {
		name         string
		request      gen.GetNodeNetworkRouteByInterfaceRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject)
	}{
		{
			name: "when success",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Data: entryData,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteGetEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Routes)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "", InterfaceName: "eth0",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRouteByInterface400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when validation error empty interface",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "server1", InterfaceName: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRouteByInterface400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job client error",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRouteByInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1", Error: "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Equal(gen.RouteGetEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast success",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: entryData},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
			},
		},
		{
			name: "when broadcast error",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRouteByInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteGetEntryStatusFailed, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "unsupported",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteGetEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast success with nil data",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: nil},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteGetEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Routes)
				s.Empty(*r.Results[0].Routes)
			},
		},
		{
			name: "when success with route metric",
			request: gen.GetNodeNetworkRouteByInterfaceRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
			},
			setupMock: func() {
				entryWithMetric := route.Entry{
					Interface: "eth0",
					Routes: []route.Route{
						{To: "10.0.0.0/8", Via: "192.168.1.1", Metric: 100},
						{To: "", Via: ""},
					},
				}
				data, _ := json.Marshal(entryWithMetric)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Data: data,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRouteByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				routes := *r.Results[0].Routes
				s.Len(routes, 2)
				// First route: has destination, gateway, metric.
				s.Require().NotNil(routes[0].Destination)
				s.Equal("10.0.0.0/8", *routes[0].Destination)
				s.Require().NotNil(routes[0].Gateway)
				s.Equal("192.168.1.1", *routes[0].Gateway)
				s.Require().NotNil(routes[0].Metric)
				s.Equal(100, *routes[0].Metric)
				// Second route: empty strings become nil.
				s.Nil(routes[1].Destination)
				s.Nil(routes[1].Gateway)
				s.Nil(routes[1].Metric)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.GetNodeNetworkRouteByInterface(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkRouteGetPublicTestSuite) TestGetNetworkRouteByInterfaceValidationHTTP() {
	entry := route.Entry{
		Interface: "eth0",
		Routes:    []route.Route{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
	}
	entryData, _ := json.Marshal(entry)

	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/route/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Data: entryData,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"results"`)
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/route/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
				s.Contains(rec.Body.String(), "valid_target")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			networkHandler := apinetwork.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(networkHandler, nil)
			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacRouteGetTestSigningKey = "test-signing-key-for-route-get-rbac"

func (s *NetworkRouteGetPublicTestSuite) TestGetNetworkRouteByInterfaceRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	entry := route.Entry{
		Interface: "eth0",
		Routes:    []route.Route{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
	}
	entryData, _ := json.Marshal(entry)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:      "when no token returns 401",
			setupAuth: func(_ *http.Request) {},
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
				token, _ := tokenManager.Generate(
					rbacRouteGetTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"node:write"},
				)
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
			name: "when valid token returns 200",
			setupAuth: func(req *http.Request) {
				token, _ := tokenManager.Generate(
					rbacRouteGetTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{Hostname: "server1", Data: entryData}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
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
						Security: config.ServerSecurity{SigningKey: rbacRouteGetTestSigningKey},
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

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/network/route/eth0", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNetworkRouteGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkRouteGetPublicTestSuite))
}
