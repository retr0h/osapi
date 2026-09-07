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
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/iface"
	netifMocks "github.com/osapi-io/osapi/internal/provider/network/netplan/iface/mocks"
)

type ProcessorInterfacePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorInterfacePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorInterfacePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorInterfacePublicTestSuite) TestProcessInterfaceOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() iface.Provider
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "interface provider not available",
		},
		{
			name: "invalid interface operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() iface.Provider {
				return netifMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid interface operation: interface",
		},
		{
			name: "unsupported interface sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() iface.Provider {
				return netifMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported interface operation: interface.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var ifaceProvider iface.Provider
			if tt.setupMock != nil {
				ifaceProvider = tt.setupMock()
			}

			processor := agent.NewNetworkProcessor(
				nil, nil,
				ifaceProvider,
				nil,
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
			}
		})
	}
}

func (s *ProcessorInterfacePublicTestSuite) TestProcessInterfaceList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() iface.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful interface list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]iface.InterfaceEntry{
					{Name: "eth0"},
					{Name: "eth1"},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []iface.InterfaceEntry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 2)
				s.Equal("eth0", entries[0].Name)
			},
		},
		{
			name: "interface list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
				return m
			},
			expectError: true,
			errorMsg:    "permission denied",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				tt.setupMock(),
				nil,
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

func (s *ProcessorInterfacePublicTestSuite) TestProcessInterfaceGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() iface.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful interface get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.get",
				Data:      json.RawMessage(`{"name":"eth0"}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "eth0").Return(&iface.InterfaceEntry{
					Name: "eth0",
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entry iface.InterfaceEntry
				err := json.Unmarshal(result, &entry)
				s.NoError(err)
				s.Equal("eth0", entry.Name)
			},
		},
		{
			name: "interface get with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() iface.Provider {
				return netifMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal interface get data",
		},
		{
			name: "interface get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "interface.get",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				tt.setupMock(),
				nil,
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

func (s *ProcessorInterfacePublicTestSuite) TestProcessInterfaceCreate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() iface.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful interface create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.create",
				Data:      json.RawMessage(`{"name":"eth1","addresses":["10.0.0.5/24"]}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), iface.InterfaceEntry{
					Name:      "eth1",
					Addresses: []string{"10.0.0.5/24"},
				}).Return(&iface.InterfaceResult{
					Name:    "eth1",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r iface.InterfaceResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("eth1", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "interface create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() iface.Provider {
				return netifMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal interface create data",
		},
		{
			name: "interface create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.create",
				Data:      json.RawMessage(`{"name":"eth1"}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("deploy failed"))
				return m
			},
			expectError: true,
			errorMsg:    "deploy failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				tt.setupMock(),
				nil,
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

func (s *ProcessorInterfacePublicTestSuite) TestProcessInterfaceUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() iface.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful interface update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.update",
				Data:      json.RawMessage(`{"name":"eth1","addresses":["10.0.0.10/24"]}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), iface.InterfaceEntry{
					Name:      "eth1",
					Addresses: []string{"10.0.0.10/24"},
				}).Return(&iface.InterfaceResult{
					Name:    "eth1",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r iface.InterfaceResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("eth1", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "interface update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() iface.Provider {
				return netifMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal interface update data",
		},
		{
			name: "interface update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.update",
				Data:      json.RawMessage(`{"name":"eth1"}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
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
			processor := agent.NewNetworkProcessor(
				nil, nil,
				tt.setupMock(),
				nil,
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

func (s *ProcessorInterfacePublicTestSuite) TestProcessInterfaceDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() iface.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful interface delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.delete",
				Data:      json.RawMessage(`{"name":"eth1"}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "eth1").Return(&iface.InterfaceResult{
					Name:    "eth1",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r iface.InterfaceResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("eth1", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "interface delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() iface.Provider {
				return netifMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal interface delete data",
		},
		{
			name: "interface delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "interface.delete",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() iface.Provider {
				m := netifMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				tt.setupMock(),
				nil,
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

func TestProcessorInterfacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorInterfacePublicTestSuite))
}
