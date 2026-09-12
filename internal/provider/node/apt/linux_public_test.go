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

package apt_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/node/apt"
)

type LinuxPublicTestSuite struct {
	suite.Suite

	provider *apt.Linux
}

func (suite *LinuxPublicTestSuite) SetupTest() {
	suite.provider = apt.NewLinuxProvider()
}

func (suite *LinuxPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		validateFunc func([]apt.Package, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result []apt.Package, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.List(context.Background()))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		validateFunc func(*apt.Package, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *apt.Package, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Get(context.Background(), "vim"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestInstall() {
	tests := []struct {
		name         string
		validateFunc func(*apt.Result, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *apt.Result, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Install(context.Background(), "vim"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestRemove() {
	tests := []struct {
		name         string
		validateFunc func(*apt.Result, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *apt.Result, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Remove(context.Background(), "vim"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		validateFunc func(*apt.Result, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *apt.Result, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Update(context.Background()))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestListUpdates() {
	tests := []struct {
		name         string
		validateFunc func([]apt.Update, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result []apt.Update, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.ListUpdates(context.Background()))
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
