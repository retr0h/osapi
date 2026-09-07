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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/config"
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
	nodeHost "github.com/osapi-io/osapi/internal/provider/node/host"
	hostMocks "github.com/osapi-io/osapi/internal/provider/node/host/mocks"
	loadMocks "github.com/osapi-io/osapi/internal/provider/node/load/mocks"
	memMocks "github.com/osapi-io/osapi/internal/provider/node/mem/mocks"
	processMocks "github.com/osapi-io/osapi/internal/telemetry/process/mocks"
)

type ProcessorPublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
	testAgent     *agent.Agent
}

func (s *ProcessorPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)

	appFs := memfs.New()
	appConfig := config.Config{
		NATS: config.NATS{
			Stream: config.NATSStream{Name: "test-stream"},
		},
		Agent: config.AgentConfig{
			Hostname:   "test-agent",
			QueueGroup: "test-queue",
			MaxJobs:    5,
		},
	}

	// Create mock providers
	hostMock := hostMocks.NewDefaultMockProvider(s.mockCtrl)
	hostMock.EXPECT().
		UpdateHostname("success-host").
		Return(&nodeHost.UpdateHostnameResult{Changed: true}, nil).
		AnyTimes()
	hostMock.EXPECT().
		UpdateHostname(gomock.Any()).
		Return(nil, fmt.Errorf("host: %w", provider.ErrUnsupported)).
		AnyTimes()
	diskMock := diskMocks.NewDefaultMockProvider(s.mockCtrl)
	memMock := memMocks.NewDefaultMockProvider(s.mockCtrl)
	loadMock := loadMocks.NewDefaultMockProvider(s.mockCtrl)

	// Use plain DNS mock to avoid hardcoded interface expectations
	dnsMock := dnsMocks.NewPlainMockProvider(s.mockCtrl)
	// Set up expectations for eth0 interface used in tests
	dnsMock.EXPECT().GetResolvConfByInterface("eth0").Return(&dns.GetResult{
		DNSServers:    []string{"192.168.1.1", "8.8.8.8"},
		SearchDomains: []string{"example.com"},
	}, nil).AnyTimes()
	dnsMock.EXPECT().
		UpdateResolvConfByInterface(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&dns.UpdateResult{Changed: true}, nil).
		AnyTimes()

	// Use plain ping mock to avoid hardcoded address expectations
	pingMock := pingMocks.NewPlainMockProvider(s.mockCtrl)
	// Set up expectations for addresses used in tests
	pingMock.EXPECT().Do("127.0.0.1").Return(&ping.Result{
		PacketsSent:     3,
		PacketsReceived: 3,
		PacketLoss:      0,
	}, nil).AnyTimes()
	pingMock.EXPECT().Do("8.8.8.8").Return(&ping.Result{
		PacketsSent:     3,
		PacketsReceived: 3,
		PacketLoss:      0,
	}, nil).AnyTimes()

	netinfoMock := netinfoMocks.NewDefaultMockProvider(s.mockCtrl)
	commandMock := commandMocks.NewDefaultMockProvider(s.mockCtrl)
	fMock := fileMocks.NewDefaultMockProvider(s.mockCtrl)

	s.testAgent = newTestAgent(newTestAgentParams{
		appFs:           appFs,
		appConfig:       appConfig,
		logger:          slog.Default(),
		jobClient:       s.mockJobClient,
		streamName:      "test-stream",
		hostProvider:    hostMock,
		diskProvider:    diskMock,
		memProvider:     memMock,
		loadProvider:    loadMock,
		dnsProvider:     dnsMock,
		pingProvider:    pingMock,
		netinfoProvider: netinfoMock,
		commandProvider: commandMock,
		fileProvider:    fMock,
		processProvider: processMocks.NewDefaultMockProvider(s.mockCtrl),
	})
}

func (s *ProcessorPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorPublicTestSuite) TestProcessJobOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful node hostname operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "hostname.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "hostname")
				s.IsType("", response["hostname"])
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful node status operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "status.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "hostname")
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful system uptime operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "uptime.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "uptime_seconds")
				s.Contains(response, "uptime")
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful system OS info operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "osinfo.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful system disk operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "disk.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "disks")
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful system memory operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "memory.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful system load operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "load.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful network DNS query operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "dns.get",
				Data:      json.RawMessage(`{"interface": "eth0"}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful network DNS modify operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "dns.update",
				Data: json.RawMessage(
					`{"servers": ["8.8.8.8"], "search_domains": ["example.com"], "interface": "eth0"}`,
				),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "success")
				s.Contains(response, "message")
				s.Equal(true, response["changed"])
			},
		},
		{
			name: "successful network ping operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "ping.do",
				Data:      json.RawMessage(`{"address": "8.8.8.8"}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name: "successful command exec operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "command",
				Operation: "exec.execute",
				Data:      json.RawMessage(`{"command":"ls","args":["-la"]}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "stdout")
			},
		},
		{
			name: "successful command shell operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "command",
				Operation: "shell.execute",
				Data:      json.RawMessage(`{"command":"echo hello"}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Contains(response, "stdout")
			},
		},
		{
			name: "successful file deploy operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "deploy.execute",
				Data: json.RawMessage(
					`{"object_name":"app.conf","path":"/etc/mock/file.conf","content_type":"raw"}`,
				),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(true, response["changed"])
			},
		},
		{
			name: "successful file status operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "file",
				Operation: "status.get",
				Data:      json.RawMessage(`{"path":"/etc/mock/file.conf"}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal("in-sync", response["status"])
			},
		},
		{
			name: "docker category routes to docker processor",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest"}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "docker runtime not available")
				s.Nil(result)
			},
		},
		{
			name: "unsupported job category",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "unsupported",
				Operation: "test.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported job category: unsupported")
				s.Nil(result)
			},
		},
		{
			name: "unsupported node operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "unsupported.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported node operation: unsupported.get")
				s.Nil(result)
			},
		},
		{
			name: "unsupported network operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "unsupported.get",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported network operation: unsupported.get")
				s.Nil(result)
			},
		},
		{
			name: "network ping missing address",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "ping.do",
				Data:      json.RawMessage(`{}`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "missing ping address")
				s.Nil(result)
			},
		},
		{
			name: "network ping invalid data format",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "ping.do",
				Data:      json.RawMessage(`invalid json`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "failed to parse ping data")
				s.Nil(result)
			},
		},
		{
			name: "network DNS invalid data format",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "dns.get",
				Data:      json.RawMessage(`invalid json`),
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "failed to parse DNS data")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(agent.ExportProcessJobOperation(s.testAgent, tt.jobRequest))
		})
	}
}

func (s *ProcessorPublicTestSuite) TestSystemOperations() {
	tests := []struct {
		name        string
		operation   string
		labels      map[string]string
		expectError bool
		validate    func(json.RawMessage)
	}{
		{
			name:      "get hostname",
			operation: "hostname.get",
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Contains(response, "hostname")
				s.Equal(false, response["changed"])
			},
		},
		{
			name:      "get hostname with labels",
			operation: "hostname.get",
			labels:    map[string]string{"group": "web.dev.us-east"},
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Contains(response, "hostname")
				s.Contains(response, "labels")
				labels, ok := response["labels"].(map[string]interface{})
				s.True(ok)
				s.Equal("web.dev.us-east", labels["group"])
				s.Equal(false, response["changed"])
			},
		},
		{
			name:      "get node status",
			operation: "status.get",
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Contains(response, "hostname")
				s.Equal(false, response["changed"])
			},
		},
		{
			name:      "get uptime",
			operation: "uptime.get",
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Contains(response, "uptime_seconds")
				s.Equal(false, response["changed"])
			},
		},
	}

	// Hostname update tests (TypeModify).
	modifyTests := []struct {
		name        string
		operation   string
		data        string
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name:        "update hostname returns unsupported",
			operation:   "hostname.update",
			data:        `{"hostname": "new-host"}`,
			expectError: true,
			errorMsg:    "operation not supported",
		},
		{
			name:      "update hostname succeeds",
			operation: "hostname.update",
			data:      `{"hostname": "success-host"}`,
			validate: func(result json.RawMessage) {
				var response map[string]interface{}
				err := json.Unmarshal(result, &response)
				s.NoError(err)
				s.Equal("success-host", response["hostname"])
				s.Equal(true, response["changed"])
			},
		},
		{
			name:        "update hostname with invalid data",
			operation:   "hostname.update",
			data:        `invalid json`,
			expectError: true,
			errorMsg:    "invalid hostname update data",
		},
	}

	for _, tt := range modifyTests {
		s.Run(tt.name, func() {
			request := job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: tt.operation,
				Data:      json.RawMessage(tt.data),
			}

			result, err := agent.ExportProcessNodeOperation(s.testAgent, request)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
			} else {
				s.NoError(err)
				s.NotNil(result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			cfg := agent.GetAgentAppConfig(s.testAgent)
			cfg.Agent.Labels = tt.labels
			agent.SetAgentAppConfig(s.testAgent, cfg)

			request := job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: tt.operation,
				Data:      json.RawMessage(`{}`),
			}

			result, err := agent.ExportProcessNodeOperation(s.testAgent, request)

			if tt.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.NotNil(result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}

func (s *ProcessorPublicTestSuite) TestNetworkOperations() {
	tests := []struct {
		name         string
		operation    string
		data         string
		validateFunc func(json.RawMessage, error)
	}{
		{
			name:      "DNS query with interface",
			operation: "dns.get",
			data:      `{"interface": "eth0"}`,
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name:      "DNS query without interface",
			operation: "dns.get",
			data:      `{}`,
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name:      "ping with valid address",
			operation: "ping.do",
			data:      `{"address": "127.0.0.1"}`,
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var response map[string]interface{}
				decodeErr := json.Unmarshal(result, &response)
				s.NoError(decodeErr)
				s.Equal(false, response["changed"])
			},
		},
		{
			name:      "ping without address",
			operation: "ping.execute",
			data:      `{}`,
			validateFunc: func(_ json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "missing ping address")
			},
		},
		{
			name:      "unsupported network operation",
			operation: "unknown.get",
			data:      `{}`,
			validateFunc: func(_ json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported network operation")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			request := job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: tt.operation,
				Data:      json.RawMessage(tt.data),
			}

			tt.validateFunc(agent.ExportProcessNetworkOperation(s.testAgent, request))
		})
	}
}

func (s *ProcessorPublicTestSuite) TestProviderFactoryMethods() {
	tests := []struct {
		name        string
		getProvider func() interface{}
	}{
		{
			name:        "getHostProvider",
			getProvider: func() interface{} { return agent.ExportGetHostProvider(s.testAgent) },
		},
		{
			name:        "getDiskProvider",
			getProvider: func() interface{} { return agent.ExportGetDiskProvider(s.testAgent) },
		},
		{
			name:        "getMemProvider",
			getProvider: func() interface{} { return agent.ExportGetMemProvider(s.testAgent) },
		},
		{
			name:        "getLoadProvider",
			getProvider: func() interface{} { return agent.ExportGetLoadProvider(s.testAgent) },
		},
		{
			name:        "getDNSProvider",
			getProvider: func() interface{} { return agent.ExportGetDNSProvider(s.testAgent) },
		},
		{
			name:        "getPingProvider",
			getProvider: func() interface{} { return agent.ExportGetPingProvider(s.testAgent) },
		},
		{
			name:        "getCommandProvider",
			getProvider: func() interface{} { return agent.ExportGetCommandProvider(s.testAgent) },
		},
		{
			name:        "getFileProvider",
			getProvider: func() interface{} { return agent.ExportGetFileProvider(s.testAgent) },
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			provider := tt.getProvider()
			s.NotNil(provider)
		})
	}
}

func (s *ProcessorPublicTestSuite) TestSystemOperationErrors() {
	tests := []struct {
		name        string
		operation   string
		errorMsg    string
		createAgent func() *agent.Agent
	}{
		{
			name:      "hostname provider error",
			operation: "hostname.get",
			errorMsg:  "hostname unavailable",
			createAgent: func() *agent.Agent {
				hostMock := hostMocks.NewPlainMockProvider(s.mockCtrl)
				hostMock.EXPECT().GetHostname().Return("", errors.New("hostname unavailable"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMock,
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "uptime provider error",
			operation: "uptime.get",
			errorMsg:  "uptime unavailable",
			createAgent: func() *agent.Agent {
				hostMock := hostMocks.NewPlainMockProvider(s.mockCtrl)
				hostMock.EXPECT().
					GetUptime().
					Return(time.Duration(0), errors.New("uptime unavailable"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMock,
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "OS info provider error",
			operation: "os.get",
			errorMsg:  "os info unavailable",
			createAgent: func() *agent.Agent {
				hostMock := hostMocks.NewPlainMockProvider(s.mockCtrl)
				hostMock.EXPECT().GetOSInfo().Return(nil, errors.New("os info unavailable"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMock,
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "disk provider error",
			operation: "disk.get",
			errorMsg:  "disk unavailable",
			createAgent: func() *agent.Agent {
				diskMock := diskMocks.NewPlainMockProvider(s.mockCtrl)
				diskMock.EXPECT().GetLocalUsageStats().Return(nil, errors.New("disk unavailable"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMocks.NewPlainMockProvider(s.mockCtrl),
					diskProvider:    diskMock,
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "memory provider error",
			operation: "memory.get",
			errorMsg:  "memory unavailable",
			createAgent: func() *agent.Agent {
				memMock := memMocks.NewPlainMockProvider(s.mockCtrl)
				memMock.EXPECT().GetStats().Return(nil, errors.New("memory unavailable"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMocks.NewPlainMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMock,
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "load provider error",
			operation: "load.get",
			errorMsg:  "load unavailable",
			createAgent: func() *agent.Agent {
				loadMock := loadMocks.NewPlainMockProvider(s.mockCtrl)
				loadMock.EXPECT().GetAverageStats().Return(nil, errors.New("load unavailable"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMocks.NewPlainMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMock,
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := tt.createAgent()
			request := job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: tt.operation,
				Data:      json.RawMessage(`{}`),
			}

			result, err := agent.ExportProcessNodeOperation(a, request)

			s.Error(err)
			s.Contains(err.Error(), tt.errorMsg)
			s.Nil(result)
		})
	}
}

func (s *ProcessorPublicTestSuite) TestNetworkOperationErrors() {
	tests := []struct {
		name        string
		operation   string
		jobType     job.Type
		data        string
		errorMsg    string
		createAgent func() *agent.Agent
	}{
		{
			name:      "DNS get error",
			operation: "dns.get",
			jobType:   job.TypeQuery,
			data:      `{"interface": "eth0"}`,
			errorMsg:  "DNS lookup failed",
			createAgent: func() *agent.Agent {
				dnsMock := dnsMocks.NewPlainMockProvider(s.mockCtrl)
				dnsMock.EXPECT().
					GetResolvConfByInterface("eth0").
					Return(nil, errors.New("DNS lookup failed"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMocks.NewPlainMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMock,
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "DNS update error",
			operation: "dns.update",
			jobType:   job.TypeModify,
			data:      `{"servers": ["8.8.8.8"], "search_domains": ["example.com"], "interface": "eth0"}`,
			errorMsg:  "DNS update failed",
			createAgent: func() *agent.Agent {
				dnsMock := dnsMocks.NewPlainMockProvider(s.mockCtrl)
				dnsMock.EXPECT().
					UpdateResolvConfByInterface(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("DNS update failed"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMocks.NewPlainMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMock,
					pingProvider:    pingMocks.NewPlainMockProvider(s.mockCtrl),
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
		{
			name:      "ping provider error",
			operation: "ping.do",
			jobType:   job.TypeQuery,
			data:      `{"address": "8.8.8.8"}`,
			errorMsg:  "ping failed",
			createAgent: func() *agent.Agent {
				pingMock := pingMocks.NewPlainMockProvider(s.mockCtrl)
				pingMock.EXPECT().Do("8.8.8.8").Return(nil, errors.New("ping timeout"))
				return newTestAgent(newTestAgentParams{
					appFs:           memfs.New(),
					appConfig:       config.Config{},
					jobClient:       s.mockJobClient,
					hostProvider:    hostMocks.NewPlainMockProvider(s.mockCtrl),
					diskProvider:    diskMocks.NewPlainMockProvider(s.mockCtrl),
					memProvider:     memMocks.NewPlainMockProvider(s.mockCtrl),
					loadProvider:    loadMocks.NewPlainMockProvider(s.mockCtrl),
					dnsProvider:     dnsMocks.NewPlainMockProvider(s.mockCtrl),
					pingProvider:    pingMock,
					netinfoProvider: netinfoMocks.NewPlainMockProvider(s.mockCtrl),
					commandProvider: commandMocks.NewPlainMockProvider(s.mockCtrl),
				})
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := tt.createAgent()
			request := job.Request{
				Type:      tt.jobType,
				Category:  "network",
				Operation: tt.operation,
				Data:      json.RawMessage(tt.data),
			}

			result, err := agent.ExportProcessNetworkOperation(a, request)

			s.Error(err)
			s.Contains(err.Error(), tt.errorMsg)
			s.Nil(result)
		})
	}
}

func TestProcessorPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorPublicTestSuite))
}
