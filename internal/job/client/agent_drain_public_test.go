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

package client_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/job/client"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type AgentDrainPublicTestSuite struct {
	suite.Suite

	mockCtrl       *gomock.Controller
	mockNATSClient *jobmocks.MockNATSClient
	mockKV         *jobmocks.MockKeyValue
	ctx            context.Context
}

func (s *AgentDrainPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockNATSClient = jobmocks.NewMockNATSClient(s.mockCtrl)
	s.mockKV = jobmocks.NewMockKeyValue(s.mockCtrl)
	s.ctx = context.Background()
}

func (s *AgentDrainPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AgentDrainPublicTestSuite) newClientWithState(
	stateKV *jobmocks.MockKeyValue,
) *client.Client {
	opts := &client.Options{
		Timeout:  30 * time.Second,
		KVBucket: s.mockKV,
		StateKV:  stateKV,
	}
	c, err := client.New(slog.Default(), s.mockNATSClient, opts)
	s.Require().NoError(err)

	return c
}

func (s *AgentDrainPublicTestSuite) newClientWithoutState() *client.Client {
	opts := &client.Options{
		Timeout:  30 * time.Second,
		KVBucket: s.mockKV,
	}
	c, err := client.New(slog.Default(), s.mockNATSClient, opts)
	s.Require().NoError(err)

	return c
}

func (s *AgentDrainPublicTestSuite) TestCheckDrainFlag() {
	tests := []struct {
		name         string
		hostname     string
		useState     bool
		setupMocks   func(*jobmocks.MockKeyValue)
		validateFunc func(bool)
	}{
		{
			name:     "when drain flag exists returns true",
			hostname: "server1",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				entry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				kv.EXPECT().
					Get(gomock.Any(), "drain.server1").
					Return(entry, nil)
			},
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:     "when drain flag missing returns false",
			hostname: "server1",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				kv.EXPECT().
					Get(gomock.Any(), "drain.server1").
					Return(nil, errors.New("key not found"))
			},
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:     "when stateKV is nil returns false",
			hostname: "server1",
			useState: false,
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var jobsClient *client.Client
			if tt.useState {
				stateKV := jobmocks.NewMockKeyValue(s.mockCtrl)
				if tt.setupMocks != nil {
					tt.setupMocks(stateKV)
				}
				jobsClient = s.newClientWithState(stateKV)
			} else {
				jobsClient = s.newClientWithoutState()
			}

			result := jobsClient.CheckDrainFlag(s.ctx, tt.hostname)
			tt.validateFunc(result)
		})
	}
}

func (s *AgentDrainPublicTestSuite) TestSetDrainFlag() {
	tests := []struct {
		name        string
		hostname    string
		useState    bool
		setupMocks  func(*jobmocks.MockKeyValue)
		expectError bool
		errorMsg    string
	}{
		{
			name:     "when write succeeds sets drain flag",
			hostname: "server1",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				kv.EXPECT().
					Put(gomock.Any(), "drain.server1", []byte("1")).
					Return(uint64(1), nil)
			},
		},
		{
			name:     "when KV put fails returns error",
			hostname: "server1",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				kv.EXPECT().
					Put(gomock.Any(), "drain.server1", []byte("1")).
					Return(uint64(0), errors.New("kv connection failed"))
			},
			expectError: true,
			errorMsg:    "set drain flag",
		},
		{
			name:        "when stateKV is nil returns error",
			hostname:    "server1",
			useState:    false,
			expectError: true,
			errorMsg:    "agent state bucket not configured",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var jobsClient *client.Client
			if tt.useState {
				stateKV := jobmocks.NewMockKeyValue(s.mockCtrl)
				if tt.setupMocks != nil {
					tt.setupMocks(stateKV)
				}
				jobsClient = s.newClientWithState(stateKV)
			} else {
				jobsClient = s.newClientWithoutState()
			}

			err := jobsClient.SetDrainFlag(s.ctx, tt.hostname)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *AgentDrainPublicTestSuite) TestDeleteDrainFlag() {
	tests := []struct {
		name        string
		hostname    string
		useState    bool
		setupMocks  func(*jobmocks.MockKeyValue)
		expectError bool
		errorMsg    string
	}{
		{
			name:     "when delete succeeds removes drain flag",
			hostname: "server1",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				kv.EXPECT().
					Delete(gomock.Any(), "drain.server1").
					Return(nil)
			},
		},
		{
			name:     "when KV delete fails returns error",
			hostname: "server1",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				kv.EXPECT().
					Delete(gomock.Any(), "drain.server1").
					Return(errors.New("kv connection failed"))
			},
			expectError: true,
			errorMsg:    "delete drain flag",
		},
		{
			name:        "when stateKV is nil returns error",
			hostname:    "server1",
			useState:    false,
			expectError: true,
			errorMsg:    "agent state bucket not configured",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var jobsClient *client.Client
			if tt.useState {
				stateKV := jobmocks.NewMockKeyValue(s.mockCtrl)
				if tt.setupMocks != nil {
					tt.setupMocks(stateKV)
				}
				jobsClient = s.newClientWithState(stateKV)
			} else {
				jobsClient = s.newClientWithoutState()
			}

			err := jobsClient.DeleteDrainFlag(s.ctx, tt.hostname)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *AgentDrainPublicTestSuite) TestOverlayDrainState() {
	tests := []struct {
		name         string
		useState     bool
		setupMocks   func(*jobmocks.MockKeyValue)
		validateFunc func(string)
	}{
		{
			name:     "when drain flag exists sets state to Cordoned",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				entry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				kv.EXPECT().
					Get(gomock.Any(), "drain.abc123").
					Return(entry, nil)
			},
			validateFunc: func(got string) {
				s.Equal(job.AgentStateCordoned, got)
			},
		},
		{
			name:     "when drain flag missing keeps original state",
			useState: true,
			setupMocks: func(kv *jobmocks.MockKeyValue) {
				kv.EXPECT().
					Get(gomock.Any(), "drain.abc123").
					Return(nil, errors.New("key not found"))
			},
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
		{
			name:     "when stateKV is nil keeps original state",
			useState: false,
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			registryKV := jobmocks.NewMockKeyValue(s.mockCtrl)

			// Set up the registry KV to return agent data with machine_id
			entry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
			entry.EXPECT().Value().Return(agentRegistrationJSON("server1"))
			registryKV.EXPECT().
				Get(gomock.Any(), "agents.abc123").
				Return(entry, nil)

			opts := &client.Options{
				Timeout:    30 * time.Second,
				KVBucket:   s.mockKV,
				RegistryKV: registryKV,
			}

			if tt.useState {
				stateKV := jobmocks.NewMockKeyValue(s.mockCtrl)
				if tt.setupMocks != nil {
					tt.setupMocks(stateKV)
				}
				opts.StateKV = stateKV
				// GetAgent also calls GetAgentTimeline which uses stateKV
				stateKV.EXPECT().
					Keys(gomock.Any()).
					Return(nil, errors.New("nats: no keys found"))
			}

			jobsClient, err := client.New(
				slog.Default(),
				s.mockNATSClient,
				opts,
			)
			s.Require().NoError(err)

			info, err := jobsClient.GetAgent(s.ctx, "abc123")
			s.NoError(err)
			tt.validateFunc(info.State)
		})
	}
}

func TestAgentDrainPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentDrainPublicTestSuite))
}
