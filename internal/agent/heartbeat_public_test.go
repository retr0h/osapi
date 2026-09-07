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
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/agent/identity"
	"github.com/osapi-io/osapi/internal/agent/pki"
	"github.com/osapi-io/osapi/internal/config"
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
	processMocks "github.com/osapi-io/osapi/internal/telemetry/process/mocks"
)

type HeartbeatPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	mockKV        *mocks.MockKeyValue
	appFs         avfs.VFS
	appConfig     config.Config
	logger        *slog.Logger
}

func (s *HeartbeatPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)
	s.mockKV = mocks.NewMockKeyValue(s.mockCtrl)
	s.appFs = memfs.New()
	s.logger = slog.Default()

	// Mock identity so tests don't depend on /etc/machine-id.
	agent.SetGetIdentityFn(func(_ avfs.VFS, _ string) (*identity.Identity, error) {
		return &identity.Identity{
			MachineID: "test-machine-id",
			Hostname:  "test-agent",
		}, nil
	})

	s.appConfig = config.Config{
		NATS: config.NATS{
			Stream: config.NATSStream{Name: "test-stream"},
			Registry: config.NATSRegistry{
				Bucket:   "agent-registry",
				TTL:      "30s",
				Storage:  "file",
				Replicas: 1,
			},
		},
		Agent: config.AgentConfig{
			Hostname:   "test-agent",
			QueueGroup: "test-queue",
			MaxJobs:    5,
			Labels:     map[string]string{"group": "web"},
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

func (s *HeartbeatPublicTestSuite) TearDownTest() {
	agent.ResetGetIdentityFn()
	s.mockCtrl.Finish()
}

func (s *HeartbeatPublicTestSuite) TestStartWithHeartbeat() {
	tests := []struct {
		name      string
		setupFunc func() *agent.Agent
		stopFunc  func(a *agent.Agent)
	}{
		{
			name: "when registryKV is set registers and deregisters",
			setupFunc: func() *agent.Agent {
				// Drain check on each heartbeat tick (no drain flag present).
				// machineID is resolved from the real system at Start(), so use Any().
				s.mockJobClient.EXPECT().
					CheckDrainFlag(gomock.Any(), gomock.Any()).
					Return(false).
					AnyTimes()

				// Heartbeat initial write (machineID-based key from real system)
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					MinTimes(1)

				// Deregister on stop
				s.mockKV.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)

					// 3 base + 1 label = 4 consumers per job type, x2 (query+modify) = 8
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(8)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(8)

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
					registryKV:      s.mockKV,
				})
			},
			stopFunc: func(a *agent.Agent) {
				stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				a.Stop(stopCtx)
			},
		},
		{
			name: "when registryKV is nil skips heartbeat",
			setupFunc: func() *agent.Agent {
				// 3 base + 1 label = 4 consumers per job type, x2 (query+modify) = 8
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(8)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(8)

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
				})
			},
			stopFunc: func(a *agent.Agent) {
				stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

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

func TestHeartbeatPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeartbeatPublicTestSuite))
}

// HeartbeatLowLevelPublicTestSuite tests the lower-level heartbeat methods.
type HeartbeatLowLevelPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	mockKV        *mocks.MockKeyValue
	testAgent     *agent.Agent
}

func (s *HeartbeatLowLevelPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)
	s.mockKV = mocks.NewMockKeyValue(s.mockCtrl)

	appConfig := config.Config{
		Agent: config.AgentConfig{
			Labels: map[string]string{"group": "web"},
		},
	}

	// Use DefaultMockProviders so provider calls during writeRegistration are satisfied.
	s.testAgent = newTestAgent(newTestAgentParams{
		appConfig:       appConfig,
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
		registryKV:      s.mockKV,
	})
	agent.SetAgentState(s.testAgent, job.AgentStateReady)
	agent.SetAgentMachineID(s.testAgent, "test-machine-id")

	// writeRegistration now calls handleDrainDetection which checks drain flag.
	// Default: no drain flag present.
	s.mockJobClient.EXPECT().
		CheckDrainFlag(gomock.Any(), "test-machine-id").
		Return(false).
		AnyTimes()
}

func (s *HeartbeatLowLevelPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
	agent.ResetMarshalJSON()
	agent.ResetHeartbeatInterval()
}

func (s *HeartbeatLowLevelPublicTestSuite) TestWriteRegistration() {
	tests := []struct {
		name         string
		setupMock    func()
		teardownMock func()
		validateFunc func(before time.Time)
	}{
		{
			name: "when marshal fails logs warning",
			setupMock: func() {
				agent.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, fmt.Errorf("marshal failure")
				})
			},
			teardownMock: func() {
				agent.ResetMarshalJSON()
			},
			validateFunc: func(_ time.Time) {
				// Marshal failure occurs before Put, so heartbeat time
				// should NOT be updated.
				got := s.testAgent.LastHeartbeatTime()
				s.True(got.IsZero(), "expected zero heartbeat time after marshal failure")
			},
		},
		{
			name: "when Put fails logs warning",
			setupMock: func() {
				s.mockKV.EXPECT().
					Put(gomock.Any(), "agents.test_machine_id", gomock.Any()).
					Return(uint64(0), errors.New("put failed"))
			},
			validateFunc: func(_ time.Time) {
				// Put failure means heartbeat time should NOT be updated.
				got := s.testAgent.LastHeartbeatTime()
				s.True(got.IsZero(), "expected zero heartbeat time after Put failure")
			},
		},
		{
			name: "when Put succeeds writes registration",
			setupMock: func() {
				s.mockKV.EXPECT().
					Put(gomock.Any(), "agents.test_machine_id", gomock.Any()).
					Return(uint64(1), nil)
			},
			validateFunc: func(before time.Time) {
				got := s.testAgent.LastHeartbeatTime()
				s.False(got.IsZero(), "expected non-zero heartbeat time after successful Put")
				s.True(
					!got.Before(before),
					"heartbeat time should be at or after test start",
				)
			},
		},
		{
			name: "when PKI enabled includes fingerprint in registration",
			setupMock: func() {
				m := pki.New(memfs.New(), "/keys", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				agent.SetAgentPKIManager(s.testAgent, m)

				s.mockKV.EXPECT().
					Put(gomock.Any(), "agents.test_machine_id", gomock.Any()).
					DoAndReturn(func(
						_ interface{},
						_ string,
						data []byte,
					) (uint64, error) {
						s.Contains(string(data), "fingerprint")
						s.Contains(string(data), "SHA256:")
						return uint64(1), nil
					})
			},
			teardownMock: func() {
				agent.SetAgentPKIManager(s.testAgent, nil)
			},
			validateFunc: func(_ time.Time) {},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			if tt.teardownMock != nil {
				defer tt.teardownMock()
			}
			before := time.Now()
			agent.ExportWriteRegistration(
				context.Background(),
				s.testAgent,
				"test-machine-id",
				"test-agent",
			)
			tt.validateFunc(before)
		})
	}
}

func (s *HeartbeatLowLevelPublicTestSuite) TestWriteRegistrationStoresHeartbeatTime() {
	tests := []struct {
		name string
	}{
		{
			name: "when Put succeeds stores last heartbeat time",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockKV.EXPECT().
				Put(gomock.Any(), "agents.test_machine_id", gomock.Any()).
				Return(uint64(1), nil)

			before := time.Now()
			agent.ExportWriteRegistration(
				context.Background(),
				s.testAgent,
				"test-machine-id",
				"test-agent",
			)
			after := time.Now()

			got := s.testAgent.LastHeartbeatTime()
			s.False(got.IsZero(), "expected non-zero heartbeat time after successful Put")
			s.True(
				!got.Before(before) && !got.After(after),
				"heartbeat time should be between before and after write",
			)
		})
	}
}

func (s *HeartbeatLowLevelPublicTestSuite) TestDeregister() {
	tests := []struct {
		name      string
		setupMock func()
	}{
		{
			name: "when Delete fails logs warning",
			setupMock: func() {
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "agents.test_machine_id").
					Return(errors.New("delete failed"))
			},
		},
		{
			name: "when Delete succeeds logs deregistration",
			setupMock: func() {
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "agents.test_machine_id").
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			// Deregister is best-effort (fire-and-forget). It deletes a
			// KV key and logs the outcome but does not mutate agent state.
			// The gomock expectations above verify the correct KV call was
			// made; beyond that, we only verify the function completes
			// without panicking.
			s.NotPanics(func() {
				agent.ExportDeregister(s.testAgent, "test-machine-id")
			})
		})
	}
}

func (s *HeartbeatLowLevelPublicTestSuite) TestStartHeartbeatRefresh() {
	tests := []struct {
		name      string
		setupMock func()
	}{
		{
			name: "ticker fires and refreshes registration",
			setupMock: func() {
				// Initial write + at least 1 ticker refresh
				s.mockKV.EXPECT().
					Put(gomock.Any(), "agents.test_machine_id", gomock.Any()).
					Return(uint64(1), nil).
					MinTimes(2)

				// Deregister on cancel
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "agents.test_machine_id").
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			// Return the same hostname to prevent hostname change detection.
			agent.SetGetAgentHostnameFn(func(_ string) (string, error) {
				return "test-agent", nil
			})
			defer agent.ResetGetAgentHostnameFn()

			agent.SetHeartbeatInterval(10 * time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())
			agent.ExportStartHeartbeat(ctx, s.testAgent, "test-machine-id", "test-agent")

			// Wait for at least one ticker refresh
			time.Sleep(50 * time.Millisecond)
			cancel()

			// Wait for goroutine to finish
			agent.WaitAgentWG(s.testAgent)
		})
	}
}

func (s *HeartbeatLowLevelPublicTestSuite) TestStartHeartbeatHostnameChange() {
	tests := []struct {
		name            string
		initialHostname string
		hostnameReply   string
		expectChanged   bool
	}{
		{
			name:            "when hostname changes updates cached hostname",
			initialHostname: "old-host",
			hostnameReply:   "new-host",
			expectChanged:   true,
		},
		{
			name:            "when hostname unchanged does not resubscribe",
			initialHostname: "same-host",
			hostnameReply:   "same-host",
			expectChanged:   false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create fresh mocks per subtest to avoid gomock expectation leaks.
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()

			mockKV := mocks.NewMockKeyValue(ctrl)
			mockJobClient := mocks.NewMockJobClient(ctrl)

			appConfig := config.Config{
				Agent: config.AgentConfig{
					Labels: map[string]string{"group": "web"},
				},
			}

			testAgent := newTestAgent(newTestAgentParams{
				appConfig:       appConfig,
				jobClient:       mockJobClient,
				streamName:      "test-stream",
				hostProvider:    hostMocks.NewDefaultMockProvider(ctrl),
				diskProvider:    diskMocks.NewDefaultMockProvider(ctrl),
				memProvider:     memMocks.NewDefaultMockProvider(ctrl),
				loadProvider:    loadMocks.NewDefaultMockProvider(ctrl),
				dnsProvider:     dnsMocks.NewDefaultMockProvider(ctrl),
				pingProvider:    pingMocks.NewDefaultMockProvider(ctrl),
				netinfoProvider: netinfoMocks.NewDefaultMockProvider(ctrl),
				commandProvider: commandMocks.NewDefaultMockProvider(ctrl),
				processProvider: processMocks.NewDefaultMockProvider(ctrl),
				registryKV:      mockKV,
			})
			agent.SetAgentState(testAgent, job.AgentStateReady)
			agent.SetAgentMachineID(testAgent, "test-machine-id")
			agent.SetAgentHostname(testAgent, tt.initialHostname)

			// Drain check — no drain flag present.
			mockJobClient.EXPECT().
				CheckDrainFlag(gomock.Any(), "test-machine-id").
				Return(false).
				AnyTimes()

			reply := tt.hostnameReply
			agent.SetGetAgentHostnameFn(func(_ string) (string, error) {
				return reply, nil
			})
			defer agent.ResetGetAgentHostnameFn()

			// Consumers use machine ID (permanent), so hostname changes
			// do NOT trigger consumer stop/start — only the cached
			// hostname is updated for registry entries and logging.

			// Initial write + at least 1 ticker refresh
			mockKV.EXPECT().
				Put(gomock.Any(), "agents.test_machine_id", gomock.Any()).
				Return(uint64(1), nil).
				MinTimes(2)

			// Deregister on cancel
			mockKV.EXPECT().
				Delete(gomock.Any(), "agents.test_machine_id").
				Return(nil).
				Times(1)

			agent.SetHeartbeatInterval(10 * time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())
			agent.ExportStartHeartbeat(ctx, testAgent, "test-machine-id", tt.initialHostname)

			// Wait for at least one ticker refresh
			time.Sleep(50 * time.Millisecond)
			cancel()

			// Wait for goroutine to finish
			agent.WaitAgentWG(testAgent)

			got := agent.GetAgentHostname(testAgent)
			s.Equal(tt.hostnameReply, got)
		})
	}
}

func (s *HeartbeatLowLevelPublicTestSuite) TestRegistryKey() {
	tests := []struct {
		name      string
		machineID string
		expected  string
	}{
		{
			name:      "simple machine ID",
			machineID: "abc-123-def",
			expected:  "agents.abc_123_def",
		},
		{
			name:      "machine ID with dots",
			machineID: "A1B2C3D4-E5F6.7890",
			expected:  "agents.A1B2C3D4_E5F6_7890",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportRegistryKey(tt.machineID)
			s.Equal(tt.expected, result)
		})
	}
}

func TestHeartbeatLowLevelPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HeartbeatLowLevelPublicTestSuite))
}
