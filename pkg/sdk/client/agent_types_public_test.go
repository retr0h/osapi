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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type AgentTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *AgentTypesPublicTestSuite) TestAgentFromGen() {
	now := time.Now().UTC().Truncate(time.Second)
	startedAt := now.Add(-1 * time.Hour)

	tests := []struct {
		name         string
		input        *gen.AgentInfo
		validateFunc func(client.Agent)
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.AgentInfo {
				labels := map[string]string{"group": "web", "env": "prod"}
				arch := "amd64"
				cpuCount := 8
				fqdn := "web-01.example.com"
				kernelVersion := "5.15.0-generic"
				packageMgr := "apt"
				serviceMgr := "systemd"
				primaryIface := "eth0"
				routeMask := "255.255.255.0"
				routeFlags := "UG"
				routeMetric := 100
				uptime := "5d 3h 22m"
				family := gen.NetworkInterfaceResponseFamily("inet")
				ipv4 := "192.168.1.10"
				ipv6 := "fe80::1"
				mac := "00:11:22:33:44:55"
				facts := map[string]interface{}{"custom_key": "custom_value"}
				state := gen.AgentInfoStateReady
				reason := "load avg 0.50 < 4.00"
				condTime := now.Add(-30 * time.Minute)
				hostname := "web-01"
				message := "agent started"
				errMsg := "connection lost"

				machineID := "abc123-machine-id"
				fingerprint := "SHA256:abc123def456"

				return &gen.AgentInfo{
					Hostname:      "web-01",
					MachineId:     &machineID,
					Fingerprint:   &fingerprint,
					Status:        gen.AgentInfoStatus("Ready"),
					Labels:        &labels,
					Architecture:  &arch,
					CpuCount:      &cpuCount,
					Fqdn:          &fqdn,
					KernelVersion: &kernelVersion,
					PackageMgr:    &packageMgr,
					ServiceMgr:    &serviceMgr,
					LoadAverage: &gen.LoadAverageResponse{
						N1min:  0.5,
						N5min:  1.2,
						N15min: 0.8,
					},
					Memory: &gen.MemoryResponse{
						Total: 8589934592,
						Used:  4294967296,
						Free:  4294967296,
					},
					OsInfo: &gen.OSInfoResponse{
						Distribution: "Ubuntu",
						Version:      "22.04",
					},
					PrimaryInterface: &primaryIface,
					Interfaces: &[]gen.NetworkInterfaceResponse{
						{
							Name:   "eth0",
							Family: &family,
							Ipv4:   &ipv4,
							Ipv6:   &ipv6,
							Mac:    &mac,
						},
					},
					Routes: &[]gen.RouteResponse{
						{
							Destination: "0.0.0.0",
							Gateway:     "192.168.1.1",
							Interface:   "eth0",
							Mask:        &routeMask,
							Flags:       &routeFlags,
							Metric:      &routeMetric,
						},
					},
					Uptime:       &uptime,
					StartedAt:    &startedAt,
					RegisteredAt: &now,
					Facts:        &facts,
					State:        &state,
					Conditions: &[]gen.NodeCondition{
						{
							Type:               gen.HighLoad,
							Status:             false,
							Reason:             &reason,
							LastTransitionTime: condTime,
						},
					},
					Timeline: &[]gen.TimelineEvent{
						{
							Timestamp: startedAt,
							Event:     "AgentStarted",
							Hostname:  &hostname,
							Message:   &message,
						},
						{
							Timestamp: now,
							Event:     "AgentFailed",
							Hostname:  &hostname,
							Error:     &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(a client.Agent) {
				suite.Equal("web-01", a.Hostname)
				suite.Equal("abc123-machine-id", a.MachineID)
				suite.Equal("SHA256:abc123def456", a.Fingerprint)
				suite.Equal("Ready", a.Status)
				suite.Equal("Ready", a.State)
				suite.Equal(map[string]string{"group": "web", "env": "prod"}, a.Labels)
				suite.Equal("amd64", a.Architecture)
				suite.Equal(8, a.CPUCount)
				suite.Equal("web-01.example.com", a.Fqdn)
				suite.Equal("5.15.0-generic", a.KernelVersion)
				suite.Equal("apt", a.PackageMgr)
				suite.Equal("systemd", a.ServiceMgr)

				suite.Require().NotNil(a.LoadAverage)
				suite.InDelta(0.5, float64(a.LoadAverage.OneMin), 0.001)
				suite.InDelta(1.2, float64(a.LoadAverage.FiveMin), 0.001)
				suite.InDelta(0.8, float64(a.LoadAverage.FifteenMin), 0.001)

				suite.Require().NotNil(a.Memory)
				suite.Equal(8589934592, a.Memory.Total)
				suite.Equal(4294967296, a.Memory.Used)
				suite.Equal(4294967296, a.Memory.Free)

				suite.Require().NotNil(a.OSInfo)
				suite.Equal("Ubuntu", a.OSInfo.Distribution)
				suite.Equal("22.04", a.OSInfo.Version)

				suite.Equal("eth0", a.PrimaryInterface)

				suite.Require().Len(a.Interfaces, 1)
				suite.Equal("eth0", a.Interfaces[0].Name)
				suite.Equal("inet", a.Interfaces[0].Family)
				suite.Equal("192.168.1.10", a.Interfaces[0].IPv4)
				suite.Equal("fe80::1", a.Interfaces[0].IPv6)
				suite.Equal("00:11:22:33:44:55", a.Interfaces[0].MAC)

				suite.Require().Len(a.Routes, 1)
				suite.Equal("0.0.0.0", a.Routes[0].Destination)
				suite.Equal("192.168.1.1", a.Routes[0].Gateway)
				suite.Equal("eth0", a.Routes[0].Interface)
				suite.Equal("255.255.255.0", a.Routes[0].Mask)
				suite.Equal("UG", a.Routes[0].Flags)
				suite.Equal(100, a.Routes[0].Metric)

				suite.Equal("5d 3h 22m", a.Uptime)
				suite.Equal(startedAt, a.StartedAt)
				suite.Equal(now, a.RegisteredAt)
				suite.Equal(map[string]any{"custom_key": "custom_value"}, a.Facts)

				suite.Require().Len(a.Conditions, 1)
				suite.Equal("HighLoad", a.Conditions[0].Type)
				suite.False(a.Conditions[0].Status)
				suite.Equal("load avg 0.50 < 4.00", a.Conditions[0].Reason)
				suite.Equal(
					now.Add(-30*time.Minute),
					a.Conditions[0].LastTransitionTime,
				)

				suite.Require().Len(a.Timeline, 2)
				suite.Equal("AgentStarted", a.Timeline[0].Event)
				suite.Equal(startedAt.Format(time.RFC3339), a.Timeline[0].Timestamp)
				suite.Equal("web-01", a.Timeline[0].Hostname)
				suite.Equal("agent started", a.Timeline[0].Message)
				suite.Empty(a.Timeline[0].Error)

				suite.Equal("AgentFailed", a.Timeline[1].Event)
				suite.Equal(now.Format(time.RFC3339), a.Timeline[1].Timestamp)
				suite.Equal("web-01", a.Timeline[1].Hostname)
				suite.Empty(a.Timeline[1].Message)
				suite.Equal("connection lost", a.Timeline[1].Error)
			},
		},
		{
			name: "when only required fields are set",
			input: &gen.AgentInfo{
				Hostname: "minimal-host",
				Status:   gen.AgentInfoStatus("Ready"),
			},
			validateFunc: func(a client.Agent) {
				suite.Equal("minimal-host", a.Hostname)
				suite.Empty(a.MachineID)
				suite.Equal("Ready", a.Status)
				suite.Empty(a.State)
				suite.Nil(a.Labels)
				suite.Empty(a.Architecture)
				suite.Zero(a.CPUCount)
				suite.Empty(a.Fqdn)
				suite.Empty(a.KernelVersion)
				suite.Empty(a.PackageMgr)
				suite.Empty(a.ServiceMgr)
				suite.Nil(a.LoadAverage)
				suite.Nil(a.Memory)
				suite.Nil(a.OSInfo)
				suite.Empty(a.PrimaryInterface)
				suite.Nil(a.Interfaces)
				suite.Nil(a.Routes)
				suite.Nil(a.Conditions)
				suite.Nil(a.Timeline)
				suite.Empty(a.Uptime)
				suite.True(a.StartedAt.IsZero())
				suite.True(a.RegisteredAt.IsZero())
				suite.Nil(a.Facts)
			},
		},
		{
			name: "when interfaces list is empty",
			input: func() *gen.AgentInfo {
				ifaces := []gen.NetworkInterfaceResponse{}

				return &gen.AgentInfo{
					Hostname:   "no-ifaces",
					Status:     gen.AgentInfoStatus("Ready"),
					Interfaces: &ifaces,
				}
			}(),
			validateFunc: func(a client.Agent) {
				suite.Equal("no-ifaces", a.Hostname)
				suite.NotNil(a.Interfaces)
				suite.Empty(a.Interfaces)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportAgentFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *AgentTypesPublicTestSuite) TestAgentListFromGen() {
	tests := []struct {
		name         string
		input        *gen.ListAgentsResponse
		validateFunc func(client.AgentList)
	}{
		{
			name: "when list contains agents",
			input: &gen.ListAgentsResponse{
				Agents: []gen.AgentInfo{
					{
						Hostname: "web-01",
						Status:   gen.AgentInfoStatus("Ready"),
					},
					{
						Hostname: "web-02",
						Status:   gen.AgentInfoStatus("Ready"),
					},
				},
				Total: 2,
			},
			validateFunc: func(al client.AgentList) {
				suite.Equal(2, al.Total)
				suite.Require().Len(al.Agents, 2)
				suite.Equal("web-01", al.Agents[0].Hostname)
				suite.Equal("web-02", al.Agents[1].Hostname)
			},
		},
		{
			name: "when list is empty",
			input: &gen.ListAgentsResponse{
				Agents: []gen.AgentInfo{},
				Total:  0,
			},
			validateFunc: func(al client.AgentList) {
				suite.Equal(0, al.Total)
				suite.Empty(al.Agents)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportAgentListFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestAgentTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AgentTypesPublicTestSuite))
}
