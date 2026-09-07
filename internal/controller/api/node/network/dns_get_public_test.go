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
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
	"github.com/osapi-io/osapi/internal/validation"
)

type NetworkDNSGetByInterfacePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkDNSGetByInterfacePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkDNSGetByInterfacePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkDNSGetByInterfacePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkDNSGetByInterfacePublicTestSuite) TestGetNodeNetworkDNSByInterface() {
	tests := []struct {
		name         string
		request      gen.GetNodeNetworkDNSByInterfaceRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject)
	}{
		{
			name: "when success",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_any",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				dnsResult := dns.GetResult{
					DNSServers:    []string{"8.8.8.8"},
					SearchDomains: []string{"example.com"},
				}
				data, _ := json.Marshal(dnsResult)
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Data:     json.RawMessage(data),
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Servers)
				s.Equal([]string{"8.8.8.8"}, *r.Results[0].Servers)
				s.Require().NotNil(r.Results[0].SearchDomains)
				s.Equal([]string{"example.com"}, *r.Results[0].SearchDomains)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "",
				InterfaceName: "eth0",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation error empty interface name",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_any",
				InterfaceName: "",
			},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when job client error",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_any",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkDNSByInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "server1",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.DNSConfigResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_all",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				dns1 := dns.GetResult{
					DNSServers:    []string{"8.8.8.8"},
					SearchDomains: []string{"example.com"},
				}
				dns2 := dns.GetResult{
					DNSServers:    []string{"1.1.1.1"},
					SearchDomains: []string{"test.com"},
				}
				data1, _ := json.Marshal(dns1)
				data2, _ := json.Marshal(dns2)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
							"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				for _, result := range r.Results {
					s.Require().NotNil(result.Changed)
					s.False(*result.Changed)
				}
			},
		},
		{
			name: "when broadcast all with errors",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_all",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				dns1 := dns.GetResult{
					DNSServers:    []string{"8.8.8.8"},
					SearchDomains: []string{"example.com"},
				}
				data1, _ := json.Marshal(dns1)
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "interface not found",
								Hostname: "server2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface200JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, h := range r.Results {
					if h.Error != nil {
						foundError = true
						s.Equal("server2", h.Hostname)
						s.Equal("interface not found", *h.Error)
					}
				}
				s.True(foundError)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_all",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "host: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.DNSConfigResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_all",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				r, ok := resp.(gen.GetNodeNetworkDNSByInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.DNSConfigResponseStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all error",
			request: gen.GetNodeNetworkDNSByInterfaceRequestObject{
				Hostname:      "_all",
				InterfaceName: "eth0",
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeNetworkDNSByInterfaceResponseObject) {
				_, ok := resp.(gen.GetNodeNetworkDNSByInterface500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeNetworkDNSByInterface(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkDNSGetByInterfacePublicTestSuite) TestGetNetworkDNSByInterfaceValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/dns/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				dnsResult := dns.GetResult{
					DNSServers:    []string{"8.8.8.8"},
					SearchDomains: []string{"example.com"},
				}
				data, _ := json.Marshal(dnsResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
				return mock
			},
			wantCode: http.StatusOK,
			wantContains: []string{
				`"results"`,
				`"servers"`,
				`"8.8.8.8"`,
				`"search_domains"`,
				`"example.com"`,
			},
		},
		{
			name: "when fact reference interface name passes validation",
			path: "/api/node/server1/network/dns/@fact.interface.primary",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				dnsResult := dns.GetResult{
					DNSServers: []string{"8.8.8.8"},
				}
				data, _ := json.Marshal(dnsResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"8.8.8.8"`},
		},
		{
			name: "when partial fact reference rejected",
			path: "/api/node/server1/network/dns/@fact",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "alphanum_or_fact"},
		},
		{
			name: "when non-alphanum interface name",
			path: "/api/node/server1/network/dns/eth-0!",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "alphanum_or_fact"},
		},
		{
			name: "when unknown fact key rejected",
			path: "/api/node/server1/network/dns/@fact.primary_interface",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "alphanum_or_fact"},
		},
		{
			name: "when broadcast all",
			path: "/api/node/_all/network/dns/eth0",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				dnsResult := dns.GetResult{
					DNSServers:    []string{"8.8.8.8"},
					SearchDomains: []string{"example.com"},
				}
				data, _ := json.Marshal(dnsResult)
				mock.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data)},
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"8.8.8.8"`},
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

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacDNSGetTestSigningKey = "test-signing-key-for-dns-get-rbac"

func (s *NetworkDNSGetByInterfacePublicTestSuite) TestGetNetworkDNSByInterfaceRBACHTTP() {
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
					rbacDNSGetTestSigningKey,
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
			name: "when valid token with network:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacDNSGetTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				dnsResult := dns.GetResult{
					DNSServers:    []string{"8.8.8.8"},
					SearchDomains: []string{"example.com"},
				}
				data, _ := json.Marshal(dnsResult)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "network", job.OperationNetworkDNSGet, gomock.Any()).
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
			wantContains: []string{`"results"`, `"8.8.8.8"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacDNSGetTestSigningKey,
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

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/network/dns/eth0", nil)
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

func TestNetworkDNSGetByInterfacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkDNSGetByInterfacePublicTestSuite))
}
