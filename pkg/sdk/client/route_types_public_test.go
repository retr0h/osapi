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

type RouteTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *RouteTypesPublicTestSuite) TestRouteInfoFromGen() {
	tests := []struct {
		name         string
		input        gen.RouteInfo
		validateFunc func(client.RouteInfo)
	}{
		{
			name: "when all fields are populated",
			input: func() gen.RouteInfo {
				dest := "10.0.0.0/8"
				gw := "192.168.1.1"
				iface := "eth0"
				metric := 100
				scope := "global"

				return gen.RouteInfo{
					Destination: &dest,
					Gateway:     &gw,
					Interface:   &iface,
					Metric:      &metric,
					Scope:       &scope,
				}
			}(),
			validateFunc: func(info client.RouteInfo) {
				suite.Equal("10.0.0.0/8", info.Destination)
				suite.Equal("192.168.1.1", info.Gateway)
				suite.Equal("eth0", info.Interface)
				suite.Equal(100, info.Metric)
				suite.Equal("global", info.Scope)
			},
		},
		{
			name:  "when all fields are nil",
			input: gen.RouteInfo{},
			validateFunc: func(info client.RouteInfo) {
				suite.Empty(info.Destination)
				suite.Empty(info.Gateway)
				suite.Empty(info.Interface)
				suite.Zero(info.Metric)
				suite.Empty(info.Scope)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.RouteInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *RouteTypesPublicTestSuite) TestRouteListCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.RouteListResponse
		validateFunc func(client.Collection[client.RouteListResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.RouteListResponse {
				dest := "10.0.0.0/8"
				gw := "192.168.1.1"

				return &gen.RouteListResponse{
					JobId: &testUUID,
					Results: []gen.RouteListEntry{
						{
							Hostname: "web-01",
							Status:   gen.RouteListEntryStatusOk,
							Routes: &[]gen.RouteInfo{
								{
									Destination: &dest,
									Gateway:     &gw,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.RouteListResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Routes, 1)
				suite.Equal("10.0.0.0/8", r.Routes[0].Destination)
				suite.Equal("192.168.1.1", r.Routes[0].Gateway)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.RouteListResponse {
				errMsg := "permission denied"

				return &gen.RouteListResponse{
					Results: []gen.RouteListEntry{
						{
							Hostname: "web-01",
							Status:   gen.RouteListEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.RouteListResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Routes)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.RouteListCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *RouteTypesPublicTestSuite) TestRouteGetCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.RouteGetResponse
		validateFunc func(client.Collection[client.RouteGetResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.RouteGetResponse {
				dest := "10.0.0.0/8"
				gw := "192.168.1.1"

				return &gen.RouteGetResponse{
					JobId: &testUUID,
					Results: []gen.RouteGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.RouteGetEntryStatusOk,
							Routes: &[]gen.RouteInfo{
								{
									Destination: &dest,
									Gateway:     &gw,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.RouteGetResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Require().Len(r.Routes, 1)
				suite.Equal("10.0.0.0/8", r.Routes[0].Destination)
			},
		},
		{
			name: "when routes is nil",
			input: &gen.RouteGetResponse{
				Results: []gen.RouteGetEntry{
					{
						Hostname: "web-01",
						Status:   gen.RouteGetEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.RouteGetResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Nil(r.Routes)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.RouteGetCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *RouteTypesPublicTestSuite) TestRouteMutationCollectionFromCreate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.RouteMutationResponse
		validateFunc func(client.Collection[client.RouteMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.RouteMutationResponse {
				iface := "eth0"
				changed := true

				return &gen.RouteMutationResponse{
					JobId: &testUUID,
					Results: []gen.RouteMutationEntry{
						{
							Hostname:  "web-01",
							Status:    gen.RouteMutationEntryStatusOk,
							Interface: &iface,
							Changed:   &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.RouteMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("eth0", r.Interface)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.RouteMutationResponse {
				errMsg := "unsupported"

				return &gen.RouteMutationResponse{
					Results: []gen.RouteMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.RouteMutationEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.RouteMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Empty(r.Interface)
				suite.False(r.Changed)
				suite.Equal("unsupported", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.RouteMutationCollectionFromCreate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *RouteTypesPublicTestSuite) TestRouteMutationCollectionFromUpdate() {
	tests := []struct {
		name         string
		input        *gen.RouteMutationResponse
		validateFunc func(client.Collection[client.RouteMutationResult])
	}{
		{
			name: "when minimal with nil pointers",
			input: &gen.RouteMutationResponse{
				Results: []gen.RouteMutationEntry{
					{
						Hostname: "web-01",
						Status:   gen.RouteMutationEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.RouteMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Interface)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.RouteMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *RouteTypesPublicTestSuite) TestRouteMutationCollectionFromDelete() {
	tests := []struct {
		name         string
		input        *gen.RouteMutationResponse
		validateFunc func(client.Collection[client.RouteMutationResult])
	}{
		{
			name: "when minimal with nil pointers",
			input: &gen.RouteMutationResponse{
				Results: []gen.RouteMutationEntry{
					{
						Hostname: "web-01",
						Status:   gen.RouteMutationEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.RouteMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Interface)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.RouteMutationCollectionFromDelete(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestRouteTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RouteTypesPublicTestSuite))
}
