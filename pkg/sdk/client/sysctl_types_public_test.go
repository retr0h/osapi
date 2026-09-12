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

type SysctlTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *SysctlTypesPublicTestSuite) TestSysctlEntryCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SysctlCollectionResponse
		validateFunc func(client.Collection[client.SysctlEntryResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SysctlCollectionResponse {
				key := "net.ipv4.ip_forward"
				value := "1"

				return &gen.SysctlCollectionResponse{
					JobId: &testUUID,
					Results: []gen.SysctlEntry{
						{
							Hostname: "web-01",
							Status:   gen.SysctlEntryStatusOk,
							Key:      &key,
							Value:    &value,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlEntryResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("net.ipv4.ip_forward", r.Key)
				suite.Equal("1", r.Value)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.SysctlCollectionResponse {
				errMsg := "permission denied"

				return &gen.SysctlCollectionResponse{
					Results: []gen.SysctlEntry{
						{
							Hostname: "web-01",
							Status:   gen.SysctlEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlEntryResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Empty(r.Key)
				suite.Empty(r.Value)
				suite.Equal("permission denied", r.Error)
			},
		},
		{
			name: "when multiple results",
			input: func() *gen.SysctlCollectionResponse {
				key1 := "net.ipv4.ip_forward"
				val1 := "1"
				key2 := "vm.swappiness"
				val2 := "60"

				return &gen.SysctlCollectionResponse{
					JobId: &testUUID,
					Results: []gen.SysctlEntry{
						{
							Hostname: "web-01",
							Status:   gen.SysctlEntryStatusOk,
							Key:      &key1,
							Value:    &val1,
						},
						{
							Hostname: "web-01",
							Status:   gen.SysctlEntryStatusOk,
							Key:      &key2,
							Value:    &val2,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlEntryResult]) {
				suite.Require().Len(c.Results, 2)
				suite.Equal("net.ipv4.ip_forward", c.Results[0].Key)
				suite.Equal("vm.swappiness", c.Results[1].Key)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SysctlEntryCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *SysctlTypesPublicTestSuite) TestSysctlEntryCollectionFromGet() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SysctlGetResponse
		validateFunc func(client.Collection[client.SysctlEntryResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SysctlGetResponse {
				key := "net.ipv4.ip_forward"
				value := "1"

				return &gen.SysctlGetResponse{
					JobId: &testUUID,
					Results: []gen.SysctlEntry{
						{
							Hostname: "web-01",
							Status:   gen.SysctlEntryStatusOk,
							Key:      &key,
							Value:    &value,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlEntryResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("net.ipv4.ip_forward", r.Key)
				suite.Equal("1", r.Value)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.SysctlGetResponse{
				Results: []gen.SysctlEntry{
					{
						Hostname: "web-01",
						Status:   gen.SysctlEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.SysctlEntryResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Key)
				suite.Empty(r.Value)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SysctlEntryCollectionFromGet(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *SysctlTypesPublicTestSuite) TestSysctlMutationCollectionFromCreate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SysctlCreateResponse
		validateFunc func(client.Collection[client.SysctlMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SysctlCreateResponse {
				key := "net.ipv4.ip_forward"
				changed := true

				return &gen.SysctlCreateResponse{
					JobId: &testUUID,
					Results: []gen.SysctlMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.SysctlMutationResultStatusOk,
							Key:      &key,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("net.ipv4.ip_forward", r.Key)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.SysctlCreateResponse {
				errMsg := "permission denied"

				return &gen.SysctlCreateResponse{
					Results: []gen.SysctlMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.SysctlMutationResultStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Empty(r.Key)
				suite.False(r.Changed)
				suite.Equal("permission denied", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SysctlMutationCollectionFromCreate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *SysctlTypesPublicTestSuite) TestSysctlMutationCollectionFromUpdate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SysctlUpdateResponse
		validateFunc func(client.Collection[client.SysctlMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SysctlUpdateResponse {
				key := "net.ipv4.ip_forward"
				changed := true

				return &gen.SysctlUpdateResponse{
					JobId: &testUUID,
					Results: []gen.SysctlMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.SysctlMutationResultStatusOk,
							Key:      &key,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("net.ipv4.ip_forward", r.Key)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.SysctlUpdateResponse{
				Results: []gen.SysctlMutationResult{
					{
						Hostname: "web-01",
						Status:   gen.SysctlMutationResultStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.SysctlMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Key)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SysctlMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *SysctlTypesPublicTestSuite) TestSysctlMutationCollectionFromDelete() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.SysctlDeleteResponse
		validateFunc func(client.Collection[client.SysctlMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.SysctlDeleteResponse {
				key := "net.ipv4.ip_forward"
				changed := true

				return &gen.SysctlDeleteResponse{
					JobId: &testUUID,
					Results: []gen.SysctlMutationResult{
						{
							Hostname: "web-01",
							Status:   gen.SysctlMutationResultStatusOk,
							Key:      &key,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.SysctlMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("net.ipv4.ip_forward", r.Key)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with nil pointers",
			input: &gen.SysctlDeleteResponse{
				Results: []gen.SysctlMutationResult{
					{
						Hostname: "web-01",
						Status:   gen.SysctlMutationResultStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.SysctlMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Key)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.SysctlMutationCollectionFromDelete(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestSysctlTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SysctlTypesPublicTestSuite))
}
