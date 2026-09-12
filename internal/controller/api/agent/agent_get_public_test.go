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

package agent_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apiagent "github.com/osapi-io/osapi/internal/controller/api/agent"
	"github.com/osapi-io/osapi/internal/controller/api/agent/gen"
	jobtypes "github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/host"
	"github.com/osapi-io/osapi/internal/provider/node/load"
	"github.com/osapi-io/osapi/internal/provider/node/mem"
)

type AgentGetPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiagent.Agent
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *AgentGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiagent.New(slog.Default(), s.mockJobClient, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *AgentGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentGetPublicTestSuite) TestGetAgentDetails() {
	tests := []struct {
		name         string
		hostname     string
		skipMock     bool
		mockAgent    *jobtypes.AgentInfo
		mockError    error
		validateFunc func(resp gen.GetAgentDetailsResponseObject)
	}{
		{
			name:     "returns 400 when hostname is empty",
			hostname: "",
			skipMock: true,
			validateFunc: func(resp gen.GetAgentDetailsResponseObject) {
				_, ok := resp.(gen.GetAgentDetails400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "returns 400 when hostname exceeds max length",
			hostname: strings.Repeat("a", 256),
			skipMock: true,
			validateFunc: func(resp gen.GetAgentDetailsResponseObject) {
				_, ok := resp.(gen.GetAgentDetails400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "success returns agent details",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				Hostname:     "server1",
				Labels:       map[string]string{"group": "web"},
				RegisteredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				StartedAt:    time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
				OSInfo:       &host.Result{Distribution: "Ubuntu", Version: "24.04"},
				Uptime:       5 * time.Hour,
				LoadAverages: &load.Result{Load1: 0.5, Load5: 0.3, Load15: 0.2},
				MemoryStats:  &mem.Result{Total: 8388608, Free: 4194304, Cached: 2097152},
			},
			validateFunc: func(resp gen.GetAgentDetailsResponseObject) {
				r, ok := resp.(gen.GetAgentDetails200JSONResponse)
				s.True(ok)
				s.Equal("server1", r.Hostname)
				s.Equal(gen.AgentInfoStatusReady, r.Status)
				s.NotNil(r.Labels)
				s.NotNil(r.OsInfo)
				s.Equal("Ubuntu", r.OsInfo.Distribution)
				s.NotNil(r.LoadAverage)
				s.NotNil(r.Memory)
				s.NotNil(r.Uptime)
			},
		},
		{
			name:      "agent not found returns 404",
			hostname:  "unknown",
			mockError: fmt.Errorf("agent not found: unknown"),
			validateFunc: func(resp gen.GetAgentDetailsResponseObject) {
				_, ok := resp.(gen.GetAgentDetails404JSONResponse)
				s.True(ok)
			},
		},
		{
			name:      "client error returns 500",
			hostname:  "server1",
			mockError: fmt.Errorf("connection failed"),
			validateFunc: func(resp gen.GetAgentDetailsResponseObject) {
				_, ok := resp.(gen.GetAgentDetails500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if !tt.skipMock {
				s.mockJobClient.EXPECT().
					GetAgent(gomock.Any(), tt.hostname).
					Return(tt.mockAgent, tt.mockError)
			}

			resp, err := s.handler.GetAgentDetails(s.ctx, gen.GetAgentDetailsRequestObject{
				Hostname: tt.hostname,
			})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AgentGetPublicTestSuite) TestGetAgentDetailsHTTP() {
	tests := []struct {
		name         string
		hostname     string
		setupJobMock func() *jobmocks.MockJobClient
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:     "when hostname exceeds max length returns 400",
			hostname: strings.Repeat("a", 256),
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
		{
			name:     "when agent exists returns details",
			hostname: "server1",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					GetAgent(gomock.Any(), "server1").
					Return(&jobtypes.AgentInfo{
						Hostname:     "server1",
						Labels:       map[string]string{"group": "web"},
						RegisteredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						StartedAt:    time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
						OSInfo:       &host.Result{Distribution: "Ubuntu", Version: "24.04"},
						Uptime:       5 * time.Hour,
						LoadAverages: &load.Result{Load1: 0.5, Load5: 0.3, Load15: 0.2},
						MemoryStats:  &mem.Result{Total: 8388608, Free: 4194304, Cached: 2097152},
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"server1"`)
				s.Contains(rec.Body.String(), `"Ready"`)
				s.Contains(rec.Body.String(), `"Ubuntu"`)
			},
		},
		{
			name:     "when agent not found returns 404",
			hostname: "unknown",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					GetAgent(gomock.Any(), "unknown").
					Return(nil, fmt.Errorf("agent not found: unknown"))
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusNotFound, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
		{
			name:     "when client error returns 500",
			hostname: "server1",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					GetAgent(gomock.Any(), "server1").
					Return(nil, fmt.Errorf("connection failed"))
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			agentHandler := apiagent.New(s.logger, jobMock, nil)
			strictHandler := gen.NewStrictHandler(agentHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/api/agent/%s", tc.hostname),
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacAgentGetTestSigningKey = "test-signing-key-for-rbac-agent-get"

func (s *AgentGetPublicTestSuite) TestGetAgentDetailsRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

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
					rbacAgentGetTestSigningKey,
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
			name: "when valid token with agent:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAgentGetTestSigningKey,
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
					GetAgent(gomock.Any(), "server1").
					Return(&jobtypes.AgentInfo{
						Hostname: "server1",
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"server1"`)
				s.Contains(rec.Body.String(), `"Ready"`)
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
							SigningKey: rbacAgentGetTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apiagent.Handler(
				s.logger,
				jobMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/agent/server1",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestAgentGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentGetPublicTestSuite))
}
