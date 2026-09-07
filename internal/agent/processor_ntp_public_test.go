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
	"github.com/osapi-io/osapi/internal/provider/node/ntp"
	ntpMocks "github.com/osapi-io/osapi/internal/provider/node/ntp/mocks"
)

type ProcessorNtpPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorNtpPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorNtpPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorNtpPublicTestSuite) TestProcessNtpOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() ntp.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "ntp.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "ntp provider not available",
		},
		{
			name: "invalid ntp operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "ntp",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() ntp.Provider {
				return ntpMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid ntp operation: ntp",
		},
		{
			name: "unsupported ntp sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "ntp.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() ntp.Provider {
				return ntpMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported ntp operation: ntp.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var ntpProvider ntp.Provider
			if tt.setupMock != nil {
				ntpProvider = tt.setupMock()
			}

			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil,
				ntpProvider, nil, nil,
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

func (s *ProcessorNtpPublicTestSuite) TestProcessNtpGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() ntp.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful ntp get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "ntp.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any()).Return(&ntp.Status{
					Synchronized:  true,
					Stratum:       2,
					CurrentSource: "time.cloudflare.com",
					Servers:       []string{"time.cloudflare.com", "ntp.ubuntu.com"},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var status ntp.Status
				err := json.Unmarshal(result, &status)
				s.NoError(err)
				s.True(status.Synchronized)
				s.Equal(2, status.Stratum)
				s.Equal("time.cloudflare.com", status.CurrentSource)
				s.Len(status.Servers, 2)
			},
		},
		{
			name: "ntp get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "ntp.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any()).Return(nil, errors.New("chronyc not found"))
				return m
			},
			expectError: true,
			errorMsg:    "chronyc not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil,
				tt.setupMock(), nil, nil,
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

func (s *ProcessorNtpPublicTestSuite) TestProcessNtpCreate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() ntp.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful ntp create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.create",
				Data:      json.RawMessage(`{"servers":["time.cloudflare.com","ntp.ubuntu.com"]}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), ntp.Config{
					Servers: []string{"time.cloudflare.com", "ntp.ubuntu.com"},
				}).Return(&ntp.CreateResult{
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r ntp.CreateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.True(r.Changed)
			},
		},
		{
			name: "ntp create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() ntp.Provider {
				return ntpMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal ntp create data",
		},
		{
			name: "ntp create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.create",
				Data:      json.RawMessage(`{"servers":["time.cloudflare.com"]}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
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
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil,
				tt.setupMock(), nil, nil,
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

func (s *ProcessorNtpPublicTestSuite) TestProcessNtpUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() ntp.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful ntp update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.update",
				Data:      json.RawMessage(`{"servers":["pool.ntp.org"]}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), ntp.Config{
					Servers: []string{"pool.ntp.org"},
				}).Return(&ntp.UpdateResult{
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r ntp.UpdateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.True(r.Changed)
			},
		},
		{
			name: "ntp update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() ntp.Provider {
				return ntpMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal ntp update data",
		},
		{
			name: "ntp update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.update",
				Data:      json.RawMessage(`{"servers":["pool.ntp.org"]}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("config not managed"))
				return m
			},
			expectError: true,
			errorMsg:    "config not managed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil,
				tt.setupMock(), nil, nil,
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

func (s *ProcessorNtpPublicTestSuite) TestProcessNtpDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() ntp.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful ntp delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.delete",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any()).Return(&ntp.DeleteResult{
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r ntp.DeleteResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.True(r.Changed)
			},
		},
		{
			name: "ntp delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "ntp.delete",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() ntp.Provider {
				m := ntpMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any()).Return(nil, errors.New("config not managed"))
				return m
			},
			expectError: true,
			errorMsg:    "config not managed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil,
				tt.setupMock(), nil, nil,
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

func TestProcessorNtpPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorNtpPublicTestSuite))
}
