// Copyright (c) 2025 John Dewey

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
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/agent/identity"
	"github.com/osapi-io/osapi/internal/config"
	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/job/mocks"
	commandMocks "github.com/osapi-io/osapi/internal/provider/command/mocks"
	netinfoMocks "github.com/osapi-io/osapi/internal/provider/network/netinfo/mocks"
	dnsMocks "github.com/osapi-io/osapi/internal/provider/network/netplan/dns/mocks"
	pingMocks "github.com/osapi-io/osapi/internal/provider/network/ping/mocks"
	diskMocks "github.com/osapi-io/osapi/internal/provider/node/disk/mocks"
	hostMocks "github.com/osapi-io/osapi/internal/provider/node/host/mocks"
	loadMocks "github.com/osapi-io/osapi/internal/provider/node/load/mocks"
	memMocks "github.com/osapi-io/osapi/internal/provider/node/mem/mocks"
	"github.com/osapi-io/osapi/internal/telemetry/metrics"
	processMocks "github.com/osapi-io/osapi/internal/telemetry/process/mocks"
)

type AgentPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	appFs         avfs.VFS
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *AgentPublicTestSuite) getFreePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port
}

func (s *AgentPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)
	s.appFs = memfs.New()
	s.logger = slog.Default()

	// Mock identity so tests don't depend on /etc/machine-id existing.
	agent.SetGetIdentityFn(func(_ avfs.VFS, _ string) (*identity.Identity, error) {
		return &identity.Identity{
			MachineID: "test-machine-id",
			Hostname:  "test-agent",
		}, nil
	})

	s.appConfig = config.Config{
		NATS: config.NATS{
			Stream: config.NATSStream{Name: "test-stream"},
		},
		Agent: config.AgentConfig{
			Hostname:   "test-agent",
			QueueGroup: "test-queue",
			MaxJobs:    5,
			Consumer: config.AgentConsumer{
				AckWait:       "30s",
				BackOff:       []string{"1s", "2s", "5s"},
				MaxDeliver:    3,
				MaxAckPending: 10,
				ReplayPolicy:  "instant",
			},
		},
	}
}

func (s *AgentPublicTestSuite) TearDownTest() {
	agent.ResetGetIdentityFn()
	s.mockCtrl.Finish()
}

func (s *AgentPublicTestSuite) TearDownSubTest() {
	// Restore the default identity mock between table rows so tests
	// that override getIdentityFn (e.g., identity-fails) don't leak
	// into subsequent rows (e.g., pending-state).
	agent.SetGetIdentityFn(func(_ avfs.VFS, _ string) (*identity.Identity, error) {
		return &identity.Identity{
			MachineID: "test-machine-id",
			Hostname:  "test-agent",
		}, nil
	})
}

func (s *AgentPublicTestSuite) buildAgent() *agent.Agent {
	return newTestAgent(newTestAgentParams{
		appFs:           s.appFs,
		appConfig:       s.appConfig,
		logger:          s.logger,
		jobClient:       s.mockJobClient,
		streamName:      "test-stream",
		hostProvider:    hostMocks.NewDefaultMockProvider(s.mockCtrl),
		diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
		memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
		loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
		dnsProvider:     dnsMocks.NewDefaultMockProvider(s.mockCtrl),
		pingProvider:    pingMocks.NewDefaultMockProvider(s.mockCtrl),
		netinfoProvider: netinfoMocks.NewDefaultMockProvider(s.mockCtrl),
		commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
		processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
	})
}

func (s *AgentPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		validateFunc func(*agent.Agent)
	}{
		{
			name: "creates agent with all providers",
			validateFunc: func(a *agent.Agent) {
				s.NotNil(a)

				a.SetSubComponents(map[string]job.SubComponentInfo{
					"agent.heartbeat": {Status: "ok"},
				})
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(s.buildAgent())
		})
	}
}

func (s *AgentPublicTestSuite) TestStart() {
	tests := []struct {
		name      string
		setupFunc func() *agent.Agent
		stopFunc  func(a *agent.Agent)
	}{
		{
			name: "starts and stops gracefully",
			setupFunc: func() *agent.Agent {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(6)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(6)

				return s.buildAgent()
			},
			stopFunc: func(a *agent.Agent) {
				stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				a.Stop(stopCtx)
			},
		},
		{
			name: "returns early when identity resolution fails",
			setupFunc: func() *agent.Agent {
				agent.SetGetIdentityFn(func(_ avfs.VFS, _ string) (*identity.Identity, error) {
					return nil, fmt.Errorf("machine-id not found")
				})

				return s.buildAgent()
			},
			stopFunc: func(a *agent.Agent) {
				// Agent should not have started — identity failed.
				err := a.IsReady()
				s.Error(err)
			},
		},
		{
			name: "returns early when PKI enrollment fails",
			setupFunc: func() *agent.Agent {
				cfg := s.appConfig
				cfg.Agent.PKI = config.AgentPKI{
					Enabled: true,
					KeyDir:  "/nonexistent/keys",
				}

				// Use failfs to make the key directory creation fail.
				ffs := failfs.New(memfs.New())
				_ = ffs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnMkdirAll {
						return errors.New("permission denied")
					}
					return nil
				})

				return newTestAgent(newTestAgentParams{
					appFs:           ffs,
					appConfig:       cfg,
					logger:          s.logger,
					jobClient:       s.mockJobClient,
					streamName:      "test-stream",
					hostProvider:    hostMocks.NewDefaultMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewDefaultMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewDefaultMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewDefaultMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
					processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
				})
			},
			stopFunc: func(a *agent.Agent) {
				// Agent should not have started — PKI enrollment failed.
				err := a.IsReady()
				s.Error(err)
			},
		},
		{
			name: "enters pending state when PKI enabled and not enrolled skips consumers",
			setupFunc: func() *agent.Agent {
				fs := memfs.New()

				cfg := s.appConfig
				cfg.Agent.PKI = config.AgentPKI{
					Enabled: true,
					KeyDir:  "/keys",
				}

				// No CreateOrUpdateConsumer/ConsumeJobs expectations:
				// consumers must NOT be started in pending state.

				return newTestAgent(newTestAgentParams{
					appFs:           fs,
					appConfig:       cfg,
					logger:          s.logger,
					jobClient:       s.mockJobClient,
					streamName:      "test-stream",
					hostProvider:    hostMocks.NewDefaultMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewDefaultMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewDefaultMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewDefaultMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
					processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
				})
			},
			stopFunc: func(a *agent.Agent) {
				// Agent should be in pending state — consumers not started.
				state := agent.GetAgentState(a)
				s.Equal(job.AgentStatePending, state)

				stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				a.Stop(stopCtx)
			},
		},
		{
			name: "returns early when preflight fails",
			setupFunc: func() *agent.Agent {
				mockExecMgr := execmocks.NewMockManager(s.mockCtrl)
				mockExecMgr.EXPECT().
					RunCmd("sudo", gomock.Any()).
					Return("", fmt.Errorf("sudo: a password is required")).
					AnyTimes()

				cfg := s.appConfig
				cfg.Agent.PrivilegeEscalation = config.PrivilegeEscalation{
					Enabled: true,
				}

				return newTestAgent(newTestAgentParams{
					appFs:           s.appFs,
					appConfig:       cfg,
					logger:          s.logger,
					jobClient:       s.mockJobClient,
					streamName:      "test-stream",
					hostProvider:    hostMocks.NewDefaultMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewDefaultMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewDefaultMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewDefaultMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
					processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
					execManager:     mockExecMgr,
				})
			},
			stopFunc: func(a *agent.Agent) {
				// Agent should not have started consumers, so IsReady fails.
				err := a.IsReady()
				s.Error(err)
			},
		},
		{
			name: "starts when preflight passes",
			setupFunc: func() *agent.Agent {
				mockExecMgr := execmocks.NewMockManager(s.mockCtrl)
				mockExecMgr.EXPECT().
					RunCmd("sudo", gomock.Any()).
					Return("/usr/bin/something", nil).
					AnyTimes()

				// Write a fake proc status file with all capabilities set.
				tmpDir := s.T().TempDir()
				path := filepath.Join(tmpDir, "status")
				err := os.WriteFile(
					path,
					[]byte("Name:\tosapi\nCapEff:\t000000000000102f\n"),
					0o644,
				)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
				s.T().Cleanup(func() {
					agent.ResetProcStatusPath()
				})

				cfg := s.appConfig
				cfg.Agent.PrivilegeEscalation = config.PrivilegeEscalation{
					Enabled: true,
				}

				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(6)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(6)

				return newTestAgent(newTestAgentParams{
					appFs:           s.appFs,
					appConfig:       cfg,
					logger:          s.logger,
					jobClient:       s.mockJobClient,
					streamName:      "test-stream",
					hostProvider:    hostMocks.NewDefaultMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewDefaultMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewDefaultMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewDefaultMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
					processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
					execManager:     mockExecMgr,
				})
			},
			stopFunc: func(a *agent.Agent) {
				stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				a.Stop(stopCtx)
			},
		},
		{
			name: "stop times out when agents are slow to finish",
			setupFunc: func() *agent.Agent {
				blockCh := make(chan struct{})

				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(6)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						_ string,
						_ interface{},
						_ interface{},
					) error {
						<-blockCh
						return nil
					}).
					Times(6)

				a := s.buildAgent()

				// Schedule cleanup after Stop returns
				s.T().Cleanup(func() {
					close(blockCh)
					time.Sleep(10 * time.Millisecond)
				})

				return a
			},
			stopFunc: func(a *agent.Agent) {
				stopCtx, cancel := context.WithCancel(context.Background())
				cancel()

				a.Stop(stopCtx)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := tt.setupFunc()
			a.Start()
			tt.stopFunc(a)
		})
	}
}

func (s *AgentPublicTestSuite) TestIsReady() {
	tests := []struct {
		name         string
		setupFunc    func() *agent.Agent
		validateFunc func(error)
	}{
		{
			name: "returns error when agent not started",
			setupFunc: func() *agent.Agent {
				return s.buildAgent()
			},
			validateFunc: func(err error) {
				s.Error(err)
				s.Contains(err.Error(), "agent not started")
			},
		},
		{
			name: "returns nil when agent is started",
			setupFunc: func() *agent.Agent {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(6)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(6)

				a := s.buildAgent()
				a.Start()
				s.T().Cleanup(func() {
					stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					a.Stop(stopCtx)
				})

				return a
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := tt.setupFunc()
			tt.validateFunc(a.IsReady())
		})
	}
}

func (s *AgentPublicTestSuite) TestSetMeterProvider() {
	tests := []struct {
		name         string
		validateFunc func(*metrics.Server, *agent.Agent)
	}{
		{
			name: "creates OTEL instruments without panic",
			validateFunc: func(srv *metrics.Server, a *agent.Agent) {
				s.Require().NotNil(srv)

				s.NotPanics(func() {
					a.SetMeterProvider(srv.MeterProvider())
				})
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := s.buildAgent()

			port := s.getFreePort()
			srv := metrics.New("127.0.0.1", port, slog.Default())
			tt.validateFunc(srv, a)

			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()

			srv.Stop(ctx)
		})
	}
}

func (s *AgentPublicTestSuite) TestLastHeartbeatTime() {
	tests := []struct {
		name         string
		validateFunc func(time.Time)
	}{
		{
			name: "returns zero time before any heartbeat",
			validateFunc: func(got time.Time) {
				s.True(got.IsZero())
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := s.buildAgent()

			tt.validateFunc(a.LastHeartbeatTime())
		})
	}
}

func TestAgentPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentPublicTestSuite))
}
