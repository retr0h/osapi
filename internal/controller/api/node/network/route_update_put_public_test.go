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
	"strings"
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

type NetworkRouteUpdatePutPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkRouteUpdatePutPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkRouteUpdatePutPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkRouteUpdatePutPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkRouteUpdatePutPublicTestSuite) TestPutNodeNetworkRoute() {
	trueVal := true

	tests := []struct {
		name         string
		request      gen.PutNodeNetworkRouteRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeNetworkRouteResponseObject)
	}{
		{
			name: "when success",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1", Changed: &trueVal,
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.RouteMutationEntryStatusOk, r.Results[0].Status)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkRoute400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when validation error empty interface name",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkRoute400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when body validation error",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{Routes: []gen.RouteItem{}},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkRoute400JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job client error",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkRoute500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "server1", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status: job.StatusSkipped, Hostname: "server1", Error: "unsupported",
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Equal(gen.RouteMutationEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast success",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Changed: &trueVal},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				s.NotNil(resp)
			},
		},
		{
			name: "when broadcast with failed and skipped hosts",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
						"server2": {
							Status:   job.StatusSkipped,
							Error:    "unsupported",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkRoute200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				statuses := map[gen.RouteMutationEntryStatus]bool{}
				for _, item := range r.Results {
					statuses[item.Status] = true
					s.Require().NotNil(item.Error)
					s.Require().NotNil(item.Changed)
					s.False(*item.Changed)
				}
				s.True(statuses[gen.RouteMutationEntryStatusFailed])
				s.True(statuses[gen.RouteMutationEntryStatusSkipped])
			},
		},
		{
			name: "when broadcast error",
			request: gen.PutNodeNetworkRouteRequestObject{
				Hostname: "_all", InterfaceName: "eth0",
				Body: &gen.RouteConfigRequest{
					Routes: []gen.RouteItem{{To: "10.0.0.0/8", Via: "192.168.1.1"}},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNetworkRouteResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkRoute500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			resp, err := s.handler.PutNodeNetworkRoute(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkRouteUpdatePutPublicTestSuite) TestPutNetworkRouteValidationHTTP() {
	trueVal := true

	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/route/eth0",
			body: `{"routes":[{"to":"10.0.0.0/8","via":"192.168.1.1"}]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{Hostname: "server1", Changed: &trueVal}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "results")
			},
		},
		{
			name: "when invalid route",
			path: "/api/node/server1/network/route/eth0",
			body: `{"routes":[{"to":"bad","via":"bad"}]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
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

			req := httptest.NewRequest(http.MethodPut, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacRouteUpdateTestSigningKey = "test-signing-key-for-route-update-rbac"

func (s *NetworkRouteUpdatePutPublicTestSuite) TestPutNetworkRouteRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	trueVal := true

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
					rbacRouteUpdateTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"network:read"},
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
					rbacRouteUpdateTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkRouteUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{Hostname: "server1", Changed: &trueVal}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
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
						Security: config.ServerSecurity{SigningKey: rbacRouteUpdateTestSigningKey},
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
				http.MethodPut,
				"/api/node/server1/network/route/eth0",
				strings.NewReader(`{"routes":[{"to":"10.0.0.0/8","via":"192.168.1.1"}]}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNetworkRouteUpdatePutPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkRouteUpdatePutPublicTestSuite))
}
