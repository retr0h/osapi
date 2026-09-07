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

type NetworkInterfaceListGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkInterfaceListGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkInterfaceListGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkInterfaceListGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkInterfaceListGetPublicTestSuite) TestGetNodeNetworkInterface() {
	trueVal := true
	entries := []iface.InterfaceEntry{
		{Name: "eth0", DHCP4: &trueVal},
	}
	entryData, _ := json.Marshal(entries)

	tests := []struct {
		name         string
		request      gen.GetNodeNetworkInterfaceRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeNetworkInterfaceResponseObject)
	}{
		{
			name: "when success",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
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
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.InterfaceListEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Interfaces)
				s.Len(*r.Results[0].Interfaces, 1)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job client error",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceListEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: entryData},
							"server2": {Hostname: "server2", Data: entryData},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
			},
		},
		{
			name: "when broadcast error",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
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
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceListEntryStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
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
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceListEntryStatusSkipped, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("unsupported", *r.Results[0].Error)
			},
		},
		{
			name: "when broadcast success with nil data",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: nil},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceListEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Interfaces)
				s.Empty(*r.Results[0].Interfaces)
			},
		},
		{
			name: "when success with full interface entry fields",
			request: gen.GetNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
			},
			setupMock: func() {
				fullEntries := []iface.InterfaceEntry{
					{
						Name:       "eth0",
						DHCP4:      &trueVal,
						DHCP6:      &trueVal,
						Addresses:  []string{"10.0.0.1/24"},
						Gateway4:   "10.0.0.1",
						Gateway6:   "fe80::1",
						MTU:        1500,
						MACAddress: "00:11:22:33:44:55",
						WakeOnLAN:  &trueVal,
					},
					{
						Name: "lo",
					},
				}
				data, _ := json.Marshal(fullEntries)
				s.mockJobClient.EXPECT().
					Query(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceList,
						nil,
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "server1",
							Data:     data,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Require().NotNil(r.Results[0].Interfaces)
				ifaces := *r.Results[0].Interfaces
				s.Len(ifaces, 2)

				// First entry has all optional fields populated.
				s.Equal("eth0", *ifaces[0].Name)
				s.Require().NotNil(ifaces[0].Dhcp4)
				s.True(*ifaces[0].Dhcp4)
				s.Require().NotNil(ifaces[0].Dhcp6)
				s.True(*ifaces[0].Dhcp6)
				s.Require().NotNil(ifaces[0].Addresses)
				s.Equal([]string{"10.0.0.1/24"}, *ifaces[0].Addresses)
				s.Require().NotNil(ifaces[0].Gateway4)
				s.Equal("10.0.0.1", *ifaces[0].Gateway4)
				s.Require().NotNil(ifaces[0].Gateway6)
				s.Equal("fe80::1", *ifaces[0].Gateway6)
				s.Require().NotNil(ifaces[0].Mtu)
				s.Equal(1500, *ifaces[0].Mtu)
				s.Require().NotNil(ifaces[0].MacAddress)
				s.Equal("00:11:22:33:44:55", *ifaces[0].MacAddress)
				s.Require().NotNil(ifaces[0].Wakeonlan)
				s.True(*ifaces[0].Wakeonlan)

				// Second entry has no optional fields.
				s.Equal("lo", *ifaces[1].Name)
				s.Nil(ifaces[1].Addresses)
				s.Nil(ifaces[1].Gateway4)
				s.Nil(ifaces[1].Gateway6)
				s.Nil(ifaces[1].Mtu)
				s.Nil(ifaces[1].MacAddress)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeNetworkInterface(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkInterfaceListGetPublicTestSuite) TestGetNetworkInterfaceListValidationHTTP() {
	trueVal := true
	entries := []iface.InterfaceEntry{
		{Name: "eth0", DHCP4: &trueVal},
	}
	entryData, _ := json.Marshal(entries)

	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/interface",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Data:     entryData,
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"server1"`},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/interface",
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

			req := httptest.NewRequest(
				http.MethodGet,
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

const rbacInterfaceListGetTestSigningKey = "test-signing-key-for-interface-list-get-rbac"

func (s *NetworkInterfaceListGetPublicTestSuite) TestGetNetworkInterfaceListRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	trueVal := true
	entries := []iface.InterfaceEntry{
		{Name: "eth0", DHCP4: &trueVal},
	}
	entryData, _ := json.Marshal(entries)

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
					rbacInterfaceListGetTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with network:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacInterfaceListGetTestSigningKey,
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
					Query(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceList, nil).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Data:     entryData,
					}, nil)
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
						Security: config.ServerSecurity{
							SigningKey: rbacInterfaceListGetTestSigningKey,
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
				"/api/node/server1/network/interface",
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

func TestNetworkInterfaceListGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkInterfaceListGetPublicTestSuite))
}
