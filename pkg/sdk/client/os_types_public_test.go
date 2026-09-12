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

type OSTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *OSTypesPublicTestSuite) TestOSInfoFromGen() {
	tests := []struct {
		name         string
		input        *gen.OSInfoResponse
		validateFunc func(*client.OSInfo)
	}{
		{
			name: "when populated",
			input: &gen.OSInfoResponse{
				Distribution: "Ubuntu",
				Version:      "22.04",
			},
			validateFunc: func(oi *client.OSInfo) {
				suite.Require().NotNil(oi)
				suite.Equal("Ubuntu", oi.Distribution)
				suite.Equal("22.04", oi.Version)
			},
		},
		{
			name:  "when nil",
			input: nil,
			validateFunc: func(oi *client.OSInfo) {
				suite.Nil(oi)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportOSInfoFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestOSTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OSTypesPublicTestSuite))
}
