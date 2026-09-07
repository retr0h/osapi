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
)

type ResponsePublicTestSuite struct {
	suite.Suite
}

func (suite *ResponsePublicTestSuite) TestRawJSON() {
	tests := []struct {
		name         string
		rawJSON      []byte
		validateFunc func(*client.Response[string])
	}{
		{
			name:    "when RawJSON returns the raw bytes",
			rawJSON: []byte(`{"hostname":"web-01"}`),
			validateFunc: func(resp *client.Response[string]) {
				suite.Equal(
					[]byte(`{"hostname":"web-01"}`),
					resp.RawJSON(),
				)
			},
		},
		{
			name:    "when RawJSON returns nil for empty response",
			rawJSON: nil,
			validateFunc: func(resp *client.Response[string]) {
				suite.Nil(resp.RawJSON())
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			resp := client.NewResponse("test", tc.rawJSON)
			tc.validateFunc(resp)
		})
	}
}

func (suite *ResponsePublicTestSuite) TestData() {
	tests := []struct {
		name         string
		data         string
		rawJSON      []byte
		validateFunc func(*client.Response[string])
	}{
		{
			name:    "when Data contains the domain type",
			data:    "web-01",
			rawJSON: []byte(`{"hostname":"web-01"}`),
			validateFunc: func(resp *client.Response[string]) {
				suite.Equal("web-01", resp.Data)
			},
		},
		{
			name:    "when Data contains an empty string",
			data:    "",
			rawJSON: []byte(`{}`),
			validateFunc: func(resp *client.Response[string]) {
				suite.Empty(resp.Data)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			resp := client.NewResponse(tc.data, tc.rawJSON)
			tc.validateFunc(resp)
		})
	}
}

func TestResponsePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ResponsePublicTestSuite))
}
