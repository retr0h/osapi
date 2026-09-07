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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apiagent "github.com/osapi-io/osapi/internal/controller/api/agent"
	"github.com/osapi-io/osapi/internal/controller/api/agent/gen"
	agentmocks "github.com/osapi-io/osapi/internal/controller/api/agent/mocks"
	"github.com/osapi-io/osapi/internal/controller/enrollment"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type AgentPendingPublicTestSuite struct {
	suite.Suite

	mockCtrl       *gomock.Controller
	mockJobClient  *jobmocks.MockJobClient
	mockEnrollment *agentmocks.MockEnrollmentManager
	handler        *apiagent.Agent
	ctx            context.Context
	appConfig      config.Config
	logger         *slog.Logger
}

func (s *AgentPendingPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.mockEnrollment = agentmocks.NewMockEnrollmentManager(s.mockCtrl)
	s.handler = apiagent.New(slog.Default(), s.mockJobClient, s.mockEnrollment)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *AgentPendingPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentPendingPublicTestSuite) TestGetAgentsPending() {
	fixedTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		useNilEnroll bool
		mockPending  []enrollment.PendingAgent
		mockErr      error
		validateFunc func(resp gen.GetAgentsPendingResponseObject)
	}{
		{
			name:         "returns 500 when enrollment not enabled",
			useNilEnroll: true,
			validateFunc: func(resp gen.GetAgentsPendingResponseObject) {
				r, ok := resp.(gen.GetAgentsPending500JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "enrollment not enabled")
			},
		},
		{
			name:        "returns empty list when no pending agents",
			mockPending: nil,
			validateFunc: func(resp gen.GetAgentsPendingResponseObject) {
				r, ok := resp.(gen.GetAgentsPending200JSONResponse)
				s.True(ok)
				s.Empty(r.Agents)
				s.Equal(0, r.Total)
			},
		},
		{
			name: "returns pending agents",
			mockPending: []enrollment.PendingAgent{
				{
					MachineID:   "abc123",
					Hostname:    "web-01",
					Fingerprint: "SHA256:deadbeef",
					RequestedAt: fixedTime,
				},
			},
			validateFunc: func(resp gen.GetAgentsPendingResponseObject) {
				r, ok := resp.(gen.GetAgentsPending200JSONResponse)
				s.True(ok)
				s.Len(r.Agents, 1)
				s.Equal(1, r.Total)
				s.Equal("abc123", r.Agents[0].MachineId)
				s.Equal("web-01", r.Agents[0].Hostname)
				s.Equal("SHA256:deadbeef", r.Agents[0].Fingerprint)
				s.Equal(fixedTime, r.Agents[0].RequestedAt)
			},
		},
		{
			name:    "returns 500 on list error",
			mockErr: fmt.Errorf("kv connection failed"),
			validateFunc: func(resp gen.GetAgentsPendingResponseObject) {
				r, ok := resp.(gen.GetAgentsPending500JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "kv connection failed")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var handler *apiagent.Agent
			if tt.useNilEnroll {
				handler = apiagent.New(slog.Default(), s.mockJobClient, nil)
			} else {
				s.mockEnrollment.EXPECT().
					ListPending(gomock.Any()).
					Return(tt.mockPending, tt.mockErr)
				handler = s.handler
			}

			resp, err := handler.GetAgentsPending(s.ctx, gen.GetAgentsPendingRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AgentPendingPublicTestSuite) TestGetAgentsPendingHTTP() {
	tests := []struct {
		name         string
		setupMocks   func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when pending agents exist returns 200",
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				jm := jobmocks.NewMockJobClient(s.mockCtrl)
				em := agentmocks.NewMockEnrollmentManager(s.mockCtrl)
				em.EXPECT().
					ListPending(gomock.Any()).
					Return([]enrollment.PendingAgent{
						{
							MachineID:   "abc123",
							Hostname:    "web-01",
							Fingerprint: "SHA256:deadbeef",
							RequestedAt: time.Now(),
						},
					}, nil)
				return jm, em
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "agents")
				s.Contains(rec.Body.String(), "web-01")
				s.Contains(rec.Body.String(), "total")
			},
		},
		{
			name: "when list fails returns 500",
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				jm := jobmocks.NewMockJobClient(s.mockCtrl)
				em := agentmocks.NewMockEnrollmentManager(s.mockCtrl)
				em.EXPECT().
					ListPending(gomock.Any()).
					Return(nil, fmt.Errorf("kv error"))
				return jm, em
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "error")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jm, em := tc.setupMocks()

			agentHandler := apiagent.New(s.logger, jm, em)
			strictHandler := gen.NewStrictHandler(agentHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/agent/pending", nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacAgentPendingTestSigningKey = "test-signing-key-for-rbac-agent-pending"

func (s *AgentPendingPublicTestSuite) TestGetAgentsPendingRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupMocks   func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:      "when no token returns 401",
			setupAuth: func(_ *http.Request) {},
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				return jobmocks.NewMockJobClient(
						s.mockCtrl,
					), agentmocks.NewMockEnrollmentManager(
						s.mockCtrl,
					)
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
					rbacAgentPendingTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"node:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				return jobmocks.NewMockJobClient(
						s.mockCtrl,
					), agentmocks.NewMockEnrollmentManager(
						s.mockCtrl,
					)
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
					rbacAgentPendingTestSigningKey,
					[]string{"read"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				jm := jobmocks.NewMockJobClient(s.mockCtrl)
				em := agentmocks.NewMockEnrollmentManager(s.mockCtrl)
				em.EXPECT().ListPending(gomock.Any()).Return(nil, nil)
				return jm, em
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "agents")
				s.Contains(rec.Body.String(), "total")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jm, em := tc.setupMocks()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacAgentPendingTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apiagent.Handler(
				s.logger,
				jm,
				appConfig.Controller.API.Security.SigningKey,
				nil,
				em,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/agent/pending", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestAgentPendingPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentPendingPublicTestSuite))
}
