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

type NTPTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *NTPTypesPublicTestSuite) TestNtpStatusCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.NtpCollectionResponse
		validateFunc func(client.Collection[client.NtpStatusResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.NtpCollectionResponse {
				synced := true
				stratum := 2
				offset := "+0.001s"
				source := "pool.ntp.org"
				servers := []string{"0.pool.ntp.org", "1.pool.ntp.org"}

				return &gen.NtpCollectionResponse{
					JobId: &testUUID,
					Results: []gen.NtpStatusEntry{
						{
							Hostname:      "web-01",
							Status:        gen.NtpStatusEntryStatusOk,
							Synchronized:  &synced,
							Stratum:       &stratum,
							Offset:        &offset,
							CurrentSource: &source,
							Servers:       &servers,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpStatusResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.True(r.Synchronized)
				suite.Equal(2, r.Stratum)
				suite.Equal("+0.001s", r.Offset)
				suite.Equal("pool.ntp.org", r.CurrentSource)
				suite.Equal([]string{"0.pool.ntp.org", "1.pool.ntp.org"}, r.Servers)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.NtpCollectionResponse {
				errMsg := "permission denied"

				return &gen.NtpCollectionResponse{
					Results: []gen.NtpStatusEntry{
						{
							Hostname: "web-01",
							Status:   gen.NtpStatusEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpStatusResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.False(r.Synchronized)
				suite.Zero(r.Stratum)
				suite.Empty(r.Offset)
				suite.Empty(r.CurrentSource)
				suite.Nil(r.Servers)
				suite.Equal("permission denied", r.Error)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.NtpCollectionResponse {
				synced1 := true
				synced2 := false

				return &gen.NtpCollectionResponse{
					JobId: &testUUID,
					Results: []gen.NtpStatusEntry{
						{
							Hostname:     "web-01",
							Status:       gen.NtpStatusEntryStatusOk,
							Synchronized: &synced1,
						},
						{
							Hostname:     "web-02",
							Status:       gen.NtpStatusEntryStatusOk,
							Synchronized: &synced2,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpStatusResult]) {
				suite.Require().Len(c.Results, 2)
				suite.True(c.Results[0].Synchronized)
				suite.False(c.Results[1].Synchronized)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.NtpStatusCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *NTPTypesPublicTestSuite) TestNtpMutationCollectionFromCreate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.NtpCreateResponse
		validateFunc func(client.Collection[client.NtpMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.NtpCreateResponse {
				changed := true

				return &gen.NtpCreateResponse{
					JobId: &testUUID,
					Results: []gen.NtpMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.NtpMutationResultStatusOk,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.NtpCreateResponse {
				errMsg := "permission denied"

				return &gen.NtpCreateResponse{
					Results: []gen.NtpMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.NtpMutationResultStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.False(r.Changed)
				suite.Equal("permission denied", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.NtpMutationCollectionFromCreate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *NTPTypesPublicTestSuite) TestNtpMutationCollectionFromUpdate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.NtpUpdateResponse
		validateFunc func(client.Collection[client.NtpMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.NtpUpdateResponse {
				changed := true

				return &gen.NtpUpdateResponse{
					JobId: &testUUID,
					Results: []gen.NtpMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.NtpMutationResultStatusOk,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.NtpUpdateResponse{
				Results: []gen.NtpMutationResult{
					{
						Hostname: "web-01",
						Status:   gen.NtpMutationResultStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.NtpMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.NtpMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *NTPTypesPublicTestSuite) TestNtpMutationCollectionFromDelete() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.NtpDeleteResponse
		validateFunc func(client.Collection[client.NtpMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.NtpDeleteResponse {
				changed := true

				return &gen.NtpDeleteResponse{
					JobId: &testUUID,
					Results: []gen.NtpMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.NtpMutationResultStatusOk,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.NtpMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.NtpDeleteResponse{
				Results: []gen.NtpMutationResult{
					{
						Hostname: "web-01",
						Status:   gen.NtpMutationResultStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.NtpMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.NtpMutationCollectionFromDelete(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestNTPTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(NTPTypesPublicTestSuite))
}
