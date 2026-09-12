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
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/job/client"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

// registrationJSON returns a minimal agent registration JSON for the given hostname.
func registrationJSON(
	hostname string,
) []byte {
	return []byte(fmt.Sprintf(
		`{"hostname":%q,"registered_at":"2026-01-01T00:00:00Z"}`,
		hostname,
	))
}

// setupRegistryKV configures mockRegistryKV to return the provided hostnames
// from Keys() and then return registration data for each.
func setupRegistryKV(
	ctrl *gomock.Controller,
	hostnames []string,
) *jobmocks.MockKeyValue {
	mockRegistryKV := jobmocks.NewMockKeyValue(ctrl)

	keys := make([]string, len(hostnames))
	for i, h := range hostnames {
		keys[i] = "agents." + job.SanitizeHostname(h)
	}

	mockRegistryKV.EXPECT().
		Keys(gomock.Any()).
		Return(keys, nil)

	for _, h := range hostnames {
		key := "agents." + job.SanitizeHostname(h)
		entry := jobmocks.NewMockKeyValueEntry(ctrl)
		entry.EXPECT().Value().Return(registrationJSON(h))
		mockRegistryKV.EXPECT().
			Get(gomock.Any(), key).
			Return(entry, nil)
	}

	return mockRegistryKV
}

type ClientPublicTestSuite struct {
	suite.Suite

	mockCtrl       *gomock.Controller
	mockNATSClient *jobmocks.MockNATSClient
	mockKV         *jobmocks.MockKeyValue
	jobsClient     *client.Client
	ctx            context.Context
}

func (s *ClientPublicTestSuite) SetupTest() {
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

func (s *ClientPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ClientPublicTestSuite) TestQuery() {
	const (
		target    = "server1"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
		subject   = "jobs.query.host.server1"
	)

	successResp := `{"status":"completed","hostname":"server1"}`
	failedResp := `{"status":"failed","error":"provider error"}`
	skippedResp := `{"status":"skipped","error":"unsupported"}`

	tests := []struct {
		name         string
		data         any
		withMeter    bool
		setupMocks   func()
		expectedErr  string
		validateFunc func(jobID string, resp *job.Response)
	}{
		{
			name:      "when succeeds with meter provider",
			data:      nil,
			withMeter: true,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					successResp,
					nil,
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
			},
		},
		{
			name: "when succeeds",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					successResp,
					nil,
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
				s.Equal(job.StatusCompleted, resp.Status)
			},
		},
		{
			name: "when job failed",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					failedResp,
					nil,
				)
			},
			expectedErr: "job failed: provider error",
		},
		{
			name: "when job skipped",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					skippedResp,
					nil,
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
				s.Equal(job.StatusSkipped, resp.Status)
				s.Equal("unsupported", resp.Error)
			},
		},
		{
			name: "when data marshal fails",
			// A channel cannot be marshaled to JSON.
			data:        make(chan int),
			setupMocks:  func() {},
			expectedErr: "marshal data",
		},
		{
			name: "when publish error",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					"",
					errors.New("kv put failed"),
				)
			},
			expectedErr: "failed to publish and wait",
		},
		{
			name: "when nats publish error",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocksWithOpts(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndWaitMockOpts{
						mockError: errors.New("nats publish failed"),
						errorMode: errorOnPublish,
					},
				)
			},
			expectedErr: "failed to publish and wait",
		},
		{
			name: "when watch error",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocksWithOpts(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndWaitMockOpts{
						mockError: errors.New("watch failed"),
						errorMode: errorOnWatch,
					},
				)
			},
			expectedErr: "failed to publish and wait",
		},
		{
			name: "when timeout waiting for response",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocksWithOpts(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndWaitMockOpts{
						mockError: errors.New("timeout"),
						errorMode: errorOnTimeout,
					},
				)
			},
			expectedErr: "failed to publish and wait",
		},
		{
			name: "when nil entry received before real entry succeeds",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocksWithOpts(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndWaitMockOpts{
						responseData: successResp,
						sendNilFirst: true,
					},
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
				s.Equal(job.StatusCompleted, resp.Status)
			},
		},
		{
			name: "when unmarshal error on response",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocksWithOpts(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndWaitMockOpts{
						responseData: "not-valid-json",
					},
				)
			},
			expectedErr: "failed to publish and wait",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.withMeter {
				s.jobsClient.SetMeterProvider(sdkmetric.NewMeterProvider())
			}

			tt.setupMocks()

			jobID, resp, err := s.jobsClient.Query(
				s.ctx,
				target,
				category,
				operation,
				tt.data,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				s.Empty(jobID)
				s.Nil(resp)
			} else {
				s.NoError(err)
				if tt.validateFunc != nil {
					tt.validateFunc(jobID, resp)
				}
			}
		})
	}
}

func (s *ClientPublicTestSuite) TestQueryBroadcast() {
	const (
		target    = "_all"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
		subject   = "jobs.query._all"
	)

	host1Resp := `{"status":"completed","hostname":"server1"}`
	host2FailResp := `{"status":"failed","error":"provider error","hostname":"server2"}`

	tests := []struct {
		name         string
		data         any
		withMeter    bool
		setupMocks   func() *client.Client
		expectedErr  string
		validateFunc func(jobID string, results map[string]*job.Response)
	}{
		{
			name:      "when succeeds with meter provider",
			data:      nil,
			withMeter: true,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
			},
		},
		{
			name: "when succeeds",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
				s.Contains(results, "server1")
				s.Equal(job.StatusCompleted, results["server1"].Status)
			},
		},
		{
			name: "when data marshal fails in broadcast",
			// A channel cannot be marshaled to JSON.
			data: make(chan int),
			setupMocks: func() *client.Client {
				return s.jobsClient
			},
			expectedErr: "marshal data",
		},
		{
			name: "when job failed",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{
							`{"status":"failed","error":"provider error","hostname":"server1"}`,
						},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
				s.Equal(job.StatusFailed, results["server1"].Status)
				s.Equal("provider error", results["server1"].Error)
			},
		},
		{
			name: "when job skipped",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{
							`{"status":"skipped","hostname":"server1"}`,
						},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
				s.Equal(job.StatusSkipped, results["server1"].Status)
			},
		},
		{
			name: "when publish error",
			data: nil,
			setupMocks: func() *client.Client {
				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						mockError: errors.New("kv put failed"),
						errorMode: errorOnKVPut,
					},
				)
				return s.jobsClient
			},
			expectedErr: "failed to collect broadcast responses",
		},
		{
			name: "when partial failure",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1", "server2"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp, host2FailResp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 2)
				s.Contains(results, "server1")
				s.Equal(job.StatusCompleted, results["server1"].Status)
				s.Contains(results, "server2")
				s.Equal(job.StatusFailed, results["server2"].Status)
				s.Equal("provider error", results["server2"].Error)
			},
		},
		{
			name: "when ListAgents fails falls back to full timeout and collects responses",
			data: nil,
			setupMocks: func() *client.Client {
				// registryKV returns error so ListAgents fails — warn path is exercised.
				registryKV := jobmocks.NewMockKeyValue(s.mockCtrl)
				registryKV.EXPECT().
					Keys(gomock.Any()).
					Return(nil, errors.New("registry unavailable"))
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
			},
		},
		{
			name: "when nats publish error in broadcast",
			data: nil,
			setupMocks: func() *client.Client {
				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						mockError: errors.New("nats publish failed"),
						errorMode: errorOnPublish,
					},
				)
				return s.jobsClient
			},
			expectedErr: "failed to collect broadcast responses",
		},
		{
			name: "when watch error in broadcast",
			data: nil,
			setupMocks: func() *client.Client {
				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						mockError: errors.New("watch failed"),
						errorMode: errorOnWatch,
					},
				)
				return s.jobsClient
			},
			expectedErr: "failed to collect broadcast responses",
		},
		{
			name: "when nil entry received before real entry succeeds",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
						sendNilFirst:    true,
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
			},
		},
		{
			name: "when broadcast response has invalid JSON skips entry then collects valid",
			data: nil,
			setupMocks: func() *client.Client {
				// Two agents in registry: expectedCount=2. Send bad JSON first,
				// then one valid response; the bad one is skipped, the valid one
				// counts as 1 of 2 and the second slot stays open until timeout.
				// Use a tiny timeout so this doesn't wait 30s.
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				opts := &client.Options{
					Timeout:    50 * time.Millisecond,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
					RegistryKV: registryKV,
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						// bad JSON followed by valid JSON; bad is skipped,
						// valid gives us 1 response which satisfies expectedCount=1.
						responseEntries: []string{"not-valid-json", host1Resp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				// The valid response was collected after skipping the bad one.
				s.Len(results, 1)
			},
		},
		{
			name: "when broadcast times out after partial responses returns what was collected",
			data: nil,
			setupMocks: func() *client.Client {
				// No registry so expectedCount=0 (full timeout path).
				// Send one valid response then block. Timeout fires with
				// len(responses)>0 so the non-error timeout branch executes.
				opts := &client.Options{
					Timeout:    50 * time.Millisecond,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
						mockError:       errors.New("partial"),
						errorMode:       errorOnTimeoutWithPartialResponse,
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
			},
		},
		{
			name: "when broadcast response has empty hostname uses unknown",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				// Response has no hostname field — falls back to "unknown" key.
				noHostnameResp := `{"status":"completed"}`
				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{noHostnameResp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Contains(results, "unknown")
			},
		},
		{
			name: "when broadcast times out injects timeout entries for missing agents",
			data: nil,
			setupMocks: func() *client.Client {
				// Registry reports 2 agents but only server1 responds.
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1", "server2"})
				opts := &client.Options{
					Timeout:    50 * time.Millisecond,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
					RegistryKV: registryKV,
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
						mockError:       errors.New("partial"),
						errorMode:       errorOnTimeoutWithPartialResponse,
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 2)
				s.Contains(results, "server1")
				s.Equal(job.StatusCompleted, results["server1"].Status)
				s.Contains(results, "server2")
				s.Equal(job.StatusFailed, results["server2"].Status)
				s.Equal("timeout: agent did not respond", results["server2"].Error)
			},
		},
		{
			name: "when broadcast times out with no responses returns error",
			data: nil,
			setupMocks: func() *client.Client {
				opts := &client.Options{
					Timeout:    50 * time.Millisecond,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						mockError: errors.New("timeout"),
						errorMode: errorOnTimeout,
					},
				)
				return c
			},
			expectedErr: "timeout waiting for broadcast responses: no agents responded",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := tt.setupMocks()
			if tt.withMeter {
				c.SetMeterProvider(sdkmetric.NewMeterProvider())
			}

			jobID, results, err := c.QueryBroadcast(
				s.ctx,
				target,
				category,
				operation,
				tt.data,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				s.Empty(jobID)
				s.Nil(results)
			} else {
				s.NoError(err)
				if tt.validateFunc != nil {
					tt.validateFunc(jobID, results)
				}
			}
		})
	}
}

func (s *ClientPublicTestSuite) TestModify() {
	const (
		target    = "server1"
		category  = "node"
		operation = job.OperationType("node.hostname.set")
		subject   = "jobs.modify.host.server1"
	)

	successResp := `{"status":"completed","hostname":"server1","changed":true}`
	failedResp := `{"status":"failed","error":"permission denied"}`
	skippedResp := `{"status":"skipped","error":"unsupported"}`

	tests := []struct {
		name         string
		data         any
		withMeter    bool
		setupMocks   func()
		expectedErr  string
		validateFunc func(jobID string, resp *job.Response)
	}{
		{
			name:      "when succeeds with meter provider",
			data:      nil,
			withMeter: true,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					successResp,
					nil,
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
			},
		},
		{
			name: "when succeeds",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					successResp,
					nil,
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
				s.Equal(job.StatusCompleted, resp.Status)
			},
		},
		{
			name: "when job failed",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					failedResp,
					nil,
				)
			},
			expectedErr: "job failed: permission denied",
		},
		{
			name: "when job skipped",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					skippedResp,
					nil,
				)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
				s.Equal(job.StatusSkipped, resp.Status)
				s.Equal("unsupported", resp.Error)
			},
		},
		{
			name:        "when data marshal fails",
			data:        make(chan int),
			setupMocks:  func() {},
			expectedErr: "marshal data",
		},
		{
			name: "when publish error",
			data: nil,
			setupMocks: func() {
				setupPublishAndWaitMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					"",
					errors.New("kv put failed"),
				)
			},
			expectedErr: "failed to publish and wait",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.withMeter {
				s.jobsClient.SetMeterProvider(sdkmetric.NewMeterProvider())
			}

			tt.setupMocks()

			jobID, resp, err := s.jobsClient.Modify(
				s.ctx,
				target,
				category,
				operation,
				tt.data,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				s.Empty(jobID)
				s.Nil(resp)
			} else {
				s.NoError(err)
				if tt.validateFunc != nil {
					tt.validateFunc(jobID, resp)
				}
			}
		})
	}
}

func (s *ClientPublicTestSuite) TestModifyBroadcast() {
	const (
		target    = "_all"
		category  = "node"
		operation = job.OperationType("node.hostname.set")
		subject   = "jobs.modify._all"
	)

	host1Resp := `{"status":"completed","hostname":"server1","changed":true}`
	host2FailResp := `{"status":"failed","error":"permission denied","hostname":"server2"}`

	tests := []struct {
		name         string
		data         any
		withMeter    bool
		setupMocks   func() *client.Client
		expectedErr  string
		validateFunc func(jobID string, results map[string]*job.Response)
	}{
		{
			name:      "when succeeds with meter provider",
			data:      nil,
			withMeter: true,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
			},
		},
		{
			name: "when succeeds",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
				s.Contains(results, "server1")
				s.Equal(job.StatusCompleted, results["server1"].Status)
			},
		},
		{
			name: "when data marshal fails in modify broadcast",
			// A channel cannot be marshaled to JSON.
			data: make(chan int),
			setupMocks: func() *client.Client {
				return s.jobsClient
			},
			expectedErr: "marshal data",
		},
		{
			name: "when job failed",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{
							`{"status":"failed","error":"permission denied","hostname":"server1"}`,
						},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
				s.Equal(job.StatusFailed, results["server1"].Status)
				s.Equal("permission denied", results["server1"].Error)
			},
		},
		{
			name: "when job skipped",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{
							`{"status":"skipped","hostname":"server1"}`,
						},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 1)
				s.Equal(job.StatusSkipped, results["server1"].Status)
			},
		},
		{
			name: "when publish error",
			data: nil,
			setupMocks: func() *client.Client {
				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						mockError: errors.New("kv put failed"),
						errorMode: errorOnKVPut,
					},
				)
				return s.jobsClient
			},
			expectedErr: "failed to collect broadcast responses",
		},
		{
			name: "when partial failure",
			data: nil,
			setupMocks: func() *client.Client {
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1", "server2"})
				c := s.newClientWithRegistry(registryKV)

				setupPublishAndCollectMocks(
					s.mockCtrl,
					s.mockKV,
					s.mockNATSClient,
					subject,
					&publishAndCollectMockOpts{
						responseEntries: []string{host1Resp, host2FailResp},
					},
				)
				return c
			},
			validateFunc: func(jobID string, results map[string]*job.Response) {
				s.NotEmpty(jobID)
				s.Len(results, 2)
				s.Contains(results, "server1")
				s.Equal(job.StatusCompleted, results["server1"].Status)
				s.Contains(results, "server2")
				s.Equal(job.StatusFailed, results["server2"].Status)
				s.Equal("permission denied", results["server2"].Error)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := tt.setupMocks()
			if tt.withMeter {
				c.SetMeterProvider(sdkmetric.NewMeterProvider())
			}

			jobID, results, err := c.ModifyBroadcast(
				s.ctx,
				target,
				category,
				operation,
				tt.data,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				s.Empty(jobID)
				s.Nil(results)
			} else {
				s.NoError(err)
				if tt.validateFunc != nil {
					tt.validateFunc(jobID, results)
				}
			}
		})
	}
}

// newClientWithRegistry creates a new client using the suite's existing mocks
// plus the provided registryKV.
func (s *ClientPublicTestSuite) newClientWithRegistry(
	registryKV *jobmocks.MockKeyValue,
) *client.Client {
	opts := &client.Options{
		Timeout:    30 * time.Second,
		KVBucket:   s.mockKV,
		StreamName: "JOBS",
		RegistryKV: registryKV,
	}
	c, err := client.New(slog.Default(), s.mockNATSClient, opts)
	s.Require().NoError(err)
	return c
}

func (s *ClientPublicTestSuite) TestQueryWithPKISigner() {
	const (
		target    = "server1"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
		subject   = "jobs.query.host.server1"
	)

	signer, _ := newSigner(gomock.NewController(s.T()))

	tests := []struct {
		name         string
		setupMocks   func()
		expectedErr  string
		validateFunc func(jobID string, resp *job.Response)
	}{
		{
			name: "when PKI signs job data and unwraps response",
			setupMocks: func() {
				// KV Put should receive a signed envelope.
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, _ string, data []byte) (uint64, error) {
						// Verify the stored data is a valid signed envelope.
						var envelope job.SignedEnvelope
						s.NoError(json.Unmarshal(data, &envelope))
						s.NotEmpty(envelope.Payload)
						s.NotEmpty(envelope.Signature)
						s.Equal("SHA256:test-fingerprint", envelope.Fingerprint)
						return uint64(1), nil
					})

				// Publish succeeds.
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), subject, gomock.Any()).
					Return(nil)

				// Watch returns a response.
				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(
					[]byte(`{"status":"completed","hostname":"server1"}`),
				)
				ch := make(chan jetstream.KeyValueEntry, 1)
				ch <- mockEntry

				mockWatcher := jobmocks.NewMockKeyWatcher(s.mockCtrl)
				mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
				mockWatcher.EXPECT().Stop().Return(nil)

				s.mockKV.EXPECT().
					Watch(gomock.Any(), gomock.Any()).
					Return(mockWatcher, nil)
			},
			validateFunc: func(jobID string, resp *job.Response) {
				s.NotEmpty(jobID)
				s.NotNil(resp)
				s.Equal(job.StatusCompleted, resp.Status)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create a client with PKI signer.
			opts := &client.Options{
				Timeout:    30 * time.Second,
				KVBucket:   s.mockKV,
				StreamName: "JOBS",
				PKISigner:  signer,
			}
			c, err := client.New(slog.Default(), s.mockNATSClient, opts)
			s.Require().NoError(err)

			tt.setupMocks()

			jobID, resp, err := c.Query(
				s.ctx,
				target,
				category,
				operation,
				nil,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				return
			}

			s.NoError(err)
			if tt.validateFunc != nil {
				tt.validateFunc(jobID, resp)
			}
		})
	}
}

func (s *ClientPublicTestSuite) TestQueryWithPKISignerSignError() {
	const (
		target    = "server1"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
	)

	signer, _ := newSigner(gomock.NewController(s.T()))

	tests := []struct {
		name         string
		setupFn      func()
		validateFunc func(string, *job.Response, error)
	}{
		{
			name: "when signing marshal fails returns sign error",
			setupFn: func() {
				client.SetSigningMarshalFn(func(_ any) ([]byte, error) {
					return nil, errors.New("marshal boom")
				})
			},
			validateFunc: func(_ string, _ *job.Response, err error) {
				s.Error(err)
				s.Contains(err.Error(), "failed to sign job data")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupFn()

			opts := &client.Options{
				Timeout:    30 * time.Second,
				KVBucket:   s.mockKV,
				StreamName: "JOBS",
				PKISigner:  signer,
			}
			c, err := client.New(slog.Default(), s.mockNATSClient, opts)
			s.Require().NoError(err)

			tt.validateFunc(c.Query(s.ctx, target, category, operation, nil))
		})
	}
}

func (s *ClientPublicTestSuite) TearDownSubTest() {
	client.ResetSigningMarshalFn()
}

func (s *ClientPublicTestSuite) TestQueryWithPKISignerUnwrapPaths() {
	const (
		target    = "server1"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
		subject   = "jobs.query.host.server1"
	)

	signer, _ := newSignerWithControllerKey(gomock.NewController(s.T()))

	tests := []struct {
		name         string
		responseData func() []byte
		validateFunc func(jobID string, resp *job.Response)
	}{
		{
			name: "when response is a valid signed envelope unwraps successfully",
			responseData: func() []byte {
				inner := []byte(`{"status":"completed","hostname":"server1"}`)
				wrapped, _ := client.ExportWrapInSignedEnvelope(signer, inner)
				return wrapped
			},
			validateFunc: func(_ string, resp *job.Response) {
				s.NotNil(resp)
				s.Equal(job.StatusCompleted, resp.Status)
			},
		},
		{
			name: "when response envelope has invalid signature warns and uses raw envelope data",
			responseData: func() []byte {
				// Build an envelope with a tampered signature so verification fails.
				inner := []byte(`{"status":"completed","hostname":"server1"}`)
				wrapped, _ := client.ExportWrapInSignedEnvelope(signer, inner)
				var envelope job.SignedEnvelope
				_ = json.Unmarshal(wrapped, &envelope)
				envelope.Signature[0] ^= 0xFF
				corrupted, _ := json.Marshal(envelope)
				return corrupted
			},
			// The warn path fires. The raw responseData (the envelope JSON) is then
			// unmarshalled as a job.Response — unrecognized fields are ignored,
			// so status ends up as zero value.
			validateFunc: func(_ string, resp *job.Response) {
				s.NotNil(resp)
				// Status is empty because the raw envelope JSON doesn't have a
				// "status" field at the top level.
				s.Equal(job.Status(""), resp.Status)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			opts := &client.Options{
				Timeout:    30 * time.Second,
				KVBucket:   s.mockKV,
				StreamName: "JOBS",
				PKISigner:  signer,
			}
			c, err := client.New(slog.Default(), s.mockNATSClient, opts)
			s.Require().NoError(err)

			// KV Put succeeds (signed envelope).
			s.mockKV.EXPECT().
				Put(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(uint64(1), nil)
			s.mockNATSClient.EXPECT().
				Publish(gomock.Any(), subject, gomock.Any()).
				Return(nil)

			mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
			mockEntry.EXPECT().Value().Return(tt.responseData())
			ch := make(chan jetstream.KeyValueEntry, 1)
			ch <- mockEntry

			mockWatcher := jobmocks.NewMockKeyWatcher(s.mockCtrl)
			mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
			mockWatcher.EXPECT().Stop().Return(nil)

			s.mockKV.EXPECT().
				Watch(gomock.Any(), gomock.Any()).
				Return(mockWatcher, nil)

			jobID, resp, err := c.Query(s.ctx, target, category, operation, nil)
			s.NoError(err)
			tt.validateFunc(jobID, resp)
		})
	}
}

func (s *ClientPublicTestSuite) TestModifyBroadcastWithPKISigner() {
	const (
		target    = "_all"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
		subject   = "jobs.modify._all"
	)

	signer, _ := newSigner(gomock.NewController(s.T()))

	// Use a signer with ControllerPublicKey set so verification is attempted.
	signerWithCtrl, _ := newSignerWithControllerKey(gomock.NewController(s.T()))

	tests := []struct {
		name         string
		setupMocks   func()
		expectedErr  string
		validateFunc func(responses map[string]*job.Response)
	}{
		{
			name: "when PKI signs and unwraps broadcast responses",
			setupMocks: func() {
				// Registry returns one agent.
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})

				opts := &client.Options{
					Timeout:    30 * time.Second,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
					PKISigner:  signerWithCtrl,
					RegistryKV: registryKV,
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)
				s.jobsClient = c

				// KV Put succeeds.
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), subject, gomock.Any()).
					Return(nil)

				// Return a signed envelope response.
				inner := []byte(`{"status":"completed","hostname":"server1"}`)
				wrapped, _ := client.ExportWrapInSignedEnvelope(signerWithCtrl, inner)

				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(wrapped)
				ch := make(chan jetstream.KeyValueEntry, 1)
				ch <- mockEntry

				mockWatcher := jobmocks.NewMockKeyWatcher(s.mockCtrl)
				mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
				mockWatcher.EXPECT().Stop().Return(nil)

				s.mockKV.EXPECT().
					Watch(gomock.Any(), gomock.Any()).
					Return(mockWatcher, nil)
			},
			validateFunc: func(responses map[string]*job.Response) {
				s.Len(responses, 1)
				s.Equal(job.StatusCompleted, responses["server1"].Status)
			},
		},
		{
			name: "when broadcast response has invalid signature warns and skips",
			setupMocks: func() {
				// Registry returns one agent.
				registryKV := setupRegistryKV(s.mockCtrl, []string{"server1"})

				opts := &client.Options{
					Timeout:    1 * time.Second,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
					PKISigner:  signerWithCtrl,
					RegistryKV: registryKV,
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)
				s.jobsClient = c

				// KV Put succeeds.
				s.mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				s.mockNATSClient.EXPECT().
					Publish(gomock.Any(), subject, gomock.Any()).
					Return(nil)

				// Return a signed envelope with tampered signature.
				inner := []byte(`{"status":"completed","hostname":"server1"}`)
				wrapped, _ := client.ExportWrapInSignedEnvelope(signerWithCtrl, inner)
				var envelope job.SignedEnvelope
				_ = json.Unmarshal(wrapped, &envelope)
				envelope.Signature[0] ^= 0xFF
				corrupted, _ := json.Marshal(envelope)

				mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
				mockEntry.EXPECT().Value().Return(corrupted)
				ch := make(chan jetstream.KeyValueEntry, 1)
				ch <- mockEntry

				mockWatcher := jobmocks.NewMockKeyWatcher(s.mockCtrl)
				mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
				mockWatcher.EXPECT().Stop().Return(nil)

				s.mockKV.EXPECT().
					Watch(gomock.Any(), gomock.Any()).
					Return(mockWatcher, nil)
			},
			// The warn fires, raw envelope JSON is unmarshalled to a Response
			// with empty hostname, so it goes under "unknown". Then timeout
			// fires because expected agents (server1) never responded.
			validateFunc: func(responses map[string]*job.Response) {
				s.NotEmpty(responses)
			},
		},
		{
			name: "when PKI signing marshal fails returns error",
			setupMocks: func() {
				client.SetSigningMarshalFn(func(_ any) ([]byte, error) {
					return nil, errors.New("marshal boom")
				})

				opts := &client.Options{
					Timeout:    30 * time.Second,
					KVBucket:   s.mockKV,
					StreamName: "JOBS",
					PKISigner:  signer,
				}
				c, err := client.New(slog.Default(), s.mockNATSClient, opts)
				s.Require().NoError(err)
				s.jobsClient = c
			},
			expectedErr: "failed to sign job data",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMocks()

			_, responses, err := s.jobsClient.ModifyBroadcast(
				s.ctx, target, category, operation, nil,
			)

			if tt.expectedErr != "" {
				s.Error(err)
				s.Contains(err.Error(), tt.expectedErr)
				return
			}

			s.NoError(err)
			if tt.validateFunc != nil {
				tt.validateFunc(responses)
			}
		})
	}
}

func (s *ClientPublicTestSuite) TestQueryWithTargetResolver() {
	const (
		hostname  = "web-01"
		machineID = "abc123def456"
		category  = "node"
		operation = job.OperationType("node.hostname.get")
	)

	// The resolver translates hostname → machineID.
	resolver := func(target string) string {
		if target == hostname {
			return machineID
		}
		return target
	}

	// Expect subject to use the resolved machineID, not the hostname.
	expectedSubject := "jobs.query.host." + machineID

	s.mockKV.EXPECT().
		Put(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(uint64(1), nil)
	s.mockNATSClient.EXPECT().
		Publish(gomock.Any(), expectedSubject, gomock.Any()).
		Return(nil)

	mockEntry := jobmocks.NewMockKeyValueEntry(s.mockCtrl)
	mockEntry.EXPECT().Value().Return(
		[]byte(`{"status":"completed","hostname":"web-01"}`),
	)
	ch := make(chan jetstream.KeyValueEntry, 1)
	ch <- mockEntry

	mockWatcher := jobmocks.NewMockKeyWatcher(s.mockCtrl)
	mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
	mockWatcher.EXPECT().Stop().Return(nil)

	s.mockKV.EXPECT().
		Watch(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(mockWatcher, nil)

	opts := &client.Options{
		Timeout:        30 * time.Second,
		KVBucket:       s.mockKV,
		StreamName:     "JOBS",
		TargetResolver: resolver,
	}
	c, err := client.New(slog.Default(), s.mockNATSClient, opts)
	s.Require().NoError(err)

	_, resp, err := c.Query(s.ctx, hostname, category, operation, nil)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(job.StatusCompleted, resp.Status)
}

func TestClientPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ClientPublicTestSuite))
}

// publishAndWaitErrorMode controls which step of the publishAndWait flow fails.
type publishAndWaitErrorMode int

const (
	// errorOnKVPut makes kv.Put return the error.
	errorOnKVPut publishAndWaitErrorMode = iota
	// errorOnPublish makes natsClient.Publish return the error.
	errorOnPublish
	// errorOnWatch makes kv.Watch return the error.
	errorOnWatch
	// errorOnTimeout simulates a timeout by providing a channel that never sends.
	errorOnTimeout
	// errorOnTimeoutWithPartialResponse sends one valid response then blocks,
	// simulating a timeout after partial collection. Only valid for
	// publishAndCollect (broadcast) flows.
	errorOnTimeoutWithPartialResponse
)

// publishAndWaitMockOpts configures the mock behavior for publishAndWait tests.
type publishAndWaitMockOpts struct {
	// responseData is the JSON response to return from the watcher entry.
	responseData string
	// mockError is the error to inject (used with errorMode).
	mockError error
	// errorMode controls which step fails when mockError is set.
	errorMode publishAndWaitErrorMode
	// sendNilFirst sends a nil entry before the real entry on the watcher channel.
	sendNilFirst bool
}

// setupPublishAndWaitMocks configures mocks for the publishAndWait flow.
func setupPublishAndWaitMocks(
	ctrl *gomock.Controller,
	mockKV *jobmocks.MockKeyValue,
	mockNATSClient *jobmocks.MockNATSClient,
	subject string,
	responseData string,
	mockError error,
) {
	setupPublishAndWaitMocksWithOpts(ctrl, mockKV, mockNATSClient, subject, &publishAndWaitMockOpts{
		responseData: responseData,
		mockError:    mockError,
		errorMode:    errorOnKVPut,
	})
}

// publishAndCollectMockOpts configures mock behavior for publishAndCollect tests.
type publishAndCollectMockOpts struct {
	// responseEntries is a list of JSON response strings to send on the watcher.
	responseEntries []string
	// mockError is the error to inject (used with errorMode).
	mockError error
	// errorMode controls which step fails when mockError is set.
	errorMode publishAndWaitErrorMode
	// sendNilFirst sends a nil entry before the response entries on the watcher channel.
	sendNilFirst bool
}

// setupPublishAndCollectMocks configures mocks for the publishAndCollect flow.
func setupPublishAndCollectMocks(
	ctrl *gomock.Controller,
	mockKV *jobmocks.MockKeyValue,
	mockNATSClient *jobmocks.MockNATSClient,
	subject string,
	opts *publishAndCollectMockOpts,
) {
	if opts.mockError != nil && opts.errorMode == errorOnKVPut {
		mockKV.EXPECT().
			Put(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(uint64(0), opts.mockError)
		return
	}

	// kv.Put succeeds
	mockKV.EXPECT().
		Put(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(uint64(1), nil)

	if opts.mockError != nil && opts.errorMode == errorOnPublish {
		mockNATSClient.EXPECT().
			Publish(gomock.Any(), subject, gomock.Any()).
			Return(opts.mockError)
		return
	}

	// natsClient.Publish succeeds
	mockNATSClient.EXPECT().
		Publish(gomock.Any(), subject, gomock.Any()).
		Return(nil)

	if opts.mockError != nil && opts.errorMode == errorOnWatch {
		mockKV.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			Return(nil, opts.mockError)
		return
	}

	if opts.mockError != nil && opts.errorMode == errorOnTimeout {
		// Return a channel that never sends anything, causing timeout with 0 responses
		ch := make(chan jetstream.KeyValueEntry)

		mockWatcher := jobmocks.NewMockKeyWatcher(ctrl)
		mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
		mockWatcher.EXPECT().Stop().Return(nil)

		mockKV.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			Return(mockWatcher, nil)
		return
	}

	if opts.mockError != nil && opts.errorMode == errorOnTimeoutWithPartialResponse {
		// Send one valid response then block so the timeout fires with partial results.
		// The first responseEntry provides the partial response data.
		ch := make(chan jetstream.KeyValueEntry, 1)
		mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
		responseData := ""
		if len(opts.responseEntries) > 0 {
			responseData = opts.responseEntries[0]
		}
		mockEntry.EXPECT().Value().Return([]byte(responseData))
		ch <- mockEntry

		mockWatcher := jobmocks.NewMockKeyWatcher(ctrl)
		mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
		mockWatcher.EXPECT().Stop().Return(nil)

		mockKV.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			Return(mockWatcher, nil)
		return
	}

	// Create buffered channel with all response entries
	bufSize := len(opts.responseEntries)
	if opts.sendNilFirst {
		bufSize++
	}
	ch := make(chan jetstream.KeyValueEntry, bufSize)
	if opts.sendNilFirst {
		ch <- nil
	}
	for _, data := range opts.responseEntries {
		if data == "" {
			continue
		}
		mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
		mockEntry.EXPECT().Value().Return([]byte(data))
		ch <- mockEntry
	}

	// Create mock watcher
	mockWatcher := jobmocks.NewMockKeyWatcher(ctrl)
	mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
	mockWatcher.EXPECT().Stop().Return(nil)

	// kv.Watch returns the mock watcher
	mockKV.EXPECT().
		Watch(gomock.Any(), gomock.Any()).
		Return(mockWatcher, nil)
}

// setupPublishAndWaitMocksWithOpts configures mocks with fine-grained control.
func setupPublishAndWaitMocksWithOpts(
	ctrl *gomock.Controller,
	mockKV *jobmocks.MockKeyValue,
	mockNATSClient *jobmocks.MockNATSClient,
	subject string,
	opts *publishAndWaitMockOpts,
) {
	if opts.mockError != nil && opts.errorMode == errorOnKVPut {
		mockKV.EXPECT().
			Put(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(uint64(0), opts.mockError)
		return
	}

	// kv.Put succeeds
	mockKV.EXPECT().
		Put(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(uint64(1), nil)

	if opts.mockError != nil && opts.errorMode == errorOnPublish {
		mockNATSClient.EXPECT().
			Publish(gomock.Any(), subject, gomock.Any()).
			Return(opts.mockError)
		return
	}

	// natsClient.Publish succeeds
	mockNATSClient.EXPECT().
		Publish(gomock.Any(), subject, gomock.Any()).
		Return(nil)

	if opts.mockError != nil && opts.errorMode == errorOnWatch {
		mockKV.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			Return(nil, opts.mockError)
		return
	}

	if opts.mockError != nil && opts.errorMode == errorOnTimeout {
		// Return a channel that never sends anything, causing timeout
		ch := make(chan jetstream.KeyValueEntry)

		mockWatcher := jobmocks.NewMockKeyWatcher(ctrl)
		mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
		mockWatcher.EXPECT().Stop().Return(nil)

		mockKV.EXPECT().
			Watch(gomock.Any(), gomock.Any()).
			Return(mockWatcher, nil)
		return
	}

	// Create mock entry with response data
	mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
	mockEntry.EXPECT().Value().Return([]byte(opts.responseData))

	// Create buffered channel and optionally send nil first
	bufSize := 1
	if opts.sendNilFirst {
		bufSize = 2
	}
	ch := make(chan jetstream.KeyValueEntry, bufSize)
	if opts.sendNilFirst {
		ch <- nil
	}
	ch <- mockEntry

	// Create mock watcher
	mockWatcher := jobmocks.NewMockKeyWatcher(ctrl)
	mockWatcher.EXPECT().Updates().Return(ch).AnyTimes()
	mockWatcher.EXPECT().Stop().Return(nil)

	// kv.Watch returns the mock watcher
	mockKV.EXPECT().
		Watch(gomock.Any(), gomock.Any()).
		Return(mockWatcher, nil)
}
