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

type MemoryTypesPublicTestSuite struct {
	suite.Suite
}

func (suite *MemoryTypesPublicTestSuite) TestMemoryFromGen() {
	tests := []struct {
		name         string
		input        *gen.MemoryResponse
		validateFunc func(*client.Memory)
	}{
		{
			name: "when populated",
			input: &gen.MemoryResponse{
				Total: 8589934592,
				Used:  4294967296,
				Free:  4294967296,
			},
			validateFunc: func(m *client.Memory) {
				suite.Require().NotNil(m)
				suite.Equal(8589934592, m.Total)
				suite.Equal(4294967296, m.Used)
				suite.Equal(4294967296, m.Free)
			},
		},
		{
			name:  "when nil",
			input: nil,
			validateFunc: func(m *client.Memory) {
				suite.Nil(m)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := client.ExportMemoryFromGen(tc.input)
			tc.validateFunc(result)
		})
	}
}

func TestMemoryTypesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MemoryTypesPublicTestSuite))
}
