// Copyright (c) 2025 John Dewey

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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/job/client"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

// newMockKeyLister returns a MockKeyLister that yields the given keys then closes.
func newMockKeyLister(
	ctrl *gomock.Controller,
	keys []string,
) *jobmocks.MockKeyLister {
	ch := make(chan string, len(keys))
	for _, k := range keys {
		ch <- k
	}
	close(ch)
	m := jobmocks.NewMockKeyLister(ctrl)
	m.EXPECT().Keys().Return(ch)
	m.EXPECT().Stop().Return(nil).AnyTimes()
	return m
}

type JobsPublicTestSuite struct {
	suite.Suite

	mockCtrl       *gomock.Controller
	mockNATSClient *jobmocks.MockNATSClient
	mockKV         *jobmocks.MockKeyValue
	jobsClient     *client.Client
	ctx            context.Context
}

func (s *JobsPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockNATSClient = jobmocks.NewMockNATSClient(s.mockCtrl)
	s.mockKV = jobmocks.NewMockKeyValue(s.mockCtrl)
	s.ctx = context.Background()

	opts := &client.Options{
		Timeout:    30 * time.Second,
		KVBucket:   s.mockKV,
		StreamName: "JOBS",
	}
	var err error
	s.jobsClient, err = client.New(slog.Default(), s.mockNATSClient, opts)
	s.Require().NoError(err)
}

func (s *JobsPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *JobsPublicTestSuite) TearDownSubTest() {
	client.ResetSigningMarshalFn()
}

func (s *JobsPublicTestSuite) TestNew() {
	tests := []struct {
		name        string
		opts        *client.Options
		expectedErr string
	}{
		{
			name:        "nil options",
			opts:        nil,
			expectedErr: "options cannot be nil",
		},
		{
			name:        "nil KV bucket",
			opts:        &client.Options{},
			expectedErr: "kvBucket cannot be nil",
		},
		{
			name: "valid options",
			opts: &client.Options{
				KVBucket: s.mockKV,
				Timeout:  30 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c, err := client.New(slog.Default(), s.mockNATSClient, tt.opts)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				s.Nil(c)
			} else {
				s.NoError(err)
				s.NotNil(c)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestGetJobStatus() {
	now := time.Now().Format(time.RFC3339)

	tests := []struct {
		name           string
		jobID          string
		expectedErr    string
		expectedStatus string
		expectedError  string
		agentCount     int
		responseCount  int
		setupMocks     func()
		validateFunc   func(qj *job.QueuedJob)
	}{
		{
			name:  "successful job status retrieval",
			jobID: "job-123",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(`{
					"id": "job-123",
					"status": "completed",
					"created": "2024-01-01T00:00:00Z",
					"subject": "jobs.query.server1",
					"operation": {"type": "node.hostname.get"}
				}`))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-123").Return(mockEntry, nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-123.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-123.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)
			},
			expectedStatus: "submitted",
		},
		{
			name:        "job not found",
			jobID:       "nonexistent",
			expectedErr: "job not found: nonexistent",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs.nonexistent").
					Return(nil, errors.New("key not found"))
			},
		},
		{
			name:        "invalid job data",
			jobID:       "job-invalid",
			expectedErr: "failed to parse job data",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(`invalid json`))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-invalid").Return(mockEntry, nil)
			},
		},
		{
			name:  "completed agent with response",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z","subject":"jobs.query.server1"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.acknowledged.agent1.100",
						"status.job-1.started.agent1.200",
						"status.job-1.completed.agent1.300",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"responses.job-1.agent1.400",
					}), nil)

				ackEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				ackEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"acknowledged","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.acknowledged.agent1.100").
					Return(ackEntry, nil)

				startEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				startEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.200").
					Return(startEntry, nil)

				compEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				compEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"completed","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.300").
					Return(compEntry, nil)

				respEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				respEntry.EXPECT().Value().Return([]byte(
					`{"status":"completed","data":"eyJ0ZXN0IjogdHJ1ZX0="}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-1.agent1.400").
					Return(respEntry, nil)
			},
			expectedStatus: "completed",
			agentCount:     1,
			responseCount:  1,
		},
		{
			name:  "completed agent response propagates changed to queued job",
			jobID: "job-changed",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-changed","operation":{"type":"file.deploy.execute"},"created":"2024-01-01T00:00:00Z","subject":"jobs.modify.server1"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-changed").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-changed.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-changed.completed.agent1.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-changed.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"responses.job-changed.agent1.200",
					}), nil)

				compEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				compEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-changed","event":"completed","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-changed.completed.agent1.100").
					Return(compEntry, nil)

				respEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				respEntry.EXPECT().Value().Return([]byte(
					`{"status":"completed","data":"eyJ0ZXN0IjogdHJ1ZX0=","changed":true,"hostname":"agent1"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-changed.agent1.200").
					Return(respEntry, nil)
			},
			expectedStatus: "completed",
			agentCount:     1,
			responseCount:  1,
			validateFunc: func(qj *job.QueuedJob) {
				s.NotNil(qj.Changed, "QueuedJob.Changed should be set from response")
				s.True(*qj.Changed)
			},
		},
		{
			name:  "failed agent with error",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.started.agent1.100",
						"status.job-1.failed.agent1.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				startEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				startEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.100").
					Return(startEntry, nil)

				failEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				failEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"failed","hostname":"agent1","timestamp":"%s","data":{"error":"disk full"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.failed.agent1.200").
					Return(failEntry, nil)
			},
			expectedStatus: "failed",
			expectedError:  "disk full",
			agentCount:     1,
		},
		{
			name:  "partial failure with multiple agents",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.completed.agent1.100",
						"status.job-1.failed.agent2.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				compEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				compEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"completed","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.100").
					Return(compEntry, nil)

				failEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				failEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"failed","hostname":"agent2","timestamp":"%s","data":{"error":"timeout"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.failed.agent2.200").
					Return(failEntry, nil)
			},
			expectedStatus: "partial_failure",
			agentCount:     2,
		},
		{
			name:  "processing state",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.acknowledged.agent1.100",
						"status.job-1.started.agent1.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				ackEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				ackEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"acknowledged","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.acknowledged.agent1.100").
					Return(ackEntry, nil)

				startEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				startEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.200").
					Return(startEntry, nil)
			},
			expectedStatus: "processing",
			agentCount:     1,
		},
		{
			name:  "event get error skipped gracefully",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.completed.agent1.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.100").
					Return(nil, errors.New("kv error"))
			},
			expectedStatus: "submitted",
		},
		{
			name:  "invalid event JSON skipped gracefully",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.completed.agent1.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				invalidEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				invalidEntry.EXPECT().Value().Return([]byte(`invalid json`))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.100").
					Return(invalidEntry, nil)
			},
			expectedStatus: "submitted",
		},
		{
			name:        "status keys error returns error",
			jobID:       "job-1",
			expectedErr: "failed to get status events",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(nil, errors.New("connection failed"))
			},
		},
		{
			name:  "response parse error skipped",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"responses.job-1.agent1.100",
					}), nil)

				respEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				respEntry.EXPECT().Value().Return([]byte(`not json`))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-1.agent1.100").
					Return(respEntry, nil)
			},
			expectedStatus: "submitted",
			responseCount:  0,
		},
		{
			name:  "response get error skipped",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"responses.job-1.agent1.100",
					}), nil)

				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-1.agent1.100").
					Return(nil, errors.New("kv error"))
			},
			expectedStatus: "submitted",
			responseCount:  0,
		},
		{
			name:  "when events are out of order completed wins over started",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.completed.agent1.300",
						"status.job-1.started.agent1.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				compEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				compEntry.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"completed","hostname":"agent1","timestamp":"2024-01-01T00:00:05Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.300").
					Return(compEntry, nil)

				startEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				startEntry.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"2024-01-01T00:00:01Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.100").
					Return(startEntry, nil)
			},
			expectedStatus: "completed",
			agentCount:     1,
		},
		{
			name:  "acknowledged only agent shows processing",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.acknowledged.agent1.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				ackEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				ackEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-1","event":"acknowledged","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.acknowledged.agent1.100").
					Return(ackEntry, nil)
			},
			expectedStatus: "processing",
			agentCount:     1,
		},
		{
			name:  "redelivered job has positive duration",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				// Simulate NATS redelivery: two started/failed cycles
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.started.agent1.100",
						"status.job-1.failed.agent1.200",
						"status.job-1.started.agent1.300",
						"status.job-1.failed.agent1.400",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				// First attempt: started at T+0, failed at T+1s
				start1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				start1.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"2024-01-01T00:00:01Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.100").
					Return(start1, nil)

				fail1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				fail1.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"failed","hostname":"agent1","timestamp":"2024-01-01T00:00:02Z","data":{"error":"attempt 1"}}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.failed.agent1.200").
					Return(fail1, nil)

				// Second attempt (redelivery): started at T+60s, failed at T+61s
				start2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				start2.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"2024-01-01T00:01:00Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.300").
					Return(start2, nil)

				fail2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				fail2.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"failed","hostname":"agent1","timestamp":"2024-01-01T00:01:01Z","data":{"error":"attempt 2"}}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.failed.agent1.400").
					Return(fail2, nil)
			},
			expectedStatus: "failed",
			expectedError:  "attempt 2",
			agentCount:     1,
			validateFunc: func(qj *job.QueuedJob) {
				ws := qj.AgentStates["agent1"]
				// Duration should span from first start to last failure (60s)
				// and must be positive (not negative like the old bug)
				s.Equal("1m0s", ws.Duration)
				s.False(ws.StartTime.IsZero())
				s.False(ws.EndTime.IsZero())
				s.True(ws.EndTime.After(ws.StartTime))
			},
		},
		{
			name:  "completed job has sub-second duration with RFC3339Nano",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.started.agent1.100",
						"status.job-1.completed.agent1.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				startEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				startEntry.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"started","hostname":"agent1","timestamp":"2024-01-01T00:00:01.000000000Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.started.agent1.100").
					Return(startEntry, nil)

				compEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				compEntry.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"completed","hostname":"agent1","timestamp":"2024-01-01T00:00:01.045000000Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.200").
					Return(compEntry, nil)
			},
			expectedStatus: "completed",
			agentCount:     1,
			validateFunc: func(qj *job.QueuedJob) {
				ws := qj.AgentStates["agent1"]
				s.Equal("45ms", ws.Duration)
			},
		},
		{
			name:  "invalid timestamp skipped",
			jobID: "job-1",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-1.completed.agent1.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				badEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				badEntry.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"completed","hostname":"agent1","timestamp":"not-a-date"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.completed.agent1.100").
					Return(badEntry, nil)
			},
			expectedStatus: "submitted",
		},
		{
			name:        "response keys collection error returns error",
			jobID:       "job-1",
			expectedErr: "failed to get response keys",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-1.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-1.>").
					Return(nil, errors.New("connection failed"))
			},
		},
		{
			name:  "single agent skipped sets overall skipped status",
			jobID: "job-skip",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-skip","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-skip").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-skip.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-skip.started.agent1.100",
						"status.job-skip.skipped.agent1.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-skip.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				startEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				startEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-skip","event":"started","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-skip.started.agent1.100").
					Return(startEntry, nil)

				skipEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				skipEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-skip","event":"skipped","hostname":"agent1","timestamp":"%s","data":{"error":"operation not supported on this OS family"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-skip.skipped.agent1.200").
					Return(skipEntry, nil)
			},
			expectedStatus: "skipped",
			agentCount:     1,
			validateFunc: func(qj *job.QueuedJob) {
				ws := qj.AgentStates["agent1"]
				s.Equal("skipped", ws.Status)

				// Verify timeline includes skipped event with message
				found := false
				for _, ev := range qj.Timeline {
					if ev.Event == "skipped" {
						found = true
						s.Contains(ev.Message, "unsupported OS family")
						s.Equal("operation not supported on this OS family", ev.Error)
					}
				}
				s.True(found, "timeline should contain skipped event")
			},
		},
		{
			name:  "multi-agent all skipped sets overall skipped",
			jobID: "job-allskip",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-allskip","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-allskip").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-allskip.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-allskip.skipped.agent1.100",
						"status.job-allskip.skipped.agent2.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-allskip.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				skip1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				skip1.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-allskip","event":"skipped","hostname":"agent1","timestamp":"%s","data":{"error":"unsupported"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-allskip.skipped.agent1.100").
					Return(skip1, nil)

				skip2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				skip2.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-allskip","event":"skipped","hostname":"agent2","timestamp":"%s","data":{"error":"unsupported"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-allskip.skipped.agent2.200").
					Return(skip2, nil)
			},
			expectedStatus: "skipped",
			agentCount:     2,
		},
		{
			name:  "broadcast aggregates Changed from multiple responses",
			jobID: "job-bc",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-bc","operation":{"type":"file.deploy.execute"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-bc").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-bc.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-bc.completed.agent1.100",
						"status.job-bc.completed.agent2.200",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-bc.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"responses.job-bc.agent1.300",
						"responses.job-bc.agent2.400",
					}), nil)

				comp1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				comp1.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-bc","event":"completed","hostname":"agent1","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-bc.completed.agent1.100").
					Return(comp1, nil)

				comp2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				comp2.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-bc","event":"completed","hostname":"agent2","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-bc.completed.agent2.200").
					Return(comp2, nil)

				resp1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				resp1.EXPECT().Value().Return([]byte(
					`{"status":"completed","data":"eyJ0ZXN0IjogdHJ1ZX0=","changed":true,"hostname":"agent1"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-bc.agent1.300").
					Return(resp1, nil)

				resp2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				resp2.EXPECT().Value().Return([]byte(
					`{"status":"completed","data":"eyJ0ZXN0IjogdHJ1ZX0=","changed":false,"hostname":"agent2"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-bc.agent2.400").
					Return(resp2, nil)
			},
			expectedStatus: "completed",
			agentCount:     2,
			responseCount:  2,
			validateFunc: func(qj *job.QueuedJob) {
				s.NotNil(qj.Changed, "broadcast Changed should be set")
				s.True(*qj.Changed, "broadcast Changed should be true when any host changed")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			jobStatus, err := s.jobsClient.GetJobStatus(s.ctx, tt.jobID)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
				s.NotNil(jobStatus)
				if tt.expectedStatus != "" {
					s.Equal(tt.expectedStatus, jobStatus.Status)
				}
				if tt.expectedError != "" {
					s.Equal(tt.expectedError, jobStatus.Error)
				}
				if tt.agentCount > 0 {
					s.Len(jobStatus.AgentStates, tt.agentCount)
				}
				if tt.responseCount > 0 {
					s.Len(jobStatus.Responses, tt.responseCount)
				}
				if tt.validateFunc != nil {
					tt.validateFunc(jobStatus)
				}
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestListJobs() {
	tests := []struct {
		name               string
		statusFilter       string
		limit              int
		offset             int
		expectedErr        string
		setupMocks         func()
		expectedJobs       int
		expectedTotalCount int
	}{
		{
			name:         "no jobs found",
			statusFilter: "",
			setupMocks: func() {
				s.mockKV.EXPECT().Keys(gomock.Any()).Return(nil, jetstream.ErrNoKeysFound)
			},
			expectedJobs:       0,
			expectedTotalCount: 0,
		},
		{
			name:        "kv error",
			expectedErr: "error fetching jobs",
			setupMocks: func() {
				s.mockKV.EXPECT().Keys(gomock.Any()).Return(nil, errors.New("connection failed"))
			},
		},
		{
			name: "returns all jobs default limit",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2"}, nil)

				mockEntry1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry1.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockEntry1, nil)

				mockEntry2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry2.EXPECT().Value().Return([]byte(
					`{"id":"job-2","operation":{"type":"node.status.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-2").Return(mockEntry2, nil)
			},
			expectedJobs:       2,
			expectedTotalCount: 2,
		},
		{
			name:         "filters out non matching status",
			statusFilter: "completed",
			setupMocks: func() {
				// No status keys → default "submitted", doesn't match "completed"
				// Two-pass: no kv.Get needed for filtering
				s.mockKV.EXPECT().Keys(gomock.Any()).Return([]string{"jobs.job-1"}, nil)
			},
			expectedJobs:       0,
			expectedTotalCount: 0,
		},
		{
			name: "empty job ID after trim skipped",
			setupMocks: func() {
				s.mockKV.EXPECT().Keys(gomock.Any()).Return([]string{"jobs."}, nil)
			},
			expectedJobs:       0,
			expectedTotalCount: 0,
		},
		{
			name: "getJobStatusFromKeys error skipped",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-bad"}, nil)

				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs.job-bad").
					Return(nil, errors.New("kv error"))
			},
			expectedJobs:       0,
			expectedTotalCount: 1,
		},
		{
			name: "only processes jobs prefix keys",
			setupMocks: func() {
				s.mockKV.EXPECT().Keys(gomock.Any()).Return(
					[]string{
						"status.job-1.submitted._api.123",
						"jobs.job-1",
						"responses.job-1.host.123",
					},
					nil,
				)

				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockJobEntry, nil)

				mockStatusEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockStatusEntry.EXPECT().Value().Return([]byte(
					`{"job_id":"job-1","event":"submitted","hostname":"_api","timestamp":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-1.submitted._api.123").
					Return(mockStatusEntry, nil)

				mockRespEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockRespEntry.EXPECT().Value().Return([]byte(
					`{"status":"completed","data":"{}"}`,
				))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "responses.job-1.host.123").
					Return(mockRespEntry, nil)
			},
			expectedJobs:       1,
			expectedTotalCount: 1,
		},
		{
			name:  "limit restricts returned jobs",
			limit: 1,
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2"}, nil)

				// Only job-2 is fetched (newest-first, limit 1)
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-2","operation":{"type":"node.status.get"},"created":"2024-01-02T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-2").Return(mockEntry, nil)
			},
			expectedJobs:       1,
			expectedTotalCount: 2,
		},
		{
			name:   "offset skips jobs",
			offset: 1,
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2"}, nil)

				// After reversing: [job-2, job-1], offset 1 skips job-2, returns job-1
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockEntry, nil)
			},
			expectedJobs:       1,
			expectedTotalCount: 2,
		},
		{
			name:   "offset beyond total returns empty",
			offset: 10,
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2"}, nil)
			},
			expectedJobs:       0,
			expectedTotalCount: 2,
		},
		{
			name:         "filter with offset skips matching jobs",
			statusFilter: "submitted",
			offset:       1,
			setupMocks: func() {
				// No status keys → all default "submitted"
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2", "jobs.job-3"}, nil)

				// Reversed: job-3, job-2, job-1. Offset 1 → page = [job-2, job-1]
				// Only page items get kv.Get in Pass 2
				mockEntry2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry2.EXPECT().Value().Return([]byte(
					`{"id":"job-2","operation":{"type":"node.status.get"},"created":"2024-01-02T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-2").Return(mockEntry2, nil)

				mockEntry1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry1.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockEntry1, nil)
			},
			expectedJobs:       2,
			expectedTotalCount: 3,
		},
		{
			name:         "filter with limit restricts results",
			statusFilter: "submitted",
			limit:        1,
			setupMocks: func() {
				// No status keys → both default "submitted"
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2"}, nil)

				// Reversed: job-2, job-1. Limit 1 → page = [job-2]
				// Only page item gets kv.Get in Pass 2
				mockEntry2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry2.EXPECT().Value().Return([]byte(
					`{"id":"job-2","operation":{"type":"node.status.get"},"created":"2024-01-02T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-2").Return(mockEntry2, nil)
			},
			expectedJobs:       1,
			expectedTotalCount: 2,
		},
		{
			name:         "filter skips jobs with get error",
			statusFilter: "submitted",
			setupMocks: func() {
				// No status keys → both default "submitted"
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-bad", "jobs.job-good"}, nil)

				// Reversed: job-good, job-bad — both match filter
				// Pass 2: job-good Get fails, job-bad Get succeeds
				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs.job-good").
					Return(nil, errors.New("kv error"))

				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-bad","operation":{"type":"node.status.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-bad").Return(mockEntry, nil)
			},
			expectedJobs: 1,
			// totalCount is 2 because key-name-based counting finds
			// both jobs matching the filter before Pass 2 Get errors
			expectedTotalCount: 2,
		},
		{
			name: "getJobStatusFromKeys with invalid JSON",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1"}, nil)

				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(`not valid json`))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockEntry, nil)
			},
			expectedJobs:       0,
			expectedTotalCount: 1,
		},
		{
			name:  "limit exceeding max capped to default",
			limit: 200,
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2"}, nil)

				// Limit 200 → capped to DefaultPageSize (10)
				// Both jobs are within page
				mockEntry2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry2.EXPECT().Value().Return([]byte(
					`{"id":"job-2","operation":{"type":"node.status.get"},"created":"2024-01-02T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-2").Return(mockEntry2, nil)

				mockEntry1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry1.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockEntry1, nil)
			},
			expectedJobs:       2,
			expectedTotalCount: 2,
		},
		{
			name: "newest first ordering",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{"jobs.job-1", "jobs.job-2", "jobs.job-3"}, nil)

				// Reversed: job-3, job-2, job-1
				mockEntry3 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry3.EXPECT().Value().Return([]byte(
					`{"id":"job-3","operation":{"type":"node.status.get"},"created":"2024-01-03T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-3").Return(mockEntry3, nil)

				mockEntry2 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry2.EXPECT().Value().Return([]byte(
					`{"id":"job-2","operation":{"type":"node.status.get"},"created":"2024-01-02T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-2").Return(mockEntry2, nil)

				mockEntry1 := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry1.EXPECT().Value().Return([]byte(
					`{"id":"job-1","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-1").Return(mockEntry1, nil)
			},
			expectedJobs:       3,
			expectedTotalCount: 3,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			result, err := s.jobsClient.ListJobs(
				s.ctx,
				tt.statusFilter,
				tt.limit,
				tt.offset,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
				s.NotNil(result)
				s.Len(result.Jobs, tt.expectedJobs)
				s.Equal(tt.expectedTotalCount, result.TotalCount)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestDeleteJob() {
	tests := []struct {
		name        string
		jobID       string
		expectedErr string
		setupMocks  func()
	}{
		{
			name:  "successful deletion",
			jobID: "job-123",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(`{"id":"job-123"}`)).AnyTimes()
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-123").Return(mockEntry, nil)
				s.mockKV.EXPECT().Delete(gomock.Any(), "jobs.job-123").Return(nil)
			},
		},
		{
			name:        "job not found",
			jobID:       "nonexistent",
			expectedErr: "job not found: nonexistent",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs.nonexistent").
					Return(nil, errors.New("key not found"))
			},
		},
		{
			name:        "delete error",
			jobID:       "job-456",
			expectedErr: "failed to delete job",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(`{"id":"job-456"}`)).AnyTimes()
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-456").Return(mockEntry, nil)
				s.mockKV.EXPECT().
					Delete(gomock.Any(), "jobs.job-456").
					Return(errors.New("storage failure"))
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			err := s.jobsClient.DeleteJob(s.ctx, tt.jobID)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestRetriedEventInTimeline() {
	now := time.Now().Format(time.RFC3339)

	tests := []struct {
		name         string
		jobID        string
		setupMocks   func()
		validateFunc func(qj *job.QueuedJob)
	}{
		{
			name:  "retried event appears in timeline with new job ID",
			jobID: "job-original",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-original","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-original").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-original.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-original.submitted._api.100",
						"status.job-original.failed.agent1.200",
						"status.job-original.retried._api.300",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-original.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				submitEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				submitEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-original","event":"submitted","hostname":"_api","timestamp":"%s"}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-original.submitted._api.100").
					Return(submitEntry, nil)

				failEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				failEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-original","event":"failed","hostname":"agent1","timestamp":"%s","data":{"error":"timeout"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-original.failed.agent1.200").
					Return(failEntry, nil)

				retriedEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				retriedEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-original","event":"retried","hostname":"_api","timestamp":"%s","data":{"new_job_id":"job-new-123","target_hostname":"_any"}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-original.retried._api.300").
					Return(retriedEntry, nil)
			},
			validateFunc: func(qj *job.QueuedJob) {
				s.Equal("failed", qj.Status)
				s.Len(qj.Timeline, 3)

				// Find retried event in timeline
				var found bool
				for _, te := range qj.Timeline {
					if te.Event == "retried" {
						s.Contains(te.Message, "job-new-123")
						found = true
					}
				}
				s.True(found, "retried event should appear in timeline")
			},
		},
		{
			name:  "retried event without new_job_id shows fallback message",
			jobID: "job-original-2",
			setupMocks: func() {
				mockJobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockJobEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-original-2","operation":{"type":"node.hostname.get"},"created":"2024-01-01T00:00:00Z"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-original-2").Return(mockJobEntry, nil)

				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status.job-original-2.>").
					Return(newMockKeyLister(s.mockCtrl, []string{
						"status.job-original-2.retried._api.100",
					}), nil)
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses.job-original-2.>").
					Return(newMockKeyLister(s.mockCtrl, nil), nil)

				retriedEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				retriedEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"job_id":"job-original-2","event":"retried","hostname":"_api","timestamp":"%s","data":{}}`,
					now,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "status.job-original-2.retried._api.100").
					Return(retriedEntry, nil)
			},
			validateFunc: func(qj *job.QueuedJob) {
				s.Len(qj.Timeline, 1)
				s.Equal("retried", qj.Timeline[0].Event)
				s.Equal("Job retried", qj.Timeline[0].Message)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			qj, err := s.jobsClient.GetJobStatus(s.ctx, tt.jobID)
			s.NoError(err)
			s.NotNil(qj)
			tt.validateFunc(qj)
		})
	}
}

func (s *JobsPublicTestSuite) TestCreateJob() {
	tests := []struct {
		name        string
		opData      map[string]interface{}
		target      string
		expectedErr string
		setupMocks  func()
	}{
		{
			name: "missing type field returns error",
			opData: map[string]interface{}{
				"data": "no-type",
			},
			target:      "_any",
			expectedErr: "invalid operation format: missing type field",
			setupMocks:  func() {},
		},
		{
			name: "marshal failure returns error",
			opData: map[string]interface{}{
				"type":          "node.hostname.get",
				"unmarshalable": make(chan int),
			},
			target:      "_any",
			expectedErr: "failed to marshal job with status",
			setupMocks: func() {
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
			},
		},
		{
			name: "modify operation uses modify prefix",
			opData: map[string]interface{}{
				"type": "network.dns.update",
				"data": map[string]interface{}{},
			},
			target: "web-01",
			setupMocks: func() {
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					Times(2)
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "status event put failure is logged not returned",
			opData: map[string]interface{}{
				"type": "node.hostname.get",
				"data": map[string]interface{}{},
			},
			target: "_any",
			setupMocks: func() {
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				// First put succeeds (job storage)
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				// Second put fails (status event)
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("status event write failed"))
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "publish failure returns error",
			opData: map[string]interface{}{
				"type": "node.hostname.get",
				"data": map[string]interface{}{},
			},
			target: "_any",
			setupMocks: func() {
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					Times(2)
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("publish failed"))
			},
			expectedErr: "failed to send notification",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			result, err := s.jobsClient.CreateJob(s.ctx, tt.opData, tt.target)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
				s.NotEmpty(result.JobID)
				s.Equal("created", result.Status)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestRetryJob() {
	tests := []struct {
		name        string
		jobID       string
		target      string
		expectedErr string
		setupMocks  func()
	}{
		{
			name:   "successful retry",
			jobID:  "job-123",
			target: "_any",
			setupMocks: func() {
				// Read original job
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-123","operation":{"type":"node.hostname.get","data":{}},"subject":"jobs.query._any"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-123").Return(mockEntry, nil)

				// CreateJob: store new job + status event + publish
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					Times(2)
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				// Write retried event on original job
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(2), nil)
			},
		},
		{
			name:        "job not found",
			jobID:       "nonexistent",
			target:      "_any",
			expectedErr: "job not found: nonexistent",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs.nonexistent").
					Return(nil, errors.New("key not found"))
			},
		},
		{
			name:        "invalid job JSON",
			jobID:       "job-bad",
			target:      "_any",
			expectedErr: "failed to parse job data",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(`not json`))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-bad").Return(mockEntry, nil)
			},
		},
		{
			name:        "missing operation field",
			jobID:       "job-no-op",
			target:      "_any",
			expectedErr: "job has no operation data",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-no-op","subject":"jobs.query._any"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-no-op").Return(mockEntry, nil)
			},
		},
		{
			name:        "create job fails",
			jobID:       "job-456",
			target:      "_any",
			expectedErr: "failed to create retry job",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-456","operation":{"type":"node.hostname.get","data":{}},"subject":"jobs.query._any"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-456").Return(mockEntry, nil)

				// CreateJob fails on KV put
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("kv error"))
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
			},
		},
		{
			name:   "retried event put error is logged not returned",
			jobID:  "job-789",
			target: "_any",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-789","operation":{"type":"node.hostname.get","data":{}},"subject":"jobs.query._any"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-789").Return(mockEntry, nil)

				// CreateJob succeeds
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					Times(2)
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				// Retried event put fails (should be logged, not returned)
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("event put failed"))
			},
		},
		{
			name:   "empty target defaults to any",
			jobID:  "job-empty-target",
			target: "",
			setupMocks: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return([]byte(
					`{"id":"job-empty-target","operation":{"type":"node.hostname.get","data":{}},"subject":"jobs.query._any"}`,
				))
				s.mockKV.EXPECT().Get(gomock.Any(), "jobs.job-empty-target").Return(mockEntry, nil)

				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					Times(2)
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				// Retried event
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(2), nil)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			result, err := s.jobsClient.RetryJob(s.ctx, tt.jobID, tt.target)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
				s.NotEmpty(result.JobID)
				s.Equal("created", result.Status)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestGetQueueSummary() {
	tests := []struct {
		name               string
		expectedErr        string
		setupMocks         func()
		expectedTotalJobs  int
		expectedSubmitted  int
		expectedProcessing int
		expectedCompleted  int
		expectedFailed     int
		expectedDLQ        int
	}{
		{
			name: "when no keys found returns empty stats",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
			},
			expectedTotalJobs:  0,
			expectedSubmitted:  0,
			expectedProcessing: 0,
			expectedCompleted:  0,
			expectedFailed:     0,
			expectedDLQ:        0,
		},
		{
			name:        "when keys error returns error",
			expectedErr: "error fetching keys",
			setupMocks: func() {
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return(nil, errors.New("connection failed"))
			},
		},
		{
			name: "when jobs exist returns correct status counts",
			setupMocks: func() {
				keys := []string{
					// Job 1: submitted + acknowledged + started + completed → completed
					"status.job-1.submitted._api.100",
					"status.job-1.acknowledged.agent1.101",
					"status.job-1.started.agent1.102",
					"status.job-1.completed.agent1.103",
					// Job 2: submitted + acknowledged + started → processing
					"status.job-2.submitted._api.200",
					"status.job-2.acknowledged.agent1.201",
					"status.job-2.started.agent1.202",
					// Job 3: submitted + failed → failed
					"status.job-3.submitted._api.300",
					"status.job-3.failed.agent1.301",
					// Job 4: submitted only → submitted
					"status.job-4.submitted._api.400",
					// Job 5: retried → completed
					"status.job-5.submitted._api.500",
					"status.job-5.retried.agent1.501",
					// Non-status keys should be ignored
					"jobs.job-1",
					"responses.job-1.agent1.103",
				}
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return(keys, nil)
				s.mockNATSClient.EXPECT().
					GetStreamInfo(gomock.Any(), "JOBS-DLQ").
					Return(nil, errors.New("no stream"))
			},
			expectedTotalJobs:  5,
			expectedSubmitted:  1,
			expectedProcessing: 1,
			expectedCompleted:  2,
			expectedFailed:     1,
			expectedDLQ:        0,
		},
		{
			name: "when DLQ has messages includes DLQ count",
			setupMocks: func() {
				keys := []string{
					"status.job-1.submitted._api.100",
				}
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return(keys, nil)
				s.mockNATSClient.EXPECT().
					GetStreamInfo(gomock.Any(), "JOBS-DLQ").
					Return(&jetstream.StreamInfo{
						State: jetstream.StreamState{Msgs: 3},
					}, nil)
			},
			expectedTotalJobs:  1,
			expectedSubmitted:  1,
			expectedProcessing: 0,
			expectedCompleted:  0,
			expectedFailed:     0,
			expectedDLQ:        3,
		},
		{
			name: "when malformed status key is skipped",
			setupMocks: func() {
				keys := []string{
					"status.incomplete",
					"status.job-1.submitted._api.100",
					"status.job-1.completed.agent1.101",
				}
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return(keys, nil)
				s.mockNATSClient.EXPECT().
					GetStreamInfo(gomock.Any(), "JOBS-DLQ").
					Return(nil, errors.New("no stream"))
			},
			expectedTotalJobs:  1,
			expectedSubmitted:  0,
			expectedProcessing: 0,
			expectedCompleted:  1,
			expectedFailed:     0,
			expectedDLQ:        0,
		},
		{
			name: "when DLQ error returns zero DLQ count",
			setupMocks: func() {
				keys := []string{
					"status.job-1.submitted._api.100",
					"status.job-1.completed.agent1.101",
				}
				s.mockKV.EXPECT().
					Keys(gomock.Any()).
					Return(keys, nil)
				s.mockNATSClient.EXPECT().
					GetStreamInfo(gomock.Any(), "JOBS-DLQ").
					Return(nil, errors.New("stream not found"))
			},
			expectedTotalJobs:  1,
			expectedSubmitted:  0,
			expectedProcessing: 0,
			expectedCompleted:  1,
			expectedFailed:     0,
			expectedDLQ:        0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			stats, err := s.jobsClient.GetQueueSummary(s.ctx)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
				s.NotNil(stats)
				s.Equal(tt.expectedTotalJobs, stats.TotalJobs)
				s.Equal(tt.expectedSubmitted, stats.StatusCounts["submitted"])
				s.Equal(tt.expectedProcessing, stats.StatusCounts["processing"])
				s.Equal(tt.expectedCompleted, stats.StatusCounts["completed"])
				s.Equal(tt.expectedFailed, stats.StatusCounts["failed"])
				s.Equal(tt.expectedDLQ, stats.DLQCount)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestComputeStatusFromKeyNames() {
	tests := []struct {
		name             string
		keys             []string
		expectedOrderIDs []string
		validateFunc     func(map[string]string)
	}{
		{
			name:             "empty keys",
			keys:             []string{},
			expectedOrderIDs: nil,
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{}, got)
			},
		},
		{
			name:             "only jobs keys no status events",
			keys:             []string{"jobs.job-1", "jobs.job-2"},
			expectedOrderIDs: []string{"job-2", "job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{}, got)
			},
		},
		{
			name: "single agent completed",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.acknowledged.agent1.101",
				"status.job-1.started.agent1.102",
				"status.job-1.completed.agent1.103",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
				}, got)
			},
		},
		{
			name: "single agent failed",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.acknowledged.agent1.101",
				"status.job-1.started.agent1.102",
				"status.job-1.failed.agent1.103",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "failed",
				}, got)
			},
		},
		{
			name: "single agent processing",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.acknowledged.agent1.101",
				"status.job-1.started.agent1.102",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "processing",
				}, got)
			},
		},
		{
			name: "submitted only via api",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "submitted",
				}, got)
			},
		},
		{
			name: "acknowledged only agent shows processing",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.acknowledged.agent1.101",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "processing",
				}, got)
			},
		},
		{
			name: "multi-agent partial failure",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.completed.agent1.101",
				"status.job-1.failed.agent2.102",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "partial_failure",
				}, got)
			},
		},
		{
			name: "multi-agent all completed",
			keys: []string{
				"jobs.job-1",
				"status.job-1.completed.agent1.101",
				"status.job-1.completed.agent2.102",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
				}, got)
			},
		},
		{
			name: "multi-agent one still processing",
			keys: []string{
				"jobs.job-1",
				"status.job-1.completed.agent1.101",
				"status.job-1.started.agent2.102",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "processing",
				}, got)
			},
		},
		{
			name: "retried counts as completed",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.retried.agent1.101",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
				}, got)
			},
		},
		{
			name: "multiple jobs mixed statuses",
			keys: []string{
				"jobs.job-1",
				"jobs.job-2",
				"jobs.job-3",
				"status.job-1.completed.agent1.101",
				"status.job-2.started.agent1.201",
				"status.job-3.failed.agent1.301",
			},
			expectedOrderIDs: []string{"job-3", "job-2", "job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
					"job-2": "processing",
					"job-3": "failed",
				}, got)
			},
		},
		{
			name: "malformed status key skipped",
			keys: []string{
				"jobs.job-1",
				"status.incomplete",
				"status.job-1.completed.agent1.101",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
				}, got)
			},
		},
		{
			name: "non-job non-status keys ignored",
			keys: []string{
				"jobs.job-1",
				"responses.job-1.agent1.100",
				"status.job-1.completed.agent1.101",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
				}, got)
			},
		},
		{
			name:             "empty job ID after trim skipped",
			keys:             []string{"jobs."},
			expectedOrderIDs: nil,
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{}, got)
			},
		},
		{
			name: "single agent skipped",
			keys: []string{
				"jobs.job-1",
				"status.job-1.submitted._api.100",
				"status.job-1.acknowledged.agent1.101",
				"status.job-1.started.agent1.102",
				"status.job-1.skipped.agent1.103",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "skipped",
				}, got)
			},
		},
		{
			name: "multi-agent all skipped",
			keys: []string{
				"jobs.job-1",
				"status.job-1.skipped.agent1.101",
				"status.job-1.skipped.agent2.102",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "skipped",
				}, got)
			},
		},
		{
			name: "multi-agent skipped and completed shows completed",
			keys: []string{
				"jobs.job-1",
				"status.job-1.skipped.agent1.101",
				"status.job-1.completed.agent2.102",
			},
			expectedOrderIDs: []string{"job-1"},
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{
					"job-1": "completed",
				}, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			orderedIDs, statuses := client.ExportComputeStatusFromKeyNames(tt.keys)

			s.Equal(tt.expectedOrderIDs, orderedIDs)
			tt.validateFunc(statuses)
		})
	}
}

func (s *JobsPublicTestSuite) TestCreateJobWithPKISigner() {
	signer, _ := newSigner(gomock.NewController(s.T()))

	tests := []struct {
		name        string
		opData      map[string]interface{}
		target      string
		setupFn     func()
		setupMocks  func()
		expectedErr string
	}{
		{
			name: "when PKI signs job data successfully",
			opData: map[string]interface{}{
				"type": "node.hostname.get",
				"data": map[string]interface{}{},
			},
			target:  "_any",
			setupFn: func() {},
			setupMocks: func() {
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil).
					Times(2)
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "when PKI signing marshal fails returns error",
			opData: map[string]interface{}{
				"type": "node.hostname.get",
				"data": map[string]interface{}{},
			},
			target: "_any",
			setupFn: func() {
				client.SetSigningMarshalFn(func(_ any) ([]byte, error) {
					return nil, errors.New("marshal boom")
				})
			},
			setupMocks: func() {
				s.mockKV.EXPECT().Bucket().Return("test-bucket").AnyTimes()
			},
			expectedErr: "failed to sign job data",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupFn()
			tt.setupMocks()

			opts := &client.Options{
				Timeout:    30 * time.Second,
				KVBucket:   s.mockKV,
				StreamName: "JOBS",
				PKISigner:  signer,
			}
			c, err := client.New(slog.Default(), s.mockNATSClient, opts)
			s.Require().NoError(err)

			result, err := c.CreateJob(s.ctx, tt.opData, tt.target)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
			} else {
				s.NoError(err)
				s.NotEmpty(result.JobID)
				s.Equal("created", result.Status)
			}
		})
	}
}

func (s *JobsPublicTestSuite) TestGetJobStatusWithPKISigner() {
	signer, _ := newSignerWithControllerKey(gomock.NewController(s.T()))
	jobID := "pki-job-123"

	tests := []struct {
		name         string
		setupMocks   func()
		validateFunc func(qj *job.QueuedJob)
	}{
		{
			name: "when response has signed envelope unwraps successfully",
			setupMocks: func() {
				// Job entry.
				jobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				jobEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"id":"%s","status":"unprocessed","created":"2026-01-01T00:00:00Z","subject":"jobs.query.host.server1","operation":{"type":"node.hostname.get"}}`,
					jobID,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs."+jobID).
					Return(jobEntry, nil)

				// Status keys — empty.
				statusLister := newMockKeyLister(s.mockCtrl, []string{})
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status."+jobID+".>").
					Return(statusLister, nil)

				// Response keys — one signed response.
				responseKey := "responses." + jobID + ".server1.12345"
				responseLister := newMockKeyLister(s.mockCtrl, []string{responseKey})
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses."+jobID+".>").
					Return(responseLister, nil)

				// The response entry is a signed envelope.
				inner := []byte(
					`{"status":"completed","hostname":"server1","data":{"hostname":"web-01"}}`,
				)
				wrapped, _ := client.ExportWrapInSignedEnvelope(signer, inner)

				respEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				respEntry.EXPECT().Value().Return(wrapped)
				s.mockKV.EXPECT().
					Get(gomock.Any(), responseKey).
					Return(respEntry, nil)
			},
			validateFunc: func(qj *job.QueuedJob) {
				s.Require().NotNil(qj)
				s.Len(qj.Responses, 1)
				resp, ok := qj.Responses["server1"]
				s.True(ok)
				s.Equal("completed", string(resp.Status))
			},
		},
		{
			name: "when response envelope has invalid signature skips response",
			setupMocks: func() {
				// Job entry.
				jobEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				jobEntry.EXPECT().Value().Return([]byte(fmt.Sprintf(
					`{"id":"%s","status":"unprocessed","created":"2026-01-01T00:00:00Z","subject":"jobs.query.host.server1","operation":{"type":"node.hostname.get"}}`,
					jobID,
				)))
				s.mockKV.EXPECT().
					Get(gomock.Any(), "jobs."+jobID).
					Return(jobEntry, nil)

				// Status keys — empty.
				statusLister := newMockKeyLister(s.mockCtrl, []string{})
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "status."+jobID+".>").
					Return(statusLister, nil)

				// Response keys — one response with tampered signature.
				responseKey := "responses." + jobID + ".server1.12345"
				responseLister := newMockKeyLister(s.mockCtrl, []string{responseKey})
				s.mockKV.EXPECT().
					ListKeysFiltered(gomock.Any(), "responses."+jobID+".>").
					Return(responseLister, nil)

				// Build a valid signed envelope then corrupt the signature.
				inner := []byte(`{"status":"completed","hostname":"server1"}`)
				wrapped, _ := client.ExportWrapInSignedEnvelope(signer, inner)
				var envelope job.SignedEnvelope
				_ = json.Unmarshal(wrapped, &envelope)
				envelope.Signature[0] ^= 0xFF
				corrupted, _ := json.Marshal(envelope)

				respEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				respEntry.EXPECT().Value().Return(corrupted)
				s.mockKV.EXPECT().
					Get(gomock.Any(), responseKey).
					Return(respEntry, nil)
			},
			validateFunc: func(qj *job.QueuedJob) {
				// unwrapSignedEnvelope returns an error for the bad signature,
				// so getJobResponses skips this response.
				s.Require().NotNil(qj)
				s.Len(qj.Responses, 0)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			opts := &client.Options{
				Timeout:    30 * time.Second,
				KVBucket:   s.mockKV,
				StreamName: "JOBS",
				PKISigner:  signer,
			}
			c, err := client.New(slog.Default(), s.mockNATSClient, opts)
			s.Require().NoError(err)

			qj, err := c.GetJobStatus(s.ctx, jobID)
			s.NoError(err)
			tt.validateFunc(qj)
		})
	}
}

func TestJobsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(JobsPublicTestSuite))
}
