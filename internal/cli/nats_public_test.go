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

package cli_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	natsclient "github.com/osapi-io/nats-client/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/cli"
	climocks "github.com/osapi-io/osapi/internal/cli/mocks"
	"github.com/osapi-io/osapi/internal/config"
)

type NATSPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func TestNATSPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NATSPublicTestSuite))
}

func (suite *NATSPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
}

func (suite *NATSPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *NATSPublicTestSuite) TestCloseNATSClient() {
	tests := []struct {
		name    string
		setupFn func() func()
	}{
		{
			name: "when real client with nil NC does not panic",
			setupFn: func() func() {
				client := &natsclient.Client{}

				return func() {
					cli.CloseNATSClient(client)
				}
			},
		},
		{
			name: "when real client with non-nil NC closes connection",
			setupFn: func() func() {
				mockConn := climocks.NewMockNATSConnector(suite.ctrl)
				mockConn.EXPECT().Close()
				client := &natsclient.Client{NC: mockConn}

				return func() {
					cli.CloseNATSClient(client)
				}
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			closeFn := tc.setupFn()

			assert.NotPanics(suite.T(), closeFn)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildNATSAuthOptions() {
	tests := []struct {
		name         string
		auth         config.NATSAuth
		validateFunc func(natsclient.AuthOptions)
	}{
		{
			name: "when user_pass returns user pass auth",
			auth: config.NATSAuth{
				Type:     "user_pass",
				Username: "osapi",
				Password: "secret",
			},
			validateFunc: func(got natsclient.AuthOptions) {
				assert.Equal(suite.T(), natsclient.AuthOptions{
					AuthType: natsclient.UserPassAuth,
					Username: "osapi",
					Password: "secret",
				}, got)
			},
		},
		{
			name: "when nkey returns nkey auth",
			auth: config.NATSAuth{
				Type:     "nkey",
				NKeyFile: "/path/to/nkey",
			},
			validateFunc: func(got natsclient.AuthOptions) {
				assert.Equal(suite.T(), natsclient.AuthOptions{
					AuthType: natsclient.NKeyAuth,
					NKeyFile: "/path/to/nkey",
				}, got)
			},
		},
		{
			name: "when none returns no auth",
			auth: config.NATSAuth{
				Type: "none",
			},
			validateFunc: func(got natsclient.AuthOptions) {
				assert.Equal(suite.T(), natsclient.AuthOptions{
					AuthType: natsclient.NoAuth,
				}, got)
			},
		},
		{
			name: "when empty type defaults to no auth",
			auth: config.NATSAuth{},
			validateFunc: func(got natsclient.AuthOptions) {
				assert.Equal(suite.T(), natsclient.AuthOptions{
					AuthType: natsclient.NoAuth,
				}, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildNATSAuthOptions(tc.auth)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildJobKVConfig() {
	tests := []struct {
		name         string
		namespace    string
		kvCfg        config.NATSKV
		validateFunc func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			kvCfg: config.NATSKV{
				Bucket:         "job-queue",
				ResponseBucket: "job-responses",
				TTL:            "1h",
				MaxBytes:       104857600,
				Storage:        "file",
				Replicas:       1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-job-queue", cfg.Bucket)
				assert.Equal(suite.T(), 1*time.Hour, cfg.TTL)
				assert.Equal(suite.T(), int64(104857600), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			kvCfg: config.NATSKV{
				Bucket:   "job-queue",
				TTL:      "30m",
				MaxBytes: 52428800,
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "job-queue", cfg.Bucket)
				assert.Equal(suite.T(), 30*time.Minute, cfg.TTL)
				assert.Equal(suite.T(), int64(52428800), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
		{
			name:      "when TTL is invalid defaults to zero",
			namespace: "",
			kvCfg: config.NATSKV{
				Bucket:   "job-queue",
				TTL:      "invalid",
				MaxBytes: 0,
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), time.Duration(0), cfg.TTL)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildJobKVConfig(tc.namespace, tc.kvCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildResponseKVConfig() {
	tests := []struct {
		name         string
		namespace    string
		kvCfg        config.NATSKV
		validateFunc func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			kvCfg: config.NATSKV{
				Bucket:         "job-queue",
				ResponseBucket: "job-responses",
				TTL:            "1h",
				MaxBytes:       104857600,
				Storage:        "file",
				Replicas:       1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-job-responses", cfg.Bucket)
				assert.Equal(suite.T(), 1*time.Hour, cfg.TTL)
				assert.Equal(suite.T(), int64(104857600), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			kvCfg: config.NATSKV{
				Bucket:         "job-queue",
				ResponseBucket: "job-responses",
				TTL:            "30m",
				MaxBytes:       52428800,
				Storage:        "memory",
				Replicas:       3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "job-responses", cfg.Bucket)
				assert.Equal(suite.T(), 30*time.Minute, cfg.TTL)
				assert.Equal(suite.T(), int64(52428800), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildResponseKVConfig(tc.namespace, tc.kvCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildRegistryKVConfig() {
	tests := []struct {
		name         string
		namespace    string
		registryCfg  config.NATSRegistry
		validateFunc func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			registryCfg: config.NATSRegistry{
				Bucket:   "agent-registry",
				TTL:      "30s",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-agent-registry", cfg.Bucket)
				assert.Equal(suite.T(), 30*time.Second, cfg.TTL)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			registryCfg: config.NATSRegistry{
				Bucket:   "agent-registry",
				TTL:      "1m",
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "agent-registry", cfg.Bucket)
				assert.Equal(suite.T(), 1*time.Minute, cfg.TTL)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
		{
			name:      "when TTL is invalid defaults to zero",
			namespace: "",
			registryCfg: config.NATSRegistry{
				Bucket:   "agent-registry",
				TTL:      "invalid",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), time.Duration(0), cfg.TTL)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildRegistryKVConfig(tc.namespace, tc.registryCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildFactsKVConfig() {
	tests := []struct {
		name         string
		namespace    string
		factsCfg     config.NATSFacts
		validateFunc func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			factsCfg: config.NATSFacts{
				Bucket:   "agent-facts",
				TTL:      "1h",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-agent-facts", cfg.Bucket)
				assert.Equal(suite.T(), 1*time.Hour, cfg.TTL)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			factsCfg: config.NATSFacts{
				Bucket:   "agent-facts",
				TTL:      "30m",
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "agent-facts", cfg.Bucket)
				assert.Equal(suite.T(), 30*time.Minute, cfg.TTL)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
		{
			name:      "when TTL is invalid defaults to zero",
			namespace: "",
			factsCfg: config.NATSFacts{
				Bucket:   "agent-facts",
				TTL:      "invalid",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), time.Duration(0), cfg.TTL)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildFactsKVConfig(tc.namespace, tc.factsCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildStateKVConfig() {
	tests := []struct {
		name         string
		namespace    string
		stateCfg     config.NATSState
		validateFunc func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			stateCfg: config.NATSState{
				Bucket:   "agent-state",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-agent-state", cfg.Bucket)
				assert.Equal(suite.T(), time.Duration(0), cfg.TTL)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			stateCfg: config.NATSState{
				Bucket:   "agent-state",
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "agent-state", cfg.Bucket)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildStateKVConfig(tc.namespace, tc.stateCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildAuditStreamConfig() {
	tests := []struct {
		name         string
		namespace    string
		auditCfg     config.NATSAudit
		validateFunc func(jetstream.StreamConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			auditCfg: config.NATSAudit{
				Stream:   "AUDIT",
				Subject:  "audit",
				MaxAge:   "720h",
				MaxBytes: 52428800,
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.StreamConfig) {
				assert.Equal(suite.T(), "osapi-AUDIT", cfg.Name)
				assert.Equal(suite.T(), []string{"osapi.audit.>"}, cfg.Subjects)
				assert.Equal(suite.T(), 720*time.Hour, cfg.MaxAge)
				assert.Equal(suite.T(), int64(52428800), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
				assert.Equal(suite.T(), jetstream.DiscardOld, cfg.Discard)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			auditCfg: config.NATSAudit{
				Stream:   "AUDIT",
				Subject:  "audit",
				MaxAge:   "24h",
				MaxBytes: 1048576,
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.StreamConfig) {
				assert.Equal(suite.T(), "AUDIT", cfg.Name)
				assert.Equal(suite.T(), []string{"audit.>"}, cfg.Subjects)
				assert.Equal(suite.T(), 24*time.Hour, cfg.MaxAge)
				assert.Equal(suite.T(), int64(1048576), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
				assert.Equal(suite.T(), jetstream.DiscardOld, cfg.Discard)
			},
		},
		{
			name:      "when MaxAge is invalid defaults to zero",
			namespace: "",
			auditCfg: config.NATSAudit{
				Stream:   "AUDIT",
				Subject:  "audit",
				MaxAge:   "invalid",
				MaxBytes: 0,
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.StreamConfig) {
				assert.Equal(suite.T(), time.Duration(0), cfg.MaxAge)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildAuditStreamConfig(tc.namespace, tc.auditCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildObjectStoreConfig() {
	tests := []struct {
		name         string
		namespace    string
		objectsCfg   config.NATSObjects
		validateFunc func(jetstream.ObjectStoreConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			objectsCfg: config.NATSObjects{
				Bucket:   "file-objects",
				MaxBytes: 104857600,
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.ObjectStoreConfig) {
				assert.Equal(suite.T(), "osapi-file-objects", cfg.Bucket)
				assert.Equal(suite.T(), int64(104857600), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			objectsCfg: config.NATSObjects{
				Bucket:   "file-objects",
				MaxBytes: 52428800,
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.ObjectStoreConfig) {
				assert.Equal(suite.T(), "file-objects", cfg.Bucket)
				assert.Equal(suite.T(), int64(52428800), cfg.MaxBytes)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
		{
			name:      "when max_bytes is zero",
			namespace: "osapi",
			objectsCfg: config.NATSObjects{
				Bucket:   "file-objects",
				MaxBytes: 0,
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.ObjectStoreConfig) {
				assert.Equal(suite.T(), "osapi-file-objects", cfg.Bucket)
				assert.Equal(suite.T(), int64(0), cfg.MaxBytes)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildObjectStoreConfig(tc.namespace, tc.objectsCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildEnrollmentKVConfig() {
	tests := []struct {
		name          string
		namespace     string
		enrollmentCfg config.NATSEnrollment
		validateFunc  func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			enrollmentCfg: config.NATSEnrollment{
				Bucket:   "agent-enrollment",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-agent-enrollment", cfg.Bucket)
				assert.Equal(suite.T(), time.Duration(0), cfg.TTL)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			enrollmentCfg: config.NATSEnrollment{
				Bucket:   "agent-enrollment",
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "agent-enrollment", cfg.Bucket)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildEnrollmentKVConfig(tc.namespace, tc.enrollmentCfg)

			tc.validateFunc(got)
		})
	}
}

func (suite *NATSPublicTestSuite) TestBuildFileStateKVConfig() {
	tests := []struct {
		name         string
		namespace    string
		fileStateCfg config.NATSFileState
		validateFunc func(jetstream.KeyValueConfig)
	}{
		{
			name:      "when namespace is set",
			namespace: "osapi",
			fileStateCfg: config.NATSFileState{
				Bucket:   "file-state",
				Storage:  "file",
				Replicas: 1,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "osapi-file-state", cfg.Bucket)
				assert.Equal(suite.T(), time.Duration(0), cfg.TTL)
				assert.Equal(suite.T(), jetstream.FileStorage, cfg.Storage)
				assert.Equal(suite.T(), 1, cfg.Replicas)
			},
		},
		{
			name:      "when namespace is empty",
			namespace: "",
			fileStateCfg: config.NATSFileState{
				Bucket:   "file-state",
				Storage:  "memory",
				Replicas: 3,
			},
			validateFunc: func(cfg jetstream.KeyValueConfig) {
				assert.Equal(suite.T(), "file-state", cfg.Bucket)
				assert.Equal(suite.T(), jetstream.MemoryStorage, cfg.Storage)
				assert.Equal(suite.T(), 3, cfg.Replicas)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := cli.BuildFileStateKVConfig(tc.namespace, tc.fileStateCfg)

			tc.validateFunc(got)
		})
	}
}
