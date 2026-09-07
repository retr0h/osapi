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

type InterfaceTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *InterfaceTypesPublicTestSuite) TestInterfaceInfoFromGen() {
	tests := []struct {
		name         string
		input        gen.InterfaceInfo
		validateFunc func(client.InterfaceInfo)
	}{
		{
			name: "when all fields are populated",
			input: func() gen.InterfaceInfo {
				name := "eth0"
				dhcp4 := true
				dhcp6 := false
				gw4 := "192.168.1.1"
				gw6 := "fe80::1"
				mtu := 1500
				mac := "00:11:22:33:44:55"
				wol := true
				state := "up"
				addrs := []string{"192.168.1.10/24", "10.0.0.1/8"}

				return gen.InterfaceInfo{
					Name:       &name,
					Dhcp4:      &dhcp4,
					Dhcp6:      &dhcp6,
					Gateway4:   &gw4,
					Gateway6:   &gw6,
					Mtu:        &mtu,
					MacAddress: &mac,
					Wakeonlan:  &wol,
					State:      &state,
					Addresses:  &addrs,
				}
			}(),
			validateFunc: func(info client.InterfaceInfo) {
				suite.Equal("eth0", info.Name)
				suite.True(info.DHCP4)
				suite.False(info.DHCP6)
				suite.Equal("192.168.1.1", info.Gateway4)
				suite.Equal("fe80::1", info.Gateway6)
				suite.Equal(1500, info.MTU)
				suite.Equal("00:11:22:33:44:55", info.MACAddress)
				suite.True(info.WakeOnLAN)
				suite.Equal("up", info.State)
				suite.Equal([]string{"192.168.1.10/24", "10.0.0.1/8"}, info.Addresses)
			},
		},
		{
			name:  "when all fields are nil",
			input: gen.InterfaceInfo{},
			validateFunc: func(info client.InterfaceInfo) {
				suite.Empty(info.Name)
				suite.False(info.DHCP4)
				suite.False(info.DHCP6)
				suite.Empty(info.Gateway4)
				suite.Empty(info.Gateway6)
				suite.Zero(info.MTU)
				suite.Empty(info.MACAddress)
				suite.False(info.WakeOnLAN)
				suite.Empty(info.State)
				suite.Nil(info.Addresses)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.InterfaceInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *InterfaceTypesPublicTestSuite) TestInterfaceListCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.InterfaceListResponse
		validateFunc func(client.Collection[client.InterfaceListResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.InterfaceListResponse {
				name := "eth0"
				dhcp4 := true

				return &gen.InterfaceListResponse{
					JobId: &testUUID,
					Results: []gen.InterfaceListEntry{
						{
							Hostname: "web-01",
							Status:   gen.InterfaceListEntryStatusOk,
							Interfaces: &[]gen.InterfaceInfo{
								{
									Name:  &name,
									Dhcp4: &dhcp4,
								},
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.InterfaceListResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Empty(r.Error)
				suite.Require().Len(r.Interfaces, 1)
				suite.Equal("eth0", r.Interfaces[0].Name)
				suite.True(r.Interfaces[0].DHCP4)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.InterfaceListResponse {
				errMsg := "permission denied"

				return &gen.InterfaceListResponse{
					Results: []gen.InterfaceListEntry{
						{
							Hostname: "web-01",
							Status:   gen.InterfaceListEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.InterfaceListResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Equal("permission denied", r.Error)
				suite.Nil(r.Interfaces)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.InterfaceListCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *InterfaceTypesPublicTestSuite) TestInterfaceGetCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.InterfaceGetResponse
		validateFunc func(client.Collection[client.InterfaceGetResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.InterfaceGetResponse {
				name := "eth0"
				dhcp4 := true

				return &gen.InterfaceGetResponse{
					JobId: &testUUID,
					Results: []gen.InterfaceGetEntry{
						{
							Hostname: "web-01",
							Status:   gen.InterfaceGetEntryStatusOk,
							Interface: &gen.InterfaceInfo{
								Name:  &name,
								Dhcp4: &dhcp4,
							},
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.InterfaceGetResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.NotNil(r.Interface)
				suite.Equal("eth0", r.Interface.Name)
				suite.True(r.Interface.DHCP4)
			},
		},
		{
			name: "when interface is nil",
			input: &gen.InterfaceGetResponse{
				Results: []gen.InterfaceGetEntry{
					{
						Hostname: "web-01",
						Status:   gen.InterfaceGetEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.InterfaceGetResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Nil(r.Interface)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.InterfaceGetCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *InterfaceTypesPublicTestSuite) TestInterfaceMutationCollectionFromCreate() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00,
		0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66,
		0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.InterfaceMutationResponse
		validateFunc func(client.Collection[client.InterfaceMutationResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.InterfaceMutationResponse {
				name := "eth0"
				changed := true

				return &gen.InterfaceMutationResponse{
					JobId: &testUUID,
					Results: []gen.InterfaceMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.InterfaceMutationEntryStatusOk,
							Name:     &name,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.InterfaceMutationResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("ok", r.Status)
				suite.Equal("eth0", r.Name)
				suite.True(r.Changed)
				suite.Empty(r.Error)
			},
		},
		{
			name: "when minimal with error",
			input: func() *gen.InterfaceMutationResponse {
				errMsg := "unsupported"

				return &gen.InterfaceMutationResponse{
					Results: []gen.InterfaceMutationEntry{
						{
							Hostname: "web-01",
							Status:   gen.InterfaceMutationEntryStatusFailed,
							Error:    &errMsg,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.InterfaceMutationResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("failed", r.Status)
				suite.Empty(r.Name)
				suite.False(r.Changed)
				suite.Equal("unsupported", r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.InterfaceMutationCollectionFromCreate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *InterfaceTypesPublicTestSuite) TestInterfaceMutationCollectionFromUpdate() {
	tests := []struct {
		name         string
		input        *gen.InterfaceMutationResponse
		validateFunc func(client.Collection[client.InterfaceMutationResult])
	}{
		{
			name: "when minimal with nil pointers",
			input: &gen.InterfaceMutationResponse{
				Results: []gen.InterfaceMutationEntry{
					{
						Hostname: "web-01",
						Status:   gen.InterfaceMutationEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.InterfaceMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Name)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.InterfaceMutationCollectionFromUpdate(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *InterfaceTypesPublicTestSuite) TestInterfaceMutationCollectionFromDelete() {
	tests := []struct {
		name         string
		input        *gen.InterfaceMutationResponse
		validateFunc func(client.Collection[client.InterfaceMutationResult])
	}{
		{
			name: "when minimal with nil pointers",
			input: &gen.InterfaceMutationResponse{
				Results: []gen.InterfaceMutationEntry{
					{
						Hostname: "web-01",
						Status:   gen.InterfaceMutationEntryStatusSkipped,
					},
				},
			},
			validateFunc: func(c client.Collection[client.InterfaceMutationResult]) {
				suite.Require().Len(c.Results, 1)

				r := c.Results[0]
				suite.Equal("web-01", r.Hostname)
				suite.Equal("skipped", r.Status)
				suite.Empty(r.Name)
				suite.False(r.Changed)
				suite.Empty(r.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.InterfaceMutationCollectionFromDelete(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestInterfaceTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InterfaceTypesPublicTestSuite))
}
