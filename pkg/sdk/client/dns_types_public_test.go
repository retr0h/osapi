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

type DNSTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *DNSTypesPublicTestSuite) TestDNSConfigCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.DNSConfigCollectionResponse
		validateFunc func(client.Collection[client.DNSConfig])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.DNSConfigCollectionResponse {
				servers := []string{"8.8.8.8", "8.8.4.4"}
				searchDomains := []string{"example.com", "local"}
				changed := false

				return &gen.DNSConfigCollectionResponse{
					JobId: &testUUID,
					Results: []gen.DNSConfigResponse{
						{
							Hostname:      "web-01",
							Changed:       &changed,
							Servers:       &servers,
							SearchDomains: &searchDomains,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.DNSConfig]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 1)

				dc := c.Results[0]
				suite.Equal("web-01", dc.Hostname)
				suite.Empty(dc.Error)
				suite.False(dc.Changed)
				suite.Equal([]string{"8.8.8.8", "8.8.4.4"}, dc.Servers)
				suite.Equal([]string{"example.com", "local"}, dc.SearchDomains)
			},
		},
		{
			name: "when minimal",
			input: &gen.DNSConfigCollectionResponse{
				Results: []gen.DNSConfigResponse{
					{Hostname: "web-01"},
				},
			},
			validateFunc: func(c client.Collection[client.DNSConfig]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)
				suite.Equal("web-01", c.Results[0].Hostname)
				suite.False(c.Results[0].Changed)
				suite.Nil(c.Results[0].Servers)
				suite.Nil(c.Results[0].SearchDomains)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportDNSConfigCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func (suite *DNSTypesPublicTestSuite) TestDNSUpdateCollectionFromGen() {
	tests := []struct {
		name         string
		input        *gen.DNSUpdateCollectionResponse
		validateFunc func(client.Collection[client.DNSUpdateResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.DNSUpdateCollectionResponse {
				changed := true

				return &gen.DNSUpdateCollectionResponse{
					Results: []gen.DNSUpdateResultItem{
						{
							Hostname: "web-01",
							Status:   gen.DNSUpdateResultItemStatus("applied"),
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.DNSUpdateResult]) {
				suite.Require().Len(c.Results, 1)

				dr := c.Results[0]
				suite.Equal("web-01", dr.Hostname)
				suite.Equal("applied", dr.Status)
				suite.True(dr.Changed)
				suite.Empty(dr.Error)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportDNSUpdateCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestDNSTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DNSTypesPublicTestSuite))
}
