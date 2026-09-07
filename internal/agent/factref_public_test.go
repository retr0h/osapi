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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/job"
)

type FactRefPublicTestSuite struct {
	suite.Suite
}

func (s *FactRefPublicTestSuite) TestResolveFacts() {
	tests := []struct {
		name         string
		params       map[string]any
		facts        *job.FactsRegistration
		hostname     string
		wantErr      bool
		errContains  string
		validateFunc func(result map[string]any)
	}{
		{
			name: "when simple interface.primary substitution",
			params: map[string]any{
				"interface_name": "@fact.interface.primary",
			},
			facts: &job.FactsRegistration{
				PrimaryInterface: "eth0",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("eth0", result["interface_name"])
			},
		},
		{
			name: "when hostname substitution",
			params: map[string]any{
				"target": "@fact.hostname",
			},
			facts:    &job.FactsRegistration{},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("web-01", result["target"])
			},
		},
		{
			name: "when arch substitution",
			params: map[string]any{
				"arch": "@fact.arch",
			},
			facts: &job.FactsRegistration{
				Architecture: "x86_64",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("x86_64", result["arch"])
			},
		},
		{
			name: "when kernel substitution",
			params: map[string]any{
				"kernel": "@fact.kernel",
			},
			facts: &job.FactsRegistration{
				KernelVersion: "6.1.0",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("6.1.0", result["kernel"])
			},
		},
		{
			name: "when fqdn substitution",
			params: map[string]any{
				"fqdn": "@fact.fqdn",
			},
			facts: &job.FactsRegistration{
				FQDN: "web-01.example.com",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("web-01.example.com", result["fqdn"])
			},
		},
		{
			name: "when containerized is true",
			params: map[string]any{
				"in_container": "@fact.containerized",
			},
			facts: &job.FactsRegistration{
				Containerized: true,
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("true", result["in_container"])
			},
		},
		{
			name: "when containerized is false",
			params: map[string]any{
				"in_container": "@fact.containerized",
			},
			facts: &job.FactsRegistration{
				Containerized: false,
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("false", result["in_container"])
			},
		},
		{
			name:        "when os substitution returns error",
			params:      map[string]any{"os": "@fact.os"},
			facts:       &job.FactsRegistration{},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "os fact not available",
		},
		{
			name: "when custom fact substitution",
			params: map[string]any{
				"env": "@fact.custom.environment",
			},
			facts: &job.FactsRegistration{
				Facts: map[string]any{
					"environment": "production",
				},
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("production", result["env"])
			},
		},
		{
			name: "when multiple references in one string",
			params: map[string]any{
				"desc": "@fact.interface.primary on @fact.hostname",
			},
			facts: &job.FactsRegistration{
				PrimaryInterface: "eth0",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("eth0 on web-01", result["desc"])
			},
		},
		{
			name: "when nested map values",
			params: map[string]any{
				"config": map[string]any{
					"iface": "@fact.interface.primary",
					"host":  "@fact.hostname",
				},
			},
			facts: &job.FactsRegistration{
				PrimaryInterface: "eth0",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				config := result["config"].(map[string]any)
				s.Equal("eth0", config["iface"])
				s.Equal("web-01", config["host"])
			},
		},
		{
			name: "when no fact references params unchanged",
			params: map[string]any{
				"address":        "192.168.1.1",
				"count":          4,
				"interface_name": "eth0",
			},
			facts:    &job.FactsRegistration{},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal("192.168.1.1", result["address"])
				s.Equal(4, result["count"])
				s.Equal("eth0", result["interface_name"])
			},
		},
		{
			name:     "when nil params returns nil",
			params:   nil,
			facts:    &job.FactsRegistration{},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Nil(result)
			},
		},
		{
			name: "when nil facts returns error for any reference",
			params: map[string]any{
				"iface": "@fact.interface.primary",
			},
			facts:       nil,
			hostname:    "web-01",
			wantErr:     true,
			errContains: "facts not available",
		},
		{
			name: "when unknown fact reference returns error",
			params: map[string]any{
				"value": "@fact.nonexistent",
			},
			facts:       &job.FactsRegistration{},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "unknown fact key",
		},
		{
			name: "when custom fact not found returns error",
			params: map[string]any{
				"val": "@fact.custom.missing",
			},
			facts:       &job.FactsRegistration{},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "custom fact \"missing\" not found",
		},
		{
			name: "when custom fact key exists but facts map is nil",
			params: map[string]any{
				"val": "@fact.custom.key",
			},
			facts:       &job.FactsRegistration{Facts: nil},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "custom fact \"key\" not found",
		},
		{
			name: "when custom fact key missing from non-nil facts map",
			params: map[string]any{
				"val": "@fact.custom.missing",
			},
			facts: &job.FactsRegistration{
				Facts: map[string]any{"other": "value"},
			},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "custom fact \"missing\" not found",
		},
		{
			name: "when primary interface not set returns error",
			params: map[string]any{
				"iface": "@fact.interface.primary",
			},
			facts:       &job.FactsRegistration{PrimaryInterface: ""},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "primary interface not set",
		},
		{
			name: "when hostname not set returns error",
			params: map[string]any{
				"host": "@fact.hostname",
			},
			facts:       &job.FactsRegistration{},
			hostname:    "",
			wantErr:     true,
			errContains: "hostname not set",
		},
		{
			name: "when arch not set returns error",
			params: map[string]any{
				"arch": "@fact.arch",
			},
			facts:       &job.FactsRegistration{Architecture: ""},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "architecture not set",
		},
		{
			name: "when kernel not set returns error",
			params: map[string]any{
				"kernel": "@fact.kernel",
			},
			facts:       &job.FactsRegistration{KernelVersion: ""},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "kernel version not set",
		},
		{
			name: "when fqdn not set returns error",
			params: map[string]any{
				"fqdn": "@fact.fqdn",
			},
			facts:       &job.FactsRegistration{FQDN: ""},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "fqdn not set",
		},
		{
			name: "when non-string values pass through unchanged",
			params: map[string]any{
				"count":   42,
				"enabled": true,
				"ratio":   3.14,
			},
			facts:    &job.FactsRegistration{},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				s.Equal(42, result["count"])
				s.Equal(true, result["enabled"])
				s.Equal(3.14, result["ratio"])
			},
		},
		{
			name: "when slice values are resolved",
			params: map[string]any{
				"args": []any{"addr", "show", "dev", "@fact.interface.primary"},
			},
			facts: &job.FactsRegistration{
				PrimaryInterface: "eth0",
			},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				args := result["args"].([]any)
				s.Equal("addr", args[0])
				s.Equal("show", args[1])
				s.Equal("dev", args[2])
				s.Equal("eth0", args[3])
			},
		},
		{
			name: "when slice error propagates",
			params: map[string]any{
				"args": []any{"ok", "@fact.nonexistent"},
			},
			facts:       &job.FactsRegistration{},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "unknown fact key",
		},
		{
			name: "when nested slice in map is resolved",
			params: map[string]any{
				"config": map[string]any{
					"hosts": []any{"@fact.hostname", "other"},
				},
			},
			facts:    &job.FactsRegistration{},
			hostname: "web-01",
			validateFunc: func(result map[string]any) {
				config := result["config"].(map[string]any)
				hosts := config["hosts"].([]any)
				s.Equal("web-01", hosts[0])
				s.Equal("other", hosts[1])
			},
		},
		{
			name: "when nested map error propagates",
			params: map[string]any{
				"config": map[string]any{
					"bad": "@fact.nonexistent",
				},
			},
			facts:       &job.FactsRegistration{},
			hostname:    "web-01",
			wantErr:     true,
			errContains: "unknown fact key",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			result, err := agent.ResolveFacts(tc.params, tc.facts, tc.hostname)

			if tc.wantErr {
				s.Error(err)
				s.Contains(err.Error(), tc.errContains)
			} else {
				s.NoError(err)
				if tc.validateFunc != nil {
					tc.validateFunc(result)
				}
			}
		})
	}
}

func TestFactRefPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FactRefPublicTestSuite))
}
