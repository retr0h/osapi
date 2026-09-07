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
	"github.com/osapi-io/osapi/internal/job/mocks"
	diskMocks "github.com/osapi-io/osapi/internal/provider/node/disk/mocks"
	hostMocks "github.com/osapi-io/osapi/internal/provider/node/host/mocks"
	loadMocks "github.com/osapi-io/osapi/internal/provider/node/load/mocks"
	memMocks "github.com/osapi-io/osapi/internal/provider/node/mem/mocks"
	"github.com/osapi-io/osapi/internal/provider/scheduled/cron"
	cronMocks "github.com/osapi-io/osapi/internal/provider/scheduled/cron/mocks"
)

type ProcessorSchedulePublicTestSuite struct {
	suite.Suite

	mockCtrl      *gomock.Controller
	mockJobClient *mocks.MockJobClient
}

func (s *ProcessorSchedulePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockJobClient = mocks.NewMockJobClient(s.mockCtrl)
}

func (s *ProcessorSchedulePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorSchedulePublicTestSuite) TestProcessScheduleOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() cron.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: nil,
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "cron provider not available")
				s.Nil(result)
			},
		},
		{
			name: "dispatches to cron operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]cron.Entry{}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var entries []cron.Entry
				decodeErr := json.Unmarshal(result, &entries)
				s.NoError(decodeErr)
				s.Empty(entries)
			},
		},
		{
			name: "unsupported schedule operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "unknown.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported schedule operation: unknown.list")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var cronProvider cron.Provider
			if tt.setupMock != nil {
				cronProvider = tt.setupMock()
			}

			processor := agent.NewScheduleProcessor(cronProvider, slog.Default())
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorSchedulePublicTestSuite) TestProcessCronOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() cron.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "invalid cron operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "invalid cron operation: cron")
				s.Nil(result)
			},
		},
		{
			name: "successful cron list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]cron.Entry{
					{
						Name:     "backup",
						Schedule: "0 2 * * *",
						User:     "root",
						Object:   "/usr/local/bin/backup.sh",
					},
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var entries []cron.Entry
				decodeErr := json.Unmarshal(result, &entries)
				s.NoError(decodeErr)
				s.Len(entries, 1)
				s.Equal("backup", entries[0].Name)
				s.Equal("0 2 * * *", entries[0].Schedule)
			},
		},
		{
			name: "cron list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Nil(result)
			},
		},
		{
			name: "successful cron get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.get",
				Data:      json.RawMessage(`{"name":"backup"}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "backup").Return(&cron.Entry{
					Name:     "backup",
					Schedule: "0 2 * * *",
					User:     "root",
					Object:   "/usr/local/bin/backup.sh",
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var entry cron.Entry
				decodeErr := json.Unmarshal(result, &entry)
				s.NoError(decodeErr)
				s.Equal("backup", entry.Name)
			},
		},
		{
			name: "cron get with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal cron get data")
				s.Nil(result)
			},
		},
		{
			name: "cron get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.get",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "not found")
				s.Nil(result)
			},
		},
		{
			name: "successful cron create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.create",
				Data: json.RawMessage(
					`{"name":"logrotate","schedule":"0 0 * * *","user":"root","command":"/usr/sbin/logrotate"}`,
				),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, entry cron.Entry) (*cron.CreateResult, error) {
						s.Equal("logrotate", entry.Name)
						return &cron.CreateResult{
							Name:    "logrotate",
							Changed: true,
						}, nil
					},
				)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r cron.CreateResult
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("logrotate", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "cron create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal cron create data")
				s.Nil(result)
			},
		},
		{
			name: "cron create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.create",
				Data: json.RawMessage(
					`{"name":"dup","schedule":"* * * * *","user":"root","command":"echo"}`,
				),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("already exists"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "already exists")
				s.Nil(result)
			},
		},
		{
			name: "successful cron update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.update",
				Data: json.RawMessage(
					`{"name":"backup","schedule":"0 3 * * *","user":"root","command":"/usr/local/bin/backup.sh"}`,
				),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, entry cron.Entry) (*cron.UpdateResult, error) {
						s.Equal("backup", entry.Name)
						s.Equal("0 3 * * *", entry.Schedule)
						return &cron.UpdateResult{
							Name:    "backup",
							Changed: true,
						}, nil
					},
				)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r cron.UpdateResult
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("backup", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "cron update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal cron update data")
				s.Nil(result)
			},
		},
		{
			name: "cron update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.update",
				Data: json.RawMessage(
					`{"name":"missing","schedule":"* * * * *","user":"root","command":"echo"}`,
				),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errors.New("not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "not found")
				s.Nil(result)
			},
		},
		{
			name: "successful cron delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.delete",
				Data:      json.RawMessage(`{"name":"backup"}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "backup").Return(&cron.DeleteResult{
					Name:    "backup",
					Changed: true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r cron.DeleteResult
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("backup", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "cron delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal cron delete data")
				s.Nil(result)
			},
		},
		{
			name: "cron delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "schedule",
				Operation: "cron.delete",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "not found")
				s.Nil(result)
			},
		},
		{
			name: "unsupported cron sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				return cronMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported cron operation: cron.unknown")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewScheduleProcessor(tt.setupMock(), slog.Default())
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorSchedulePublicTestSuite) TestProcessJobOperationScheduleCategory() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() cron.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "schedule category dispatches correctly",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				m := cronMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]cron.Entry{}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var entries []cron.Entry
				decodeErr := json.Unmarshal(result, &entries)
				s.NoError(decodeErr)
				s.Empty(entries)
			},
		},
		{
			name: "schedule category with nil provider",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "schedule",
				Operation: "cron.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() cron.Provider {
				return nil
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "cron provider not available")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			a := newTestAgent(newTestAgentParams{
				jobClient:    s.mockJobClient,
				hostProvider: hostMocks.NewDefaultMockProvider(s.mockCtrl),
				diskProvider: diskMocks.NewDefaultMockProvider(s.mockCtrl),
				memProvider:  memMocks.NewDefaultMockProvider(s.mockCtrl),
				loadProvider: loadMocks.NewDefaultMockProvider(s.mockCtrl),
				cronProvider: tt.setupMock(),
			})

			tt.validateFunc(agent.ExportProcessJobOperation(a, tt.jobRequest))
		})
	}
}

func TestProcessorSchedulePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorSchedulePublicTestSuite))
}
