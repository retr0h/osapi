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

package hostname_test

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
	apihostname "github.com/osapi-io/osapi/internal/controller/api/node/hostname"
	"github.com/osapi-io/osapi/internal/controller/api/node/hostname/gen"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/validation"
)

type HostnameGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apihostname.Hostname
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *HostnameGetPublicTestSuite) SetupSuite() {
	validation.RegisterTargetValidator(func(_ context.Context) ([]validation.AgentTarget, error) {
		return []validation.AgentTarget{
			{Hostname: "server1", Labels: map[string]string{"group": "web"}},
			{Hostname: "server2"},
		}, nil
	})
}

func (s *HostnameGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apihostname.New(slog.Default(), s.mockJobClient)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *HostnameGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HostnameGetPublicTestSuite) TestGetNodeHostname() {
	tests := []struct {
		name         string
		request      gen.GetNodeHostnameRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetNodeHostnameResponseObject)
	}{
		{
			name:    "success",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_any"},
			setupMock: func() {
				data, _ := json.Marshal(map[string]any{
					"hostname": "my-hostname",
					"labels":   map[string]string{"group": "web"},
				})
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("my-hostname", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Labels)
				s.Equal(map[string]string{"group": "web"}, *r.Results[0].Labels)
				s.Require().NotNil(r.Results[0].Changed)
				s.False(*r.Results[0].Changed)
			},
		},
		{
			name:    "empty hostname falls back to agent hostname",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_any"},
			setupMock: func() {
				data, _ := json.Marshal(map[string]any{
					"hostname": "",
				})
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("agent1", r.Results[0].Hostname)
			},
		},
		{
			name:      "validation error empty hostname",
			request:   gen.GetNodeHostnameRequestObject{Hostname: ""},
			setupMock: func() {},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "required")
			},
		},
		{
			name:    "job client error",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_any"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "_any", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				_, ok := resp.(gen.GetNodeHostname500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when job skipped",
			request: gen.GetNodeHostnameRequestObject{Hostname: "server1"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Status:   job.StatusSkipped,
						Hostname: "server1",
						Error:    "host: operation not supported on this OS family",
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.HostnameResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all success",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_all"},
			setupMock: func() {
				data1, _ := json.Marshal(map[string]any{
					"hostname": "host1",
					"labels":   map[string]string{"group": "web"},
				})
				data2, _ := json.Marshal(map[string]any{
					"hostname": "host2",
				})
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 2)
				for _, result := range r.Results {
					s.Require().NotNil(result.Changed)
					s.False(*result.Changed)
				}
			},
		},
		{
			name:    "broadcast all with errors",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_all"},
			setupMock: func() {
				data1, _ := json.Marshal(map[string]any{
					"hostname": "host1",
				})
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {
							Status:   job.StatusFailed,
							Error:    "interface not found",
							Hostname: "server2",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
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
			name:    "broadcast with skipped host",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusSkipped,
							Error:    "host: operation not supported on this OS family",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("host: operation not supported on this OS family", *r.Results[0].Error)
				s.Equal(gen.HostnameResponseStatusSkipped, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast with failed host",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {
							Status:   job.StatusFailed,
							Error:    "permission denied",
							Hostname: "server1",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				r, ok := resp.(gen.GetNodeHostname200JSONResponse)
				s.True(ok)
				s.Require().Len(r.Results, 1)
				s.Equal("server1", r.Results[0].Hostname)
				s.Require().NotNil(r.Results[0].Error)
				s.Equal("permission denied", *r.Results[0].Error)
				s.Equal(gen.HostnameResponseStatusFailed, r.Results[0].Status)
			},
		},
		{
			name:    "broadcast all error",
			request: gen.GetNodeHostnameRequestObject{Hostname: "_all"},
			setupMock: func() {
				s.mockJobClient.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("", nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetNodeHostnameResponseObject) {
				_, ok := resp.(gen.GetNodeHostname500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetNodeHostname(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *HostnameGetPublicTestSuite) TestGetNodeHostnameValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantBody     string
		wantContains []string
	}{
		{
			name: "when empty hostname returns 400",
			path: "/api/node/%20/hostname",
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
		{
			name: "when get Ok",
			path: "/api/node/server1/hostname",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				data, _ := json.Marshal(map[string]any{
					"hostname": "default-hostname",
				})
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", &job.Response{
						Hostname: "agent1",
						Data:     json.RawMessage(data),
					}, nil)
				return mock
			},
			wantCode: http.StatusOK,
			wantBody: `{"job_id":"550e8400-e29b-41d4-a716-446655440000","results":[{"changed":false,"hostname":"default-hostname","status":"ok"}]}`,
		},
		{
			name: "when job client errors",
			path: "/api/node/server1/hostname",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("", nil, assert.AnError)
				return mock
			},
			wantCode: http.StatusInternalServerError,
			wantBody: `{"error":"assert.AnError general error for testing"}`,
		},
		{
			name: "when broadcast all",
			path: "/api/node/_all/hostname",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				data1, _ := json.Marshal(map[string]any{"hostname": "host1"})
				data2, _ := json.Marshal(map[string]any{"hostname": "host2"})
				mock.EXPECT().
					QueryBroadcast(gomock.Any(), "_all", "node", job.OperationNodeHostnameGet, gomock.Any()).
					Return("550e8400-e29b-41d4-a716-446655440000", map[string]*job.Response{
						"server1": {Hostname: "server1", Data: json.RawMessage(data1)},
						"server2": {Hostname: "server2", Data: json.RawMessage(data2)},
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"results"`, `"host1"`, `"host2"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			hostnameHandler := apihostname.New(s.logger, jobMock)
			strictHandler := gen.NewStrictHandler(hostnameHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			if tc.wantBody != "" {
				s.JSONEq(tc.wantBody, rec.Body.String())
			}
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *HostnameGetPublicTestSuite) TestGetNodeHostnameRBACHTTP() {
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
					rbacTestSigningKey,
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
			name: "when valid token with node:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				data, _ := json.Marshal(map[string]any{"hostname": "test-host"})
				mock.EXPECT().
					Query(gomock.Any(), "server1", "node", job.OperationNodeHostnameGet, gomock.Any()).
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
			wantContains: []string{`"hostname":"test-host"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apihostname.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/node/server1/hostname", nil)
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

func TestHostnameGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnameGetPublicTestSuite))
}
