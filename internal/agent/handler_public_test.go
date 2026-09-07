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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	agentmocks "github.com/osapi-io/osapi/internal/agent/mocks"
	"github.com/osapi-io/osapi/internal/agent/pki"
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider"
	commandMocks "github.com/osapi-io/osapi/internal/provider/command/mocks"
	fileMocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
	netinfoMocks "github.com/osapi-io/osapi/internal/provider/network/netinfo/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
	dnsMocks "github.com/osapi-io/osapi/internal/provider/network/netplan/dns/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/ping"
	pingMocks "github.com/osapi-io/osapi/internal/provider/network/ping/mocks"
	diskMocks "github.com/osapi-io/osapi/internal/provider/node/disk/mocks"
	hostMocks "github.com/osapi-io/osapi/internal/provider/node/host/mocks"
	loadMocks "github.com/osapi-io/osapi/internal/provider/node/load/mocks"
	memMocks "github.com/osapi-io/osapi/internal/provider/node/mem/mocks"
	"github.com/osapi-io/osapi/internal/telemetry/metrics"
	processMocks "github.com/osapi-io/osapi/internal/telemetry/process/mocks"
)

// newTestMsg creates a jetstream.Msg mock that returns the given subject,
// data, and nil headers. All three methods are expected any number of times,
// mirroring how the handler accesses message fields.
func newTestMsg(
	ctrl *gomock.Controller,
	subject string,
	data []byte,
) jetstream.Msg {
	m := agentmocks.NewMockMsg(ctrl)
	m.EXPECT().Subject().Return(subject).AnyTimes()
	m.EXPECT().Data().Return(data).AnyTimes()
	m.EXPECT().Headers().Return(nil).AnyTimes()
	return m
}

type HandlerPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	testAgent     *agent.Agent
}

func (s *HandlerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)

	// Use plain DNS mock with appropriate expectations
	dnsMock := dnsMocks.NewPlainMockProvider(s.mockCtrl)
	dnsMock.EXPECT().GetResolvConfByInterface(gomock.Any()).Return(&dns.GetResult{
		DNSServers:    []string{"192.168.1.1", "8.8.8.8"},
		SearchDomains: []string{"example.com"},
	}, nil).AnyTimes()
	dnsMock.EXPECT().
		UpdateResolvConfByInterface(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&dns.UpdateResult{Changed: true}, nil).
		AnyTimes()

	// Use plain ping mock with appropriate expectations
	pingMock := pingMocks.NewPlainMockProvider(s.mockCtrl)
	pingMock.EXPECT().Do(gomock.Any()).Return(&ping.Result{
		PacketsSent:     3,
		PacketsReceived: 3,
		PacketLoss:      0,
	}, nil).AnyTimes()

	s.testAgent = newTestAgent(newTestAgentParams{
		jobClient:       s.mockJobClient,
		streamName:      "test-stream",
		hostProvider:    hostMocks.NewDefaultMockProvider(s.mockCtrl),
		diskProvider:    diskMocks.NewDefaultMockProvider(s.mockCtrl),
		memProvider:     memMocks.NewDefaultMockProvider(s.mockCtrl),
		loadProvider:    loadMocks.NewDefaultMockProvider(s.mockCtrl),
		dnsProvider:     dnsMock,
		pingProvider:    pingMock,
		netinfoProvider: netinfoMocks.NewDefaultMockProvider(s.mockCtrl),
		commandProvider: commandMocks.NewDefaultMockProvider(s.mockCtrl),
		fileProvider:    fileMocks.NewDefaultMockProvider(s.mockCtrl),
		processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
	})

	// Set up a real MeterProvider so the instrumentation nil-guards are exercised.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		port := l.Addr().(*net.TCPAddr).Port
		_ = l.Close()
		srv := metrics.New("127.0.0.1", port, slog.Default())
		s.testAgent.SetMeterProvider(srv.MeterProvider())
	}
}

func (s *HandlerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *HandlerPublicTestSuite) TestWriteStatusEvent() {
	tests := []struct {
		name        string
		jobID       string
		event       string
		data        map[string]interface{}
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name:  "when successful status event write",
			jobID: "test-job-123",
			event: "started",
			data:  map[string]interface{}{"agent_version": "1.0.0", "pid": 12345},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-123", "started", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name:  "when status event write with nil data",
			jobID: "test-job-456",
			event: "completed",
			data:  nil,
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-456", "completed", gomock.Any(), nil).
					Return(nil)
			},
			expectError: false,
		},
		{
			name:  "when status event write failure",
			jobID: "test-job-789",
			event: "failed",
			data:  map[string]interface{}{"error": "processing failed"},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-789", "failed", gomock.Any(), gomock.Any()).
					Return(errors.New("KV storage failed"))
			},
			expectError: true,
			errorMsg:    "KV storage failed",
		},
		{
			name:  "when empty job ID",
			jobID: "",
			event: "started",
			data:  map[string]interface{}{},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "", "started", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			err := agent.ExportWriteStatusEvent(
				context.Background(),
				s.testAgent,
				tt.jobID,
				tt.event,
				tt.data,
			)

			if tt.expectError {
				s.Error(err)
				if tt.errorMsg != "" {
					s.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *HandlerPublicTestSuite) TestHandleJobMessage() {
	tests := []struct {
		name        string
		setupMsg    func(ctrl *gomock.Controller) jetstream.Msg
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "when successful job processing",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("test-job-123"))
			},
			setupMocks: func() {
				// Mock job data retrieval
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.test-job-123").
					Return([]byte(`{
						"id": "test-job-123",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				// Mock status event writes
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-123", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-123", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-123", "completed", gomock.Any(), gomock.Any()).
					Return(nil)

				// Mock response write
				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "test-job-123", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when job processing fails",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("test-job-456"))
			},
			setupMocks: func() {
				// Mock job data retrieval
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.test-job-456").
					Return([]byte(`{
						"id": "test-job-456",
						"operation": {
							"type": "node.unsupported.get",
							"data": {}
						}
					}`), nil)

				// Mock status event writes
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-456", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-456", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "test-job-456", "failed", gomock.Any(), gomock.Any()).
					Return(nil)

				// Mock response write for failed job
				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "test-job-456", gomock.Any(), gomock.Any(), "failed", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: true,
			errorMsg:    "job processing failed",
		},
		{
			name: "when invalid subject format",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "invalid", []byte("test-job-789"))
			},
			setupMocks: func() {
				// No mocks needed as it should fail early
			},
			expectError: true,
			errorMsg:    "failed to parse subject",
		},
		{
			name: "when job not found",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("nonexistent-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.nonexistent-job").
					Return(nil, errors.New("job not found"))
			},
			expectError: true,
			errorMsg:    "job not found",
		},
		{
			name: "when invalid job data format",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("invalid-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.invalid-job").
					Return([]byte(`invalid json`), nil)
			},
			expectError: true,
			errorMsg:    "failed to parse job data",
		},
		{
			name: "when missing job ID",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("missing-id-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.missing-id-job").
					Return([]byte(`{
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)
			},
			expectError: true,
			errorMsg:    "invalid job format: missing id",
		},
		{
			name: "when missing operation",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("missing-op-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.missing-op-job").
					Return([]byte(`{
						"id": "missing-op-job"
					}`), nil)
			},
			expectError: true,
			errorMsg:    "invalid job format: missing operation",
		},
		{
			name: "when missing operation type",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("missing-type-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.missing-type-job").
					Return([]byte(`{
						"id": "missing-type-job",
						"operation": {
							"data": {}
						}
					}`), nil)
			},
			expectError: true,
			errorMsg:    "invalid operation format: missing type field",
		},
		{
			name: "when invalid operation type format",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("invalid-type-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.invalid-type-job").
					Return([]byte(`{
						"id": "invalid-type-job",
						"operation": {
							"type": "invalid",
							"data": {}
						}
					}`), nil)
			},
			expectError: true,
			errorMsg:    "invalid operation type format",
		},
		{
			name: "when acknowledged write error logged",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("ack-err-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.ack-err-job").
					Return([]byte(`{
						"id": "ack-err-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "ack-err-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(errors.New("ack write failed"))

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "ack-err-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "ack-err-job", "completed", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "ack-err-job", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when started write error logged",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("start-err-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.start-err-job").
					Return([]byte(`{
						"id": "start-err-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "start-err-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "start-err-job", "started", gomock.Any(), gomock.Any()).
					Return(errors.New("started write failed"))

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "start-err-job", "completed", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "start-err-job", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when completed write error logged",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("comp-err-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.comp-err-job").
					Return([]byte(`{
						"id": "comp-err-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "comp-err-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "comp-err-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "comp-err-job", "completed", gomock.Any(), gomock.Any()).
					Return(errors.New("completed write failed"))

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "comp-err-job", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when failed write error logged",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("fail-err-job"))
			},
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.fail-err-job").
					Return([]byte(`{
						"id": "fail-err-job",
						"operation": {
							"type": "node.unsupported.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fail-err-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fail-err-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fail-err-job", "failed", gomock.Any(), gomock.Any()).
					Return(errors.New("failed write failed"))

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "fail-err-job", gomock.Any(), gomock.Any(), "failed", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: true,
			errorMsg:    "job processing failed",
		},
		{
			name: "when processor returns ErrUnsupported sets skipped status",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("skip-job"))
			},
			setupMocks: func() {
				// Replace host provider with one that returns ErrUnsupported,
				// and restore the original after the mock expectations are set.
				origHost := agent.GetAgentHostProvider(s.testAgent)
				unsupportedHost := hostMocks.NewPlainMockProvider(s.mockCtrl)
				unsupportedHost.EXPECT().
					GetHostname().
					DoAndReturn(func() (string, error) {
						agent.SetAgentHostProvider(s.testAgent, origHost)
						return "", fmt.Errorf("linux: %w", provider.ErrUnsupported)
					})
				agent.SetAgentHostProvider(s.testAgent, unsupportedHost)

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.skip-job").
					Return([]byte(`{
						"id": "skip-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "skip-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "skip-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "skip-job", "skipped", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "skip-job", gomock.Any(), gomock.Any(), "skipped", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when skipped write error logged",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("skip-err-job"))
			},
			setupMocks: func() {
				origHost := agent.GetAgentHostProvider(s.testAgent)
				unsupportedHost := hostMocks.NewPlainMockProvider(s.mockCtrl)
				unsupportedHost.EXPECT().
					GetHostname().
					DoAndReturn(func() (string, error) {
						agent.SetAgentHostProvider(s.testAgent, origHost)
						return "", fmt.Errorf("linux: %w", provider.ErrUnsupported)
					})
				agent.SetAgentHostProvider(s.testAgent, unsupportedHost)

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.skip-err-job").
					Return([]byte(`{
						"id": "skip-err-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "skip-err-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "skip-err-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "skip-err-job", "skipped", gomock.Any(), gomock.Any()).
					Return(errors.New("skipped write failed"))

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "skip-err-job", gomock.Any(), gomock.Any(), "skipped", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when fact reference resolved in job data",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("fact-resolve-job"))
			},
			setupMocks: func() {
				agent.SetAgentCachedFacts(s.testAgent, &job.FactsRegistration{
					PrimaryInterface: "eth0",
				})

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.fact-resolve-job").
					Return([]byte(`{
						"id": "fact-resolve-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {"iface": "@fact.interface.primary"}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-resolve-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-resolve-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-resolve-job", "completed", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "fact-resolve-job", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when fact reference with nil cached facts writes error to KV",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("fact-nil-job"))
			},
			setupMocks: func() {
				agent.SetAgentCachedFacts(s.testAgent, nil)

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.fact-nil-job").
					Return([]byte(`{
						"id": "fact-nil-job",
						"operation": {
							"type": "network.dns.get",
							"data": {"interface": "@fact.interface.primary"}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-nil-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-nil-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-nil-job", "failed", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "fact-nil-job", gomock.Any(), gomock.Any(), "failed", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: true,
			errorMsg:    "facts not available",
		},
		{
			name: "when unresolvable fact reference writes error to KV",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("fact-fail-job"))
			},
			setupMocks: func() {
				agent.SetAgentCachedFacts(s.testAgent, &job.FactsRegistration{})

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.fact-fail-job").
					Return([]byte(`{
						"id": "fact-fail-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {"iface": "@fact.nonexistent"}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-fail-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-fail-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "fact-fail-job", "failed", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "fact-fail-job", gomock.Any(), gomock.Any(), "failed", gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expectError: true,
			errorMsg:    "failed to resolve fact references",
		},
		{
			name: "when response storage failure",
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("storage-fail-job"))
			},
			setupMocks: func() {
				// Mock successful job processing
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.storage-fail-job").
					Return([]byte(`{
						"id": "storage-fail-job",
						"operation": {
							"type": "node.hostname.get",
							"data": {}
						}
					}`), nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "storage-fail-job", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "storage-fail-job", "started", gomock.Any(), gomock.Any()).
					Return(nil)

				// Response is written before the status event; failure
				// here prevents the completed event from being written.
				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "storage-fail-job", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(errors.New("storage failure"))
			},
			expectError: true,
			errorMsg:    "failed to store job response",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			msg := tt.setupMsg(s.mockCtrl)
			err := agent.ExportHandleJobMessage(s.testAgent, msg)

			if tt.expectError {
				s.Error(err)
				if tt.errorMsg != "" {
					s.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *HandlerPublicTestSuite) TestHandleJobMessageModifyJobs() {
	tests := []struct {
		name        string
		subject     string
		jobData     string
		setupMocks  func()
		expectError bool
	}{
		{
			name:    "when modify job type identification",
			subject: "jobs.modify.test-agent",
			jobData: `{
				"id": "modify-job-123",
				"operation": {
					"type": "network.dns.update",
					"data": {
						"servers": ["8.8.8.8"],
						"search_domains": ["example.com"],
						"interface": "eth0"
					}
				}
			}`,
			setupMocks: func() {
				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.modify-job-123").
					Return([]byte(`{
						"id": "modify-job-123",
						"operation": {
							"type": "network.dns.update",
							"data": {
								"servers": ["8.8.8.8"],
								"search_domains": ["example.com"],
								"interface": "eth0"
							}
						}
					}`), nil)

				// Mock status events with specific job ID.
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "modify-job-123", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					AnyTimes()

				// Mock response write — verify job ID and completed status.
				s.mockJobClient.EXPECT().
					WriteJobResponse(
						gomock.Any(),
						"modify-job-123",
						gomock.Any(),
						gomock.Any(),
						"completed",
						gomock.Any(),
						gomock.Any(),
					).
					Return(nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			msg := newTestMsg(s.mockCtrl, tt.subject, []byte("modify-job-123"))
			err := agent.ExportHandleJobMessage(s.testAgent, msg)

			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *HandlerPublicTestSuite) TestExtractChanged() {
	tests := []struct {
		name string
		data json.RawMessage
		want *bool
	}{
		{
			name: "when empty data returns nil",
			data: nil,
			want: nil,
		},
		{
			name: "when invalid JSON returns nil",
			data: json.RawMessage(`not json`),
			want: nil,
		},
		{
			name: "when changed key missing returns nil",
			data: json.RawMessage(`{"success":true}`),
			want: nil,
		},
		{
			name: "when changed is non-bool returns nil",
			data: json.RawMessage(`{"changed":"yes"}`),
			want: nil,
		},
		{
			name: "when changed is true returns true",
			data: json.RawMessage(`{"changed":true}`),
			want: boolPtr(true),
		},
		{
			name: "when changed is false returns false",
			data: json.RawMessage(`{"changed":false}`),
			want: boolPtr(false),
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := agent.ExportExtractChanged(tt.data)

			s.Equal(tt.want, got)
		})
	}
}

func (s *HandlerPublicTestSuite) TestUnwrapJobEnvelope() {
	// Generate a keypair for signing.
	controllerPub, controllerPriv, err := ed25519.GenerateKey(rand.Reader)
	s.Require().NoError(err)

	tests := []struct {
		name        string
		setupPKI    func() *pki.Manager
		data        func() []byte
		wantPayload string
		expectError bool
		errorMsg    string
	}{
		{
			name: "when PKI disabled passes through raw data",
			setupPKI: func() *pki.Manager {
				return nil
			},
			data: func() []byte {
				return []byte(`{"id":"test","operation":{"type":"node.hostname.get"}}`)
			},
			wantPayload: `{"id":"test","operation":{"type":"node.hostname.get"}}`,
			expectError: false,
		},
		{
			name: "when valid signed envelope with correct controller key",
			setupPKI: func() *pki.Manager {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				m.SetControllerPublicKey(controllerPub)
				return m
			},
			data: func() []byte {
				payload := []byte(`{"id":"signed-test"}`)
				sig := ed25519.Sign(controllerPriv, payload)
				envelope := job.SignedEnvelope{
					Payload:     payload,
					Signature:   sig,
					Fingerprint: "SHA256:controller-fp",
				}
				data, _ := json.Marshal(envelope)
				return data
			},
			wantPayload: `{"id":"signed-test"}`,
			expectError: false,
		},
		{
			name: "when signed envelope with invalid signature",
			setupPKI: func() *pki.Manager {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				m.SetControllerPublicKey(controllerPub)
				return m
			},
			data: func() []byte {
				payload := []byte(`{"id":"bad-sig"}`)
				envelope := job.SignedEnvelope{
					Payload: payload,
					Signature: []byte(
						"invalid-signature-data-that-is-long-enough-for-ed25519-64-bytes!!",
					),
					Fingerprint: "SHA256:bad-fp",
				}
				data, _ := json.Marshal(envelope)
				return data
			},
			wantPayload: "",
			expectError: true,
			errorMsg:    "invalid controller signature",
		},
		{
			name: "when signed envelope without controller key skips verification",
			setupPKI: func() *pki.Manager {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				// No controller public key set.
				return m
			},
			data: func() []byte {
				payload := []byte(`{"id":"no-ctrl-key"}`)
				sig := ed25519.Sign(controllerPriv, payload)
				envelope := job.SignedEnvelope{
					Payload:     payload,
					Signature:   sig,
					Fingerprint: "SHA256:controller-fp",
				}
				data, _ := json.Marshal(envelope)
				return data
			},
			wantPayload: `{"id":"no-ctrl-key"}`,
			expectError: false,
		},
		{
			name: "when raw JSON with PKI enabled passes through",
			setupPKI: func() *pki.Manager {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				m.SetControllerPublicKey(controllerPub)
				return m
			},
			data: func() []byte {
				return []byte(`{"id":"raw-job","operation":{"type":"node.hostname.get"}}`)
			},
			wantPayload: `{"id":"raw-job","operation":{"type":"node.hostname.get"}}`,
			expectError: false,
		},
		{
			name: "when invalid JSON with PKI enabled passes through",
			setupPKI: func() *pki.Manager {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				return m
			},
			data: func() []byte {
				return []byte(`not json at all`)
			},
			wantPayload: "not json at all",
			expectError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			pkiMgr := tt.setupPKI()
			agent.SetAgentPKIManager(s.testAgent, pkiMgr)
			defer agent.SetAgentPKIManager(s.testAgent, nil)

			data := tt.data()
			result, err := agent.ExportUnwrapJobEnvelope(s.testAgent, data)

			if tt.expectError {
				s.Error(err)
				if tt.errorMsg != "" {
					s.Contains(err.Error(), tt.errorMsg)
				}
				return
			}

			s.NoError(err)
			s.Equal(tt.wantPayload, string(result))
		})
	}
}

func (s *HandlerPublicTestSuite) TestHandleJobMessageWithSignedEnvelope() {
	// Generate controller keypair for signing job data.
	controllerPub, controllerPriv, err := ed25519.GenerateKey(rand.Reader)
	s.Require().NoError(err)

	tests := []struct {
		name        string
		setupPKI    func()
		cleanupPKI  func()
		setupMsg    func(ctrl *gomock.Controller) jetstream.Msg
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "when signed job data processed successfully",
			setupPKI: func() {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				m.SetControllerPublicKey(controllerPub)
				agent.SetAgentPKIManager(s.testAgent, m)
			},
			cleanupPKI: func() {
				agent.SetAgentPKIManager(s.testAgent, nil)
			},
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("signed-job-ok"))
			},
			setupMocks: func() {
				// Create signed job data.
				payload := []byte(
					`{"id":"signed-job-ok","operation":{"type":"node.hostname.get","data":{}}}`,
				)
				sig := ed25519.Sign(controllerPriv, payload)
				envelope := job.SignedEnvelope{
					Payload:     payload,
					Signature:   sig,
					Fingerprint: "SHA256:ctrl",
				}
				envelopeJSON, _ := json.Marshal(envelope)

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.signed-job-ok").
					Return(envelopeJSON, nil)

				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "signed-job-ok", "acknowledged", gomock.Any(), gomock.Any()).
					Return(nil)
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "signed-job-ok", "started", gomock.Any(), gomock.Any()).
					Return(nil)
				s.mockJobClient.EXPECT().
					WriteStatusEvent(gomock.Any(), "signed-job-ok", "completed", gomock.Any(), gomock.Any()).
					Return(nil)
				s.mockJobClient.EXPECT().
					WriteJobResponse(gomock.Any(), "signed-job-ok", gomock.Any(), gomock.Any(), "completed", "", gomock.Any()).
					Return(nil)
			},
			expectError: false,
		},
		{
			name: "when signed job data with invalid signature fails",
			setupPKI: func() {
				m := pki.New(memfs.New(), "/tmp/pki", "agent")
				s.Require().NoError(m.LoadOrGenerate())
				m.SetControllerPublicKey(controllerPub)
				agent.SetAgentPKIManager(s.testAgent, m)
			},
			cleanupPKI: func() {
				agent.SetAgentPKIManager(s.testAgent, nil)
			},
			setupMsg: func(ctrl *gomock.Controller) jetstream.Msg {
				return newTestMsg(ctrl, "jobs.query.test-agent", []byte("bad-sig-job"))
			},
			setupMocks: func() {
				// Create job data with invalid signature.
				payload := []byte(`{"id":"bad-sig-job","operation":{"type":"node.hostname.get"}}`)
				envelope := job.SignedEnvelope{
					Payload: payload,
					Signature: []byte(
						"bad-signature-data-that-is-at-least-64-bytes-long-for-ed25519!!!",
					),
					Fingerprint: "SHA256:bad",
				}
				envelopeJSON, _ := json.Marshal(envelope)

				s.mockJobClient.EXPECT().
					GetJobData(gomock.Any(), "jobs.bad-sig-job").
					Return(envelopeJSON, nil)
			},
			expectError: true,
			errorMsg:    "job signature verification failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupPKI()
			defer tt.cleanupPKI()
			tt.setupMocks()

			msg := tt.setupMsg(s.mockCtrl)
			err := agent.ExportHandleJobMessage(s.testAgent, msg)

			if tt.expectError {
				s.Error(err)
				if tt.errorMsg != "" {
					s.Contains(err.Error(), tt.errorMsg)
				}
			} else {
				s.NoError(err)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func TestHandlerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HandlerPublicTestSuite))
}
