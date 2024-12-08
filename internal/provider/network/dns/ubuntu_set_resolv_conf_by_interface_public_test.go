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
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/exec/mocks"
	"github.com/retr0h/osapi/internal/provider/network/dns"
)

type UbuntuSetResolvConfByInterfacePublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	logger *slog.Logger
}

func (suite *UbuntuSetResolvConfByInterfacePublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *UbuntuSetResolvConfByInterfacePublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *UbuntuSetResolvConfByInterfacePublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *UbuntuSetResolvConfByInterfacePublicTestSuite) TestSetResolvConfByInterface() {
	tests := []struct {
		name          string
		setupMock     func() *mocks.MockManager
		servers       []string
		searchDomains []string
		interfaceName string
		want          *dns.Config
		wantErr       bool
		wantErrType   error
	}{
		{
			name: "when SetResolvConf Ok",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewSetResolvConfManager(suite.ctrl)

				return mock
			},
			interfaceName: "wlp0s20f3",
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			want: &dns.Config{
				DNSServers: []string{
					"8.8.8.8",
					"9.9.9.9",
				},
				SearchDomains: []string{
					"foo.local",
					"bar.local",
				},
			},
			wantErr: false,
		},
		{
			name: "when SetResolvConf preserves existing servers Ok",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewSetResolvConfManagerPreserveDNSServers(suite.ctrl)

				return mock
			},
			interfaceName: "wlp0s20f3",
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			want: &dns.Config{
				DNSServers: []string{
					"1.1.1.1",
					"2.2.2.2",
				},
				SearchDomains: []string{
					"foo.local",
					"bar.local",
				},
			},
			wantErr: false,
		},
		{
			name: "when SetResolvConf preserves existing search domains Ok",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewSetResolvConfManagerPreserveDNSDomain(suite.ctrl)

				return mock
			},
			interfaceName: "wlp0s20f3",
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			want: &dns.Config{
				DNSServers: []string{
					"8.8.8.8",
					"9.9.9.9",
				},
				SearchDomains: []string{
					"foo.example.com",
					"bar.example.com",
				},
			},
			wantErr: false,
		},
		{
			name:    "when SetResolvConf missing args errors",
			wantErr: true,
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainManager(suite.ctrl)

				return mock
			},
			interfaceName: "wlp0s20f3",
			wantErrType:   fmt.Errorf("no DNS servers or search domains provided; nothing to update"),
		},
		{
			name: "when GetResolvConfByInterface errors",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainManager(suite.ctrl)

				mock.EXPECT().
					RunCmd(mocks.ResolveCommand, []string{"status", mocks.NetworkInterfaceName}).
					Return("", assert.AnError).
					AnyTimes()

				return mock
			},
			interfaceName: "wlp0s20f3",
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when exec.RunCmd setting DNS Domain errors",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewSetResolvConfManagerSetDNSDomainError(suite.ctrl)

				return mock
			},
			interfaceName: "wlp0s20f3",
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when exec.RunCmd setting DNS Servers errors",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewSetResolvConfManagerSetDNSServersError(suite.ctrl)

				return mock
			},
			interfaceName: "wlp0s20f3",
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.Run(tc.name, func() {
				mock := tc.setupMock()

				net := dns.NewUbuntuProvider(suite.logger, mock)
				err := net.SetResolvConfByInterface(tc.servers, tc.searchDomains, tc.interfaceName)

				if tc.wantErr {
					suite.Error(err)
					suite.Contains(err.Error(), tc.wantErrType.Error())
				} else {
					suite.NoError(err)

					got, err := net.GetResolvConfByInterface(tc.interfaceName)
					suite.Equal(tc.want, got)
					suite.NoError(err)
				}
			})
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuSetResolvConfByInterfacePublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuSetResolvConfByInterfacePublicTestSuite))
}
