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
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/provider/node/sysctl"
	sysctlMocks "github.com/osapi-io/osapi/internal/provider/node/sysctl/mocks"
)

type ProcessorSysctlPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorSysctlPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorSysctlPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorSysctlPublicTestSuite) TestProcessSysctlOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() sysctl.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "sysctl provider not available",
		},
		{
			name: "invalid sysctl operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() sysctl.Provider {
				return sysctlMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid sysctl operation: sysctl",
		},
		{
			name: "unsupported sysctl sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() sysctl.Provider {
				return sysctlMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported sysctl operation: sysctl.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var sysctlProvider sysctl.Provider
			if tt.setupMock != nil {
				sysctlProvider = tt.setupMock()
			}

			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				sysctlProvider,
				nil, nil, nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
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

func (s *ProcessorSysctlPublicTestSuite) TestProcessSysctlList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() sysctl.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful sysctl list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]sysctl.Entry{
					{Key: "net.ipv4.ip_forward", Value: "1"},
					{Key: "vm.swappiness", Value: "10"},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []sysctl.Entry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 2)
				s.Equal("net.ipv4.ip_forward", entries[0].Key)
				s.Equal("1", entries[0].Value)
			},
		},
		{
			name: "sysctl list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
				return m
			},
			expectError: true,
			errorMsg:    "permission denied",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil, nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
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

func (s *ProcessorSysctlPublicTestSuite) TestProcessSysctlGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() sysctl.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful sysctl get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.get",
				Data:      json.RawMessage(`{"key":"net.ipv4.ip_forward"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "net.ipv4.ip_forward").Return(&sysctl.Entry{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entry sysctl.Entry
				err := json.Unmarshal(result, &entry)
				s.NoError(err)
				s.Equal("net.ipv4.ip_forward", entry.Key)
				s.Equal("1", entry.Value)
			},
		},
		{
			name: "sysctl get with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() sysctl.Provider {
				return sysctlMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal sysctl get data",
		},
		{
			name: "sysctl get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sysctl.get",
				Data:      json.RawMessage(`{"key":"missing.key"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "missing.key").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil, nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
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

func (s *ProcessorSysctlPublicTestSuite) TestProcessSysctlCreate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() sysctl.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful sysctl create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.create",
				Data:      json.RawMessage(`{"key":"net.ipv4.ip_forward","value":"1"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), sysctl.Entry{
					Key:   "net.ipv4.ip_forward",
					Value: "1",
				}).Return(&sysctl.CreateResult{
					Key:     "net.ipv4.ip_forward",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r sysctl.CreateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("net.ipv4.ip_forward", r.Key)
				s.True(r.Changed)
			},
		},
		{
			name: "sysctl create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() sysctl.Provider {
				return sysctlMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal sysctl create data",
		},
		{
			name: "sysctl create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.create",
				Data:      json.RawMessage(`{"key":"invalid.param","value":"bad"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("invalid parameter"))
				return m
			},
			expectError: true,
			errorMsg:    "invalid parameter",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil, nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
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

func (s *ProcessorSysctlPublicTestSuite) TestProcessSysctlUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() sysctl.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful sysctl update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.update",
				Data:      json.RawMessage(`{"key":"net.ipv4.ip_forward","value":"0"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), sysctl.Entry{
					Key:   "net.ipv4.ip_forward",
					Value: "0",
				}).Return(&sysctl.UpdateResult{
					Key:     "net.ipv4.ip_forward",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r sysctl.UpdateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("net.ipv4.ip_forward", r.Key)
				s.True(r.Changed)
			},
		},
		{
			name: "sysctl update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() sysctl.Provider {
				return sysctlMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal sysctl update data",
		},
		{
			name: "sysctl update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.update",
				Data:      json.RawMessage(`{"key":"net.ipv4.ip_forward","value":"0"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not managed"))
				return m
			},
			expectError: true,
			errorMsg:    "not managed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil, nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
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

func (s *ProcessorSysctlPublicTestSuite) TestProcessSysctlDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() sysctl.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful sysctl delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.delete",
				Data:      json.RawMessage(`{"key":"net.ipv4.ip_forward"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "net.ipv4.ip_forward").Return(&sysctl.DeleteResult{
					Key:     "net.ipv4.ip_forward",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r sysctl.DeleteResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("net.ipv4.ip_forward", r.Key)
				s.True(r.Changed)
			},
		},
		{
			name: "sysctl delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() sysctl.Provider {
				return sysctlMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal sysctl delete data",
		},
		{
			name: "sysctl delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sysctl.delete",
				Data:      json.RawMessage(`{"key":"missing.key"}`),
			},
			setupMock: func() sysctl.Provider {
				m := sysctlMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "missing.key").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil, nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
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

func TestProcessorSysctlPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorSysctlPublicTestSuite))
}
