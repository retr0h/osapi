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
	"github.com/osapi-io/osapi/internal/provider/node/user"
	userMocks "github.com/osapi-io/osapi/internal/provider/node/user/mocks"
)

type ProcessorUserPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorUserPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorUserPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorUserPublicTestSuite) newProcessor(
	userProvider user.Provider,
) agent.ProcessorFunc {
	return agent.NewNodeProcessor(
		nil, nil, nil, nil,
		nil, nil, nil, nil,
		nil,
		userProvider,
		nil,
		nil,
		nil,
		config.Config{},
		slog.Default(),
	)
}

func (s *ProcessorUserPublicTestSuite) TestProcessUserOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "user provider not available",
		},
		{
			name: "invalid user operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid user operation: user",
		},
		{
			name: "unsupported user sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported user operation: user.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var userProvider user.Provider
			if tt.setupMock != nil {
				userProvider = tt.setupMock()
			}

			processor := s.newProcessor(userProvider)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
			}
		})
	}
}

func (s *ProcessorUserPublicTestSuite) TestProcessUserList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful user list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListUsers(gomock.Any()).Return([]user.User{
					{Name: "root", UID: 0, GID: 0, Home: "/root", Shell: "/bin/bash"},
					{Name: "john", UID: 1000, GID: 1000, Home: "/home/john", Shell: "/bin/bash"},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var users []user.User
				err := json.Unmarshal(result, &users)
				s.NoError(err)
				s.Len(users, 2)
				s.Equal("root", users[0].Name)
				s.Equal("john", users[1].Name)
			},
		},
		{
			name: "user list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListUsers(gomock.Any()).Return(nil, errors.New("permission denied"))
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

func (s *ProcessorUserPublicTestSuite) TestProcessUserGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful user get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.get",
				Data:      json.RawMessage(`{"name":"john"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().GetUser(gomock.Any(), "john").Return(&user.User{
					Name: "john", UID: 1000, GID: 1000,
					Home: "/home/john", Shell: "/bin/bash",
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var u user.User
				err := json.Unmarshal(result, &u)
				s.NoError(err)
				s.Equal("john", u.Name)
				s.Equal(1000, u.UID)
			},
		},
		{
			name: "user get with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal user get data",
		},
		{
			name: "user get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "user.get",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().GetUser(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
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

func (s *ProcessorUserPublicTestSuite) TestProcessUserCreate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful user create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.create",
				Data:      json.RawMessage(`{"name":"newuser","shell":"/bin/bash"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().CreateUser(gomock.Any(), user.CreateUserOpts{
					Name:  "newuser",
					Shell: "/bin/bash",
				}).Return(&user.Result{
					Name:    "newuser",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("newuser", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "user create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal user create data",
		},
		{
			name: "user create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.create",
				Data:      json.RawMessage(`{"name":"existing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("user already exists"))
				return m
			},
			expectError: true,
			errorMsg:    "user already exists",
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

func (s *ProcessorUserPublicTestSuite) TestProcessUserUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful user update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.update",
				Data:      json.RawMessage(`{"name":"john","opts":{"shell":"/bin/zsh"}}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().UpdateUser(gomock.Any(), "john", user.UpdateUserOpts{
					Shell: "/bin/zsh",
				}).Return(&user.Result{
					Name:    "john",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("john", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "user update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal user update data",
		},
		{
			name: "user update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.update",
				Data:      json.RawMessage(`{"name":"john","opts":{"shell":"/bin/zsh"}}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
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

func (s *ProcessorUserPublicTestSuite) TestProcessUserDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful user delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.delete",
				Data:      json.RawMessage(`{"name":"olduser"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().DeleteUser(gomock.Any(), "olduser").Return(&user.Result{
					Name:    "olduser",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("olduser", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "user delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal user delete data",
		},
		{
			name: "user delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.delete",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().DeleteUser(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
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

func (s *ProcessorUserPublicTestSuite) TestProcessUserPassword() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful password change",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.password",
				Data:      json.RawMessage(`{"name":"john","password":"newpass123"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					ChangePassword(gomock.Any(), "john", "newpass123").
					Return(&user.Result{
						Name:    "john",
						Changed: true,
					}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.Result
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("john", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "password change with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.password",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal user password data",
		},
		{
			name: "password change provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "user.password",
				Data:      json.RawMessage(`{"name":"john","password":"newpass123"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					ChangePassword(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("permission denied"))
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

func (s *ProcessorUserPublicTestSuite) TestProcessGroupOperation() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock:   nil,
			expectError: true,
			errorMsg:    "user provider not available",
		},
		{
			name: "invalid group operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "invalid group operation: group",
		},
		{
			name: "unsupported group sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.unknown",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unsupported group operation: group.unknown",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var userProvider user.Provider
			if tt.setupMock != nil {
				userProvider = tt.setupMock()
			}

			processor := s.newProcessor(userProvider)
			result, err := processor(tt.jobRequest)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
			}
		})
	}
}

func (s *ProcessorUserPublicTestSuite) TestProcessGroupList() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful group list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListGroups(gomock.Any()).Return([]user.Group{
					{Name: "root", GID: 0},
					{Name: "sudo", GID: 27, Members: []string{"john"}},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var groups []user.Group
				err := json.Unmarshal(result, &groups)
				s.NoError(err)
				s.Len(groups, 2)
				s.Equal("root", groups[0].Name)
				s.Equal("sudo", groups[1].Name)
			},
		},
		{
			name: "group list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.list",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListGroups(gomock.Any()).Return(nil, errors.New("permission denied"))
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

func (s *ProcessorUserPublicTestSuite) TestProcessGroupGet() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful group get",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.get",
				Data:      json.RawMessage(`{"name":"sudo"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().GetGroup(gomock.Any(), "sudo").Return(&user.Group{
					Name: "sudo", GID: 27, Members: []string{"john"},
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var g user.Group
				err := json.Unmarshal(result, &g)
				s.NoError(err)
				s.Equal("sudo", g.Name)
				s.Equal(27, g.GID)
			},
		},
		{
			name: "group get with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.get",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal group get data",
		},
		{
			name: "group get provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "group.get",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().GetGroup(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
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

func (s *ProcessorUserPublicTestSuite) TestProcessGroupCreate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful group create",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.create",
				Data:      json.RawMessage(`{"name":"developers"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().CreateGroup(gomock.Any(), user.CreateGroupOpts{
					Name: "developers",
				}).Return(&user.GroupResult{
					Name:    "developers",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.GroupResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("developers", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "group create with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.create",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal group create data",
		},
		{
			name: "group create provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.create",
				Data:      json.RawMessage(`{"name":"existing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					CreateGroup(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("group already exists"))
				return m
			},
			expectError: true,
			errorMsg:    "group already exists",
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

func (s *ProcessorUserPublicTestSuite) TestProcessGroupUpdate() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful group update",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.update",
				Data: json.RawMessage(
					`{"name":"developers","opts":{"members":["john","jane"]}}`,
				),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().UpdateGroup(gomock.Any(), "developers", user.UpdateGroupOpts{
					Members: []string{"john", "jane"},
				}).Return(&user.GroupResult{
					Name:    "developers",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.GroupResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("developers", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "group update with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.update",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal group update data",
		},
		{
			name: "group update provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.update",
				Data:      json.RawMessage(`{"name":"missing","opts":{}}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					UpdateGroup(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
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

func (s *ProcessorUserPublicTestSuite) TestProcessGroupDelete() {
	tests := []struct {
		name        string
		jobRequest  job.Request
		setupMock   func() user.Provider
		expectError bool
		errorMsg    string
		validate    func(json.RawMessage)
	}{
		{
			name: "successful group delete",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.delete",
				Data:      json.RawMessage(`{"name":"oldgroup"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().DeleteGroup(gomock.Any(), "oldgroup").Return(&user.GroupResult{
					Name:    "oldgroup",
					Changed: true,
				}, nil)
				return m
			},
			validate: func(result json.RawMessage) {
				var r user.GroupResult
				err := json.Unmarshal(result, &r)
				s.NoError(err)
				s.Equal("oldgroup", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "group delete with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.delete",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			expectError: true,
			errorMsg:    "unmarshal group delete data",
		},
		{
			name: "group delete provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "group.delete",
				Data:      json.RawMessage(`{"name":"missing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().DeleteGroup(gomock.Any(), "missing").Return(nil, errors.New("not found"))
				return m
			},
			expectError: true,
			errorMsg:    "not found",
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

func TestProcessorUserPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorUserPublicTestSuite))
}
