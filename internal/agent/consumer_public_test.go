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
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	agentmocks "github.com/osapi-io/osapi/internal/agent/mocks"
	"github.com/osapi-io/osapi/internal/config"
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

type ConsumerPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	testAgent     *agent.Agent
}

func (s *ConsumerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)

	appConfig := config.Config{
		NATS: config.NATS{
			Stream: config.NATSStream{
				Name: "test-stream",
			},
		},
		Agent: config.AgentConfig{
			Hostname:   "test-agent",
			QueueGroup: "test-queue",
			MaxJobs:    5,
			Consumer: config.AgentConsumer{
				AckWait:       "30s",
				BackOff:       []string{"1s", "2s"},
				MaxDeliver:    3,
				MaxAckPending: 10,
				ReplayPolicy:  "instant",
			},
		},
	}

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
	})
}

func (s *ConsumerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ConsumerPublicTestSuite) TestConsumerNamePrefix() {
	tests := []struct {
		name         string
		consumerName string
		validateFunc func(result string)
	}{
		{
			name:         "when consumer name is configured returns name with underscore",
			consumerName: "my-consumer",
			validateFunc: func(result string) {
				s.Equal("my-consumer_", result)
			},
		},
		{
			name:         "when consumer name is empty returns empty string",
			consumerName: "",
			validateFunc: func(result string) {
				s.Equal("", result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfg := agent.GetAgentAppConfig(s.testAgent)
			cfg.Agent.Consumer.Name = tt.consumerName
			agent.SetAgentAppConfig(s.testAgent, cfg)

			result := agent.ExportConsumerNamePrefix(s.testAgent)
			tt.validateFunc(result)
		})
	}
}

func (s *ConsumerPublicTestSuite) TestConsumeQueryJobs() {
	tests := []struct {
		name       string
		hostname   string
		labels     map[string]string
		setupMocks func()
		expectErr  bool
	}{
		{
			name:     "successful query job consumption",
			hostname: "test-agent",
			setupMocks: func() {
				// Expect consumer creation for all 3 query patterns
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(3)

				// Expect job consumption for all consumers
				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(3)
			},
			expectErr: false,
		},
		{
			name:     "consumer creation failure",
			hostname: "test-agent",
			setupMocks: func() {
				// Fail all consumer creations
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(errors.New("consumer creation failed")).
					Times(3)

				// No consumption should happen since creation failed
			},
			expectErr: false, // Should not return error, just log and continue
		},
		{
			name:     "partial consumer creation failure",
			hostname: "test-agent",
			setupMocks: func() {
				// First consumer succeeds
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(1)

				// Second and third consumers fail
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(errors.New("consumer creation failed")).
					Times(2)

				// Only one consumption should happen
				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(1)
			},
			expectErr: false,
		},
		{
			name:     "empty hostname",
			hostname: "",
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(3)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(3)
			},
			expectErr: false,
		},
		{
			name:     "consume error logged",
			hostname: "test-agent",
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(3)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("connection lost")).
					Times(3)
			},
			expectErr: false,
		},
		{
			name:     "with labels creates extra consumers",
			hostname: "test-agent",
			labels: map[string]string{
				"group": "web.dev.us-east",
			},
			setupMocks: func() {
				// 3 base + 3 prefix levels for group:web.dev.us-east = 6 total
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(6)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(6)
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cfg := agent.GetAgentAppConfig(s.testAgent)
			cfg.Agent.Labels = tt.labels
			agent.SetAgentAppConfig(s.testAgent, cfg)

			tt.setupMocks()

			err := agent.ExportConsumeQueryJobs(ctx, s.testAgent, tt.hostname)

			if tt.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}

			// Allow goroutines to execute before cleanup
			time.Sleep(10 * time.Millisecond)
		})
	}
}

func (s *ConsumerPublicTestSuite) TestConsumeModifyJobs() {
	tests := []struct {
		name       string
		hostname   string
		labels     map[string]string
		setupMocks func()
		expectErr  bool
	}{
		{
			name:     "successful modify job consumption",
			hostname: "test-agent",
			setupMocks: func() {
				// Expect consumer creation for all 3 modify patterns
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(3)

				// Expect job consumption for all consumers
				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(3)
			},
			expectErr: false,
		},
		{
			name:     "consumer creation failure",
			hostname: "test-agent",
			setupMocks: func() {
				// Fail all consumer creations
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(errors.New("consumer creation failed")).
					Times(3)
			},
			expectErr: false, // Should not return error, just log and continue
		},
		{
			name:     "hostname with special characters",
			hostname: "test-agent.domain.com",
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(3)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(3)
			},
			expectErr: false,
		},
		{
			name:     "consume error logged",
			hostname: "test-agent",
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(3)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("connection lost")).
					Times(3)
			},
			expectErr: false,
		},
		{
			name:     "with labels creates extra consumers",
			hostname: "test-agent",
			labels: map[string]string{
				"group": "web.dev.us-east",
			},
			setupMocks: func() {
				// 3 base + 3 prefix levels for group:web.dev.us-east = 6 total
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(nil).
					Times(6)

				s.mockJobClient.EXPECT().
					ConsumeJobs(gomock.Any(), "test-stream", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Canceled).
					Times(6)
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cfg := agent.GetAgentAppConfig(s.testAgent)
			cfg.Agent.Labels = tt.labels
			agent.SetAgentAppConfig(s.testAgent, cfg)

			tt.setupMocks()

			err := agent.ExportConsumeModifyJobs(ctx, s.testAgent, tt.hostname)

			if tt.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}

			// Allow goroutines to execute before cleanup
			time.Sleep(10 * time.Millisecond)
		})
	}
}

func (s *ConsumerPublicTestSuite) TestCreateConsumer() {
	tests := []struct {
		name          string
		streamName    string
		consumerName  string
		filterSubject string
		agentConsumer config.AgentConsumer
		setupMocks    func()
		expectErr     bool
		errorMsg      string
	}{
		{
			name:          "successful consumer creation with instant replay",
			streamName:    "test-stream",
			consumerName:  "test-consumer",
			filterSubject: "jobs.query._any",
			agentConsumer: config.AgentConsumer{
				AckWait:       "30s",
				BackOff:       []string{"1s", "2s", "5s"},
				MaxDeliver:    3,
				MaxAckPending: 10,
				ReplayPolicy:  "instant",
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Do(func(_ context.Context, _ string, cfg jetstream.ConsumerConfig) {
						s.Equal("test-consumer", cfg.Durable)
						s.Equal("jobs.query._any", cfg.FilterSubject)
						s.Equal(jetstream.AckExplicitPolicy, cfg.AckPolicy)
						s.Equal(jetstream.DeliverAllPolicy, cfg.DeliverPolicy)
						s.Equal(3, cfg.MaxDeliver)
						s.Equal(30*time.Second, cfg.AckWait)
						s.Len(cfg.BackOff, 3)
						s.Equal(10, cfg.MaxAckPending)
						s.Equal(jetstream.ReplayInstantPolicy, cfg.ReplayPolicy)
					}).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name:          "successful consumer creation with original replay",
			streamName:    "test-stream",
			consumerName:  "test-consumer-orig",
			filterSubject: "jobs.modify._any",
			agentConsumer: config.AgentConsumer{
				AckWait:       "60s",
				BackOff:       []string{"2s"},
				MaxDeliver:    5,
				MaxAckPending: 20,
				ReplayPolicy:  "original",
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Do(func(_ context.Context, _ string, cfg jetstream.ConsumerConfig) {
						s.Equal("test-consumer-orig", cfg.Durable)
						s.Equal("jobs.modify._any", cfg.FilterSubject)
						s.Equal(60*time.Second, cfg.AckWait)
						s.Len(cfg.BackOff, 1)
						s.Equal(5, cfg.MaxDeliver)
						s.Equal(20, cfg.MaxAckPending)
						s.Equal(jetstream.ReplayOriginalPolicy, cfg.ReplayPolicy)
					}).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name:          "consumer creation failure",
			streamName:    "test-stream",
			consumerName:  "test-consumer",
			filterSubject: "jobs.query._any",
			agentConsumer: config.AgentConsumer{
				AckWait:       "30s",
				BackOff:       []string{"1s"},
				MaxDeliver:    3,
				MaxAckPending: 10,
				ReplayPolicy:  "instant",
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Return(errors.New("stream not found"))
			},
			expectErr: true,
			errorMsg:  "stream not found",
		},
		{
			name:          "invalid duration in config",
			streamName:    "test-stream",
			consumerName:  "test-consumer",
			filterSubject: "jobs.query._any",
			agentConsumer: config.AgentConsumer{
				AckWait:       "invalid-duration",
				BackOff:       []string{"invalid", "2s"},
				MaxDeliver:    3,
				MaxAckPending: 10,
				ReplayPolicy:  "instant",
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					CreateOrUpdateConsumer(gomock.Any(), "test-stream", gomock.Any()).
					Do(func(_ context.Context, _ string, cfg jetstream.ConsumerConfig) {
						// Should have default AckWait (0) and only valid BackOff durations
						s.Equal(time.Duration(0), cfg.AckWait)
						s.Len(cfg.BackOff, 1) // Only valid "2s" should be included
					}).
					Return(nil)
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Update agent config for this test
			cfg := agent.GetAgentAppConfig(s.testAgent)
			cfg.Agent.Consumer = tt.agentConsumer
			agent.SetAgentAppConfig(s.testAgent, cfg)

			tt.setupMocks()

			err := agent.ExportCreateConsumer(
				context.Background(),
				s.testAgent,
				tt.streamName,
				tt.consumerName,
				tt.filterSubject,
			)

			if tt.expectErr {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *ConsumerPublicTestSuite) TestHandleJobMessageJS() {
	tests := []struct {
		name       string
		msgData    []byte
		msgSubject string
		setupMocks func()
		expectErr  bool
		errorMsg   string
	}{
		{
			name:       "successful message handling",
			msgData:    []byte("test-job-key"),
			msgSubject: "jobs.query.test-agent",
			setupMocks: func() {
				// Mock successful job data retrieval and processing
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.test-job-key").
					Return([]byte(`{
						"id": "test-job-123",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				// Mock status event writes
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-key", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-key", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-key", "completed", gomock.Any(), gomock.Any()).
					Return(nil)

				// Mock response write
				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "test-job-key", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectErr: false,
		},
		{
			name:       "job processing failure",
			msgData:    []byte("failed-job-key"),
			msgSubject: "jobs.query.test-agent",
			setupMocks: func() {
				// Mock job data retrieval failure
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.failed-job-key").
					Return(nil, errors.New("job not found"))
			},
			expectErr: true,
			errorMsg:  "job not found",
		},
		{
			name:       "invalid subject",
			msgData:    []byte("test-job-key"),
			msgSubject: "invalid.subject",
			setupMocks: func() {
				// No mocks needed as it should fail early
			},
			expectErr: true,
			errorMsg:  "failed to parse subject",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			mockMsg := agentmocks.NewMockMsg(s.mockCtrl)
			mockMsg.EXPECT().Subject().Return(tt.msgSubject).AnyTimes()
			mockMsg.EXPECT().Data().Return(tt.msgData).AnyTimes()
			mockMsg.EXPECT().Headers().Return(nil).AnyTimes()

			err := agent.ExportHandleJobMessageJS(s.testAgent, mockMsg)

			if tt.expectErr {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func TestConsumerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConsumerPublicTestSuite))
}
