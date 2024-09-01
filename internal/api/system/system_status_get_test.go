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

package system

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SystemStatusTestSuite struct {
	suite.Suite
}

func (suite *SystemStatusTestSuite) SetupTest() {
}

func (suite *SystemStatusTestSuite) TestFormatDurationOk() {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{
			name: "0 days, 0 hours, 0 minutes",
			d:    time.Duration(0) * time.Second,
			want: "0 days, 0 hours, 0 minutes",
		},
		{
			name: "0 days, 0 hours, 1 minute",
			d:    time.Duration(60) * time.Second,
			want: "0 days, 0 hours, 1 minute",
		},
		{
			name: "0 days, 1 hour, 0 minutes",
			d:    time.Duration(3600) * time.Second,
			want: "0 days, 1 hour, 0 minutes",
		},
		{
			name: "1 day, 0 hours, 0 minutes",
			d:    time.Duration(24*3600) * time.Second,
			want: "1 day, 0 hours, 0 minutes",
		},
		{
			name: "1 day, 1 hour, 1 minute",
			d:    time.Duration(24*3600+3600+60) * time.Second,
			want: "1 day, 1 hour, 1 minute",
		},
		{
			name: "4 days, 1 hour, 25 minutes",
			d:    time.Duration(int64(math.Trunc(350735.47))) * time.Second,
			want: "4 days, 1 hour, 25 minutes",
		},
		{
			name: "2 days, 2 hours, 2 minutes",
			d:    time.Duration(2*24*3600+2*3600+2*60) * time.Second,
			want: "2 days, 2 hours, 2 minutes",
		},
		{
			name: "0 days, 0 hours, 59 minutes",
			d:    time.Duration(59) * time.Minute,
			want: "0 days, 0 hours, 59 minutes",
		},
		{
			name: "0 days, 23 hours, 59 minutes",
			d:    time.Duration(23*3600+59*60) * time.Second,
			want: "0 days, 23 hours, 59 minutes",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := formatDuration(tc.d)
			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestSystemStatusTestSuite(t *testing.T) {
	suite.Run(t, new(SystemStatusTestSuite))
}
