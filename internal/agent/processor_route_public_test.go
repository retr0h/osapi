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
	"github.com/osapi-io/osapi/internal/provider/network/netplan/route"
	routeMocks "github.com/osapi-io/osapi/internal/provider/network/netplan/route/mocks"
)

type ProcessorRoutePublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorRoutePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorRoutePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorRoutePublicTestSuite) TestProcessRouteOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() route.Provider
		validateFunc func(any, error)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: nil,
			validateFunc: func(result any, err error) {
				s.Error(err)
				s.Contains(err.Error(), "route provider not available")
				s.Nil(result)
			},
		},
		{
			name: "invalid route operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() route.Provider {
				return routeMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result any, err error) {
				s.Error(err)
				s.Contains(err.Error(), "invalid route operation: route")
				s.Nil(result)
			},
		},
		{
			name: "unsupported route sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() route.Provider {
				return routeMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result any, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported route operation: route.unknown")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var routeProvider route.Provider
			if tt.setupMock != nil {
				routeProvider = tt.setupMock()
			}

			processor := agent.NewNetworkProcessor(
				nil, nil,
				nil,
				routeProvider,
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorRoutePublicTestSuite) TestProcessRouteList() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() route.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful route list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return([]route.ListEntry{
					{Destination: "10.0.0.0/24", Gateway: "10.0.0.1", Interface: "eth0"},
					{Destination: "192.168.1.0/24", Gateway: "192.168.1.1", Interface: "eth1"},
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var entries []route.ListEntry
				decodeErr := json.Unmarshal(result, &entries)
				s.NoError(decodeErr)
				s.Len(entries, 2)
				s.Equal("10.0.0.0/24", entries[0].Destination)
			},
		},
		{
			name: "route list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().List(gomock.Any()).Return(nil, errors.New("permission denied"))
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
			processor := agent.NewNetworkProcessor(
				nil, nil,
				nil,
				tt.setupMock(),
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorRoutePublicTestSuite) TestProcessRouteGet() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() route.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful route get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.get",
				Data:      json.RawMessage(`{"interface":"eth0"}`),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "eth0").Return(&route.Entry{
					Interface: "eth0",
					Routes: []route.Route{
						{To: "10.0.0.0/24", Via: "10.0.0.1"},
					},
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var entry route.Entry
				decodeErr := json.Unmarshal(result, &entry)
				s.NoError(decodeErr)
				s.Equal("eth0", entry.Interface)
				s.Len(entry.Routes, 1)
			},
		},
		{
			name: "route get with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() route.Provider {
				return routeMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal route get data")
				s.Nil(result)
			},
		},
		{
			name: "route get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "network",
				Operation: "route.get",
				Data:      json.RawMessage(`{"interface":"missing"}`),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Get(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "not found")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				nil,
				tt.setupMock(),
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorRoutePublicTestSuite) TestProcessRouteCreate() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() route.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful route create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.create",
				Data: json.RawMessage(
					`{"interface":"eth0","routes":[{"to":"10.0.0.0/24","via":"10.0.0.1"}]}`,
				),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Create(gomock.Any(), route.Entry{
					Interface: "eth0",
					Routes: []route.Route{
						{To: "10.0.0.0/24", Via: "10.0.0.1"},
					},
				}).Return(&route.Result{
					Interface: "eth0",
					Changed:   true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r route.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("eth0", r.Interface)
				s.True(r.Changed)
			},
		},
		{
			name: "route create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() route.Provider {
				return routeMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal route create data")
				s.Nil(result)
			},
		},
		{
			name: "route create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.create",
				Data: json.RawMessage(
					`{"interface":"eth0","routes":[{"to":"10.0.0.0/24","via":"10.0.0.1"}]}`,
				),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("deploy failed"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "deploy failed")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				nil,
				tt.setupMock(),
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorRoutePublicTestSuite) TestProcessRouteUpdate() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() route.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful route update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.update",
				Data: json.RawMessage(
					`{"interface":"eth0","routes":[{"to":"10.0.0.0/24","via":"10.0.0.2"}]}`,
				),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Update(gomock.Any(), route.Entry{
					Interface: "eth0",
					Routes: []route.Route{
						{To: "10.0.0.0/24", Via: "10.0.0.2"},
					},
				}).Return(&route.Result{
					Interface: "eth0",
					Changed:   true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r route.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("eth0", r.Interface)
				s.True(r.Changed)
			},
		},
		{
			name: "route update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() route.Provider {
				return routeMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal route update data")
				s.Nil(result)
			},
		},
		{
			name: "route update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.update",
				Data: json.RawMessage(
					`{"interface":"eth0","routes":[{"to":"10.0.0.0/24","via":"10.0.0.2"}]}`,
				),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not managed"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "not managed")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				nil,
				tt.setupMock(),
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorRoutePublicTestSuite) TestProcessRouteDelete() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() route.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful route delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.delete",
				Data:      json.RawMessage(`{"interface":"eth0"}`),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "eth0").Return(&route.Result{
					Interface: "eth0",
					Changed:   true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r route.Result
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.Equal("eth0", r.Interface)
				s.True(r.Changed)
			},
		},
		{
			name: "route delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() route.Provider {
				return routeMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal route delete data")
				s.Nil(result)
			},
		},
		{
			name: "route delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "network",
				Operation: "route.delete",
				Data:      json.RawMessage(`{"interface":"missing"}`),
			},
			setupMock: func() route.Provider {
				m := routeMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().Delete(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "not found")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := agent.NewNetworkProcessor(
				nil, nil,
				nil,
				tt.setupMock(),
				slog.Default(),
			)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func TestProcessorRoutePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorRoutePublicTestSuite))
}
