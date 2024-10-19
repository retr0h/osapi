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

package mem_test

import (
	"testing"

	sysMem "github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system/mem"
)

type UbuntuVMPublicTestSuite struct {
	suite.Suite
}

func (suite *UbuntuVMPublicTestSuite) SetupTest() {}

func (suite *UbuntuVMPublicTestSuite) TearDownTest() {}

func (suite *UbuntuVMPublicTestSuite) TestGetStats() {
	tests := []struct {
		name        string
		setupMock   func() func() (*sysMem.VirtualMemoryStat, error)
		want        *mem.Stats
		wantErr     bool
		wantErrType error
	}{
		{
			name:      "when GetStats Ok",
			setupMock: nil,
			want: &mem.Stats{
				Total:  1, // Expect at least 1 byte of total memory
				Free:   0, // Free memory can be zero or more
				Cached: 0, // Cached memory can be zero or more
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
				ubuntu.VirtualMemory = tc.setupMock()
			}

			got, err := ubuntu.GetStats()

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.wantErrType)
				suite.Require().Nil(got)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)

				suite.Greater(got.Total, tc.want.Total)
				suite.GreaterOrEqual(got.Free, tc.want.Free)
				suite.GreaterOrEqual(got.Cached, tc.want.Cached)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuVMPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuVMPublicTestSuite))
}
