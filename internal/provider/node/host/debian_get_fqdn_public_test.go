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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/host"
)

type DebianGetFQDNPublicTestSuite struct {
	suite.Suite
}

func (suite *DebianGetFQDNPublicTestSuite) SetupTest() {}

func (suite *DebianGetFQDNPublicTestSuite) TearDownTest() {}

func (suite *DebianGetFQDNPublicTestSuite) TestGetFQDN() {
	tests := []struct {
		name        string
		setupMock   func(u *host.Debian)
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetFQDN Ok",
			setupMock: func(u *host.Debian) {
				u.HostnameFn = func() (string, error) {
					return "node-01.example.com", nil
				}
			},
			want:    "node-01.example.com",
			wantErr: false,
		},
		{
			name: "when os.Hostname errors",
			setupMock: func(u *host.Debian) {
				u.HostnameFn = func() (string, error) {
					return "", assert.AnError
				}
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			debian := host.NewDebianProvider(nil)

			if tc.setupMock != nil {
				tc.setupMock(debian)
			}

			got, err := debian.GetFQDN()

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
func TestDebianGetFQDNPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianGetFQDNPublicTestSuite))
}
