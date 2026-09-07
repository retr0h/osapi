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
	"github.com/osapi-io/osapi/internal/provider/network/netplan/iface"
	"github.com/osapi-io/osapi/internal/validation"
)

type NetworkInterfaceGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkInterfaceGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkInterfaceGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkInterfaceGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkInterfaceGetPublicTestSuite) TestGetNodeNetworkInterfaceByName() {
	trueVal := true
	entry := iface.InterfaceEntry{Name: "eth0", DHCP4: &trueVal}
	entryData, _ := json.Marshal(entry)

	fullEntry := iface.InterfaceEntry{
		Name:       "eth0",
		DHCP4:      &trueVal,
		Addresses:  []string{"10.0.0.1/24"},
		Gateway4:   "10.0.0.1",
		Gateway6:   "fe80::1",
		MTU:        1500,
		MACAddress: "00:11:22:33:44:55",
		WakeOnLAN:  &trueVal,
	}
	fullEntryData, _ := json.Marshal(fullEntry)

	tests := []struct {
		name         string
		request      gen.GetNodeNetworkInterfaceByNameRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeNetworkInterfaceByNameResponseObject)
	}{
		{
			name: "when success",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "server1",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "server1",
							Data:     entryData,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.InterfaceGetEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Interface)
			},
		},
		{
			name: "when success with all interface fields",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "server1",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "server1",
							Data:     fullEntryData,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				ifc := r.Results[0].Interface
				s.Require().NotNil(ifc)
				s.Equal("eth0", *ifc.Name)
				s.Require().NotNil(ifc.Addresses)
				s.Equal([]string{"10.0.0.1/24"}, *ifc.Addresses)
				s.Require().NotNil(ifc.Mtu)
				s.Equal(1500, *ifc.Mtu)
				s.Require().NotNil(ifc.Gateway4)
				s.Equal("10.0.0.1", *ifc.Gateway4)
				s.Require().NotNil(ifc.Gateway6)
				s.Equal("fe80::1", *ifc.Gateway6)
				s.Require().NotNil(ifc.MacAddress)
				s.Equal("00:11:22:33:44:55", *ifc.MacAddress)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "",
				Name:     "eth0",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation error empty name",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "server1",
				Name:     "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "when job client error",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "server1",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkInterfaceByName500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "server1",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceGetEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "_all",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: entryData},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 1)
			},
		},
		{
			name: "when broadcast error",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "_all",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkInterfaceByName500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "_all",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceGetEntryStatusFailed, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "_all",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusSkipped,
								Error:    "unsupported",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceGetEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast success with nil data",
			request: gen.GetNodeNetworkInterfaceByNameRequestObject{
				Hostname: "_all",
				Name:     "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceGet,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: nil},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceByNameResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterfaceByName200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceGetEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Interface)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeNetworkInterfaceByName(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkInterfaceGetPublicTestSuite) TestGetNetworkInterfaceByNameValidationHTTP() {
	trueVal := true
	entry := iface.InterfaceEntry{Name: "eth0", DHCP4: &trueVal}
	entryData, _ := json.Marshal(entry)

	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/interface/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Data:     entryData,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "server1")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/interface/eth0",
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

const rbacInterfaceGetTestSigningKey = "test-signing-key-for-interface-get-rbac"

func (s *NetworkInterfaceGetPublicTestSuite) TestGetNetworkInterfaceByNameRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	trueVal := true
	entry := iface.InterfaceEntry{Name: "eth0", DHCP4: &trueVal}
	entryData, _ := json.Marshal(entry)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
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
					rbacInterfaceGetTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"node:write"},
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
			name: "when valid token with network:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacInterfaceGetTestSigningKey,
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
					Query(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Data:     entryData,
					}, nil)
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
						Security: config.ServerSecurity{
							SigningKey: rbacInterfaceGetTestSigningKey,
						},
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
				http.MethodGet,
				"/api/node/server1/network/interface/eth0",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNetworkInterfaceGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkInterfaceGetPublicTestSuite))
}
