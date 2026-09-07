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
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/job"
	dockerProv "github.com/osapi-io/osapi/internal/provider/container/docker"
	dockerMocks "github.com/osapi-io/osapi/internal/provider/container/docker/mocks"
)

type ProcessorDockerPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorDockerPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorDockerPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorDockerPublicTestSuite) TestProcessDockerOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func(*dockerMocks.MockProvider)
		expectError  bool
		errorMsg     string
		validateFunc func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest"}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "docker runtime not available",
		},
		{
			name: "successful container create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data: json.RawMessage(
					`{"image":"nginx:latest","name":"web","auto_start":true}`,
				),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Create(gomock.Any(), dockerProv.CreateParams{
						Image:     "nginx:latest",
						Name:      "web",
						AutoStart: true,
					}).
					Return(&dockerProv.Container{
						ID:      "abc123",
						Name:    "web",
						Image:   "nginx:latest",
						State:   "created",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("abc123", r["id"])
				s.Equal("web", r["name"])
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "successful container create with ports and volumes",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data: json.RawMessage(
					`{"image":"nginx:latest","name":"web","ports":[{"host":8080,"container":80}],"volumes":[{"host":"/data","container":"/var/data"}]}`,
				),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Create(gomock.Any(), dockerProv.CreateParams{
						Image: "nginx:latest",
						Name:  "web",
						Ports: []dockerProv.PortMapping{
							{Host: 8080, Container: 80},
						},
						Volumes: []dockerProv.VolumeMapping{
							{Host: "/data", Container: "/var/data"},
						},
					}).
					Return(&dockerProv.Container{
						ID:      "def456",
						Name:    "web",
						Image:   "nginx:latest",
						State:   "created",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("def456", r["id"])
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "successful container create with hostname and dns",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data: json.RawMessage(
					`{"image":"nginx:latest","name":"web","hostname":"web-01","dns":["8.8.8.8","8.8.4.4"]}`,
				),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Create(gomock.Any(), dockerProv.CreateParams{
						Image:    "nginx:latest",
						Name:     "web",
						Hostname: "web-01",
						DNS:      []string{"8.8.8.8", "8.8.4.4"},
					}).
					Return(&dockerProv.Container{
						ID:      "ghi789",
						Name:    "web",
						Image:   "nginx:latest",
						State:   "created",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("ghi789", r["id"])
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "unsupported docker operation",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "unknown.get",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unsupported docker operation",
		},
		{
			name: "create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal create data",
		},
		{
			name: "provider error on create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "create.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("docker error"))
			},
			expectError: true,
			errorMsg:    "docker error",
		},
		// --- start operation ---
		{
			name: "successful container start",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "start.execute",
				Data:      json.RawMessage(`{"id":"abc123"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Start(gomock.Any(), "abc123").
					Return(&dockerProv.ActionResult{
						Message: "Container started successfully",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Contains(r, "message")
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "start with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "start.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal start data",
		},
		{
			name: "provider error on start",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "start.execute",
				Data:      json.RawMessage(`{"id":"abc123"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Start(gomock.Any(), "abc123").
					Return(nil, errors.New("start failed"))
			},
			expectError: true,
			errorMsg:    "start failed",
		},
		// --- stop operation ---
		{
			name: "successful container stop with timeout",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "stop.execute",
				Data:      json.RawMessage(`{"id":"abc123","timeout":10}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Stop(gomock.Any(), "abc123", gomock.Any()).
					Return(&dockerProv.ActionResult{
						Message: "Container stopped successfully",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Contains(r, "message")
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "successful container stop without timeout",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "stop.execute",
				Data:      json.RawMessage(`{"id":"abc123"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Stop(gomock.Any(), "abc123", (*time.Duration)(nil)).
					Return(&dockerProv.ActionResult{
						Message: "Container stopped successfully",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Contains(r, "message")
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "stop with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "stop.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal stop data",
		},
		{
			name: "provider error on stop",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "stop.execute",
				Data:      json.RawMessage(`{"id":"abc123","timeout":10}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Stop(gomock.Any(), "abc123", gomock.Any()).
					Return(nil, errors.New("stop failed"))
			},
			expectError: true,
			errorMsg:    "stop failed",
		},
		// --- remove operation ---
		{
			name: "successful container remove",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "remove.execute",
				Data:      json.RawMessage(`{"id":"abc123","force":true}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Remove(gomock.Any(), "abc123", true).
					Return(&dockerProv.ActionResult{
						Message: "Container removed successfully",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Contains(r, "message")
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "remove with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "remove.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal remove data",
		},
		{
			name: "provider error on remove",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "remove.execute",
				Data:      json.RawMessage(`{"id":"abc123","force":false}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Remove(gomock.Any(), "abc123", false).
					Return(nil, errors.New("remove failed"))
			},
			expectError: true,
			errorMsg:    "remove failed",
		},
		// --- list operation ---
		{
			name: "successful container list",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "list.get",
				Data:      json.RawMessage(`{"state":"running","limit":10}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					List(gomock.Any(), dockerProv.ListParams{
						State: "running",
						Limit: 10,
					}).
					Return([]dockerProv.Container{
						{ID: "abc123"},
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r []dockerProv.Container
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Len(r, 1)
				s.Equal("abc123", r[0].ID)
			},
		},
		{
			name: "list with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "list.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal list data",
		},
		{
			name: "provider error on list",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "list.get",
				Data:      json.RawMessage(`{"state":"all"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("list failed"))
			},
			expectError: true,
			errorMsg:    "list failed",
		},
		// --- inspect operation ---
		{
			name: "successful container inspect",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "inspect.get",
				Data:      json.RawMessage(`{"id":"abc123"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Inspect(gomock.Any(), "abc123").
					Return(&dockerProv.ContainerDetail{
						Container: dockerProv.Container{ID: "abc123"},
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r dockerProv.ContainerDetail
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("abc123", r.ID)
			},
		},
		{
			name: "inspect with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "inspect.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal inspect data",
		},
		{
			name: "provider error on inspect",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "inspect.get",
				Data:      json.RawMessage(`{"id":"abc123"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Inspect(gomock.Any(), "abc123").
					Return(nil, errors.New("inspect failed"))
			},
			expectError: true,
			errorMsg:    "inspect failed",
		},
		// --- exec operation ---
		{
			name: "successful container exec",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "exec.execute",
				Data:      json.RawMessage(`{"id":"abc123","command":["ls","-la"]}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Exec(gomock.Any(), "abc123", dockerProv.ExecParams{
						Command: []string{"ls", "-la"},
					}).
					Return(&dockerProv.ExecResult{
						Stdout:   "output",
						ExitCode: 0,
						Changed:  true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("output", r["stdout"])
				s.Equal(float64(0), r["exit_code"])
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "exec with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "exec.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal exec data",
		},
		{
			name: "provider error on exec",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "exec.execute",
				Data:      json.RawMessage(`{"id":"abc123","command":["ls"]}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Exec(gomock.Any(), "abc123", gomock.Any()).
					Return(nil, errors.New("exec failed"))
			},
			expectError: true,
			errorMsg:    "exec failed",
		},
		// --- pull operation ---
		{
			name: "successful container pull",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "pull.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Pull(gomock.Any(), "nginx:latest").
					Return(&dockerProv.PullResult{
						ImageID: "sha256:abc",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("sha256:abc", r["image_id"])
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "pull with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "pull.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal pull data",
		},
		{
			name: "provider error on pull",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "pull.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest"}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					Pull(gomock.Any(), "nginx:latest").
					Return(nil, errors.New("pull failed"))
			},
			expectError: true,
			errorMsg:    "pull failed",
		},
		// --- image-remove operation ---
		{
			name: "successful image remove",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "image-remove.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest","force":false}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					ImageRemove(gomock.Any(), "nginx:latest", false).
					Return(&dockerProv.ActionResult{
						Message: "Image removed successfully",
						Changed: true,
					}, nil)
			},
			validateFunc: func(result json.RawMessage) {
				var r map[string]interface{}
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Contains(r, "message")
				s.Equal(true, r["changed"])
			},
		},
		{
			name: "image-remove with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "image-remove.execute",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock:   func(_ *dockerMocks.MockProvider) {},
			expectError: true,
			errorMsg:    "unmarshal image-remove data",
		},
		{
			name: "provider error on image-remove",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "docker",
				Operation: "image-remove.execute",
				Data:      json.RawMessage(`{"image":"nginx:latest","force":false}`),
			},
			setupMock: func(m *dockerMocks.MockProvider) {
				m.EXPECT().
					ImageRemove(gomock.Any(), "nginx:latest", false).
					Return(nil, errors.New("image-remove failed"))
			},
			expectError: true,
			errorMsg:    "image-remove failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var containerMock dockerProv.Provider
			if tt.setupMock != nil {
				m := dockerMocks.NewMockProvider(s.mockCtrl)
				tt.setupMock(m)
				containerMock = m
			}
			// nil provider case uses nil containerMock

			processor := agent.NewDockerProcessor(containerMock, slog.Default())
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

func TestProcessorDockerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorDockerPublicTestSuite))
}
