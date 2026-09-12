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

package host_test

import (
	"testing"

	sysHost "github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/host"
)

type DarwinGetKernelVersionPublicTestSuite struct {
	suite.Suite
}

func (suite *DarwinGetKernelVersionPublicTestSuite) SetupTest() {}

func (suite *DarwinGetKernelVersionPublicTestSuite) TearDownTest() {}

func (suite *DarwinGetKernelVersionPublicTestSuite) TestGetKernelVersion() {
	tests := []struct {
		name         string
		setupMock    func() func() (*sysHost.InfoStat, error)
		validateFunc func(string, error)
	}{
		{
			name: "when GetKernelVersion Ok",
			setupMock: func() func() (*sysHost.InfoStat, error) {
				return func() (*sysHost.InfoStat, error) {
					return &sysHost.InfoStat{KernelVersion: "24.3.0"}, nil
				}
			},
			validateFunc: func(got string, err error) {
				suite.NoError(err)
				suite.NotNil(got)
				suite.Equal("24.3.0", got)
			},
		},
		{
			name: "when host.Info errors",
			setupMock: func() func() (*sysHost.InfoStat, error) {
				return func() (*sysHost.InfoStat, error) {
					return nil, assert.AnError
				}
			},
			validateFunc: func(got string, err error) {
				suite.Error(err)
				suite.ErrorContains(err, assert.AnError.Error())
				suite.Empty(got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			darwin := host.NewDarwinProvider()

			if tc.setupMock != nil {
				darwin.InfoFn = tc.setupMock()
			}

			tc.validateFunc(darwin.GetKernelVersion())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDarwinGetKernelVersionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DarwinGetKernelVersionPublicTestSuite))
}
