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

package load_test

import (
	"testing"

	sysLoad "github.com/shirou/gopsutil/v4/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/load"
)

type DarwinGetAverageStatsPublicTestSuite struct {
	suite.Suite
}

func (suite *DarwinGetAverageStatsPublicTestSuite) SetupTest() {}

func (suite *DarwinGetAverageStatsPublicTestSuite) TearDownTest() {}

func (suite *DarwinGetAverageStatsPublicTestSuite) TestGetAverageStats() {
	tests := []struct {
		name         string
		setupMock    func() func() (*sysLoad.AvgStat, error)
		validateFunc func(any, error)
	}{
		{
			name: "when GetAverageStats Ok",
			setupMock: func() func() (*sysLoad.AvgStat, error) {
				return func() (*sysLoad.AvgStat, error) {
					return &sysLoad.AvgStat{
						Load1:  1.0,
						Load5:  0.5,
						Load15: 0.2,
					}, nil
				}
			},
			validateFunc: func(got any, err error) {
				suite.NoError(err)
				suite.NotNil(got)
				suite.Equal(&load.Result{
					Load1:  1.0,
					Load5:  0.5,
					Load15: 0.2,
				}, got)
			},
		},
		{
			name: "when load.Avg errors",
			setupMock: func() func() (*sysLoad.AvgStat, error) {
				return func() (*sysLoad.AvgStat, error) {
					return nil, assert.AnError
				}
			},
			validateFunc: func(got any, err error) {
				suite.Error(err)
				suite.ErrorContains(err, assert.AnError.Error())
				suite.Nil(got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			darwin := load.NewDarwinProvider()

			if tc.setupMock != nil {
				darwin.AvgFn = tc.setupMock()
			}

			tc.validateFunc(darwin.GetAverageStats())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDarwinGetAverageStatsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DarwinGetAverageStatsPublicTestSuite))
}
