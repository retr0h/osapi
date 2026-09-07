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
	"github.com/osapi-io/osapi/internal/provider/node/apt"
	aptMocks "github.com/osapi-io/osapi/internal/provider/node/apt/mocks"
)

type ProcessorPackagePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorPackagePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorPackagePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorPackagePublicTestSuite) newProcessor(
	packageProvider apt.Provider,
) agent.ProcessorFunc {
	return agent.NewNodeProcessor(
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil,
		nil,
		packageProvider,
		nil,
		nil,
		config.Config{},
		slog.Default(),
	)
}

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.list",
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "package provider not available",
		},
		{
			name: "invalid operation format (no sub-operation)",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package",
			},
			setupMock: func() apt.Provider {
				return aptMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid package operation: package",
		},
		{
			name: "unsupported package sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.unknown",
			},
			setupMock: func() apt.Provider {
				return aptMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported package operation: package.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var packageProvider apt.Provider
			if tt.setupMock != nil {
				packageProvider = tt.setupMock()
			}

			processor := s.newProcessor(packageProvider)
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

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.list",
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]apt.Package{
					{
						Name:    "curl",
						Version: "7.88.1-10+deb12u5",
						Status:  "installed",
					},
					{
						Name:    "vim",
						Version: "2:9.0.1378-2",
						Status:  "installed",
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var pkgs []apt.Package
				err := json.Unmarshal(result, &pkgs)
				s.NoError(err)
				s.Len(pkgs, 2)
				s.Equal("curl", pkgs[0].Name)
				s.Equal("vim", pkgs[1].Name)
			},
		},
		{
			name: "list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.list",
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
				return m
			},
			expectError: true,
			errorMsg:    "permission denied",
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

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.get",
				Data:      json.RawMessage(`{"name": "curl"}`),
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "curl").Return(&apt.Package{
					Name:    "curl",
					Version: "7.88.1-10+deb12u5",
					Status:  "installed",
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var pkg apt.Package
				err := json.Unmarshal(result, &pkg)
				s.NoError(err)
				s.Equal("curl", pkg.Name)
				s.Equal("installed", pkg.Status)
			},
		},
		{
			name: "get unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() apt.Provider {
				return aptMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal package get data",
		},
		{
			name: "get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.get",
				Data:      json.RawMessage(`{"name": "nonexistent"}`),
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Get(gomock.Any(), "nonexistent").
					Return(nil, errors.New("package not found"))
				return m
			},
			expectError: true,
			errorMsg:    "package not found",
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

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageInstall() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful install",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.install",
				Data:      json.RawMessage(`{"name": "nginx"}`),
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Install(gomock.Any(), "nginx").Return(&apt.Result{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r apt.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "install unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.install",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() apt.Provider {
				return aptMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal package install data",
		},
		{
			name: "install provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.install",
				Data:      json.RawMessage(`{"name": "bad-pkg"}`),
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Install(gomock.Any(), "bad-pkg").
					Return(nil, errors.New("package not found in repository"))
				return m
			},
			expectError: true,
			errorMsg:    "package not found in repository",
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

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageRemove() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful remove",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.remove",
				Data:      json.RawMessage(`{"name": "nginx"}`),
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Remove(gomock.Any(), "nginx").Return(&apt.Result{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r apt.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "remove unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.remove",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() apt.Provider {
				return aptMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal package remove data",
		},
		{
			name: "remove provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.remove",
				Data:      json.RawMessage(`{"name": "essential-pkg"}`),
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Remove(gomock.Any(), "essential-pkg").
					Return(nil, errors.New("cannot remove essential package"))
				return m
			},
			expectError: true,
			errorMsg:    "cannot remove essential package",
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

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.update",
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any()).Return(&apt.Result{
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r apt.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.True(r.Changed)
			},
		},
		{
			name: "update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "package.update",
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any()).Return(nil, errors.New("network unreachable"))
				return m
			},
			expectError: true,
			errorMsg:    "network unreachable",
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

func (s *ProcessorPackagePublicTestSuite) TestProcessPackageListUpdates() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() apt.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful list updates",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.listUpdates",
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListUpdates(gomock.Any()).Return([]apt.Update{
					{
						Name:           "curl",
						CurrentVersion: "7.88.1-10+deb12u4",
						NewVersion:     "7.88.1-10+deb12u5",
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var updates []apt.Update
				err := json.Unmarshal(result, &updates)
				s.NoError(err)
				s.Len(updates, 1)
				s.Equal("curl", updates[0].Name)
				s.Equal("7.88.1-10+deb12u5", updates[0].NewVersion)
			},
		},
		{
			name: "list updates provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "package.listUpdates",
			},
			setupMock: func() apt.Provider {
				m := aptMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListUpdates(gomock.Any()).Return(nil, errors.New("apt cache stale"))
				return m
			},
			expectError: true,
			errorMsg:    "apt cache stale",
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

func TestProcessorPackagePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorPackagePublicTestSuite))
}
