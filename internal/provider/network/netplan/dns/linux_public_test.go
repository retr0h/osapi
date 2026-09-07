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

package dns_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
)

type LinuxPublicTestSuite struct {
	suite.Suite
}

func (s *LinuxPublicTestSuite) TestGetResolvConfByInterface() {
	tests := []struct {
		name         string
		validateFunc func(any, error)
	}{
		{
			name: "returns error for linux stub",
			validateFunc: func(result any, err error) {
				s.Error(err)
				s.Nil(result)
				s.ErrorIs(err, provider.ErrUnsupported)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			l := &dns.Linux{}
			tt.validateFunc(l.GetResolvConfByInterface("eth0"))
		})
	}
}

func (s *LinuxPublicTestSuite) TestUpdateResolvConfByInterface() {
	s.Run("returns error for linux stub", func() {
		l := &dns.Linux{}
		result, err := l.UpdateResolvConfByInterface(
			[]string{"8.8.8.8"},
			[]string{"example.com"},
			"eth0",
			false,
		)

		s.Error(err)
		s.Nil(result)
		s.ErrorIs(err, provider.ErrUnsupported)
	})
}

func TestLinuxPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LinuxPublicTestSuite))
}
