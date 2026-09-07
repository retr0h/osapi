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

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type HealthTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *HealthTypesPublicTestSuite) TestSystemStatusFromGen() {
	tests := []struct {
		name               string
		input              *gen.StatusResponse
		serviceUnavailable bool
		validateFunc       func(client.SystemStatus)
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.StatusResponse {
				errMsg := "connection timeout"
				labels := "group=web"

				return &gen.StatusResponse{
					Status:  "degraded",
					Version: "1.2.3",
					Uptime:  "5d 3h",
					Components: map[string]gen.ComponentHealth{
						"nats": {
							Status: "healthy",
						},
						"store": {
							Status: "unhealthy",
							Error:  &errMsg,
						},
						"controller.api": {
							Status:  "ok",
							Address: func() *string { s := "http://0.0.0.0:8080"; return &s }(),
						},
					},
					Nats: &gen.NATSInfo{
						Url:     "nats://localhost:4222",
						Version: "2.10.0",
					},
					Agents: &gen.AgentStats{
						Total: 3,
						Ready: 2,
						Agents: &[]gen.AgentDetail{
							{
								Hostname:   "web-01",
								Labels:     &labels,
								Registered: "5m ago",
							},
							{
								Hostname:   "web-02",
								Registered: "10m ago",
							},
						},
					},
					Jobs: &gen.JobStats{
						Total:       100,
						Completed:   80,
						Failed:      5,
						Processing:  10,
						Unprocessed: 3,
						Dlq:         2,
					},
					Consumers: &gen.ConsumerStats{
						Total: 2,
						Consumers: &[]gen.ConsumerDetail{
							{
								Name:        "jobs-agent",
								Pending:     5,
								AckPending:  2,
								Redelivered: 1,
							},
						},
					},
					Streams: &[]gen.StreamInfo{
						{
							Name:      "JOBS",
							Messages:  500,
							Bytes:     1048576,
							Consumers: 2,
						},
					},
					KvBuckets: &[]gen.KVBucketInfo{
						{
							Name:  "job-queue",
							Keys:  50,
							Bytes: 524288,
						},
						{
							Name:  "audit-log",
							Keys:  200,
							Bytes: 2097152,
						},
					},
					ObjectStores: &[]gen.ObjectStoreInfo{
						{
							Name: "file-objects",
							Size: 5242880,
						},
					},
					Registry: func() *[]gen.ComponentEntry {
						agentType := "agent"
						agentHostname := "web-01"
						agentStatus := "Ready"
						agentAge := "5m"
						var agentCPU float32 = 12.5
						var agentMem int64 = 104857600
						agentConditions := []string{"MemoryPressure"}

						apiType := "api"
						apiHostname := "api-server-01"
						apiStatus := "Ready"
						apiAge := "1h"
						var apiCPU float32 = 2.1
						var apiMem int64 = 52428800

						entries := []gen.ComponentEntry{
							{
								Type:       &agentType,
								Hostname:   &agentHostname,
								Status:     &agentStatus,
								Age:        &agentAge,
								CpuPercent: &agentCPU,
								MemBytes:   &agentMem,
								Conditions: &agentConditions,
							},
							{
								Type:       &apiType,
								Hostname:   &apiHostname,
								Status:     &apiStatus,
								Age:        &apiAge,
								CpuPercent: &apiCPU,
								MemBytes:   &apiMem,
							},
						}
						return &entries
					}(),
				}
			}(),
			serviceUnavailable: false,
			validateFunc: func(s client.SystemStatus) {
				suite.Equal("degraded", s.Status)
				suite.Equal("1.2.3", s.Version)
				suite.Equal("5d 3h", s.Uptime)
				suite.False(s.ServiceUnavailable)

				suite.Require().Len(s.Components, 3)
				suite.Equal("healthy", s.Components["nats"].Status)
				suite.Empty(s.Components["nats"].Error)
				suite.Equal("unhealthy", s.Components["store"].Status)
				suite.Equal("connection timeout", s.Components["store"].Error)

				suite.Equal("ok", s.Components["controller.api"].Status)
				suite.Equal("http://0.0.0.0:8080", s.Components["controller.api"].Address)

				suite.Require().NotNil(s.NATS)
				suite.Equal("nats://localhost:4222", s.NATS.URL)
				suite.Equal("2.10.0", s.NATS.Version)

				suite.Require().NotNil(s.Agents)
				suite.Equal(3, s.Agents.Total)
				suite.Equal(2, s.Agents.Ready)
				suite.Require().Len(s.Agents.Agents, 2)
				suite.Equal("web-01", s.Agents.Agents[0].Hostname)
				suite.Equal("group=web", s.Agents.Agents[0].Labels)
				suite.Equal("5m ago", s.Agents.Agents[0].Registered)
				suite.Equal("web-02", s.Agents.Agents[1].Hostname)
				suite.Empty(s.Agents.Agents[1].Labels)
				suite.Equal("10m ago", s.Agents.Agents[1].Registered)

				suite.Require().NotNil(s.Jobs)
				suite.Equal(100, s.Jobs.Total)
				suite.Equal(80, s.Jobs.Completed)
				suite.Equal(5, s.Jobs.Failed)
				suite.Equal(10, s.Jobs.Processing)
				suite.Equal(3, s.Jobs.Unprocessed)
				suite.Equal(2, s.Jobs.Dlq)

				suite.Require().NotNil(s.Consumers)
				suite.Equal(2, s.Consumers.Total)
				suite.Require().Len(s.Consumers.Consumers, 1)
				suite.Equal("jobs-agent", s.Consumers.Consumers[0].Name)
				suite.Equal(5, s.Consumers.Consumers[0].Pending)
				suite.Equal(2, s.Consumers.Consumers[0].AckPending)
				suite.Equal(1, s.Consumers.Consumers[0].Redelivered)

				suite.Require().Len(s.Streams, 1)
				suite.Equal("JOBS", s.Streams[0].Name)
				suite.Equal(500, s.Streams[0].Messages)
				suite.Equal(1048576, s.Streams[0].Bytes)
				suite.Equal(2, s.Streams[0].Consumers)

				suite.Require().Len(s.KVBuckets, 2)
				suite.Equal("job-queue", s.KVBuckets[0].Name)
				suite.Equal(50, s.KVBuckets[0].Keys)
				suite.Equal(524288, s.KVBuckets[0].Bytes)
				suite.Equal("audit-log", s.KVBuckets[1].Name)
				suite.Equal(200, s.KVBuckets[1].Keys)
				suite.Equal(2097152, s.KVBuckets[1].Bytes)

				suite.Require().Len(s.ObjectStores, 1)
				suite.Equal("file-objects", s.ObjectStores[0].Name)
				suite.Equal(5242880, s.ObjectStores[0].Size)

				suite.Require().Len(s.Registry, 2)
				suite.Equal("agent", s.Registry[0].Type)
				suite.Equal("web-01", s.Registry[0].Hostname)
				suite.Equal("Ready", s.Registry[0].Status)
				suite.Equal("5m", s.Registry[0].Age)
				suite.InDelta(12.5, s.Registry[0].CPUPercent, 0.001)
				suite.Equal(int64(104857600), s.Registry[0].MemBytes)
				suite.Equal([]string{"MemoryPressure"}, s.Registry[0].Conditions)
				suite.Equal("api", s.Registry[1].Type)
				suite.Equal("api-server-01", s.Registry[1].Hostname)
				suite.Equal("Ready", s.Registry[1].Status)
				suite.Equal("1h", s.Registry[1].Age)
				suite.InDelta(2.1, s.Registry[1].CPUPercent, 0.001)
				suite.Equal(int64(52428800), s.Registry[1].MemBytes)
				suite.Nil(s.Registry[1].Conditions)
			},
		},
		{
			name: "when only required fields are set",
			input: &gen.StatusResponse{
				Status:     "ok",
				Version:    "1.0.0",
				Uptime:     "1h",
				Components: map[string]gen.ComponentHealth{},
			},
			serviceUnavailable: false,
			validateFunc: func(s client.SystemStatus) {
				suite.Equal("ok", s.Status)
				suite.Equal("1.0.0", s.Version)
				suite.Equal("1h", s.Uptime)
				suite.False(s.ServiceUnavailable)
				suite.Empty(s.Components)
				suite.Nil(s.NATS)
				suite.Nil(s.Agents)
				suite.Nil(s.Jobs)
				suite.Nil(s.Consumers)
				suite.Nil(s.Streams)
				suite.Nil(s.KVBuckets)
				suite.Nil(s.ObjectStores)
				suite.Nil(s.Registry)
			},
		},
		{
			name: "when registry entry has all nil optional fields",
			input: &gen.StatusResponse{
				Status:     "ok",
				Version:    "1.0.0",
				Uptime:     "1h",
				Components: map[string]gen.ComponentHealth{},
				Registry: &[]gen.ComponentEntry{
					{
						// all optional pointer fields left nil
					},
				},
			},
			serviceUnavailable: false,
			validateFunc: func(s client.SystemStatus) {
				suite.Require().Len(s.Registry, 1)
				suite.Empty(s.Registry[0].Type)
				suite.Empty(s.Registry[0].Hostname)
				suite.Empty(s.Registry[0].Status)
				suite.Empty(s.Registry[0].Age)
				suite.Equal(float64(0), s.Registry[0].CPUPercent)
				suite.Equal(int64(0), s.Registry[0].MemBytes)
				suite.Nil(s.Registry[0].Conditions)
			},
		},
		{
			name: "when service unavailable is true",
			input: &gen.StatusResponse{
				Status:     "degraded",
				Version:    "1.0.0",
				Uptime:     "30m",
				Components: map[string]gen.ComponentHealth{},
			},
			serviceUnavailable: true,
			validateFunc: func(s client.SystemStatus) {
				suite.Equal("degraded", s.Status)
				suite.True(s.ServiceUnavailable)
			},
		},
		{
			name: "when agents has nil agents list",
			input: &gen.StatusResponse{
				Status:  "ok",
				Version: "1.0.0",
				Uptime:  "1h",
				Components: map[string]gen.ComponentHealth{
					"nats": {Status: "healthy"},
				},
				Agents: &gen.AgentStats{
					Total: 0,
					Ready: 0,
				},
			},
			serviceUnavailable: false,
			validateFunc: func(s client.SystemStatus) {
				suite.Require().NotNil(s.Agents)
				suite.Equal(0, s.Agents.Total)
				suite.Equal(0, s.Agents.Ready)
				suite.Nil(s.Agents.Agents)
			},
		},
		{
			name: "when consumers has nil consumers list",
			input: &gen.StatusResponse{
				Status:     "ok",
				Version:    "1.0.0",
				Uptime:     "1h",
				Components: map[string]gen.ComponentHealth{},
				Consumers: &gen.ConsumerStats{
					Total: 0,
				},
			},
			serviceUnavailable: false,
			validateFunc: func(s client.SystemStatus) {
				suite.Require().NotNil(s.Consumers)
				suite.Equal(0, s.Consumers.Total)
				suite.Nil(s.Consumers.Consumers)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportSystemStatusFromGen(tc.input, tc.serviceUnavailable)
			tc.validateFunc(result)
		})
	}
}

func TestHealthTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HealthTypesPublicTestSuite))
}
