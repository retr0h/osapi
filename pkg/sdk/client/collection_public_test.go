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

type CollectionPublicTestSuite struct {
	suite.Suite
}

func (suite *CollectionPublicTestSuite) TestCollectionFirst() {
	tests := []struct {
		name         string
		col          client.Collection[client.HostnameResult]
		validateFunc func(client.HostnameResult, bool)
	}{
		{
			name: "returns first result and true",
			col: client.Collection[client.HostnameResult]{
				Results: []client.HostnameResult{
					{Hostname: "web-01"},
					{Hostname: "web-02"},
				},
				JobID: "job-1",
			},
			validateFunc: func(
				r client.HostnameResult,
				ok bool,
			) {
				suite.True(ok)
				suite.Equal("web-01", r.Hostname)
			},
		},
		{
			name: "returns zero value and false when empty",
			col: client.Collection[client.HostnameResult]{
				Results: []client.HostnameResult{},
			},
			validateFunc: func(
				r client.HostnameResult,
				ok bool,
			) {
				suite.False(ok)
				suite.Equal("", r.Hostname)
			},
		},
		{
			name: "returns zero value and false when nil",
			col:  client.Collection[client.HostnameResult]{},
			validateFunc: func(
				r client.HostnameResult,
				ok bool,
			) {
				suite.False(ok)
				suite.Equal("", r.Hostname)
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			r, ok := tt.col.First()
			tt.validateFunc(r, ok)
		})
	}
}

func TestCollectionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CollectionPublicTestSuite))
}
