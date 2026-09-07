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

type HostnameTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *HostnameTypesPublicTestSuite) TestHostnameCollectionFromGen() {
	testUUID := openapi_types.UUID{
		0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4,
		0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00,
	}

	tests := []struct {
		name         string
		input        *gen.HostnameCollectionResponse
		validateFunc func(client.Collection[client.HostnameResult])
	}{
		{
			name: "when all fields are populated",
			input: func() *gen.HostnameCollectionResponse {
				labels := map[string]string{"group": "web", "env": "prod"}
				errMsg := "timeout"
				changed := false

				return &gen.HostnameCollectionResponse{
					JobId: &testUUID,
					Results: []gen.HostnameResponse{
						{
							Hostname: "web-01",
							Labels:   &labels,
							Changed:  &changed,
						},
						{
							Hostname: "web-02",
							Error:    &errMsg,
							Changed:  &changed,
						},
					},
				}
			}(),
			validateFunc: func(c client.Collection[client.HostnameResult]) {
				suite.Equal("550e8400-e29b-41d4-a716-446655440000", c.JobID)
				suite.Require().Len(c.Results, 2)

				suite.Equal("web-01", c.Results[0].Hostname)
				suite.Equal(map[string]string{"group": "web", "env": "prod"}, c.Results[0].Labels)
				suite.Empty(c.Results[0].Error)
				suite.False(c.Results[0].Changed)

				suite.Equal("web-02", c.Results[1].Hostname)
				suite.Equal("timeout", c.Results[1].Error)
				suite.Nil(c.Results[1].Labels)
				suite.False(c.Results[1].Changed)
			},
		},
		{
			name: "when minimal",
			input: &gen.HostnameCollectionResponse{
				Results: []gen.HostnameResponse{
					{Hostname: "minimal-host"},
				},
			},
			validateFunc: func(c client.Collection[client.HostnameResult]) {
				suite.Empty(c.JobID)
				suite.Require().Len(c.Results, 1)
				suite.Equal("minimal-host", c.Results[0].Hostname)
				suite.Empty(c.Results[0].Error)
				suite.Nil(c.Results[0].Labels)
				suite.False(c.Results[0].Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportHostnameCollectionFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestHostnameTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(HostnameTypesPublicTestSuite))
}
