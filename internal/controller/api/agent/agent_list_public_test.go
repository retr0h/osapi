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
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

type AgentListPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *jobmocks.MockJobClient
	handler       *apiagent.Agent
	ctx           context.Context
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *AgentListPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = jobmocks.NewMockJobClient(s.mockCtrl)
	s.handler = apiagent.New(slog.Default(), s.mockJobClient, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *AgentListPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentListPublicTestSuite) TestListAgents() {
	tests := []struct {
		name         string
		mockAgents   []jobtypes.AgentInfo
		mockError    error
		validateFunc func(resp gen.GetAgentsResponseObject)
	}{
		{
			name: "success with agents",
			mockAgents: []jobtypes.AgentInfo{
				{
					Hostname:     "server1",
					MachineID:    "abc123def456",
					Fingerprint:  "SHA256:test-fingerprint",
					Labels:       map[string]string{"group": "web"},
					RegisteredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					StartedAt:    time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
					OSInfo:       &host.Result{Distribution: "Ubuntu", Version: "24.04"},
					Uptime:       5 * time.Hour,
					LoadAverages: &load.Result{Load1: 0.5, Load5: 0.3, Load15: 0.2},
					MemoryStats:  &mem.Result{Total: 8388608, Free: 4194304, Cached: 2097152},
				},
				{Hostname: "server2"},
			},
			validateFunc: func(resp gen.GetAgentsResponseObject) {
				r, ok := resp.(gen.GetAgents200JSONResponse)
				s.True(ok)
				s.Equal(2, r.Total)
				s.Len(r.Agents, 2)
				s.Equal("server1", r.Agents[0].Hostname)
				s.Equal(gen.AgentInfoStatusReady, r.Agents[0].Status)
				s.Require().NotNil(r.Agents[0].MachineId)
				s.Equal("abc123def456", *r.Agents[0].MachineId)
				s.Require().NotNil(r.Agents[0].Fingerprint)
				s.Equal("SHA256:test-fingerprint", *r.Agents[0].Fingerprint)
				s.NotNil(r.Agents[0].Labels)
				s.NotNil(r.Agents[0].RegisteredAt)
				s.NotNil(r.Agents[0].StartedAt)
				s.NotNil(r.Agents[0].OsInfo)
				s.Equal("Ubuntu", r.Agents[0].OsInfo.Distribution)
				s.NotNil(r.Agents[0].LoadAverage)
				s.NotNil(r.Agents[0].Memory)
				s.NotNil(r.Agents[0].Uptime)
				s.Equal("server2", r.Agents[1].Hostname)
				s.Equal(gen.AgentInfoStatusReady, r.Agents[1].Status)
				s.Nil(r.Agents[1].MachineId)
			},
		},
		{
			name: "success with all facts fields",
			mockAgents: []jobtypes.AgentInfo{
				{
					Hostname:      "server1",
					Architecture:  "x86_64",
					KernelVersion: "6.1.0",
					CPUCount:      8,
					FQDN:          "server1.example.com",
					ServiceMgr:    "systemd",
					PackageMgr:    "apt",
					Interfaces: []jobtypes.NetworkInterface{
						{
							Name:   "eth0",
							IPv4:   "10.0.0.1",
							IPv6:   "fe80::1",
							MAC:    "aa:bb:cc:dd:ee:ff",
							Family: "inet",
						},
						{Name: "lo", IPv4: "127.0.0.1"},
					},
					Facts: map[string]any{"env": "prod"},
				},
			},
			validateFunc: func(resp gen.GetAgentsResponseObject) {
				r, ok := resp.(gen.GetAgents200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)

				a := r.Agents[0]
				s.Equal("server1", a.Hostname)
				s.Require().NotNil(a.Architecture)
				s.Equal("x86_64", *a.Architecture)
				s.Require().NotNil(a.KernelVersion)
				s.Equal("6.1.0", *a.KernelVersion)
				s.Require().NotNil(a.CpuCount)
				s.Equal(8, *a.CpuCount)
				s.Require().NotNil(a.Fqdn)
				s.Equal("server1.example.com", *a.Fqdn)
				s.Require().NotNil(a.ServiceMgr)
				s.Equal("systemd", *a.ServiceMgr)
				s.Require().NotNil(a.PackageMgr)
				s.Equal("apt", *a.PackageMgr)
				s.Require().NotNil(a.Interfaces)
				s.Len(*a.Interfaces, 2)
				iface0 := (*a.Interfaces)[0]
				s.Equal("eth0", iface0.Name)
				s.Require().NotNil(iface0.Ipv4)
				s.Equal("10.0.0.1", *iface0.Ipv4)
				s.Require().NotNil(iface0.Ipv6)
				s.Equal("fe80::1", *iface0.Ipv6)
				s.Require().NotNil(iface0.Mac)
				s.Equal("aa:bb:cc:dd:ee:ff", *iface0.Mac)
				s.Require().NotNil(iface0.Family)
				s.Equal(gen.NetworkInterfaceResponseFamily("inet"), *iface0.Family)
				iface1 := (*a.Interfaces)[1]
				s.Equal("lo", iface1.Name)
				s.Nil(iface1.Ipv6)
				s.Nil(iface1.Mac)
				s.Nil(iface1.Family)
				s.Require().NotNil(a.Facts)
				s.Equal("prod", (*a.Facts)["env"])
			},
		},
		{
			name: "success with routes and primary interface",
			mockAgents: []jobtypes.AgentInfo{
				{
					Hostname:         "server1",
					PrimaryInterface: "eth0",
					Routes: []jobtypes.Route{
						{
							Destination: "0.0.0.0",
							Gateway:     "192.168.1.1",
							Interface:   "eth0",
							Mask:        "/0",
							Metric:      100,
							Flags:       "0003",
						},
						{
							Destination: "192.168.1.0",
							Gateway:     "0.0.0.0",
							Interface:   "eth0",
						},
					},
				},
			},
			validateFunc: func(resp gen.GetAgentsResponseObject) {
				r, ok := resp.(gen.GetAgents200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)

				a := r.Agents[0]
				s.Require().NotNil(a.PrimaryInterface)
				s.Equal("eth0", *a.PrimaryInterface)
				s.Require().NotNil(a.Routes)
				s.Len(*a.Routes, 2)
				route0 := (*a.Routes)[0]
				s.Equal("0.0.0.0", route0.Destination)
				s.Equal("192.168.1.1", route0.Gateway)
				s.Equal("eth0", route0.Interface)
				s.Require().NotNil(route0.Mask)
				s.Equal("/0", *route0.Mask)
				s.Require().NotNil(route0.Metric)
				s.Equal(100, *route0.Metric)
				s.Require().NotNil(route0.Flags)
				s.Equal("0003", *route0.Flags)
				route1 := (*a.Routes)[1]
				s.Equal("192.168.1.0", route1.Destination)
				s.Nil(route1.Mask)
				s.Nil(route1.Metric)
				s.Nil(route1.Flags)
			},
		},
		{
			name: "success with scheduling fields",
			mockAgents: []jobtypes.AgentInfo{
				{
					Hostname: "server1",
					State:    jobtypes.AgentStateDraining,
					Conditions: []jobtypes.Condition{
						{
							Type:               "MemoryPressure",
							Status:             true,
							LastTransitionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
							Reason:             "memory above threshold",
						},
						{
							Type:               "DiskPressure",
							Status:             false,
							LastTransitionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						},
					},
					Timeline: []jobtypes.TimelineEvent{
						{
							Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
							Event:     "drain",
							Hostname:  "server1",
							Message:   "Drain initiated via API",
							Error:     "some error",
						},
						{
							Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
							Event:     "ready",
						},
					},
				},
			},
			validateFunc: func(resp gen.GetAgentsResponseObject) {
				r, ok := resp.(gen.GetAgents200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)

				a := r.Agents[0]
				s.Require().NotNil(a.State)
				s.Equal(gen.AgentInfoState("Draining"), *a.State)
				s.Require().NotNil(a.Conditions)
				s.Len(*a.Conditions, 2)
				c0 := (*a.Conditions)[0]
				s.Equal(gen.NodeConditionType("MemoryPressure"), c0.Type)
				s.True(c0.Status)
				s.Require().NotNil(c0.Reason)
				s.Equal("memory above threshold", *c0.Reason)
				c1 := (*a.Conditions)[1]
				s.Nil(c1.Reason)
				s.Require().NotNil(a.Timeline)
				s.Len(*a.Timeline, 2)
				t0 := (*a.Timeline)[0]
				s.Equal("drain", t0.Event)
				s.Require().NotNil(t0.Hostname)
				s.Equal("server1", *t0.Hostname)
				s.Require().NotNil(t0.Message)
				s.Equal("Drain initiated via API", *t0.Message)
				s.Require().NotNil(t0.Error)
				s.Equal("some error", *t0.Error)
				t1 := (*a.Timeline)[1]
				s.Equal("ready", t1.Event)
				s.Nil(t1.Hostname)
				s.Nil(t1.Message)
				s.Nil(t1.Error)
			},
		},
		{
			name:       "success with no agents",
			mockAgents: []jobtypes.AgentInfo{},
			validateFunc: func(resp gen.GetAgentsResponseObject) {
				r, ok := resp.(gen.GetAgents200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Agents)
			},
		},
		{
			name:      "job client error",
			mockError: assert.AnError,
			validateFunc: func(resp gen.GetAgentsResponseObject) {
				_, ok := resp.(gen.GetAgents500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockJobClient.EXPECT().
				ListAgents(gomock.Any()).
				Return(tt.mockAgents, tt.mockError)

			resp, err := s.handler.GetAgents(s.ctx, gen.GetAgentsRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AgentListPublicTestSuite) TestListAgentsHTTP() {
	tests := []struct {
		name         string
		setupJobMock func() *jobmocks.MockJobClient
		wantCode     int
		wantContains []string
	}{
		{
			name: "when agents exist returns agent list",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					ListAgents(gomock.Any()).
					Return([]jobtypes.AgentInfo{
						{Hostname: "server1"},
						{Hostname: "server2"},
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"total":2`, `"server1"`, `"server2"`, `"status":"Ready"`},
		},
		{
			name: "when no agents returns empty list",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					ListAgents(gomock.Any()).
					Return([]jobtypes.AgentInfo{}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"total":0`},
		},
		{
			name: "when job client errors returns 500",
			setupJobMock: func() *jobmocks.MockJobClient {
				mock := jobmocks.NewMockJobClient(s.mockCtrl)
				mock.EXPECT().
					ListAgents(gomock.Any()).
					Return(nil, assert.AnError)
				return mock
			},
			wantCode:     http.StatusInternalServerError,
			wantContains: []string{`"error"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			agentHandler := apiagent.New(s.logger, jobMock, nil)
			strictHandler := gen.NewStrictHandler(agentHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/agent", nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacAgentListTestSigningKey = "test-signing-key-for-rbac-agent-list"

func (s *AgentListPublicTestSuite) TestListAgentsRBACHTTP() {
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
					rbacAgentListTestSigningKey,
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
			name: "when valid token with agent:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAgentListTestSigningKey,
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
					ListAgents(gomock.Any()).
					Return([]jobtypes.AgentInfo{
						{Hostname: "server1"},
						{Hostname: "server2"},
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"total":2`, `"server1"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			jobMock := tc.setupJobMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacAgentListTestSigningKey,
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
				"/api/agent",
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

func (s *AgentListPublicTestSuite) TestUint64ToInt() {
	maxInt := int(^uint(0) >> 1)

	tests := []struct {
		name string
		val  uint64
		want int
	}{
		{
			name: "when zero",
			val:  0,
			want: 0,
		},
		{
			name: "when normal value",
			val:  42,
			want: 42,
		},
		{
			name: "when max int value",
			val:  uint64(maxInt),
			want: maxInt,
		},
		{
			name: "when overflow",
			val:  math.MaxUint64,
			want: maxInt,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := apiagent.ExportUint64ToInt(tt.val)
			s.Equal(tt.want, got)
		})
	}
}

func TestAgentListPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentListPublicTestSuite))
}
