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
	"github.com/osapi-io/osapi/internal/provider/node/service"
	serviceMocks "github.com/osapi-io/osapi/internal/provider/node/service/mocks"
)

type ProcessorServicePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorServicePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorServicePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorServicePublicTestSuite) newNodeProcessor(
	serviceProvider service.Provider,
) agent.ProcessorFunc {
	return agent.NewNodeProcessor(
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil,
		nil,
		nil,
		nil,
		serviceProvider,
		config.Config{},
		slog.Default(),
	)
}

func (s *ProcessorServicePublicTestSuite) TestProcessServiceOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "service provider not available",
		},
		{
			name: "invalid operation format missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid service operation: service",
		},
		{
			name: "unsupported sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported service operation: service.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var serviceProvider service.Provider
			if tt.setupMock != nil {
				serviceProvider = tt.setupMock()
			}

			processor := s.newNodeProcessor(serviceProvider)
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]service.Info{
					{
						Name:    "nginx",
						Status:  "running",
						Enabled: true,
					},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var entries []service.Info
				err := json.Unmarshal(result, &entries)
				s.NoError(err)
				s.Len(entries, 1)
				s.Equal("nginx", entries[0].Name)
			},
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
				return m
			},
			expectError: true,
			errorMsg:    "permission denied",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.get",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "nginx").Return(&service.Info{
					Name:    "nginx",
					Status:  "running",
					Enabled: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var info service.Info
				err := json.Unmarshal(result, &info)
				s.NoError(err)
				s.Equal("nginx", info.Name)
				s.Equal("running", info.Status)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service get data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "service.get",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceCreate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.create",
				Data:      json.RawMessage(`{"name":"my-svc","object":"unit-obj"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, entry service.Entry) (*service.CreateResult, error) {
						s.Equal("my-svc", entry.Name)
						return &service.CreateResult{
							Name:    "my-svc",
							Changed: true,
						}, nil
					},
				)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.CreateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("my-svc", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service create data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.create",
				Data:      json.RawMessage(`{"name":"dup","object":"unit-obj"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("already exists"))
				return m
			},
			expectError: true,
			errorMsg:    "already exists",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.update",
				Data:      json.RawMessage(`{"name":"my-svc","object":"unit-obj-v2"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ interface{}, entry service.Entry) (*service.UpdateResult, error) {
						s.Equal("my-svc", entry.Name)
						return &service.UpdateResult{
							Name:    "my-svc",
							Changed: true,
						}, nil
					},
				)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.UpdateResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("my-svc", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service update data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.update",
				Data:      json.RawMessage(`{"name":"missing","object":"unit-obj"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.delete",
				Data:      json.RawMessage(`{"name":"my-svc"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "my-svc").Return(&service.DeleteResult{
					Name:    "my-svc",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.DeleteResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("my-svc", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service delete data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.delete",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "missing").
					Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceStart() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful start",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.start",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Start(gomock.Any(), "nginx").Return(&service.ActionResult{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.ActionResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.start",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service start data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.start",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Start(gomock.Any(), "nginx").
					Return(nil, errors.New("failed to start"))
				return m
			},
			expectError: true,
			errorMsg:    "failed to start",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceStop() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful stop",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.stop",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Stop(gomock.Any(), "nginx").Return(&service.ActionResult{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.ActionResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.stop",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service stop data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.stop",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Stop(gomock.Any(), "nginx").
					Return(nil, errors.New("failed to stop"))
				return m
			},
			expectError: true,
			errorMsg:    "failed to stop",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceRestart() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful restart",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.restart",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Restart(gomock.Any(), "nginx").Return(&service.ActionResult{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.ActionResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.restart",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service restart data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.restart",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Restart(gomock.Any(), "nginx").
					Return(nil, errors.New("failed to restart"))
				return m
			},
			expectError: true,
			errorMsg:    "failed to restart",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceEnable() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful enable",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.enable",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Enable(gomock.Any(), "nginx").Return(&service.ActionResult{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.ActionResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.enable",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service enable data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.enable",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Enable(gomock.Any(), "nginx").
					Return(nil, errors.New("failed to enable"))
				return m
			},
			expectError: true,
			errorMsg:    "failed to enable",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func (s *ProcessorServicePublicTestSuite) TestProcessServiceDisable() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() service.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful disable",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.disable",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Disable(gomock.Any(), "nginx").Return(&service.ActionResult{
					Name:    "nginx",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r service.ActionResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("nginx", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "unmarshal error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.disable",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() service.Provider {
				return serviceMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal service disable data",
		},
		{
			name: "provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "service.disable",
				Data:      json.RawMessage(`{"name":"nginx"}`),
			},
			setupMock: func() service.Provider {
				m := serviceMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Disable(gomock.Any(), "nginx").
					Return(nil, errors.New("failed to disable"))
				return m
			},
			expectError: true,
			errorMsg:    "failed to disable",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newNodeProcessor(tt.setupMock())
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

func TestProcessorServicePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorServicePublicTestSuite))
}
