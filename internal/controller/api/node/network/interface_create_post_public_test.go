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

type NetworkInterfaceCreatePostPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkInterfaceCreatePostPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkInterfaceCreatePostPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkInterfaceCreatePostPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkInterfaceCreatePostPublicTestSuite) TestPostNodeNetworkInterface() {
	trueVal := true

	tests := []struct {
		name         string
		request      gen.PostNodeNetworkInterfaceRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostNodeNetworkInterfaceResponseObject)
	}{
		{
			name: "when success",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "eth0",
				Body: &gen.InterfaceConfigRequest{
					Dhcp4: &trueVal,
				},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceCreate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						&job.Response{
							Hostname: "server1",
							Changed:  &trueVal,
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Equal(gen.InterfaceMutationEntryStatusOk, r.Results[0].Status)
				s.Require().NotNil(r.Results[0].Changed)
				s.True(*r.Results[0].Changed)
			},
		},
		{
			name: "when validation error empty hostname",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation error empty name",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "when job client error",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceCreate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				_, ok := resp.(gen.PostNodeNetworkInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceCreate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceMutationEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceCreate,
						gomock.Any(),
					).
					Return(
						"550e8400-e29b-41d4-a716-446655440000",
						map[string]*job.Response{
							"server1": {Hostname: "server1", Changed: &trueVal},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				s.NotNil(resp)
			},
		},
		{
			name: "when broadcast error",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceCreate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				_, ok := resp.(gen.PostNodeNetworkInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when broadcast with failed and skipped hosts",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "_all",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					ModifyBroadcast(
						gomock.Any(),
						"_all",
						"network",
						job.OperationNetworkInterfaceCreate,
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
							"server2": {
								Status:   job.StatusSkipped,
								Error:    "unsupported",
								Hostname: "server2",
							},
						},
						nil,
					)
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				statuses := map[gen.InterfaceMutationEntryStatus]bool{}
				for _, item := range r.Results {
					statuses[item.Status] = true
					s.Require().NotNil(item.Error)
					s.Require().NotNil(item.Changed)
					s.False(*item.Changed)
				}
				s.True(statuses[gen.InterfaceMutationEntryStatusFailed])
				s.True(statuses[gen.InterfaceMutationEntryStatusSkipped])
			},
		},
		{
			name: "when all optional fields provided",
			request: gen.PostNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "eth0",
				Body: func() *gen.InterfaceConfigRequest {
					dhcp6 := false
					addrs := []string{"10.0.0.1/24"}
					gw4 := "10.0.0.1"
					gw6 := "fe80::1"
					mtu := 1500
					mac := "00:11:22:33:44:55"
					wol := true
					return &gen.InterfaceConfigRequest{
						Dhcp4:      &trueVal,
						Dhcp6:      &dhcp6,
						Addresses:  &addrs,
						Gateway4:   &gw4,
						Gateway6:   &gw6,
						Mtu:        &mtu,
						MacAddress: &mac,
						Wakeonlan:  &wol,
					}
				}(),
			},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Modify(
						gomock.Any(),
						"server1",
						"network",
						job.OperationNetworkInterfaceCreate,
						gomock.Any(),
					).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						_ string,
						_ string,
						data interface{},
					) (string, *job.Response, error) {
						d := data.(map[string]any)
						s.Equal("eth0", d["name"])
						s.Equal(true, d["dhcp4"])
						s.Equal(false, d["dhcp6"])
						s.Equal([]string{"10.0.0.1/24"}, d["addresses"])
						s.Equal("10.0.0.1", d["gateway4"])
						s.Equal("fe80::1", d["gateway6"])
						s.Equal(1500, d["mtu"])
						s.Equal("00:11:22:33:44:55", d["mac_address"])
						s.Equal(true, d["wakeonlan"])
						return "550e8400-e29b-41d4-a716-446655440000", &job.Response{
							Hostname: "server1",
							Changed:  &trueVal,
						}, nil
					})
			},
			validateFunc: func(resp gen.PostNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PostNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceMutationEntryStatusOk, r.Results[0].Status)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostNodeNetworkInterface(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkInterfaceCreatePostPublicTestSuite) TestPostNetworkInterfaceValidationHTTP() {
	trueVal := true

	tests := []struct {
		name         string
		path         string
		body         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid request",
			path: "/api/node/server1/network/interface/eth0",
			body: `{"dhcp4":true}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"server1"`},
		},
		{
			name: "when invalid address CIDR",
			path: "/api/node/server1/network/interface/eth0",
			body: `{"addresses":["not-cidr"]}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`, "Addresses", "cidr"},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/interface/eth0",
			body: `{"dhcp4":true}`,
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
				http.MethodPost,
				tc.path,
				strings.NewReader(tc.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacInterfaceCreateTestSigningKey = "test-signing-key-for-interface-create-rbac"

func (s *NetworkInterfaceCreatePostPublicTestSuite) TestPostNetworkInterfaceRBACHTTP() {
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
					rbacInterfaceCreateTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with network:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacInterfaceCreateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceCreate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"changed":true`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacInterfaceCreateTestSigningKey,
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
				http.MethodPost,
				"/api/node/server1/network/interface/eth0",
				strings.NewReader(`{"dhcp4":true}`),
			)
			req.Header.Set("Content-Type", "application/json")
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

func TestNetworkInterfaceCreatePostPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkInterfaceCreatePostPublicTestSuite))
}
