//go:build ubuntu
// +build ubuntu

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

package load_test

import (
	"testing"

	sysLoad "github.com/shirou/gopsutil/v4/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system/load"
)

type UbuntuAvgPublicTestSuite struct {
	suite.Suite
}

func (suite *UbuntuAvgPublicTestSuite) SetupTest() {}

func (suite *UbuntuAvgPublicTestSuite) TearDownTest() {}

func (suite *UbuntuAvgPublicTestSuite) TestGetAverageStats() {
	tests := []struct {
		name        string
		setupMock   func() func() (*sysLoad.AvgStat, error)
		want        *load.AverageStats
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetAverageStats Ok",
			want: &load.AverageStats{
				Load1:  0.0, // Expect at least 0.0 or more for 1-minute load.
				Load5:  0.0, // Expect at least 0.0 or more for 5-minute load.
				Load15: 0.0, // Expect at least 0.0 or more for 15-minute load.
			},
			wantErr: false,
		},
		{
			name: "when load.Avg errors",
			setupMock: func() func() (*sysLoad.AvgStat, error) {
				return func() (*sysLoad.AvgStat, error) {
					return nil, assert.AnError // Simulate an error
				}
			},
			want:        nil,
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ubuntu := load.NewUbuntuProvider()

			if tc.setupMock != nil {
				ubuntu.Avg = tc.setupMock()
			}

			got, err := ubuntu.GetAverageStats()

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.wantErrType)
				suite.Require().Nil(got)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)

				suite.GreaterOrEqual(got.Load1, tc.want.Load1)
				suite.GreaterOrEqual(got.Load5, tc.want.Load5)
				suite.GreaterOrEqual(got.Load15, tc.want.Load15)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuAvgPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuAvgPublicTestSuite))
}
