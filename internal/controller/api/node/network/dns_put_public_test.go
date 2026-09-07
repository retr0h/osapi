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

type NetworkDNSPutByInterfacePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkDNSPutByInterfacePublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkDNSPutByInterfacePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkDNSPutByInterfacePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkDNSPutByInterfacePublicTestSuite) TestPutNodeNetworkDNS() {
	trueVal := true
	falseVal := false

	tests := []struct {
		name         string
		request      gen.PutNodeNetworkDNSRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeNetworkDNSResponseObject)
	}{
		{
			name: "when success",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
					SearchDomains: &[]string{"example.com"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Changed:  &trueVal,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.Equal(gen.DNSUpdateResultItemStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when override_dhcp true passes to job data",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
					OverrideDhcp:  &trueVal,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.AssignableToTypeOf(map[string]any{}),
					).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						_ string,
						_ string,
						data map[string]any,
					) (string, *job.Response, error) {
						s.Equal(true, data["override_dhcp"])
						return "550e8400-e29b-41d4-a716-446655440000",
							&job.Response{
								Hostname: "agent1",
								Changed:  &trueVal,
							}, nil
					})
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when override_dhcp false passes false to job data",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
					OverrideDhcp:  &falseVal,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.AssignableToTypeOf(map[string]any{}),
					).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						_ string,
						_ string,
						data map[string]any,
					) (string, *job.Response, error) {
						s.Equal(false, data["override_dhcp"])
						return "550e8400-e29b-41d4-a716-446655440000",
							&job.Response{
								Hostname: "agent1",
								Changed:  &trueVal,
							}, nil
					})
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when body validation error empty interface name",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "when job client error",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_any",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"_any",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkDNS500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "server1",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.DNSUpdateResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
					SearchDomains: &[]string{"example.com"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Changed: &trueVal},
							"server2": {Hostname: "server2", Changed: &falseVal},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				s.NotNil(resp)
			},
		},
		{
			name: "when broadcast all with errors",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
					SearchDomains: &[]string{"example.com"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Changed: &trueVal},
							"server2": {
								Status:   job.StatusFailed,
								Error:    "permission denied",
								Hostname: "server2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Len(r.Results, 2)
				var foundError bool
				for _, item := range r.Results {
					if item.Error != nil {
						foundError = true
						s.Equal("server2", item.Hostname)
						s.Equal(gen.DNSUpdateResultItemStatusFailed, item.Status)
					}
				}
				s.True(foundError)
			},
		},
		{
			name: "when broadcast with skipped host",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {
								Status:   job.StatusSkipped,
								Error:    "host: operation not supported on this OS family",
								Hostname: "server1",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.DNSUpdateResultItemStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast with failed host",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkDNSUpdate,
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
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkDNS202JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.DNSUpdateResultItemStatusFailed, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all error",
			request: gen.PutNodeNetworkDNSRequestObject{
				Hostname: "_all",
				Body: &gen.PutNodeNetworkDNSJSONRequestBody{
					InterfaceName: "eth0",
					Servers:       &[]string{"1.1.1.1"},
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkDNSUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNetworkDNSResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkDNS500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PutNodeNetworkDNS(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkDNSPutByInterfacePublicTestSuite) TestPutNetworkDNSValidationHTTP() {
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
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1","8.8.8.8"],"search_domains":["foo.bar"],"interface_name":"eth0"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkDNSUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "agent1")
				s.Contains(rec.Body.String(), "ok")
				s.Contains(rec.Body.String(), "changed")
			},
		},
		{
			name: "when missing interface name",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1"]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "InterfaceName")
				s.Contains(rec.Body.String(), "required")
			},
		},
		{
			name: "when fact reference interface name passes validation",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1"],"interface_name":"@fact.interface.primary"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkDNSUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "agent1")
			},
		},
		{
			name: "when partial fact reference rejected",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1"],"interface_name":"@fact"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "InterfaceName")
				s.Contains(rec.Body.String(), "alphanum_or_fact")
			},
		},
		{
			name: "when non-alphanum interface name",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1"],"interface_name":"eth-0!"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "InterfaceName")
				s.Contains(rec.Body.String(), "alphanum_or_fact")
			},
		},
		{
			name: "when invalid server IP",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["not-an-ip"],"interface_name":"eth0"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "Servers")
				s.Contains(rec.Body.String(), "ip")
			},
		},
		{
			name: "when invalid search domain",
			path: "/api/node/server1/network/dns",
			body: `{"search_domains":["not a valid hostname!"],"interface_name":"eth0"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "SearchDomains")
				s.Contains(rec.Body.String(), "hostname")
			},
		},
		{
			name: "when unknown fact key interface rejected",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1"],"interface_name":"@fact.primary_interface"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "InterfaceName")
				s.Contains(rec.Body.String(), "alphanum_or_fact")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/dns",
			body: `{"servers":["1.1.1.1"],"interface_name":"eth0"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
				s.Contains(rec.Body.String(), "valid_target")
				s.Contains(rec.Body.String(), "not found")
			},
		},
		{
			name: "when override_dhcp true in HTTP request",
			path: "/api/node/server1/network/dns",
			body: `{"servers":["1.1.1.1"],"interface_name":"eth0","override_dhcp":true}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkDNSUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "agent1")
				s.Contains(rec.Body.String(), "changed")
			},
		},
		{
			name: "when broadcast all",
			path: "/api/node/_all/network/dns",
			body: `{"servers":["1.1.1.1"],"search_domains":["foo.bar"],"interface_name":"eth0"}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					ModifyBroadcast(gomock.Any(), "_all", "network", job.OperationNetworkDNSUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Changed: &trueVal},
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "server1")
				s.Contains(rec.Body.String(), "changed")
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

			req := httptest.NewRequest(
				http.MethodPut,
				tc.path,
				strings.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacDNSPutTestSigningKey = "test-signing-key-for-dns-put-rbac"

func (s *NetworkDNSPutByInterfacePublicTestSuite) TestPutNetworkDNSRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	trueVal := true

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
					rbacDNSPutTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"network:read"},
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
			name: "when valid token with network:write returns 202",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacDNSPutTestSigningKey,
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
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkDNSUpdate, gomock.Any()).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "agent1",
							Changed:  &trueVal,
						},
						nil,
					)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusAccepted, rec.Code)
				s.Contains(rec.Body.String(), "results")
				s.Contains(rec.Body.String(), "changed")
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
							SigningKey: rbacDNSPutTestSigningKey,
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
				http.MethodPut,
				"/api/node/server1/network/dns",
				strings.NewReader(`{"servers":["8.8.8.8"],"interface_name":"eth0"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNetworkDNSPutByInterfacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkDNSPutByInterfacePublicTestSuite))
}
