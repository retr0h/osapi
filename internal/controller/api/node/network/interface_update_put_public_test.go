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

type NetworkInterfaceUpdatePutPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apinetwork.Network
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *NetworkInterfaceUpdatePutPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *NetworkInterfaceUpdatePutPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apinetwork.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *NetworkInterfaceUpdatePutPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkInterfaceUpdatePutPublicTestSuite) TestPutNodeNetworkInterface() {
	trueVal := true

	tests := []struct {
		name         string
		request      gen.PutNodeNetworkInterfaceRequestObject
		setupMock    func()
		validateFunc func(resp gen.PutNodeNetworkInterfaceResponseObject)
	}{
		{
			name: "when success",
			request: gen.PutNodeNetworkInterfaceRequestObject{
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
						job.OperationNetworkInterfaceUpdate,
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
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkInterface200JSONResponse)
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
			request: gen.PutNodeNetworkInterfaceRequestObject{
				Hostname: "",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name: "when validation error empty name",
			request: gen.PutNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "",
				Body:     &gen.InterfaceConfigRequest{Dhcp4: &trueVal},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
			},
		},
		{
			name: "when body struct validation error invalid address",
			request: gen.PutNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "eth0",
				Body: &gen.InterfaceConfigRequest{
					Addresses: &[]string{"not-a-cidr"},
				},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "Addresses")
			},
		},
		{
			name: "when at least one field error empty body",
			request: gen.PutNodeNetworkInterfaceRequestObject{
				Hostname: "server1",
				Name:     "eth0",
				Body:     &gen.InterfaceConfigRequest{},
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkInterface400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "at least one")
			},
		},
		{
			name: "when job client error",
			request: gen.PutNodeNetworkInterfaceRequestObject{
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
						job.OperationNetworkInterfaceUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkInterface500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when job skipped",
			request: gen.PutNodeNetworkInterfaceRequestObject{
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
						job.OperationNetworkInterfaceUpdate,
						gomock.Any(),
					).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				r, ok := resp.(gen.PutNodeNetworkInterface200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal(gen.InterfaceMutationEntryStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name: "when broadcast all success",
			request: gen.PutNodeNetworkInterfaceRequestObject{
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
						job.OperationNetworkInterfaceUpdate,
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
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				s.NotNil(resp)
			},
		},
		{
			name: "when broadcast error",
			request: gen.PutNodeNetworkInterfaceRequestObject{
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
						job.OperationNetworkInterfaceUpdate,
						gomock.Any(),
					).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.PutNodeNetworkInterfaceResponseObject) {
				_, ok := resp.(gen.PutNodeNetworkInterface500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PutNodeNetworkInterface(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *NetworkInterfaceUpdatePutPublicTestSuite) TestPutNetworkInterfaceValidationHTTP() {
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
			path: "/api/node/server1/network/interface/eth0",
			body: `{"dhcp4":true}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"results"`)
				s.Contains(rec.Body.String(), `"server1"`)
			},
		},
		{
			name: "when empty body",
			path: "/api/node/server1/network/interface/eth0",
			body: `{}`,
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
				s.Contains(rec.Body.String(), "at least one")
			},
		},
		{
			name: "when target agent not found",
			path: "/api/node/nonexistent/network/interface/eth0",
			body: `{"dhcp4":true}`,
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

const rbacInterfaceUpdateTestSigningKey = "test-signing-key-for-interface-update-rbac"

func (s *NetworkInterfaceUpdatePutPublicTestSuite) TestPutNetworkInterfaceRBACHTTP() {
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
					rbacInterfaceUpdateTestSigningKey,
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
			name: "when valid token with network:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacInterfaceUpdateTestSigningKey,
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
					Modify(gomock.Any(), "server1", "network", job.OperationNetworkInterfaceUpdate, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "server1",
						Changed:  &trueVal,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"results"`)
				s.Contains(rec.Body.String(), `"changed":true`)
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
							SigningKey: rbacInterfaceUpdateTestSigningKey,
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
				"/api/node/server1/network/interface/eth0",
				strings.NewReader(`{"dhcp4":true}`),
			)
			req.Header.Set("Content-Type", "application/json")
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestNetworkInterfaceUpdatePutPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NetworkInterfaceUpdatePutPublicTestSuite))
}
