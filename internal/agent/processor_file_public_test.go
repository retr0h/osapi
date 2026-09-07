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
	fileProv "github.com/osapi-io/osapi/internal/provider/file"
	fileMocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
)

type ProcessorFilePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorFilePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorFilePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorFilePublicTestSuite) TestProcessFileOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func(*fileMocks.MockProvider)
		expectError  bool
		errorMsg     string
		validateFunc func(json.RawMessage)
	}{
		{
			name: "successful deploy operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "deploy.execute",
				Data: json.RawMessage(
					`{"object_name":"app.conf","path":"/etc/app/app.conf","mode":"0644","content_type":"raw"}`,
				),
			},
			setupMock: func(m *fileMocks.MockProvider) {
				m.EXPECT().
					Deploy(gomock.Any(), fileProv.DeployRequest{
						ObjectName:  "app.conf",
						Path:        "/etc/app/app.conf",
						Mode:        "0644",
						ContentType: "raw",
					}).
					Return(&fileProv.DeployResult{
						Changed: true,
						SHA256:  "abc123def456",
						Path:    "/etc/app/app.conf",
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r fileProv.DeployResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.True(r.Changed)
				s.Equal("abc123def456", r.SHA256)
				s.Equal("/etc/app/app.conf", r.Path)
			},
		},
		{
			name: "successful status operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "file",
				Operation: "status.get",
				Data:      json.RawMessage(`{"path":"/etc/app/app.conf"}`),
			},
			setupMock: func(m *fileMocks.MockProvider) {
				m.EXPECT().
					Status(gomock.Any(), fileProv.StatusRequest{
						Path: "/etc/app/app.conf",
					}).
					Return(&fileProv.StatusResult{
						Path:   "/etc/app/app.conf",
						Status: "in-sync",
						SHA256: "abc123def456",
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r fileProv.StatusResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("in-sync", r.Status)
				s.Equal("/etc/app/app.conf", r.Path)
				s.Equal("abc123def456", r.SHA256)
			},
		},
		{
			name: "unsupported file operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "unknown.execute",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   func(_ *fileMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unsupported file operation",
		},
		{
			name: "deploy with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "deploy.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *fileMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "failed to parse file deploy data",
		},
		{
			name: "status with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "file",
				Operation: "status.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *fileMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "failed to parse file status data",
		},
		{
			name: "deploy provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "deploy.execute",
				Data: json.RawMessage(
					`{"object_name":"app.conf","path":"/etc/app/app.conf","content_type":"raw"}`,
				),
			},
			setupMock: func(m *fileMocks.MockProvider) {
				m.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("object not found"))
			},
			expectError: true,
			errorMsg:    "object not found",
		},
		{
			name: "status provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "file",
				Operation: "status.get",
				Data:      json.RawMessage(`{"path":"/etc/app/app.conf"}`),
			},
			setupMock: func(m *fileMocks.MockProvider) {
				m.EXPECT().
					Status(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("state KV unavailable"))
			},
			expectError: true,
			errorMsg:    "state KV unavailable",
		},
		{
			name: "successful undeploy operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "undeploy.execute",
				Data:      json.RawMessage(`{"path":"/etc/cron.d/backup"}`),
			},
			setupMock: func(m *fileMocks.MockProvider) {
				m.EXPECT().
					Undeploy(gomock.Any(), fileProv.UndeployRequest{
						Path: "/etc/cron.d/backup",
					}).
					Return(&fileProv.UndeployResult{
						Changed: true,
						Path:    "/etc/cron.d/backup",
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r fileProv.UndeployResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.True(r.Changed)
				s.Equal("/etc/cron.d/backup", r.Path)
			},
		},
		{
			name: "undeploy with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "undeploy.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *fileMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "failed to parse file undeploy data",
		},
		{
			name: "undeploy provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "undeploy.execute",
				Data:      json.RawMessage(`{"path":"/etc/cron.d/backup"}`),
			},
			setupMock: func(m *fileMocks.MockProvider) {
				m.EXPECT().
					Undeploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("state KV unavailable"))
			},
			expectError: true,
			errorMsg:    "state KV unavailable",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			fMock := fileMocks.NewMockProvider(s.mockCtrl)
			tt.setupMock(fMock)

			processor := agent.NewFileProcessor(fMock, slog.Default())
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
				tt.validateFunc(result)
			}
		})
	}
}

func (s *ProcessorFilePublicTestSuite) TestProcessFileOperationNilProvider() {
	tests := []struct {
		name         string
		validateFunc func(any, error)
	}{
		{
			name: "returns error when file provider is nil",
			validateFunc: func(result any, err error) {
				s.Error(err)
				s.Contains(err.Error(), "file provider not configured")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewFileProcessor(nil, slog.Default())
			tt.validateFunc(processor(job.Request{
				Type:      job.TypeModify,
				Category:  "file",
				Operation: "deploy.execute",
				Data:      json.RawMessage(`{}`),
			}))
		})
	}
}

func TestProcessorFilePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorFilePublicTestSuite))
}
