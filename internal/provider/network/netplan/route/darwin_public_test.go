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

package route_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/route"
)

type DarwinRoutePublicTestSuite struct {
	suite.Suite

	provider *route.Darwin
}

func (suite *DarwinRoutePublicTestSuite) SetupTest() {
	suite.provider = route.NewDarwinProvider()
}

func (suite *DarwinRoutePublicTestSuite) TestList() {
	tests := []struct {
		name string
	}{
		{
			name: "returns not implemented error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got, err := suite.provider.List(context.Background())

			suite.Nil(got)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *DarwinRoutePublicTestSuite) TestGet() {
	tests := []struct {
		name string
	}{
		{
			name: "returns not implemented error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got, err := suite.provider.Get(context.Background(), "eth0")

			suite.Nil(got)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *DarwinRoutePublicTestSuite) TestCreate() {
	tests := []struct {
		name string
	}{
		{
			name: "returns not implemented error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got, err := suite.provider.Create(context.Background(), route.Entry{})

			suite.Nil(got)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *DarwinRoutePublicTestSuite) TestUpdate() {
	tests := []struct {
		name string
	}{
		{
			name: "returns not implemented error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got, err := suite.provider.Update(context.Background(), route.Entry{})

			suite.Nil(got)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func (suite *DarwinRoutePublicTestSuite) TestDelete() {
	tests := []struct {
		name string
	}{
		{
			name: "returns not implemented error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got, err := suite.provider.Delete(context.Background(), "eth0")

			suite.Nil(got)
			suite.ErrorIs(err, provider.ErrUnsupported)
		})
	}
}

func TestDarwinRoutePublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(DarwinRoutePublicTestSuite))
}
