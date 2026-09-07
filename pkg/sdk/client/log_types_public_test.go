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

type LogTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *LogTypesPublicTestSuite) TestLogCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.LogCollectionResponse
		validateFunc func(client.Collection[client.LogEntryResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.LogCollectionResponse {
				ts := "2026-01-01T00:00:00Z"
				unit := "sshd.service"
				priority := "info"
				message := "Accepted publickey for root"
				pid := 1234
				hostname := "web-01"
				return &gen.LogCollectionResponse{
					JobId: &testUUID,
					Results: []gen.LogResultEntry{
						{
							Hostname: "web-01",
							Status:   gen.LogResultEntryStatusOk,
							Entries: &[]gen.LogEntryInfo{
								{
									Timestamp: &ts,
									Unit:      &unit,
									Priority:  &priority,
									Message:   &message,
									Pid:       &pid,
									Hostname:  &hostname,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.LogEntryResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Entries, 1)

				e := r.Entries[0]
				suite.Equal("2026-01-01T00:00:00Z", e.Timestamp)
				suite.Equal("sshd.service", e.Unit)
				suite.Equal("info", e.Priority)
				suite.Equal("Accepted publickey for root", e.Message)
				suite.Equal(1234, e.PID)
				suite.Equal("web-01", e.Hostname)
			},
		},
		{
			name: "when result has error",
			input: func() *gen.LogCollectionResponse {
				errMsg := "permission denied"
				return &gen.LogCollectionResponse{
					Results: []gen.LogResultEntry{
						{
							Hostname: "web-01",
							Status:   gen.LogResultEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.LogEntryResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Entries)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.LogCollectionResponse {
				msg1 := "Started"
				msg2 := "Stopped"
				errMsg := "unsupported"
				return &gen.LogCollectionResponse{
					JobId: &testUUID,
					Results: []gen.LogResultEntry{
						{
							Hostname: "web-01",
							Status:   gen.LogResultEntryStatusOk,
							Entries: &[]gen.LogEntryInfo{
								{Message: &msg1},
								{Message: &msg2},
							},
						},
						{
							Hostname: "web-02",
							Status:   gen.LogResultEntryStatusSkipped,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.LogEntryResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Len(c.Results[0].Entries, 2)
				suite.Equal("unsupported", c.Results[1].Error)
			},
		},
		{
			name: "when entries pointer is nil",
			input: &gen.LogCollectionResponse{
				Results: []gen.LogResultEntry{
					{
						Hostname: "web-01",
						Status:   gen.LogResultEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.LogEntryResult]) {
				suite.Require().Len(c.Results, 1)
				suite.Nil(c.Results[0].Entries)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.LogCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *LogTypesPublicTestSuite) TestLogEntryInfoFromGen() {
	tests := []struct {
		name         string
		input        gen.LogEntryInfo
		validateFunc func(client.LogEntry)
	}{
		{
			name: "when all fields are populated",
			input: func() gen.LogEntryInfo {
				ts := "2026-01-01T12:00:00Z"
				unit := "nginx.service"
				priority := "err"
				message := "connection refused"
				pid := 9999
				hostname := "web-01"
				return gen.LogEntryInfo{
					Timestamp: &ts,
					Unit:      &unit,
					Priority:  &priority,
					Message:   &message,
					Pid:       &pid,
					Hostname:  &hostname,
				}
			}(),
			validateFunc: func(e client.LogEntry) {
				suite.Equal("2026-01-01T12:00:00Z", e.Timestamp)
				suite.Equal("nginx.service", e.Unit)
				suite.Equal("err", e.Priority)
				suite.Equal("connection refused", e.Message)
				suite.Equal(9999, e.PID)
				suite.Equal("web-01", e.Hostname)
			},
		},
		{
			name:  "when all fields are nil",
			input: gen.LogEntryInfo{},
			validateFunc: func(e client.LogEntry) {
				suite.Empty(e.Timestamp)
				suite.Empty(e.Unit)
				suite.Empty(e.Priority)
				suite.Empty(e.Message)
				suite.Zero(e.PID)
				suite.Empty(e.Hostname)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.LogEntryInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestLogTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LogTypesPublicTestSuite))
}
