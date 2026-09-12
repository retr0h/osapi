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

package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/node/service"
)

type LinuxPublicTestSuite struct {
	suite.Suite

	provider *service.Linux
}

func (suite *LinuxPublicTestSuite) SetupTest() {
	suite.provider = service.NewLinuxProvider()
}

func (suite *LinuxPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		validateFunc func([]service.Info, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result []service.Info, err error) {
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
		validateFunc func(*service.Info, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.Info, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Get(context.Background(), "nginx"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestCreate() {
	tests := []struct {
		name         string
		validateFunc func(*service.CreateResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(got *service.CreateResult, err error) {
				suite.Nil(got)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Create(
				context.Background(),
				service.Entry{Name: "test"},
			))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		validateFunc func(*service.UpdateResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(got *service.UpdateResult, err error) {
				suite.Nil(got)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Update(
				context.Background(),
				service.Entry{Name: "test"},
			))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestDelete() {
	tests := []struct {
		name         string
		validateFunc func(*service.DeleteResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.DeleteResult, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Delete(context.Background(), "test"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestStart() {
	tests := []struct {
		name         string
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.ActionResult, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Start(context.Background(), "nginx"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestStop() {
	tests := []struct {
		name         string
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.ActionResult, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Stop(context.Background(), "nginx"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestRestart() {
	tests := []struct {
		name         string
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.ActionResult, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Restart(context.Background(), "nginx"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestEnable() {
	tests := []struct {
		name         string
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.ActionResult, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Enable(context.Background(), "nginx"))
		})
	}
}

func (suite *LinuxPublicTestSuite) TestDisable() {
	tests := []struct {
		name         string
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name: "returns not implemented error",
			validateFunc: func(result *service.ActionResult, err error) {
				suite.Nil(result)
				suite.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(suite.provider.Disable(context.Background(), "nginx"))
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
