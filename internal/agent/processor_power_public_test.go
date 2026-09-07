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
	"github.com/osapi-io/osapi/internal/provider/node/power"
	powerMocks "github.com/osapi-io/osapi/internal/provider/node/power/mocks"
)

type ProcessorPowerPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorPowerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorPowerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorPowerPublicTestSuite) TestProcessPowerOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() power.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.reboot",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: nil,
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "power provider not available")
				s.Nil(result)
			},
		},
		{
			name: "invalid operation format (no sub-operation)",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() power.Provider {
				return powerMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "invalid power operation: power")
				s.Nil(result)
			},
		},
		{
			name: "unsupported power sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() power.Provider {
				return powerMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported power operation: power.unknown")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var powerProvider power.Provider
			if tt.setupMock != nil {
				powerProvider = tt.setupMock()
			}

			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				nil, nil, nil,
				powerProvider,
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorPowerPublicTestSuite) TestProcessPowerReboot() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() power.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful reboot with opts",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.reboot",
				Data:      json.RawMessage(`{"delay":30,"message":"Scheduled reboot"}`),
			},
			setupMock: func() power.Provider {
				m := powerMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Reboot(gomock.Any(), power.Opts{
					Delay:   30,
					Message: "Scheduled reboot",
				}).Return(&power.Result{
					Action:  "reboot",
					Delay:   30,
					Changed: true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r power.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("reboot", r.Action)
				s.Equal(30, r.Delay)
				s.True(r.Changed)
			},
		},
		{
			name: "successful reboot with nil data uses zero opts",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.reboot",
				Data:      nil,
			},
			setupMock: func() power.Provider {
				m := powerMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Reboot(gomock.Any(), power.Opts{}).Return(&power.Result{
					Action:  "reboot",
					Delay:   0,
					Changed: true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r power.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("reboot", r.Action)
				s.True(r.Changed)
			},
		},
		{
			name: "reboot with invalid JSON data returns unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.reboot",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() power.Provider {
				return powerMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal power opts")
				s.Nil(result)
			},
		},
		{
			name: "reboot provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.reboot",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() power.Provider {
				m := powerMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Reboot(gomock.Any(), power.Opts{}).
					Return(nil, errors.New("permission denied"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				nil, nil, nil,
				tt.setupMock(),
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorPowerPublicTestSuite) TestProcessPowerShutdown() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() power.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful shutdown with opts",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.shutdown",
				Data:      json.RawMessage(`{"delay":60,"message":"Maintenance shutdown"}`),
			},
			setupMock: func() power.Provider {
				m := powerMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Shutdown(gomock.Any(), power.Opts{
					Delay:   60,
					Message: "Maintenance shutdown",
				}).Return(&power.Result{
					Action:  "shutdown",
					Delay:   60,
					Changed: true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r power.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("shutdown", r.Action)
				s.Equal(60, r.Delay)
				s.True(r.Changed)
			},
		},
		{
			name: "successful shutdown with nil data uses zero opts",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.shutdown",
				Data:      nil,
			},
			setupMock: func() power.Provider {
				m := powerMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Shutdown(gomock.Any(), power.Opts{}).Return(&power.Result{
					Action:  "shutdown",
					Delay:   0,
					Changed: true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r power.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("shutdown", r.Action)
				s.True(r.Changed)
			},
		},
		{
			name: "shutdown with invalid JSON data returns unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.shutdown",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() power.Provider {
				return powerMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal power opts")
				s.Nil(result)
			},
		},
		{
			name: "shutdown provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "power.shutdown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() power.Provider {
				m := powerMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Shutdown(gomock.Any(), power.Opts{}).
					Return(nil, errors.New("operation not permitted"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "operation not permitted")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil,
				nil, nil, nil,
				tt.setupMock(),
				nil,
				nil,
				nil,
				nil,
				nil,
				config.Config{},
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func TestProcessorPowerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorPowerPublicTestSuite))
}
