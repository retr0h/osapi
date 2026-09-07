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

type NetworkRouteListGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkRouteListGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkRouteListGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkRouteListGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkRouteListGetPublicTestSuite) TestGetNodeNetworkRoute() {
	entries := []route.ListEntry{
		{Destination: "10.0.0.0/8", Gateway: "192.168.1.1", Interface: "eth0"},
	}
	entryData, _ := json.Marshal(entries)

	tests := []struct {
		name         string
		request      gen.GetNodeNetworkRouteRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeNetworkRouteResponseObject)
	}{
		{
			name:    "when success",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Data: entryData,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteListEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Routes)
				s.Len(*r.Results[0].Routes, 1)
			},
		},
		{
			name:      "when validation error empty hostname",
			request:   gen.GetNodeNetworkRouteRequestObject{Hostname: ""},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRoute400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when job client error",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteList, nil).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRoute500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when job skipped",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1",
						Error: "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteListEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "when broadcast all success",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: entryData},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
			},
		},
		{
			name:    "when broadcast error",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteList, nil).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkRoute500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when broadcast with failed host",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteListEntryStatusFailed, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
			},
		},
		{
			name:    "when broadcast with skipped host",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "unsupported",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteListEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name:    "when broadcast success with nil data",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: nil},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteListEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Routes)
				s.Empty(*r.Results[0].Routes)
			},
		},
		{
			name:    "when success with full route fields",
			request: gen.GetNodeNetworkRouteRequestObject{Hostname: "server1"},
			setupMock: func() {
				fullEntries := []route.ListEntry{
					{
						Destination: "10.0.0.0/8",
						Gateway:     "192.168.1.1",
						Interface:   "eth0",
						Metric:      100,
						Flags:       "UG",
					},
					{
						Destination: "",
						Gateway:     "",
						Interface:   "",
						Metric:      0,
						Flags:       "",
					},
				}
				data, _ := json.Marshal(fullEntries)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Data: data,
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				routes := *r.Results[0].Routes
				s.Len(routes, 2)

				// First entry: all fields populated.
				s.Require().NotNil(routes[0].Destination)
				s.Equal("10.0.0.0/8", *routes[0].Destination)
				s.Require().NotNil(routes[0].Gateway)
				s.Equal("192.168.1.1", *routes[0].Gateway)
				s.Require().NotNil(routes[0].Interface)
				s.Equal("eth0", *routes[0].Interface)
				s.Require().NotNil(routes[0].Metric)
				s.Equal(100, *routes[0].Metric)
				s.Require().NotNil(routes[0].Scope)
				s.Equal("UG", *routes[0].Scope)

				// Second entry: empty strings become nil, metric 0 not set.
				s.Nil(routes[1].Destination)
				s.Nil(routes[1].Gateway)
				s.Nil(routes[1].Interface)
				s.Nil(routes[1].Metric)
				s.Nil(routes[1].Scope)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.GetNodeNetworkRoute(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkRouteListGetPublicTestSuite) TestGetNetworkRouteListValidationHTTP() {
	entries := []route.ListEntry{
		{Destination: "10.0.0.0/8", Gateway: "192.168.1.1", Interface: "eth0"},
	}
	entryData, _ := json.Marshal(entries)

	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/route",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteList, nil).
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
			path: "/api/node/nonexistent/network/route",
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

const rbacRouteListGetTestSigningKey = "test-signing-key-for-route-list-get-rbac"

func (s *NetworkRouteListGetPublicTestSuite) TestGetNetworkRouteListRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	entries := []route.ListEntry{
		{Destination: "10.0.0.0/8", Gateway: "192.168.1.1", Interface: "eth0"},
	}
	entryData, _ := json.Marshal(entries)

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
					rbacRouteListGetTestSigningKey,
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
					rbacRouteListGetTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkRouteList, nil).
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
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()
			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{SigningKey: rbacRouteListGetTestSigningKey},
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

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/network/route", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNetworkRouteListGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkRouteListGetPublicTestSuite))
}
