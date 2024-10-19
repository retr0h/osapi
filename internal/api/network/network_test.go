// Copyright (c) 2024 John Dewey

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

package network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
}

func (suite *NetworkTestSuite) SetupTest() {}

func (suite *NetworkTestSuite) TearDownTest() {}

func (suite *NetworkTestSuite) TestDurationToString() {
	tests := []struct {
		name  string
		input *time.Duration
		want  *string
	}{
		{
			name:  "nil duration",
			input: nil,
			want:  nil,
		},
		{
			name:  "zero duration",
			input: durationPtr(0),
			want:  stringPtr("0s"),
		},
		{
			name:  "one second duration",
			input: durationPtr(time.Second),
			want:  stringPtr("1s"),
		},
		{
			name:  "two minutes duration",
			input: durationPtr(2 * time.Minute),
			want:  stringPtr("2m0s"),
		},
		{
			name:  "one and a half hours duration",
			input: durationPtr(1*time.Hour + 30*time.Minute),
			want:  stringPtr("1h30m0s"),
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := durationToString(tc.input)

			suite.Equal(tc.want, got)
		})
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func stringPtr(s string) *string {
	return &s
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}
