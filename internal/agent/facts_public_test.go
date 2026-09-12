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
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
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

type FactsPublicTestSuite struct {
	suite.Suite

	mockCtrl         *gomock.Controller
	mockJobClient    *mocks.MockJobClient
	mockFactsKV      *mocks.MockKeyValue
	mockHostProvider *hostMocks.MockProvider
	mockNetinfo      *netinfoMocks.MockProvider
	testAgent        *agent.Agent
}

func (s *FactsPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)
	s.mockFactsKV = mocks.NewMockKeyValue(s.mockCtrl)
	s.mockHostProvider = hostMocks.NewDefaultMockProvider(s.mockCtrl)
	s.mockNetinfo = netinfoMocks.NewDefaultMockProvider(s.mockCtrl)

	appConfig := config.Config{
		Agent: config.AgentConfig{
			Labels: map[string]string{"group": "web"},
		},
	}

	s.testAgent = newTestAgent(newTestAgentParams{
		appConfig:       appConfig,
		jobClient:       s.mockJobClient,
		streamName:      "test-stream",
		hostProvider:    s.mockHostProvider,
		diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
		memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
		loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
		dnsProvider:     dnsMocks.NewDefaultMockProvider(s.mockCtrl),
		pingProvider:    pingMocks.NewDefaultMockProvider(s.mockCtrl),
		netinfoProvider: s.mockNetinfo,
		commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
		processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
		factsKV:         s.mockFactsKV,
	})
}

func (s *FactsPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
	agent.ResetMarshalJSON()
	agent.ResetUnmarshalJSON()
	agent.ResetDefaultFactsInterval()
}

func (s *FactsPublicTestSuite) TestWriteFacts() {
	tests := []struct {
		name         string
		setupMock    func()
		teardownMock func()
		validateFunc func()
	}{
		{
			name: "when Put succeeds writes facts",
			setupMock: func() {
				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						data []byte,
					) (uint64, error) {
						var reg job.FactsRegistration
						err := json.Unmarshal(data, &reg)
						s.NoError(err)
						s.Equal("amd64", reg.Architecture)
						s.Equal(4, reg.CPUCount)
						s.Equal("default-hostname.local", reg.FQDN)
						s.Equal("5.15.0-91-generic", reg.KernelVersion)
						s.Equal("systemd", reg.ServiceMgr)
						s.Equal("apt", reg.PackageMgr)
						s.Len(reg.Interfaces, 1)
						s.Equal("eth0", reg.Interfaces[0].Name)
						return uint64(1), nil
					})
			},
		},
		{
			name: "when Put fails logs warning",
			setupMock: func() {
				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					Return(uint64(0), errors.New("put failed"))
			},
		},
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
		},
		{
			name: "when provider errors partial data still written",
			setupMock: func() {
				// Override the default mock expectations with error-returning ones.
				errHostProvider := func() *hostMocks.MockProvider {
					m := hostMocks.NewPlainMockProvider(s.mockCtrl)
					m.EXPECT().GetArchitecture().Return("", errors.New("arch fail")).AnyTimes()
					m.EXPECT().GetKernelVersion().Return("", errors.New("kernel fail")).AnyTimes()
					m.EXPECT().GetFQDN().Return("", errors.New("fqdn fail")).AnyTimes()
					m.EXPECT().GetCPUCount().Return(0, errors.New("cpu fail")).AnyTimes()
					m.EXPECT().GetServiceManager().Return("", errors.New("svc fail")).AnyTimes()
					m.EXPECT().GetPackageManager().Return("", errors.New("pkg fail")).AnyTimes()
					return m
				}()
				agent.SetAgentHostProvider(s.testAgent, errHostProvider)

				errNetinfoProvider := func() *netinfoMocks.MockProvider {
					m := netinfoMocks.NewPlainMockProvider(s.mockCtrl)
					m.EXPECT().GetInterfaces().Return(nil, errors.New("net fail")).AnyTimes()
					m.EXPECT().GetRoutes().Return(nil, errors.New("routes fail")).AnyTimes()
					m.EXPECT().
						GetPrimaryInterface().
						Return("", errors.New("primary fail")).
						AnyTimes()
					return m
				}()
				agent.SetAgentNetinfoProvider(s.testAgent, errNetinfoProvider)

				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						_ string,
						data []byte,
					) (uint64, error) {
						var reg job.FactsRegistration
						err := json.Unmarshal(data, &reg)
						s.NoError(err)
						// All fields should be zero values since providers failed.
						s.Empty(reg.Architecture)
						s.Empty(reg.KernelVersion)
						s.Empty(reg.FQDN)
						s.Zero(reg.CPUCount)
						s.Empty(reg.ServiceMgr)
						s.Empty(reg.PackageMgr)
						s.Nil(reg.Interfaces)
						return uint64(1), nil
					})
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			if tt.teardownMock != nil {
				defer tt.teardownMock()
			}
			agent.ExportWriteFacts(
				context.Background(),
				s.testAgent,
				"test-machine-id",
				"test-agent",
			)
		})
	}
}

func (s *FactsPublicTestSuite) TestStartFactsRefresh() {
	tests := []struct {
		name      string
		setupMock func()
	}{
		{
			name: "ticker fires and refreshes facts",
			setupMock: func() {
				// Initial write + at least 1 ticker refresh
				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					Return(uint64(1), nil).
					MinTimes(2)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			agent.SetDefaultFactsInterval(10 * time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())
			agent.ExportStartFacts(ctx, s.testAgent, "test-machine-id", "test-agent")

			// Wait for at least one ticker refresh
			time.Sleep(50 * time.Millisecond)
			cancel()

			// Wait for goroutine to finish
			agent.WaitAgentWG(s.testAgent)
		})
	}
}

func (s *FactsPublicTestSuite) TestStartFactsInterval() {
	tests := []struct {
		name         string
		interval     string
		setupMock    func()
		validateFunc func()
	}{
		{
			name:     "when facts interval is configured uses configured value",
			interval: "20ms",
			setupMock: func() {
				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					Return(uint64(1), nil).
					AnyTimes()
			},
			validateFunc: func() {},
		},
		{
			name:     "when facts interval is empty uses default",
			interval: "",
			setupMock: func() {
				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					Return(uint64(1), nil).
					AnyTimes()
			},
			validateFunc: func() {},
		},
		{
			name:     "when facts interval is invalid uses default",
			interval: "7d",
			setupMock: func() {
				s.mockFactsKV.EXPECT().
					Put(gomock.Any(), "facts.test_machine_id", gomock.Any()).
					Return(uint64(1), nil).
					AnyTimes()
			},
			validateFunc: func() {},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			// Override the default to be very short so "empty" and "invalid"
			// cases also tick quickly.
			agent.SetDefaultFactsInterval(10 * time.Millisecond)

			cfg := agent.GetAgentAppConfig(s.testAgent)
			cfg.Agent.Facts.Interval = tt.interval
			agent.SetAgentAppConfig(s.testAgent, cfg)

			ctx, cancel := context.WithCancel(context.Background())
			agent.ExportStartFacts(ctx, s.testAgent, "test-machine-id", "test-agent")

			// Wait for at least one ticker refresh
			time.Sleep(50 * time.Millisecond)
			cancel()

			// Wait for goroutine to finish
			agent.WaitAgentWG(s.testAgent)

			tt.validateFunc()
		})
	}
}

func (s *FactsPublicTestSuite) TestGetFacts() {
	tests := []struct {
		name         string
		setupFunc    func()
		teardownFunc func()
		validateFunc func(result map[string]any)
	}{
		{
			name:      "when cachedFacts is nil returns nil",
			setupFunc: func() {},
			validateFunc: func(result map[string]any) {
				s.Nil(result)
			},
		},
		{
			name: "when cachedFacts populated returns fact map",
			setupFunc: func() {
				agent.SetAgentCachedFacts(s.testAgent, &job.FactsRegistration{
					Architecture: "amd64",
					CPUCount:     4,
					FQDN:         "test.local",
				})
			},
			validateFunc: func(result map[string]any) {
				s.Require().NotNil(result)
				s.Equal("amd64", result["architecture"])
				s.Equal(float64(4), result["cpu_count"])
				s.Equal("test.local", result["fqdn"])
			},
		},
		{
			name: "when marshal fails returns nil",
			setupFunc: func() {
				agent.SetAgentCachedFacts(s.testAgent, &job.FactsRegistration{
					Architecture: "amd64",
				})
				agent.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, fmt.Errorf("marshal failure")
				})
			},
			teardownFunc: func() {
				agent.ResetMarshalJSON()
			},
			validateFunc: func(result map[string]any) {
				s.Nil(result)
			},
		},
		{
			name: "when unmarshal fails returns nil",
			setupFunc: func() {
				agent.SetAgentCachedFacts(s.testAgent, &job.FactsRegistration{
					Architecture: "amd64",
				})
				agent.SetUnmarshalJSON(func(_ []byte, _ interface{}) error {
					return fmt.Errorf("unmarshal failure")
				})
			},
			teardownFunc: func() {
				agent.ResetUnmarshalJSON()
			},
			validateFunc: func(result map[string]any) {
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			agent.SetAgentCachedFacts(s.testAgent, nil)
			tt.setupFunc()
			if tt.teardownFunc != nil {
				defer tt.teardownFunc()
			}
			result := s.testAgent.GetFacts()
			tt.validateFunc(result)
		})
	}
}

func (s *FactsPublicTestSuite) TestFactsKey() {
	tests := []struct {
		name         string
		machineID    string
		validateFunc func(string)
	}{
		{
			name:      "simple machine ID",
			machineID: "abc-123-def",
			validateFunc: func(got string) {
				s.Equal("facts.abc_123_def", got)
			},
		},
		{
			name:      "machine ID with dots",
			machineID: "A1B2C3D4-E5F6.7890",
			validateFunc: func(got string) {
				s.Equal("facts.A1B2C3D4_E5F6_7890", got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportFactsKey(tt.machineID)
			tt.validateFunc(result)
		})
	}
}

func TestFactsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FactsPublicTestSuite))
}
