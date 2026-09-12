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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/host"
)

type DarwinGetFQDNPublicTestSuite struct {
	suite.Suite
}

func (suite *DarwinGetFQDNPublicTestSuite) SetupTest() {}

func (suite *DarwinGetFQDNPublicTestSuite) TearDownTest() {}

func (suite *DarwinGetFQDNPublicTestSuite) TestGetFQDN() {
	tests := []struct {
		name         string
		setupMock    func(d *host.Darwin)
		validateFunc func(string, error)
	}{
		{
			name: "when GetFQDN Ok",
			setupMock: func(d *host.Darwin) {
				d.HostnameFn = func() (string, error) {
					return "mac-01.local", nil
				}
			},
			validateFunc: func(got string, err error) {
				suite.NoError(err)
				suite.NotNil(got)
				suite.Equal("mac-01.local", got)
			},
		},
		{
			name: "when os.Hostname errors",
			setupMock: func(d *host.Darwin) {
				d.HostnameFn = func() (string, error) {
					return "", assert.AnError
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
				tc.setupMock(darwin)
			}

			tc.validateFunc(darwin.GetFQDN())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDarwinGetFQDNPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DarwinGetFQDNPublicTestSuite))
}
