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

package mem_test

import (
	"testing"

	sysMem "github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system/mem"
)

type UbuntuGetStatsPublicTestSuite struct {
	suite.Suite
}

func (suite *UbuntuGetStatsPublicTestSuite) SetupTest() {}

func (suite *UbuntuGetStatsPublicTestSuite) TearDownTest() {}

func (suite *UbuntuGetStatsPublicTestSuite) TestGetStats() {
	tests := []struct {
		name        string
		setupMock   func() func() (*sysMem.VirtualMemoryStat, error)
		want        *mem.Stats
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetStats Ok",
			setupMock: func() func() (*sysMem.VirtualMemoryStat, error) {
				return func() (*sysMem.VirtualMemoryStat, error) {
					return &sysMem.VirtualMemoryStat{
						Total:  1024,
						Free:   512,
						Cached: 256,
					}, nil
				}
			},
			want: &mem.Stats{
				Total:  1024,
				Free:   512,
				Cached: 256,
			},
			wantErr: false,
		},
		{
			name: "when mem.VirtualMemory errors",
			setupMock: func() func() (*sysMem.VirtualMemoryStat, error) {
				return func() (*sysMem.VirtualMemoryStat, error) {
					return nil, assert.AnError
				}
			},
			want:        nil,
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ubuntu := mem.NewUbuntuProvider()

			if tc.setupMock != nil {
				ubuntu.VirtualMemoryFn = tc.setupMock()
			}

			got, err := ubuntu.GetStats()

			if tc.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tc.wantErrType.Error())
				suite.Nil(got)
			} else {
				suite.NoError(err)
				suite.NotNil(got)
				suite.Equal(tc.want, got)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuGetStatsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuGetStatsPublicTestSuite))
}
