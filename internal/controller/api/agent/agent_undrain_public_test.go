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

type AgentUndrainPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiagent.Agent
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *AgentUndrainPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiagent.New(slog.Default(), s.mockJobClient, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *AgentUndrainPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentUndrainPublicTestSuite) TestUndrainAgent() {
	tests := []struct {
		name               string
		hostname           string
		skipMock           bool
		mockAgent          *jobtypes.AgentInfo
		mockGetErr         error
		mockWriteErr       error
		skipWrite          bool
		mockDeleteDrain    bool
		mockDeleteDrainErr error
		validateFunc       func(resp gen.UndrainAgentResponseObject)
	}{
		{
			name:      "returns 400 when hostname is empty",
			hostname:  "",
			skipMock:  true,
			skipWrite: true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				_, ok := resp.(gen.UndrainAgent400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:      "returns 400 when hostname exceeds max length",
			hostname:  strings.Repeat("a", 256),
			skipMock:  true,
			skipWrite: true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				_, ok := resp.(gen.UndrainAgent400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "success undrains draining agent",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateDraining,
			},
			mockDeleteDrain: true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				r, ok := resp.(gen.UndrainAgent200JSONResponse)
				s.True(ok)
				s.Contains(r.Message, "undrain initiated for agent server1")
			},
		},
		{
			name:     "success undrains cordoned agent",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateCordoned,
			},
			mockDeleteDrain: true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				r, ok := resp.(gen.UndrainAgent200JSONResponse)
				s.True(ok)
				s.Contains(r.Message, "undrain initiated for agent server1")
			},
		},
		{
			name:       "agent not found returns 404",
			hostname:   "unknown",
			mockGetErr: fmt.Errorf("agent not found: unknown"),
			skipWrite:  true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				_, ok := resp.(gen.UndrainAgent404JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "agent in ready state returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateReady,
			},
			skipWrite: true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				_, ok := resp.(gen.UndrainAgent409JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "agent with empty state returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     "",
			},
			skipWrite: true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				_, ok := resp.(gen.UndrainAgent409JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "when DeleteDrainFlag fails returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateDraining,
			},
			mockDeleteDrain:    true,
			mockDeleteDrainErr: fmt.Errorf("kv connection failed"),
			skipWrite:          true,
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				r, ok := resp.(gen.UndrainAgent409JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "failed to delete drain flag")
			},
		},
		{
			name:     "when WriteAgentTimelineEvent returns not found error returns 404",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateDraining,
			},
			mockDeleteDrain: true,
			mockWriteErr:    fmt.Errorf("agent not found: server1"),
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				_, ok := resp.(gen.UndrainAgent404JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "when WriteAgentTimelineEvent returns other error returns 409",
			hostname: "server1",
			mockAgent: &jobtypes.AgentInfo{
				MachineID: "abc123",
				Hostname:  "server1",
				State:     jobtypes.AgentStateDraining,
			},
			mockDeleteDrain: true,
			mockWriteErr:    fmt.Errorf("connection failed"),
			validateFunc: func(resp gen.UndrainAgentResponseObject) {
				r, ok := resp.(gen.UndrainAgent409JSONResponse)
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

			if tt.mockDeleteDrain {
				s.mockJobClient.EXPECT().
					DeleteDrainFlag(gomock.Any(), "abc123").
					Return(tt.mockDeleteDrainErr)
			}

			if !tt.skipWrite {
				s.mockJobClient.EXPECT().
					WriteAgentTimelineEvent(gomock.Any(), tt.hostname, "undrain", "Undrain initiated via API").
					Return(tt.mockWriteErr)
			}

			resp, err := s.handler.UndrainAgent(s.ctx, gen.UndrainAgentRequestObject{
				Hostname: tt.hostname,
			})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AgentUndrainPublicTestSuite) TestUndrainAgentHTTP() {
	tests := []struct {
		name         string
		hostname     string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name:     "when hostname exceeds max length returns 400",
			hostname: strings.Repeat("a", 256),
			setupJobMock: func() *jobmocks.MockJobClient {
				return jobmocks.NewMockJobClient(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
		{
			name:     "when draining agent exists returns 200",
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
				mock.EXPECT().
					DeleteDrainFlag(gomock.Any(), "abc123").
					Return(nil)
				mock.EXPECT().
					WriteAgentTimelineEvent(gomock.Any(), "server1", "undrain", "Undrain initiated via API").
					Return(nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"message"`, `undrain initiated`},
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
			wantCode:     http.StatusNotFound,
			wantContains: []string{`"error"`},
		},
		{
			name:     "when agent in ready state returns 409",
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
				return mock
			},
			wantCode:     http.StatusConflict,
			wantContains: []string{`"error"`, `not in draining or cordoned`},
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
				fmt.Sprintf("/api/agent/%s/undrain", tc.hostname),
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

const rbacAgentUndrainTestSigningKey = "test-signing-key-for-rbac-agent-undrain"

func (s *AgentUndrainPublicTestSuite) TestUndrainAgentRBACHTTP() {
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
					rbacAgentUndrainTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with agent:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAgentUndrainTestSigningKey,
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
						State:     jobtypes.AgentStateDraining,
					}, nil)
				mock.EXPECT().
					DeleteDrainFlag(gomock.Any(), "abc123").
					Return(nil)
				mock.EXPECT().
					WriteAgentTimelineEvent(gomock.Any(), "server1", "undrain", "Undrain initiated via API").
					Return(nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"message"`, `undrain initiated`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacAgentUndrainTestSigningKey,
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
				"/api/agent/server1/undrain",
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

func TestAgentUndrainPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentUndrainPublicTestSuite))
}
