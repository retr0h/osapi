// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package dns_test

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execMocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
)

type DarwinGetResolvConfByInterfacePublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	logger *slog.Logger
}

func (suite *DarwinGetResolvConfByInterfacePublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DarwinGetResolvConfByInterfacePublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DarwinGetResolvConfByInterfacePublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DarwinGetResolvConfByInterfacePublicTestSuite) TestGetResolvConfByInterface() {
	tests := []struct {
		name          string
		setupMock     func() *execMocks.MockManager
		interfaceName string
		want          *dns.GetResult
		wantErr       bool
		wantErrType   error
	}{
		{
			name: "when matching interface found",
			setupMock: func() *execMocks.MockManager {
				mock := execMocks.NewPlainMockManager(suite.ctrl)
				output := `
DNS configuration

resolver #1
  search domain[0] : example.com
  search domain[1] : local.lan
  nameserver[0] : 192.168.1.1
  nameserver[1] : 8.8.8.8
  if_index : 6 (en0)
  flags    : Request A records

resolver #2
  nameserver[0] : 10.0.0.1
  if_index : 7 (en1)
`
				mock.EXPECT().
					RunCmd("scutil", []string{"--dns"}).
					Return(output, nil)

				return mock
			},
			interfaceName: "en0",
			want: &dns.GetResult{
				DNSServers:    []string{"192.168.1.1", "8.8.8.8"},
				SearchDomains: []string{"example.com", "local.lan"},
			},
		},
		{
			name: "when no interface match returns error",
			setupMock: func() *execMocks.MockManager {
				mock := execMocks.NewPlainMockManager(suite.ctrl)
				output := `
DNS configuration

resolver #1
  nameserver[0] : 192.168.1.1
  nameserver[1] : 8.8.8.8
  if_index : 6 (en0)

resolver #2
  nameserver[0] : 10.0.0.1
  if_index : 7 (en1)
`
				mock.EXPECT().
					RunCmd("scutil", []string{"--dns"}).
					Return(output, nil)

				return mock
			},
			interfaceName: "en5",
			wantErr:       true,
			wantErrType:   fmt.Errorf("does not exist"),
		},
		{
			name: "when scutil command errors",
			setupMock: func() *execMocks.MockManager {
				mock := execMocks.NewPlainMockManager(suite.ctrl)

				mock.EXPECT().
					RunCmd("scutil", []string{"--dns"}).
					Return("", assert.AnError)

				return mock
			},
			interfaceName: "en0",
			wantErr:       true,
			wantErrType:   assert.AnError,
		},
		{
			name: "when no nameservers in output",
			setupMock: func() *execMocks.MockManager {
				mock := execMocks.NewPlainMockManager(suite.ctrl)
				output := `
DNS configuration

resolver #1
  search domain[0] : example.com
  if_index : 6 (en0)
`
				mock.EXPECT().
					RunCmd("scutil", []string{"--dns"}).
					Return(output, nil)

				return mock
			},
			interfaceName: "en0",
			wantErr:       true,
			wantErrType:   fmt.Errorf("no resolver blocks found"),
		},
		{
			name: "when empty output",
			setupMock: func() *execMocks.MockManager {
				mock := execMocks.NewPlainMockManager(suite.ctrl)

				mock.EXPECT().
					RunCmd("scutil", []string{"--dns"}).
					Return("", nil)

				return mock
			},
			interfaceName: "en0",
			wantErr:       true,
			wantErrType:   fmt.Errorf("no resolver blocks found"),
		},
		{
			name: "when resolver has no search domains",
			setupMock: func() *execMocks.MockManager {
				mock := execMocks.NewPlainMockManager(suite.ctrl)
				output := `
DNS configuration

resolver #1
  nameserver[0] : 8.8.8.8
  nameserver[1] : 8.8.4.4
  if_index : 6 (en0)
`
				mock.EXPECT().
					RunCmd("scutil", []string{"--dns"}).
					Return(output, nil)

				return mock
			},
			interfaceName: "en0",
			want: &dns.GetResult{
				DNSServers: []string{"8.8.8.8", "8.8.4.4"},
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()

			darwin := dns.NewDarwinProvider(suite.logger, mock)
			got, err := darwin.GetResolvConfByInterface(tc.interfaceName)

			if !tc.wantErr {
				suite.NoError(err)
				suite.Equal(tc.want, got)
			} else {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDarwinGetResolvConfByInterfacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DarwinGetResolvConfByInterfacePublicTestSuite))
}
