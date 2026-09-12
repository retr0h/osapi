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

package job_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/job"
)

type ConfigPublicTestSuite struct {
	suite.Suite
}

func (suite *ConfigPublicTestSuite) SetupTest() {}

func (suite *ConfigPublicTestSuite) SetupSubTest() {}

func (suite *ConfigPublicTestSuite) TearDownTest() {}

func (suite *ConfigPublicTestSuite) TestGetJobsStreamConfig() {
	tests := []struct {
		name         string
		streamConfig *config.NATSStream
		validateFunc func(config *jetstream.StreamConfig)
	}{
		{
			name: "when using file storage and old discard policy",
			streamConfig: &config.NATSStream{
				Name:     "JOBS",
				Subjects: "job.>",
				MaxAge:   "24h",
				MaxMsgs:  10000,
				Storage:  "file",
				Replicas: 1,
				Discard:  "old",
			},
			validateFunc: func(config *jetstream.StreamConfig) {
				suite.NotNil(config)
				suite.Equal("JOBS", config.Name)
				suite.Equal("Stream for job request and processing", config.Description)
				suite.Equal([]string{"job.>"}, config.Subjects)
				suite.Equal(jetstream.FileStorage, config.Storage)
				suite.Equal(1, config.Replicas)
				suite.Equal(24*time.Hour, config.MaxAge)
				suite.Equal(int64(10000), config.MaxMsgs)
				suite.Equal(jetstream.DiscardOld, config.Discard)
			},
		},
		{
			name: "when using memory storage and new discard policy",
			streamConfig: &config.NATSStream{
				Name:     "JOBS_MEMORY",
				Subjects: "job.memory.>",
				MaxAge:   "1h",
				MaxMsgs:  5000,
				Storage:  "memory",
				Replicas: 3,
				Discard:  "new",
			},
			validateFunc: func(config *jetstream.StreamConfig) {
				suite.NotNil(config)
				suite.Equal("JOBS_MEMORY", config.Name)
				suite.Equal("Stream for job request and processing", config.Description)
				suite.Equal([]string{"job.memory.>"}, config.Subjects)
				suite.Equal(jetstream.MemoryStorage, config.Storage)
				suite.Equal(3, config.Replicas)
				suite.Equal(1*time.Hour, config.MaxAge)
				suite.Equal(int64(5000), config.MaxMsgs)
				suite.Equal(jetstream.DiscardNew, config.Discard)
			},
		},
		{
			name: "when using unknown storage defaults to file",
			streamConfig: &config.NATSStream{
				Name:     "JOBS_UNKNOWN",
				Subjects: "job.unknown.>",
				MaxAge:   "12h",
				MaxMsgs:  1000,
				Storage:  "unknown",
				Replicas: 1,
				Discard:  "old",
			},
			validateFunc: func(config *jetstream.StreamConfig) {
				suite.NotNil(config)
				suite.Equal(jetstream.FileStorage, config.Storage)
			},
		},
		{
			name: "when using unknown discard defaults to old",
			streamConfig: &config.NATSStream{
				Name:     "JOBS_DISCARD",
				Subjects: "job.discard.>",
				MaxAge:   "6h",
				MaxMsgs:  2000,
				Storage:  "file",
				Replicas: 1,
				Discard:  "unknown",
			},
			validateFunc: func(config *jetstream.StreamConfig) {
				suite.NotNil(config)
				suite.Equal(jetstream.DiscardOld, config.Discard)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.validateFunc(job.GetJobsStreamConfig(tt.streamConfig))
		})
	}
}

func (suite *ConfigPublicTestSuite) TestGetJobsConsumerConfig() {
	tests := []struct {
		name           string
		consumerConfig *config.AgentConsumer
		streamSubjects string
		validateFunc   func(config jetstream.ConsumerConfig)
	}{
		{
			name: "when using instant replay policy",
			consumerConfig: &config.AgentConsumer{
				Name:          "jobs-agent",
				MaxDeliver:    5,
				AckWait:       "30s",
				MaxAckPending: 100,
				ReplayPolicy:  "instant",
			},
			streamSubjects: "job.>",
			validateFunc: func(config jetstream.ConsumerConfig) {
				suite.Equal("jobs-agent", config.Name)
				suite.Equal("Consumer for processing job requests", config.Description)
				suite.Equal("jobs-agent", config.Durable)
				suite.Equal(jetstream.AckExplicitPolicy, config.AckPolicy)
				suite.Equal(5, config.MaxDeliver)
				suite.Equal(30*time.Second, config.AckWait)
				suite.Equal(100, config.MaxAckPending)
				suite.Equal("job.>", config.FilterSubject)
				suite.Equal(jetstream.ReplayInstantPolicy, config.ReplayPolicy)
			},
		},
		{
			name: "when using original replay policy",
			consumerConfig: &config.AgentConsumer{
				Name:          "test-consumer",
				MaxDeliver:    3,
				AckWait:       "60s",
				MaxAckPending: 50,
				ReplayPolicy:  "original",
			},
			streamSubjects: "job.test.>",
			validateFunc: func(config jetstream.ConsumerConfig) {
				suite.Equal("test-consumer", config.Name)
				suite.Equal("test-consumer", config.Durable)
				suite.Equal(3, config.MaxDeliver)
				suite.Equal(60*time.Second, config.AckWait)
				suite.Equal(50, config.MaxAckPending)
				suite.Equal("job.test.>", config.FilterSubject)
				suite.Equal(jetstream.ReplayOriginalPolicy, config.ReplayPolicy)
			},
		},
		{
			name: "when using unknown replay policy defaults to instant",
			consumerConfig: &config.AgentConsumer{
				Name:          "unknown-consumer",
				MaxDeliver:    1,
				AckWait:       "10s",
				MaxAckPending: 10,
				ReplayPolicy:  "unknown",
			},
			streamSubjects: "job.unknown.>",
			validateFunc: func(config jetstream.ConsumerConfig) {
				suite.Equal(jetstream.ReplayInstantPolicy, config.ReplayPolicy)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.validateFunc(
				job.GetJobsConsumerConfig(tt.consumerConfig, tt.streamSubjects),
			)
		})
	}
}

func (suite *ConfigPublicTestSuite) TestGetKVBucketConfig() {
	tests := []struct {
		name         string
		kvConfig     *config.NATSKV
		validateFunc func(config jetstream.KeyValueConfig)
	}{
		{
			name: "when using file storage",
			kvConfig: &config.NATSKV{
				ResponseBucket: "job-responses",
				TTL:            "1h",
				MaxBytes:       104857600, // 100MB
				Storage:        "file",
				Replicas:       1,
			},
			validateFunc: func(config jetstream.KeyValueConfig) {
				suite.NotNil(config)
				suite.Equal("job-responses", config.Bucket)
				suite.Equal("Storage for job responses indexed by request ID", config.Description)
				suite.Equal(1*time.Hour, config.TTL)
				suite.Equal(int64(100*1024*1024), config.MaxBytes)
				suite.Equal(jetstream.FileStorage, config.Storage)
				suite.Equal(1, config.Replicas)
			},
		},
		{
			name: "when using memory storage",
			kvConfig: &config.NATSKV{
				ResponseBucket: "job-memory",
				TTL:            "30m",
				MaxBytes:       52428800, // 50MB
				Storage:        "memory",
				Replicas:       3,
			},
			validateFunc: func(config jetstream.KeyValueConfig) {
				suite.NotNil(config)
				suite.Equal("job-memory", config.Bucket)
				suite.Equal("Storage for job responses indexed by request ID", config.Description)
				suite.Equal(30*time.Minute, config.TTL)
				suite.Equal(int64(50*1024*1024), config.MaxBytes)
				suite.Equal(jetstream.MemoryStorage, config.Storage)
				suite.Equal(3, config.Replicas)
			},
		},
		{
			name: "when using unknown storage defaults to file",
			kvConfig: &config.NATSKV{
				ResponseBucket: "job-unknown",
				TTL:            "2h",
				MaxBytes:       1048576, // 1MB
				Storage:        "unknown",
				Replicas:       1,
			},
			validateFunc: func(config jetstream.KeyValueConfig) {
				suite.NotNil(config)
				suite.Equal(jetstream.FileStorage, config.Storage)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			tt.validateFunc(job.GetKVBucketConfig(tt.kvConfig))
		})
	}
}

func TestConfigPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConfigPublicTestSuite))
}
