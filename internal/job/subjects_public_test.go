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

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/job"
)

type SubjectsPublicTestSuite struct {
	suite.Suite
}

func (suite *SubjectsPublicTestSuite) SetupTest() {
	// Reset namespace to default before each test
	job.Init("")
}

func (suite *SubjectsPublicTestSuite) SetupSubTest() {}

func (suite *SubjectsPublicTestSuite) TearDownTest() {}

func (suite *SubjectsPublicTestSuite) TestBuildQuerySubject() {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{
			name:     "when building query subject for specific server",
			hostname: "server-01",
			want:     "jobs.query.server-01",
		},
		{
			name:     "when building query subject for web server",
			hostname: "web-server",
			want:     "jobs.query.web-server",
		},
		{
			name:     "when building with wildcard hostname",
			hostname: job.AllHosts,
			want:     "jobs.query.*",
		},
		{
			name:     "when building with any hostname",
			hostname: job.AnyHost,
			want:     "jobs.query._any",
		},
		{
			name:     "when building query subject for all hosts",
			hostname: "",
			want:     "jobs.query.*",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var got string
			if tt.hostname == "" {
				got = job.BuildQuerySubjectForAllHosts()
			} else {
				got = job.BuildQuerySubject(tt.hostname)
			}
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestBuildModifySubject() {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{
			name:     "when building modify subject for specific server",
			hostname: "server-01",
			want:     "jobs.modify.server-01",
		},
		{
			name:     "when building modify subject for db server",
			hostname: "db-server",
			want:     "jobs.modify.db-server",
		},
		{
			name:     "when building with wildcard hostname",
			hostname: job.AllHosts,
			want:     "jobs.modify.*",
		},
		{
			name:     "when building with any hostname",
			hostname: job.AnyHost,
			want:     "jobs.modify._any",
		},
		{
			name:     "when building modify subject for all hosts",
			hostname: "",
			want:     "jobs.modify.*",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			var got string
			if tt.hostname == "" {
				got = job.BuildModifySubjectForAllHosts()
			} else {
				got = job.BuildModifySubject(tt.hostname)
			}
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestParseSubject() {
	tests := []struct {
		name         string
		subject      string
		wantPrefix   string
		wantHostname string
		wantErr      bool
	}{
		{
			name:         "when parsing valid query subject",
			subject:      "jobs.query.server-01",
			wantPrefix:   "jobs.query",
			wantHostname: "server-01",
			wantErr:      false,
		},
		{
			name:         "when parsing valid modify subject",
			subject:      "jobs.modify.web-01",
			wantPrefix:   "jobs.modify",
			wantHostname: "web-01",
			wantErr:      false,
		},
		{
			name:         "when parsing subject with wildcard hostname",
			subject:      "jobs.query.*",
			wantPrefix:   "jobs.query",
			wantHostname: "*",
			wantErr:      false,
		},
		{
			name:         "when parsing subject with any hostname",
			subject:      "jobs.modify._any",
			wantPrefix:   "jobs.modify",
			wantHostname: "_any",
			wantErr:      false,
		},
		{
			name:    "when parsing invalid subject with too few parts",
			subject: "jobs.query",
			wantErr: true,
		},
		{
			name:         "when parsing host subject",
			subject:      "jobs.query.host.server-01",
			wantPrefix:   "jobs.query",
			wantHostname: "server-01",
			wantErr:      false,
		},
		{
			name:         "when parsing label subject",
			subject:      "jobs.query.label.group.web",
			wantPrefix:   "jobs.query",
			wantHostname: "group:web",
			wantErr:      false,
		},
		{
			name:         "when parsing hierarchical label subject",
			subject:      "jobs.query.label.group.web.dev.us-east",
			wantPrefix:   "jobs.query",
			wantHostname: "group:web.dev.us-east",
			wantErr:      false,
		},
		{
			name:    "when parsing invalid 4-part subject without host prefix",
			subject: "jobs.query.invalid.server1",
			wantErr: true,
		},
		{
			name:    "when parsing empty subject",
			subject: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			gotPrefix, gotHostname, err := job.ParseSubject(tt.subject)

			if tt.wantErr {
				suite.Error(err)
				return
			}

			suite.NoError(err)
			suite.Equal(tt.wantPrefix, gotPrefix)
			suite.Equal(tt.wantHostname, gotHostname)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestGetLocalHostname() {
	// This test uses the real system hostname
	hostname, err := job.GetLocalHostname()
	suite.NoError(err)
	suite.NotEmpty(hostname)
}

func (suite *SubjectsPublicTestSuite) TestSanitizeHostname() {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{
			name:     "when hostname has no special characters",
			hostname: "server01",
			want:     "server01",
		},
		{
			name:     "when hostname has hyphens",
			hostname: "web-server-01",
			want:     "web_server_01",
		},
		{
			name:     "when hostname has dots",
			hostname: "server.example.com",
			want:     "server_example_com",
		},
		{
			name:     "when hostname has hyphens and dots",
			hostname: "Johns-MacBook-Pro-2.local",
			want:     "Johns_MacBook_Pro_2_local",
		},
		{
			name:     "when hostname has mixed special characters",
			hostname: "test@host#123.domain!",
			want:     "test_host_123_domain_",
		},
		{
			name:     "when hostname has underscores (should be preserved)",
			hostname: "test_server_01",
			want:     "test_server_01",
		},
		{
			name:     "when hostname has numbers",
			hostname: "server123",
			want:     "server123",
		},
		{
			name:     "when hostname is empty",
			hostname: "",
			want:     "",
		},
		{
			name:     "when hostname has spaces",
			hostname: "my server name",
			want:     "my_server_name",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.SanitizeHostname(tt.hostname)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestBuildAgentSubscriptionPattern() {
	tests := []struct {
		name     string
		hostname string
		labels   map[string]string
		want     []string
	}{
		{
			name:     "when building subscription pattern for specific hostname",
			hostname: "web-server-01",
			want: []string{
				"jobs.*.host.web_server_01",
				"jobs.*._any",
				"jobs.*._all",
			},
		},
		{
			name:     "when building subscription pattern for localhost",
			hostname: "localhost",
			want: []string{
				"jobs.*.host.localhost",
				"jobs.*._any",
				"jobs.*._all",
			},
		},
		{
			name:     "when building subscription pattern with dotted hostname",
			hostname: "api.example.com",
			want: []string{
				"jobs.*.host.api_example_com",
				"jobs.*._any",
				"jobs.*._all",
			},
		},
		{
			name:     "when building with hierarchical label",
			hostname: "web-01",
			labels:   map[string]string{"group": "web.dev.us-east"},
			want: []string{
				"jobs.*.host.web_01",
				"jobs.*._any",
				"jobs.*._all",
				"jobs.*.label.group.web",
				"jobs.*.label.group.web.dev",
				"jobs.*.label.group.web.dev.us-east",
			},
		},
		{
			name:     "when building with flat label",
			hostname: "web-01",
			labels:   map[string]string{"team": "platform"},
			want: []string{
				"jobs.*.host.web_01",
				"jobs.*._any",
				"jobs.*._all",
				"jobs.*.label.team.platform",
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.BuildAgentSubscriptionPattern(tt.hostname, tt.labels)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestBuildAgentQueueGroup() {
	tests := []struct {
		name     string
		category string
		want     string
	}{
		{
			name:     "when building queue group for node category",
			category: "node",
			want:     "agents.node",
		},
		{
			name:     "when building queue group for network category",
			category: "network",
			want:     "agents.network",
		},
		{
			name:     "when building queue group for jobs category",
			category: "jobs",
			want:     "agents.jobs",
		},
		{
			name:     "when building queue group with empty category",
			category: "",
			want:     "agents.",
		},
		{
			name:     "when building queue group with complex category",
			category: "custom-service",
			want:     "agents.custom-service",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.BuildAgentQueueGroup(tt.category)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestIsSpecialHostname() {
	tests := []struct {
		name     string
		hostname string
		want     bool
	}{
		{
			name:     "when hostname is AllHosts wildcard",
			hostname: job.AllHosts,
			want:     true,
		},
		{
			name:     "when hostname is AnyHost",
			hostname: job.AnyHost,
			want:     true,
		},
		{
			name:     "when hostname is LocalHost",
			hostname: job.LocalHost,
			want:     true,
		},
		{
			name:     "when hostname is BroadcastHost",
			hostname: job.BroadcastHost,
			want:     true,
		},
		{
			name:     "when hostname is regular server name",
			hostname: "web-server-01",
			want:     false,
		},
		{
			name:     "when hostname is localhost",
			hostname: "localhost",
			want:     false,
		},
		{
			name:     "when hostname is FQDN",
			hostname: "api.example.com",
			want:     false,
		},
		{
			name:     "when hostname is empty",
			hostname: "",
			want:     false,
		},
		{
			name:     "when hostname looks like special but isn't exact",
			hostname: "_any_server",
			want:     false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.IsSpecialHostname(tt.hostname)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestValidateLabel() {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:  "when key and value are simple alphanumeric",
			key:   "role",
			value: "web",
		},
		{
			name:  "when value has hyphens and underscores",
			key:   "env",
			value: "us-east_1",
		},
		{
			name:  "when value is hierarchical with dots",
			key:   "group",
			value: "web.dev.us-east",
		},
		{
			name:    "when key contains dots",
			key:     "my.key",
			value:   "web",
			wantErr: true,
		},
		{
			name:    "when key contains colon",
			key:     "my:key",
			value:   "web",
			wantErr: true,
		},
		{
			name:    "when value segment contains spaces",
			key:     "group",
			value:   "web.dev server",
			wantErr: true,
		},
		{
			name:    "when value has empty segment",
			key:     "group",
			value:   "web..dev",
			wantErr: true,
		},
		{
			name:    "when key is empty",
			key:     "",
			value:   "web",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := job.ValidateLabel(tt.key, tt.value)
			if tt.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestParseTarget() {
	tests := []struct {
		name        string
		target      string
		wantRouting string
		wantKey     string
		wantValue   string
	}{
		{
			name:        "when target is _any",
			target:      "_any",
			wantRouting: "_any",
		},
		{
			name:        "when target is _all",
			target:      "_all",
			wantRouting: "_all",
		},
		{
			name:        "when target is a hostname",
			target:      "server1",
			wantRouting: "host",
			wantKey:     "server1",
		},
		{
			name:        "when target is a flat label",
			target:      "role:web",
			wantRouting: "label",
			wantKey:     "role",
			wantValue:   "web",
		},
		{
			name:        "when target is a hierarchical label",
			target:      "group:web.dev.us-east",
			wantRouting: "label",
			wantKey:     "group",
			wantValue:   "web.dev.us-east",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			rt, key, value := job.ParseTarget(tt.target)
			suite.Equal(tt.wantRouting, rt)
			suite.Equal(tt.wantKey, key)
			suite.Equal(tt.wantValue, value)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestBuildSubjectFromTarget() {
	tests := []struct {
		name   string
		prefix string
		target string
		want   string
	}{
		{
			name:   "when target is _any",
			prefix: "jobs.query",
			target: "_any",
			want:   "jobs.query._any",
		},
		{
			name:   "when target is _all",
			prefix: "jobs.modify",
			target: "_all",
			want:   "jobs.modify._all",
		},
		{
			name:   "when target is a hostname",
			prefix: "jobs.query",
			target: "server1",
			want:   "jobs.query.host.server1",
		},
		{
			name:   "when target is a dotted hostname",
			prefix: "jobs.query",
			target: "my-server.local",
			want:   "jobs.query.host.my_server_local",
		},
		{
			name:   "when target is a flat label",
			prefix: "jobs.query",
			target: "role:web",
			want:   "jobs.query.label.role.web",
		},
		{
			name:   "when target is a hierarchical label",
			prefix: "jobs.query",
			target: "group:web.dev.us-east",
			want:   "jobs.query.label.group.web.dev.us-east",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.BuildSubjectFromTarget(tt.prefix, tt.target)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestIsBroadcastTarget() {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{
			name:   "when target is _all",
			target: "_all",
			want:   true,
		},
		{
			name:   "when target is a label",
			target: "role:web",
			want:   true,
		},
		{
			name:   "when target is a hierarchical label",
			target: "group:web.dev",
			want:   true,
		},
		{
			name:   "when target is _any",
			target: "_any",
			want:   false,
		},
		{
			name:   "when target is a hostname",
			target: "server1",
			want:   false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.IsBroadcastTarget(tt.target)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestBuildLabelSubjects() {
	tests := []struct {
		name  string
		key   string
		value string
		want  []string
	}{
		{
			name:  "when value is flat",
			key:   "role",
			value: "web",
			want: []string{
				"jobs.*.label.role.web",
			},
		},
		{
			name:  "when value is hierarchical with two levels",
			key:   "group",
			value: "web.dev",
			want: []string{
				"jobs.*.label.group.web",
				"jobs.*.label.group.web.dev",
			},
		},
		{
			name:  "when value is hierarchical with three levels",
			key:   "group",
			value: "web.dev.us-east",
			want: []string{
				"jobs.*.label.group.web",
				"jobs.*.label.group.web.dev",
				"jobs.*.label.group.web.dev.us-east",
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.BuildLabelSubjects(tt.key, tt.value)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestInit() {
	tests := []struct {
		name             string
		namespace        string
		wantQueryPrefix  string
		wantModifyPrefix string
		wantBuildQuery   string
		wantSubscription string
		wantLabelSubject string
	}{
		{
			name:             "when namespace is empty",
			namespace:        "",
			wantQueryPrefix:  "jobs.query",
			wantModifyPrefix: "jobs.modify",
			wantBuildQuery:   "jobs.query._any",
			wantSubscription: "jobs.*._any",
			wantLabelSubject: "jobs.*.label.role.web",
		},
		{
			name:             "when namespace is set",
			namespace:        "osapi",
			wantQueryPrefix:  "osapi.jobs.query",
			wantModifyPrefix: "osapi.jobs.modify",
			wantBuildQuery:   "osapi.jobs.query._any",
			wantSubscription: "osapi.jobs.*._any",
			wantLabelSubject: "osapi.jobs.*.label.role.web",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			job.Init(tt.namespace)
			defer job.Init("")

			suite.Equal(tt.wantQueryPrefix, job.JobsQueryPrefix)
			suite.Equal(tt.wantModifyPrefix, job.JobsModifyPrefix)
			suite.Equal(tt.wantBuildQuery, job.BuildQuerySubject("_any"))
			subs := job.BuildAgentSubscriptionPattern("web-01", nil)
			suite.Contains(subs, tt.wantSubscription)
			labels := job.BuildLabelSubjects("role", "web")
			suite.Equal([]string{tt.wantLabelSubject}, labels)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestParseSubjectWithNamespace() {
	tests := []struct {
		name         string
		namespace    string
		subject      string
		wantPrefix   string
		wantHostname string
		wantErr      bool
	}{
		{
			name:         "when parsing namespaced query subject",
			namespace:    "osapi",
			subject:      "osapi.jobs.query.host.server-01",
			wantPrefix:   "osapi.jobs.query",
			wantHostname: "server-01",
		},
		{
			name:         "when parsing namespaced modify subject",
			namespace:    "osapi",
			subject:      "osapi.jobs.modify._any",
			wantPrefix:   "osapi.jobs.modify",
			wantHostname: "_any",
		},
		{
			name:         "when parsing namespaced label subject",
			namespace:    "osapi",
			subject:      "osapi.jobs.query.label.group.web.dev",
			wantPrefix:   "osapi.jobs.query",
			wantHostname: "group:web.dev",
		},
		{
			name:      "when parsing invalid namespaced subject with too few parts",
			namespace: "osapi",
			subject:   "osapi.jobs",
			wantErr:   true,
		},
		{
			name:      "when parsing namespaced subject without jobs token",
			namespace: "osapi",
			subject:   "osapi.other.query._any",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			job.Init(tt.namespace)
			defer job.Init("")

			gotPrefix, gotHostname, err := job.ParseSubject(tt.subject)

			if tt.wantErr {
				suite.Error(err)
				return
			}

			suite.NoError(err)
			suite.Equal(tt.wantPrefix, gotPrefix)
			suite.Equal(tt.wantHostname, gotHostname)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestApplyNamespaceToInfraName() {
	tests := []struct {
		name      string
		namespace string
		infraName string
		want      string
	}{
		{
			name:      "when namespace is empty",
			namespace: "",
			infraName: "JOBS",
			want:      "JOBS",
		},
		{
			name:      "when namespace is set",
			namespace: "osapi",
			infraName: "JOBS",
			want:      "osapi-JOBS",
		},
		{
			name:      "when namespace applied to KV bucket",
			namespace: "osapi",
			infraName: "job-queue",
			want:      "osapi-job-queue",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.ApplyNamespaceToInfraName(tt.namespace, tt.infraName)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestApplyNamespaceToSubjects() {
	tests := []struct {
		name      string
		namespace string
		subjects  string
		want      string
	}{
		{
			name:      "when namespace is empty",
			namespace: "",
			subjects:  "jobs.>",
			want:      "jobs.>",
		},
		{
			name:      "when namespace is set",
			namespace: "osapi",
			subjects:  "jobs.>",
			want:      "osapi.jobs.>",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.ApplyNamespaceToSubjects(tt.namespace, tt.subjects)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestCountExpectedAgents() {
	agents := []job.AgentInfo{
		{Hostname: "web-01", Labels: map[string]string{"group": "web.dev.us-east"}},
		{Hostname: "web-02", Labels: map[string]string{"group": "web.dev.us-west"}},
		{Hostname: "db-01", Labels: map[string]string{"group": "db.prod"}},
		{Hostname: "plain-01"},
	}

	tests := []struct {
		name   string
		agents []job.AgentInfo
		target string
		want   int
	}{
		{
			name:   "when target is _all returns full count",
			agents: agents,
			target: "_all",
			want:   4,
		},
		{
			name:   "when label exact match",
			agents: agents,
			target: "group:web.dev.us-east",
			want:   1,
		},
		{
			name:   "when label prefix match",
			agents: agents,
			target: "group:web",
			want:   2,
		},
		{
			name:   "when label prefix match at second level",
			agents: agents,
			target: "group:web.dev",
			want:   2,
		},
		{
			name:   "when no agents match label",
			agents: agents,
			target: "group:staging",
			want:   0,
		},
		{
			name:   "when label key does not exist on any agent",
			agents: agents,
			target: "region:us-east",
			want:   0,
		},
		{
			name:   "when agent list is empty",
			agents: []job.AgentInfo{},
			target: "_all",
			want:   0,
		},
		{
			name:   "when target is a hostname returns 0",
			agents: agents,
			target: "web-01",
			want:   0,
		},
		{
			name:   "when target is _any returns 0",
			agents: agents,
			target: "_any",
			want:   0,
		},
		{
			name: "when _all excludes cordoned agents",
			agents: []job.AgentInfo{
				{Hostname: "web-01"},
				{Hostname: "web-02", State: job.AgentStateCordoned},
				{Hostname: "web-03"},
			},
			target: "_all",
			want:   2,
		},
		{
			name: "when _all excludes draining agents",
			agents: []job.AgentInfo{
				{Hostname: "web-01"},
				{Hostname: "web-02", State: job.AgentStateDraining},
			},
			target: "_all",
			want:   1,
		},
		{
			name: "when _all excludes pending agents",
			agents: []job.AgentInfo{
				{Hostname: "web-01"},
				{Hostname: "web-02", State: job.AgentStatePending},
			},
			target: "_all",
			want:   1,
		},
		{
			name: "when label match excludes cordoned agents",
			agents: []job.AgentInfo{
				{
					Hostname: "web-01",
					Labels:   map[string]string{"group": "web.dev"},
					State:    job.AgentStateCordoned,
				},
				{Hostname: "web-02", Labels: map[string]string{"group": "web.dev"}},
			},
			target: "group:web",
			want:   1,
		},
		{
			name: "when label match excludes draining agents",
			agents: []job.AgentInfo{
				{
					Hostname: "web-01",
					Labels:   map[string]string{"group": "web.dev"},
					State:    job.AgentStateDraining,
				},
				{Hostname: "web-02", Labels: map[string]string{"group": "web.dev"}},
			},
			target: "group:web",
			want:   1,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.CountExpectedAgents(tt.agents, tt.target)
			suite.Equal(tt.want, got)
		})
	}
}

func (suite *SubjectsPublicTestSuite) TestExpectedAgentHostnames() {
	agents := []job.AgentInfo{
		{Hostname: "web-01", Labels: map[string]string{"group": "web.dev.us-east"}},
		{Hostname: "web-02", Labels: map[string]string{"group": "web.dev.us-west"}},
		{Hostname: "db-01", Labels: map[string]string{"group": "db.prod"}},
		{Hostname: "plain-01"},
	}

	tests := []struct {
		name   string
		agents []job.AgentInfo
		target string
		want   []string
	}{
		{
			name:   "when target is _all returns all hostnames",
			agents: agents,
			target: "_all",
			want:   []string{"web-01", "web-02", "db-01", "plain-01"},
		},
		{
			name:   "when label exact match returns matching hostnames",
			agents: agents,
			target: "group:web.dev.us-east",
			want:   []string{"web-01"},
		},
		{
			name:   "when label prefix match returns matching hostnames",
			agents: agents,
			target: "group:web",
			want:   []string{"web-01", "web-02"},
		},
		{
			name:   "when no agents match label returns nil",
			agents: agents,
			target: "group:staging",
			want:   nil,
		},
		{
			name:   "when agent list is empty returns nil",
			agents: []job.AgentInfo{},
			target: "_all",
			want:   nil,
		},
		{
			name:   "when target is a hostname returns nil",
			agents: agents,
			target: "web-01",
			want:   nil,
		},
		{
			name:   "when target is _any returns nil",
			agents: agents,
			target: "_any",
			want:   nil,
		},
		{
			name: "when _all excludes cordoned agents",
			agents: []job.AgentInfo{
				{Hostname: "web-01"},
				{Hostname: "web-02", State: job.AgentStateCordoned},
				{Hostname: "web-03"},
			},
			target: "_all",
			want:   []string{"web-01", "web-03"},
		},
		{
			name: "when _all excludes draining agents",
			agents: []job.AgentInfo{
				{Hostname: "web-01"},
				{Hostname: "web-02", State: job.AgentStateDraining},
			},
			target: "_all",
			want:   []string{"web-01"},
		},
		{
			name: "when _all excludes pending agents",
			agents: []job.AgentInfo{
				{Hostname: "web-01"},
				{Hostname: "web-02", State: job.AgentStatePending},
			},
			target: "_all",
			want:   []string{"web-01"},
		},
		{
			name: "when label match excludes cordoned agents",
			agents: []job.AgentInfo{
				{
					Hostname: "web-01",
					Labels:   map[string]string{"group": "web.dev"},
					State:    job.AgentStateCordoned,
				},
				{Hostname: "web-02", Labels: map[string]string{"group": "web.dev"}},
			},
			target: "group:web",
			want:   []string{"web-02"},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got := job.ExpectedAgentHostnames(tt.agents, tt.target)
			suite.Equal(tt.want, got)
		})
	}
}

func TestSubjectsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SubjectsPublicTestSuite))
}
