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

package host_test

import (
	"testing"

	sysHost "github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system/host"
)

type UbuntuGetHostnamePublicTestSuite struct {
	suite.Suite
}

func (suite *UbuntuGetHostnamePublicTestSuite) SetupTest() {}

func (suite *UbuntuGetHostnamePublicTestSuite) TearDownTest() {}

func (suite *UbuntuGetHostnamePublicTestSuite) TestGetHostname() {
	tests := []struct {
		name        string
		setupMock   func() func() (*sysHost.InfoStat, error)
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetHostname Ok",
			setupMock: func() func() (*sysHost.InfoStat, error) {
				return func() (*sysHost.InfoStat, error) {
					return &sysHost.InfoStat{Hostname: "default-hostname"}, nil
				}
			},
			want:    "default-hostname",
			wantErr: false,
		},
		{
			name: "when host.Info errors",
			setupMock: func() func() (*sysHost.InfoStat, error) {
				return func() (*sysHost.InfoStat, error) {
					return nil, assert.AnError
				}
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ubuntu := host.NewUbuntuProvider()

			if tc.setupMock != nil {
				ubuntu.InfoFn = tc.setupMock()
			}

			got, err := ubuntu.GetHostname()

			if tc.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tc.wantErrType.Error())
				suite.Empty(got)
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
func TestUbuntuGetHostnamePublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuGetHostnamePublicTestSuite))
}
