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

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apiagent "github.com/osapi-io/osapi/internal/controller/api/agent"
	"github.com/osapi-io/osapi/internal/controller/api/agent/gen"
	jobtypes "github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type AgentDrainPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiagent.Agent
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *AgentDrainPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiagent.New(slog.Default(), s.mockJobClient, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *AgentDrainPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentDrainPublicTestSuite) TestDrainAgent() {
	tests := []struct {
		name            string
		hostname        string
		skipMock        bool
		mockAgent       *jobtypes.AgentInfo
		mockGetErr      error
		mockWriteErr    error
		skipWrite       bool
		mockSetDrain    bool
		mockSetDrainErr error
		validateFunc    func(resp gen.DrainAgentResponseObject)
	}{
		{
			name:      "returns 400 when hostname is empty",
			hostname:  "",
			skipMock:  true,
			skipWrite: true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				_, ok := resp.(gen.DrainAgent400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:      "returns 400 when hostname exceeds max length",
			hostname:  strings.Repeat("a", 256),
			skipMock:  true,
			skipWrite: true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				_, ok := resp.(gen.DrainAgent400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "success drains agent",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateReady,
			},
			mockSetDrain: true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				r, ok := resp.(gen.DrainAgent200JSONResponse)
				s.True(ok)
				s.Contains(r.Message, "drain initiated for agent server1")
			},
		},
		{
			name:       "agent not found returns 404",
			hostname:   "unknown",
			mockGetErr: fmt.Errorf("agent not found: unknown"),
			skipWrite:  true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				_, ok := resp.(gen.DrainAgent404JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "agent already draining returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateDraining,
			},
			skipWrite: true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				_, ok := resp.(gen.DrainAgent409JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "agent already cordoned returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateCordoned,
			},
			skipWrite: true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				_, ok := resp.(gen.DrainAgent409JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "when SetDrainFlag fails returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateReady,
			},
			mockSetDrain:    true,
			mockSetDrainErr: fmt.Errorf("kv connection failed"),
			skipWrite:       true,
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				r, ok := resp.(gen.DrainAgent409JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "failed to set drain flag")
			},
		},
		{
			name:     "when WriteAgentTimelineEvent returns not found error returns 404",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateReady,
			},
			mockSetDrain: true,
			mockWriteErr: fmt.Errorf("agent not found: server1"),
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				_, ok := resp.(gen.DrainAgent404JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "when WriteAgentTimelineEvent returns other error returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateReady,
			},
			mockSetDrain: true,
			mockWriteErr: fmt.Errorf("connection failed"),
			validateFunc: func(resp gen.DrainAgentResponseObject) {
				r, ok := resp.(gen.DrainAgent409JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "connection failed")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if !tt.skipMock {
				s.mockJobClient.EXPECT().
					GetAgent(gomock.Any(), tt.hostname).
					Return(tt.mockAgent, tt.mockGetErr)
			}

			if tt.mockSetDrain {
				s.mockJobClient.EXPECT().
					SetDrainFlag(gomock.Any(), "abc123").
					Return(tt.mockSetDrainErr)
			}

			if !tt.skipWrite {
				s.mockJobClient.EXPECT().
					WriteAgentTimelineEvent(gomock.Any(), tt.hostname, "drain", "Drain initiated via API").
					Return(tt.mockWriteErr)
			}

			resp, err := s.handler.DrainAgent(s.ctx, gen.DrainAgentRequestObject{
				Hostname: tt.hostname,
			})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AgentDrainPublicTestSuite) TestDrainAgentHTTP() {
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
				s.Contains(rec.Body.String(), "error")
			},
		},
		{
			name:     "when agent exists returns 200",
			hostname: "server1",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					GetAgent(gomock.Any(), "server1").
					Return(&jobtypes.AgentInfo{
						MachineID: "abc123",
						Hostname:  "server1",
						State:     jobtypes.AgentStateReady,
					}, nil)
				mock.EXPECT().
					SetDrainFlag(gomock.Any(), "abc123").
					Return(nil)
				mock.EXPECT().
					WriteAgentTimelineEvent(gomock.Any(), "server1", "drain", "Drain initiated via API").
					Return(nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "message")
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
				s.Contains(rec.Body.String(), "error")
			},
		},
		{
			name:     "when agent already draining returns 409",
			hostname: "server1",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					GetAgent(gomock.Any(), "server1").
					Return(&jobtypes.AgentInfo{
						MachineID: "abc123",
						Hostname:  "server1",
						State:     jobtypes.AgentStateDraining,
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusConflict, rec.Code)
				s.Contains(rec.Body.String(), "error")
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
				http.MethodPost,
				fmt.Sprintf("/api/agent/%s/drain", tc.hostname),
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacAgentDrainTestSigningKey = "test-signing-key-for-rbac-agent-drain"

func (s *AgentDrainPublicTestSuite) TestDrainAgentRBACHTTP() {
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
					rbacAgentDrainTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"agent:read"},
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
			name: "when valid token with agent:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAgentDrainTestSigningKey,
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
						MachineID: "abc123",
						Hostname:  "server1",
						State:     jobtypes.AgentStateReady,
					}, nil)
				mock.EXPECT().
					SetDrainFlag(gomock.Any(), "abc123").
					Return(nil)
				mock.EXPECT().
					WriteAgentTimelineEvent(gomock.Any(), "server1", "drain", "Drain initiated via API").
					Return(nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "message")
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
							SigningKey: rbacAgentDrainTestSigningKey,
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
				http.MethodPost,
				"/api/agent/server1/drain",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestAgentDrainPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentDrainPublicTestSuite))
}
