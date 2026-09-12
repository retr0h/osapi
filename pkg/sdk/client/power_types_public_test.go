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

type PowerTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *PowerTypesPublicTestSuite) TestPowerCollectionFromReboot() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.PowerRebootResponse
		validateFunc func(client.Collection[client.PowerResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PowerRebootResponse {
				changed := true
				action := "reboot"
				delay := 5
				return &gen.PowerRebootResponse{
					JobId: &testUUID,
					Results: []gen.PowerResult{
						{
							Hostname: "web-01",
							Status:   gen.PowerResultStatusOk,
							Action:   &action,
							Delay:    &delay,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PowerResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("reboot", r.Action)
				suite.Equal(5, r.Delay)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.PowerRebootResponse {
				errMsg := "permission denied"
				return &gen.PowerRebootResponse{
					Results: []gen.PowerResult{
						{
							Hostname: "web-01",
							Status:   gen.PowerResultStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PowerResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Empty(r.Action)
				suite.Zero(r.Delay)
				suite.False(r.Changed)
				suite.Equal("permission denied", r.Error)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.PowerRebootResponse {
				changed1 := true
				changed2 := false
				errMsg := "unsupported"
				return &gen.PowerRebootResponse{
					JobId: &testUUID,
					Results: []gen.PowerResult{
						{
							Hostname: "web-01",
							Status:   gen.PowerResultStatusOk,
							Changed:  &changed1,
						},
						{
							Hostname: "web-02",
							Status:   gen.PowerResultStatusSkipped,
							Changed:  &changed2,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PowerResult]) {
				suite.Require().Len(c.Results, 2)
				suite.True(c.Results[0].Changed)
				suite.False(c.Results[1].Changed)
				suite.Equal("unsupported", c.Results[1].Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PowerCollectionFromReboot(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *PowerTypesPublicTestSuite) TestPowerCollectionFromShutdown() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.PowerShutdownResponse
		validateFunc func(client.Collection[client.PowerResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.PowerShutdownResponse {
				changed := true
				action := "shutdown"
				delay := 10
				return &gen.PowerShutdownResponse{
					JobId: &testUUID,
					Results: []gen.PowerResult{
						{
							Hostname: "web-01",
							Status:   gen.PowerResultStatusOk,
							Action:   &action,
							Delay:    &delay,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.PowerResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("shutdown", r.Action)
				suite.Equal(10, r.Delay)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.PowerShutdownResponse{
				Results: []gen.PowerResult{
					{
						Hostname: "web-01",
						Status:   gen.PowerResultStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.PowerResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Action)
				suite.Zero(r.Delay)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.PowerCollectionFromShutdown(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestPowerTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PowerTypesPublicTestSuite))
}
