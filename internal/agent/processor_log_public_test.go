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
	"github.com/osapi-io/osapi/internal/provider/node/log"
	logMocks "github.com/osapi-io/osapi/internal/provider/node/log/mocks"
)

type ProcessorLogPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorLogPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorLogPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorLogPublicTestSuite) newProcessor(
	logProvider log.Provider,
) agent.ProcessorFunc {
	return agent.NewNodeProcessor(
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil,
		nil,
		nil,
		logProvider,
		nil,
		config.Config{},
		slog.Default(),
	)
}

func (s *ProcessorLogPublicTestSuite) TestProcessLogOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() log.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.query",
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "log provider not available",
		},
		{
			name: "invalid operation format (no sub-operation)",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log",
			},
			setupMock: func() log.Provider {
				return logMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid log operation: log",
		},
		{
			name: "unsupported log sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.invalid",
			},
			setupMock: func() log.Provider {
				return logMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported log operation: log.invalid",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var logProvider log.Provider
			if tt.setupMock != nil {
				logProvider = tt.setupMock()
			}

			processor := s.newProcessor(logProvider)
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

func (s *ProcessorLogPublicTestSuite) TestProcessLogQuery() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() log.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "query with default opts (empty data)",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.query",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Query(gomock.Any(), log.QueryOpts{}).Return([]log.Entry{
					{
						Timestamp: "2026-01-01T00:00:00Z",
						Unit:      "systemd",
						Priority:  "info",
						Message:   "Started system",
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []log.Entry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 1)
				s.Equal("Started system", entries[0].Message)
			},
		},
		{
			name: "query with all options",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.query",
				Data:      json.RawMessage(`{"lines":50,"since":"1 hour ago","priority":"err"}`),
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Query(gomock.Any(), log.QueryOpts{
					Lines:    50,
					Since:    "1 hour ago",
					Priority: "err",
				}).Return([]log.Entry{
					{
						Timestamp: "2026-01-01T00:00:00Z",
						Priority:  "err",
						Message:   "error occurred",
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []log.Entry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 1)
				s.Equal("err", entries[0].Priority)
			},
		},
		{
			name: "query unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.query",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() log.Provider {
				return logMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal log query data",
		},
		{
			name: "query provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.query",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Query(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("journalctl failed"))
				return m
			},
			expectError: true,
			errorMsg:    "journalctl failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newProcessor(tt.setupMock())
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

func (s *ProcessorLogPublicTestSuite) TestProcessLogQueryUnit() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() log.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "queryUnit with unit name",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.queryUnit",
				Data:      json.RawMessage(`{"unit":"nginx.service"}`),
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					QueryUnit(gomock.Any(), "nginx.service", log.QueryOpts{}).
					Return([]log.Entry{
						{
							Timestamp: "2026-01-01T00:00:00Z",
							Unit:      "nginx.service",
							Priority:  "info",
							Message:   "nginx started",
						},
					}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []log.Entry
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 1)
				s.Equal("nginx.service", entries[0].Unit)
				s.Equal("nginx started", entries[0].Message)
			},
		},
		{
			name: "queryUnit unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.queryUnit",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() log.Provider {
				return logMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal log query unit data",
		},
		{
			name: "queryUnit provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.queryUnit",
				Data:      json.RawMessage(`{"unit":"nginx.service"}`),
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					QueryUnit(gomock.Any(), "nginx.service", gomock.Any()).
					Return(nil, errors.New("unit not found"))
				return m
			},
			expectError: true,
			errorMsg:    "unit not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newProcessor(tt.setupMock())
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

func (s *ProcessorLogPublicTestSuite) TestProcessLogSources() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() log.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "sources success",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.sources",
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					ListSources(gomock.Any()).
					Return([]string{"nginx", "sshd", "systemd"}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var sources []string
				err := json.Unmarshal(result, &sources)
				s.NoError(err)
				s.Equal([]string{"nginx", "sshd", "systemd"}, sources)
			},
		},
		{
			name: "sources provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "log.sources",
			},
			setupMock: func() log.Provider {
				m := logMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					ListSources(gomock.Any()).
					Return(nil, errors.New("journalctl failed"))
				return m
			},
			expectError: true,
			errorMsg:    "journalctl failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newProcessor(tt.setupMock())
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

func TestProcessorLogPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorLogPublicTestSuite))
}
