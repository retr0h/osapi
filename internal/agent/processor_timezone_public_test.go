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
	"github.com/osapi-io/osapi/internal/provider/node/timezone"
	timezoneMocks "github.com/osapi-io/osapi/internal/provider/node/timezone/mocks"
)

type ProcessorTimezonePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorTimezonePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorTimezonePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorTimezonePublicTestSuite) TestProcessTimezoneOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() timezone.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "timezone.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "timezone provider not available",
		},
		{
			name: "invalid timezone operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "timezone",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() timezone.Provider {
				return timezoneMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid timezone operation: timezone",
		},
		{
			name: "unsupported timezone sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "timezone.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() timezone.Provider {
				return timezoneMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported timezone operation: timezone.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var timezoneProvider timezone.Provider
			if tt.setupMock != nil {
				timezoneProvider = tt.setupMock()
			}

			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil, nil,
				timezoneProvider,
				nil, nil,
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

func (s *ProcessorTimezonePublicTestSuite) TestProcessTimezoneGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() timezone.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful timezone get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "timezone.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() timezone.Provider {
				m := timezoneMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any()).Return(&timezone.Info{
					Timezone:  "America/New_York",
					UTCOffset: "-05:00",
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var info timezone.Info
				err := json.Unmarshal(result, &info)
				s.NoError(err)
				s.Equal("America/New_York", info.Timezone)
				s.Equal("-05:00", info.UTCOffset)
			},
		},
		{
			name: "timezone get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "timezone.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() timezone.Provider {
				m := timezoneMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any()).Return(nil, errors.New("timedatectl not found"))
				return m
			},
			expectError: true,
			errorMsg:    "timedatectl not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil,
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

func (s *ProcessorTimezonePublicTestSuite) TestProcessTimezoneUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() timezone.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful timezone update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "timezone.update",
				Data:      json.RawMessage(`{"timezone":"America/New_York"}`),
			},
			setupMock: func() timezone.Provider {
				m := timezoneMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), "America/New_York").Return(&timezone.UpdateResult{
					Timezone: "America/New_York",
					Changed:  true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r timezone.UpdateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("America/New_York", r.Timezone)
				s.True(r.Changed)
			},
		},
		{
			name: "timezone update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "timezone.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() timezone.Provider {
				return timezoneMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal timezone update data",
		},
		{
			name: "timezone update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "timezone.update",
				Data:      json.RawMessage(`{"timezone":"Invalid/Zone"}`),
			},
			setupMock: func() timezone.Provider {
				m := timezoneMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Update(gomock.Any(), "Invalid/Zone").
					Return(nil, errors.New("invalid timezone"))
				return m
			},
			expectError: true,
			errorMsg:    "invalid timezone",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNodeProcessor(
				nil, nil, nil, nil, nil, nil,
				tt.setupMock(),
				nil, nil,
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

func TestProcessorTimezonePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorTimezonePublicTestSuite))
}
