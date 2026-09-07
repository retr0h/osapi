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

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type ProcessTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *ProcessTypesPublicTestSuite) TestProcessInfoCollectionFromList() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.ProcessCollectionResponse
		validateFunc func(client.Collection[client.ProcessInfoResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.ProcessCollectionResponse {
				pid := 1234
				name := "nginx"
				user := "www-data"
				state := "running"
				cpu := 2.5
				mem := float32(1.2)
				rss := int64(12345678)
				cmd := "nginx: worker process"
				startTime := "2026-01-01T00:00:00Z"
				return &gen.ProcessCollectionResponse{
					JobId: &testUUID,
					Results: []gen.ProcessEntry{
						{
							Hostname: "web-01",
							Status:   gen.ProcessEntryStatusOk,
							Processes: &[]gen.ProcessInfo{
								{
									Pid:        &pid,
									Name:       &name,
									User:       &user,
									State:      &state,
									CpuPercent: &cpu,
									MemPercent: &mem,
									MemRss:     &rss,
									Command:    &cmd,
									StartTime:  &startTime,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Processes, 1)

				p := r.Processes[0]
				suite.Equal(1234, p.PID)
				suite.Equal("nginx", p.Name)
				suite.Equal("www-data", p.User)
				suite.Equal("running", p.State)
				suite.InDelta(2.5, p.CPUPercent, 0.001)
				suite.InDelta(1.2, p.MemPercent, 0.001)
				suite.Equal(int64(12345678), p.MemRSS)
				suite.Equal("nginx: worker process", p.Command)
				suite.Equal("2026-01-01T00:00:00Z", p.StartTime)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.ProcessCollectionResponse {
				errMsg := "permission denied"
				return &gen.ProcessCollectionResponse{
					Results: []gen.ProcessEntry{
						{
							Hostname: "web-01",
							Status:   gen.ProcessEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessInfoResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Processes)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.ProcessCollectionResponse {
				pid1 := 1
				pid2 := 2
				errMsg := "unsupported"
				return &gen.ProcessCollectionResponse{
					JobId: &testUUID,
					Results: []gen.ProcessEntry{
						{
							Hostname: "web-01",
							Status:   gen.ProcessEntryStatusOk,
							Processes: &[]gen.ProcessInfo{
								{Pid: &pid1},
								{Pid: &pid2},
							},
						},
						{
							Hostname: "web-02",
							Status:   gen.ProcessEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessInfoResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Len(c.Results[0].Processes, 2)
				suite.Equal("unsupported", c.Results[1].Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ProcessInfoCollectionFromList(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *ProcessTypesPublicTestSuite) TestProcessInfoCollectionFromGet() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.ProcessGetResponse
		validateFunc func(client.Collection[client.ProcessInfoResult])
	}{
		{
			name: "when process found",
			input: func() *gen.ProcessGetResponse {
				pid := 42
				name := "sshd"
				return &gen.ProcessGetResponse{
					JobId: &testUUID,
					Results: []gen.ProcessGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.ProcessGetEntryStatusOk,
							Process: &gen.ProcessInfo{
								Pid:  &pid,
								Name: &name,
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessInfoResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Require().Len(r.Processes, 1)
				suite.Equal(42, r.Processes[0].PID)
				suite.Equal("sshd", r.Processes[0].Name)
			},
		},
		{
			name: "when process not found returns error entry",
			input: func() *gen.ProcessGetResponse {
				errMsg := "process not found"
				return &gen.ProcessGetResponse{
					Results: []gen.ProcessGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.ProcessGetEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Equal("failed", c.Results[0].Status)
				suite.Equal("process not found", c.Results[0].Error)
				suite.Nil(c.Results[0].Processes)
			},
		},
		{
			name: "when nil process pointer",
			input: &gen.ProcessGetResponse{
				Results: []gen.ProcessGetEntry{
					{
						Hostname: "web-01",
						Status:   gen.ProcessGetEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.ProcessInfoResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Nil(c.Results[0].Processes)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ProcessInfoCollectionFromGet(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *ProcessTypesPublicTestSuite) TestProcessSignalCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.ProcessSignalResponse
		validateFunc func(client.Collection[client.ProcessSignalResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.ProcessSignalResponse {
				changed := true
				pid := 1234
				signal := "TERM"
				return &gen.ProcessSignalResponse{
					JobId: &testUUID,
					Results: []gen.ProcessSignalResult{
						{
							Hostname: "web-01",
							Status:   gen.ProcessSignalResultStatusOk,
							Pid:      &pid,
							Signal:   &signal,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessSignalResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal(1234, r.PID)
				suite.Equal("TERM", r.Signal)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when signal failed",
			input: func() *gen.ProcessSignalResponse {
				errMsg := "no such process"
				pid := 9999
				return &gen.ProcessSignalResponse{
					Results: []gen.ProcessSignalResult{
						{
							Hostname: "web-01",
							Status:   gen.ProcessSignalResultStatusFailed,
							Pid:      &pid,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessSignalResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("failed", r.Status)
				suite.Equal(9999, r.PID)
				suite.Equal("no such process", r.Error)
				suite.False(r.Changed)
			},
		},
		{
			name: "when multiple results with nil pointers",
			input: func() *gen.ProcessSignalResponse {
				changed := true
				errMsg := "unsupported"
				return &gen.ProcessSignalResponse{
					JobId: &testUUID,
					Results: []gen.ProcessSignalResult{
						{
							Hostname: "web-01",
							Status:   gen.ProcessSignalResultStatusOk,
							Changed:  &changed,
						},
						{
							Hostname: "web-02",
							Status:   gen.ProcessSignalResultStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.ProcessSignalResult]) {
				suite.Require().Len(c.Results, 2)
				suite.True(c.Results[0].Changed)
				suite.Equal("unsupported", c.Results[1].Error)
				suite.False(c.Results[1].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ProcessSignalCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestProcessTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessTypesPublicTestSuite))
}
