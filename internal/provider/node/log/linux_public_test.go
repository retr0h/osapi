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

package log_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
	oslog "github.com/osapi-io/osapi/internal/provider/node/log"
)

type LinuxPublicTestSuite struct {
	suite.Suite

	provider *oslog.Linux
}

func (suite *LinuxPublicTestSuite) SetupTest() {
	suite.provider = oslog.NewLinuxProvider()
}

func (suite *LinuxPublicTestSuite) TestQuery() {
	tests := []struct {
		name         string
		validateFunc func(any, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result any, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Query(context.Background(), oslog.QueryOpts{}))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestQueryUnit() {
	tests := []struct {
		name string
	}{
		{
			name: "returns not implemented error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got, err := suite.provider.QueryUnit(
				context.Background(),
				"nginx.service",
				oslog.QueryOpts{},
			)

			suite.Nil(got)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *LinuxPublicTestSuite) TestListSources() {
	tests := []struct {
		name         string
		validateFunc func(any, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result any, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.ListSources(context.Background()))
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestLinuxPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LinuxPublicTestSuite))
}
