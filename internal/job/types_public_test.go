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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/provider/node/host"
	"github.com/osapi-io/osapi/internal/provider/node/load"
	"github.com/osapi-io/osapi/internal/provider/node/mem"
)

type TypesPublicTestSuite struct {
	suite.Suite
}

func (suite *TypesPublicTestSuite) SetupTest() {}

func (suite *TypesPublicTestSuite) TearDownTest() {}

func (suite *TypesPublicTestSuite) TestNetworkInterfaceJSONRoundTrip() {
	tests := []struct {
		name         string
		iface        job.NetworkInterface
		validateFunc func(job.NetworkInterface)
	}{
		{
			name: "when all fields are set",
			iface: job.NetworkInterface{
				Name:   "eth0",
				IPv4:   "192.168.1.100",
				IPv6:   "fe80::1",
				MAC:    "00:1a:2b:3c:4d:5e",
				Family: "dual",
			},
			validateFunc: func(result job.NetworkInterface) {
				suite.Equal("eth0", result.Name)
				suite.Equal("192.168.1.100", result.IPv4)
				suite.Equal("fe80::1", result.IPv6)
				suite.Equal("00:1a:2b:3c:4d:5e", result.MAC)
				suite.Equal("dual", result.Family)
			},
		},
		{
			name: "when only name is set",
			iface: job.NetworkInterface{
				Name: "lo",
			},
			validateFunc: func(result job.NetworkInterface) {
				suite.Equal("lo", result.Name)
				suite.Empty(result.IPv4)
				suite.Empty(result.IPv6)
				suite.Empty(result.MAC)
				suite.Empty(result.Family)
			},
		},
		{
			name: "when omitempty fields are absent in JSON",
			iface: job.NetworkInterface{
				Name: "wlan0",
				IPv4: "10.0.0.1",
			},
			validateFunc: func(result job.NetworkInterface) {
				suite.Equal("wlan0", result.Name)
				suite.Equal("10.0.0.1", result.IPv4)
				suite.Empty(result.MAC)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			data, err := json.Marshal(tt.iface)
			suite.NoError(err)

			var result job.NetworkInterface
			err = json.Unmarshal(data, &result)
			suite.NoError(err)

			tt.validateFunc(result)
		})
	}
}

func (suite *TypesPublicTestSuite) TestFactsRegistrationJSONRoundTrip() {
	tests := []struct {
		name         string
		reg          job.FactsRegistration
		validateFunc func(job.FactsRegistration)
	}{
		{
			name: "when all fields are set",
			reg: job.FactsRegistration{
				Architecture:  "x86_64",
				KernelVersion: "6.1.0-25-generic",
				CPUCount:      8,
				FQDN:          "web-01.example.com",
				ServiceMgr:    "systemd",
				PackageMgr:    "apt",
				Interfaces: []job.NetworkInterface{
					{
						Name: "eth0",
						IPv4: "192.168.1.100",
						MAC:  "00:1a:2b:3c:4d:5e",
					},
					{
						Name: "lo",
						IPv4: "127.0.0.1",
					},
				},
				Facts: map[string]any{
					"custom_key":    "custom_value",
					"numeric_fact":  float64(42),
					"bool_fact":     true,
					"nested_struct": map[string]any{"inner": "value"},
				},
			},
			validateFunc: func(result job.FactsRegistration) {
				suite.Equal("x86_64", result.Architecture)
				suite.Equal("6.1.0-25-generic", result.KernelVersion)
				suite.Equal(8, result.CPUCount)
				suite.Equal("web-01.example.com", result.FQDN)
				suite.Equal("systemd", result.ServiceMgr)
				suite.Equal("apt", result.PackageMgr)
				suite.Len(result.Interfaces, 2)
				suite.Equal("eth0", result.Interfaces[0].Name)
				suite.Equal("192.168.1.100", result.Interfaces[0].IPv4)
				suite.Equal("00:1a:2b:3c:4d:5e", result.Interfaces[0].MAC)
				suite.Equal("lo", result.Interfaces[1].Name)
				suite.Equal("127.0.0.1", result.Interfaces[1].IPv4)
				suite.Empty(result.Interfaces[1].MAC)
				suite.Equal("custom_value", result.Facts["custom_key"])
				suite.Equal(float64(42), result.Facts["numeric_fact"])
				suite.Equal(true, result.Facts["bool_fact"])
				nested, ok := result.Facts["nested_struct"].(map[string]any)
				suite.True(ok)
				suite.Equal("value", nested["inner"])
			},
		},
		{
			name: "when only required fields are set",
			reg: job.FactsRegistration{
				Architecture: "aarch64",
				CPUCount:     4,
			},
			validateFunc: func(result job.FactsRegistration) {
				suite.Equal("aarch64", result.Architecture)
				suite.Equal(4, result.CPUCount)
				suite.Empty(result.KernelVersion)
				suite.Empty(result.FQDN)
				suite.Empty(result.ServiceMgr)
				suite.Empty(result.PackageMgr)
				suite.Nil(result.Interfaces)
				suite.Nil(result.Facts)
			},
		},
		{
			name: "when facts map is empty it is omitted by omitempty",
			reg: job.FactsRegistration{
				Architecture: "x86_64",
				Facts:        map[string]any{},
			},
			validateFunc: func(result job.FactsRegistration) {
				suite.Equal("x86_64", result.Architecture)
				// Go 1.25 omitempty omits empty maps, so after
				// round-trip the field deserializes as nil.
				suite.Nil(result.Facts)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			data, err := json.Marshal(tt.reg)
			suite.NoError(err)

			var result job.FactsRegistration
			err = json.Unmarshal(data, &result)
			suite.NoError(err)

			tt.validateFunc(result)
		})
	}
}

func (suite *TypesPublicTestSuite) TestAgentInfoFactsFieldsJSONRoundTrip() {
	tests := []struct {
		name         string
		info         job.AgentInfo
		validateFunc func(job.AgentInfo)
	}{
		{
			name: "when facts fields are populated",
			info: job.AgentInfo{
				Hostname:     "web-01",
				Labels:       map[string]string{"group": "web"},
				RegisteredAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				StartedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				OSInfo: &host.Result{
					Distribution: "Ubuntu",
					Version:      "22.04",
				},
				Uptime:        time.Duration(3600) * time.Second,
				LoadAverages:  &load.Result{Load1: 0.5, Load5: 0.3, Load15: 0.1},
				MemoryStats:   &mem.Result{Total: 1024, Free: 512},
				AgentVersion:  "1.0.0",
				Architecture:  "x86_64",
				KernelVersion: "6.1.0-25-generic",
				CPUCount:      8,
				FQDN:          "web-01.example.com",
				ServiceMgr:    "systemd",
				PackageMgr:    "apt",
				Interfaces: []job.NetworkInterface{
					{
						Name:   "eth0",
						IPv4:   "10.0.0.1",
						IPv6:   "fe80::1",
						MAC:    "aa:bb:cc:dd:ee:ff",
						Family: "dual",
					},
				},
				Facts: map[string]any{
					"custom": "value",
				},
			},
			validateFunc: func(result job.AgentInfo) {
				suite.Equal("web-01", result.Hostname)
				suite.Equal("1.0.0", result.AgentVersion)
				suite.Equal("x86_64", result.Architecture)
				suite.Equal("6.1.0-25-generic", result.KernelVersion)
				suite.Equal(8, result.CPUCount)
				suite.Equal("web-01.example.com", result.FQDN)
				suite.Equal("systemd", result.ServiceMgr)
				suite.Equal("apt", result.PackageMgr)
				suite.Len(result.Interfaces, 1)
				suite.Equal("eth0", result.Interfaces[0].Name)
				suite.Equal("10.0.0.1", result.Interfaces[0].IPv4)
				suite.Equal("fe80::1", result.Interfaces[0].IPv6)
				suite.Equal("aa:bb:cc:dd:ee:ff", result.Interfaces[0].MAC)
				suite.Equal("dual", result.Interfaces[0].Family)
				suite.Equal("value", result.Facts["custom"])
			},
		},
		{
			name: "when facts fields are empty",
			info: job.AgentInfo{
				Hostname:     "db-01",
				AgentVersion: "1.0.0",
			},
			validateFunc: func(result job.AgentInfo) {
				suite.Equal("db-01", result.Hostname)
				suite.Equal("1.0.0", result.AgentVersion)
				suite.Empty(result.Architecture)
				suite.Empty(result.KernelVersion)
				suite.Zero(result.CPUCount)
				suite.Empty(result.FQDN)
				suite.Empty(result.ServiceMgr)
				suite.Empty(result.PackageMgr)
				suite.Nil(result.Interfaces)
				suite.Nil(result.Facts)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			data, err := json.Marshal(tt.info)
			suite.NoError(err)

			var result job.AgentInfo
			err = json.Unmarshal(data, &result)
			suite.NoError(err)

			tt.validateFunc(result)
		})
	}
}

func TestTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TypesPublicTestSuite))
}
