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
	"github.com/osapi-io/osapi/internal/provider/node/process"
	processMocks "github.com/osapi-io/osapi/internal/provider/node/process/mocks"
)

type ProcessorProcessPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorProcessPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorProcessPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorProcessPublicTestSuite) TestProcessProcessOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() process.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.list",
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "process provider not available",
		},
		{
			name: "invalid operation format (no sub-operation)",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process",
			},
			setupMock: func() process.Provider {
				return processMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid process operation: process",
		},
		{
			name: "unsupported process sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.unknown",
			},
			setupMock: func() process.Provider {
				return processMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported process operation: process.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var processProvider process.Provider
			if tt.setupMock != nil {
				processProvider = tt.setupMock()
			}

			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				processProvider,
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

func (s *ProcessorProcessPublicTestSuite) TestProcessProcessList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() process.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.list",
			},
			setupMock: func() process.Provider {
				m := processMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]process.Info{
					{
						PID:     1,
						Name:    "systemd",
						User:    "root",
						State:   "running",
						Command: "/sbin/init",
					},
					{
						PID:     42,
						Name:    "sshd",
						User:    "root",
						State:   "running",
						Command: "/usr/sbin/sshd",
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var infos []process.Info
				err := json.Unmarshal(result, &infos)
				s.NoError(err)
				s.Len(infos, 2)
				s.Equal(1, infos[0].PID)
				s.Equal("systemd", infos[0].Name)
				s.Equal(42, infos[1].PID)
			},
		},
		{
			name: "list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.list",
			},
			setupMock: func() process.Provider {
				m := processMocks.NewMockProvider(s.mockCtrl)
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
				nil, nil, nil, nil,
				tt.setupMock(),
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

func (s *ProcessorProcessPublicTestSuite) TestProcessProcessGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() process.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.get",
				Data:      json.RawMessage(`{"pid": 1234}`),
			},
			setupMock: func() process.Provider {
				m := processMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), 1234).Return(&process.Info{
					PID:     1234,
					Name:    "nginx",
					User:    "www-data",
					State:   "running",
					Command: "/usr/sbin/nginx",
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var info process.Info
				err := json.Unmarshal(result, &info)
				s.NoError(err)
				s.Equal(1234, info.PID)
				s.Equal("nginx", info.Name)
				s.Equal("www-data", info.User)
			},
		},
		{
			name: "get unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() process.Provider {
				return processMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal process get data",
		},
		{
			name: "get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "process.get",
				Data:      json.RawMessage(`{"pid": 9999}`),
			},
			setupMock: func() process.Provider {
				m := processMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), 9999).Return(nil, errors.New("process not found"))
				return m
			},
			expectError: true,
			errorMsg:    "process not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				nil, nil, nil, nil,
				tt.setupMock(),
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

func (s *ProcessorProcessPublicTestSuite) TestProcessProcessSignal() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() process.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful signal",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "process.signal",
				Data:      json.RawMessage(`{"pid": 1234, "signal": "TERM"}`),
			},
			setupMock: func() process.Provider {
				m := processMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Signal(gomock.Any(), 1234, "TERM").Return(&process.SignalResult{
					PID:     1234,
					Signal:  "TERM",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r process.SignalResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal(1234, r.PID)
				s.Equal("TERM", r.Signal)
				s.True(r.Changed)
			},
		},
		{
			name: "signal unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "process.signal",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() process.Provider {
				return processMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal process signal data",
		},
		{
			name: "signal provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "process.signal",
				Data:      json.RawMessage(`{"pid": 1234, "signal": "KILL"}`),
			},
			setupMock: func() process.Provider {
				m := processMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Signal(gomock.Any(), 1234, "KILL").
					Return(nil, errors.New("permission denied"))
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
				nil, nil, nil, nil,
				tt.setupMock(),
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

func TestProcessorProcessPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorProcessPublicTestSuite))
}
