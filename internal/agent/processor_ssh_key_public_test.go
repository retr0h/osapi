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

type ProcessorSSHKeyPublicTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *ProcessorSSHKeyPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *ProcessorSSHKeyPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ProcessorSSHKeyPublicTestSuite) newProcessor(
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

func (s *ProcessorSSHKeyPublicTestSuite) TestProcessSSHKeyOperation() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() user.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "nil provider returns error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sshKey.list",
				Data:      json.RawMessage(`{"username":"john"}`),
			},
			setupMock: nil,
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "user provider not available")
				s.Nil(result)
			},
		},
		{
			name: "invalid sshKey operation missing sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sshKey",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "invalid sshKey operation: sshKey")
				s.Nil(result)
			},
		},
		{
			name: "unsupported sshKey sub-operation",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sshKey.invalid",
				Data:      json.RawMessage(`{}`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unsupported sshKey operation: sshKey.invalid")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var userProvider user.Provider
			if tt.setupMock != nil {
				userProvider = tt.setupMock()
			}

			processor := s.newProcessor(userProvider)
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorSSHKeyPublicTestSuite) TestProcessSSHKeyList() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() user.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful ssh key list",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sshKey.list",
				Data:      json.RawMessage(`{"username":"john"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().ListKeys(gomock.Any(), "john").Return([]user.SSHKey{
					{
						Type:        "ssh-ed25519",
						Fingerprint: "SHA256:abc123",
						Comment:     "john@laptop",
					},
					{
						Type:        "ssh-rsa",
						Fingerprint: "SHA256:def456",
						Comment:     "john@desktop",
					},
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var keys []user.SSHKey
				decodeErr := json.Unmarshal(result, &keys)
				s.NoError(decodeErr)
				s.Len(keys, 2)
				s.Equal("ssh-ed25519", keys[0].Type)
				s.Equal("SHA256:abc123", keys[0].Fingerprint)
				s.Equal("ssh-rsa", keys[1].Type)
			},
		},
		{
			name: "ssh key list with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sshKey.list",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal sshKey list data")
				s.Nil(result)
			},
		},
		{
			name: "ssh key list provider error",
			jobRequest: job.Request{
				Type:      job.TypeQuery,
				Category:  "node",
				Operation: "sshKey.list",
				Data:      json.RawMessage(`{"username":"missing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					ListKeys(gomock.Any(), "missing").
					Return(nil, errors.New("user not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "user not found")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newProcessor(tt.setupMock())
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorSSHKeyPublicTestSuite) TestProcessSSHKeyAdd() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() user.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful ssh key add",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sshKey.add",
				Data: json.RawMessage(
					`{"username":"john","key":{"type":"ssh-ed25519","fingerprint":"SHA256:abc123","comment":"john@laptop","raw_line":"ssh-ed25519 AAAA... john@laptop"}}`,
				),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().AddKey(gomock.Any(), "john", user.SSHKey{
					Type:        "ssh-ed25519",
					Fingerprint: "SHA256:abc123",
					Comment:     "john@laptop",
					RawLine:     "ssh-ed25519 AAAA... john@laptop",
				}).Return(&user.SSHKeyResult{
					Changed: true,
				}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r user.SSHKeyResult
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.True(r.Changed)
			},
		},
		{
			name: "ssh key add with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sshKey.add",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal sshKey add data")
				s.Nil(result)
			},
		},
		{
			name: "ssh key add provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sshKey.add",
				Data: json.RawMessage(
					`{"username":"john","key":{"type":"ssh-ed25519","fingerprint":"SHA256:abc123","raw_line":"ssh-ed25519 AAAA..."}}`,
				),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					AddKey(gomock.Any(), gomock.Any(), gomock.Any()).
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
			processor := s.newProcessor(tt.setupMock())
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func (s *ProcessorSSHKeyPublicTestSuite) TestProcessSSHKeyRemove() {
	tests := []struct {
		name         string
		jobRequest   job.Request
		setupMock    func() user.Provider
		validateFunc func(json.RawMessage, error)
	}{
		{
			name: "successful ssh key remove",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sshKey.remove",
				Data:      json.RawMessage(`{"username":"john","fingerprint":"SHA256:abc123"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					RemoveKey(gomock.Any(), "john", "SHA256:abc123").
					Return(&user.SSHKeyResult{
						Changed: true,
					}, nil)
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.NoError(err)
				s.NotNil(result)
				var r user.SSHKeyResult
				decodeErr := json.Unmarshal(result, &r)
				s.NoError(decodeErr)
				s.True(r.Changed)
			},
		},
		{
			name: "ssh key remove with invalid JSON data",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sshKey.remove",
				Data:      json.RawMessage(`invalid json`),
			},
			setupMock: func() user.Provider {
				return userMocks.NewMockProvider(s.mockCtrl)
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "unmarshal sshKey remove data")
				s.Nil(result)
			},
		},
		{
			name: "ssh key remove provider error",
			jobRequest: job.Request{
				Type:      job.TypeModify,
				Category:  "node",
				Operation: "sshKey.remove",
				Data:      json.RawMessage(`{"username":"john","fingerprint":"SHA256:missing"}`),
			},
			setupMock: func() user.Provider {
				m := userMocks.NewMockProvider(s.mockCtrl)
				m.EXPECT().
					RemoveKey(gomock.Any(), "john", "SHA256:missing").
					Return(nil, errors.New("key not found"))
				return m
			},
			validateFunc: func(result json.RawMessage, err error) {
				s.Error(err)
				s.Contains(err.Error(), "key not found")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			processor := s.newProcessor(tt.setupMock())
			tt.validateFunc(processor(tt.jobRequest))
		})
	}
}

func TestProcessorSSHKeyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessorSSHKeyPublicTestSuite))
}
