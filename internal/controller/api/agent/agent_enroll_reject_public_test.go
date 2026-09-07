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
	agentmocks "github.com/osapi-io/osapi/internal/controller/api/agent/mocks"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type AgentEnrollRejectPublicTestSuite struct {
	suite.Suite

	mockCtrl       *gomock.Controller
	mockJobClient  *jobmocks.MockJobClient
	mockEnrollment *agentmocks.MockEnrollmentManager
	handler        *apiagent.Agent
	ctx            context.Context
	appConfig      config.Config
	logger         *slog.Logger
}

func (s *AgentEnrollRejectPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.mockEnrollment = agentmocks.NewMockEnrollmentManager(s.mockCtrl)
	s.handler = apiagent.New(slog.Default(), s.mockJobClient, s.mockEnrollment)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *AgentEnrollRejectPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentEnrollRejectPublicTestSuite) TestRejectAgent() {
	tests := []struct {
		name         string
		hostname     string
		useNilEnroll bool
		skipMock     bool
		mockErr      error
		validateFunc func(resp gen.RejectAgentResponseObject)
	}{
		{
			name:     "returns 400 when hostname is empty",
			hostname: "",
			skipMock: true,
			validateFunc: func(resp gen.RejectAgentResponseObject) {
				_, ok := resp.(gen.RejectAgent400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "returns 400 when hostname exceeds max length",
			hostname: strings.Repeat("a", 256),
			skipMock: true,
			validateFunc: func(resp gen.RejectAgentResponseObject) {
				_, ok := resp.(gen.RejectAgent400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:         "returns 500 when enrollment not enabled",
			hostname:     "web-01",
			useNilEnroll: true,
			skipMock:     true,
			validateFunc: func(resp gen.RejectAgentResponseObject) {
				r, ok := resp.(gen.RejectAgent500JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "enrollment not enabled")
			},
		},
		{
			name:     "success rejects agent",
			hostname: "web-01",
			validateFunc: func(resp gen.RejectAgentResponseObject) {
				r, ok := resp.(gen.RejectAgent200JSONResponse)
				s.True(ok)
				s.Contains(r.Message, "web-01 rejected")
			},
		},
		{
			name:     "returns 404 when not found",
			hostname: "unknown",
			mockErr:  fmt.Errorf("no pending agent with hostname"),
			validateFunc: func(resp gen.RejectAgentResponseObject) {
				_, ok := resp.(gen.RejectAgent404JSONResponse)
				s.True(ok)
			},
		},
		{
			name:     "returns 500 on internal error",
			hostname: "web-01",
			mockErr:  fmt.Errorf("kv connection failed"),
			validateFunc: func(resp gen.RejectAgentResponseObject) {
				r, ok := resp.(gen.RejectAgent500JSONResponse)
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
				handler = s.handler
			}

			if !tt.skipMock && !tt.useNilEnroll {
				s.mockEnrollment.EXPECT().
					RejectByHostname(gomock.Any(), tt.hostname, "rejected via API").
					Return(tt.mockErr)
			}

			resp, err := handler.RejectAgent(s.ctx, gen.RejectAgentRequestObject{
				Hostname: tt.hostname,
			})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AgentEnrollRejectPublicTestSuite) TestRejectAgentHTTP() {
	tests := []struct {
		name         string
		hostname     string
		setupMocks   func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name:     "when hostname exceeds max length returns 400",
			hostname: strings.Repeat("a", 256),
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				return jobmocks.NewMockJobClient(
						s.mockCtrl,
					), agentmocks.NewMockEnrollmentManager(
						s.mockCtrl,
					)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "error")
			},
		},
		{
			name:     "when agent rejected returns 200",
			hostname: "web-01",
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				jm := jobmocks.NewMockJobClient(s.mockCtrl)
				em := agentmocks.NewMockEnrollmentManager(s.mockCtrl)
				em.EXPECT().RejectByHostname(gomock.Any(), "web-01", "rejected via API").Return(nil)
				return jm, em
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "message")
			},
		},
		{
			name:     "when not found returns 404",
			hostname: "unknown",
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				jm := jobmocks.NewMockJobClient(s.mockCtrl)
				em := agentmocks.NewMockEnrollmentManager(s.mockCtrl)
				em.EXPECT().RejectByHostname(gomock.Any(), "unknown", "rejected via API").
					Return(fmt.Errorf("no pending agent with hostname"))
				return jm, em
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusNotFound, rec.Code)
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

			req := httptest.NewRequest(
				http.MethodPost,
				fmt.Sprintf("/api/agent/%s/reject", tc.hostname),
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacAgentRejectTestSigningKey = "test-signing-key-for-rbac-agent-reject"

func (s *AgentEnrollRejectPublicTestSuite) TestRejectAgentRBACHTTP() {
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
					rbacAgentRejectTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"agent:read"},
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
			name: "when valid token with agent:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAgentRejectTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMocks: func() (*jobmocks.MockJobClient, *agentmocks.MockEnrollmentManager) {
				jm := jobmocks.NewMockJobClient(s.mockCtrl)
				em := agentmocks.NewMockEnrollmentManager(s.mockCtrl)
				em.EXPECT().RejectByHostname(gomock.Any(), "web-01", "rejected via API").Return(nil)
				return jm, em
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "message")
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
							SigningKey: rbacAgentRejectTestSigningKey,
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

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/agent/web-01/reject",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestAgentEnrollRejectPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentEnrollRejectPublicTestSuite))
}
