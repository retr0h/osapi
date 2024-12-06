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
	"log/slog"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/exec/mocks"
	"github.com/retr0h/osapi/internal/provider/network/dns"
)

type UbuntuGetResolvConfPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	logger *slog.Logger
}

func (suite *UbuntuGetResolvConfPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *UbuntuGetResolvConfPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *UbuntuGetResolvConfPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *UbuntuGetResolvConfPublicTestSuite) TestGetResolvConf() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockManager
		want        *dns.Config
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetResolvConf Ok",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewGetResolvConfManager(suite.ctrl)

				return mock
			},
			want: &dns.Config{
				DNSServers: []string{
					"192.168.1.1",
					"8.8.8.8",
					"8.8.4.4",
					"2001:4860:4860::8888",
					"2001:4860:4860::8844",
				},
				SearchDomains: []string{
					"example.com",
					"local.lan",
				},
			},
			wantErr: false,
		},
		{
			name: "when default DNS Domain",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewGetResolvConfManagerNoDNSDomain(suite.ctrl)

				return mock
			},
			want: &dns.Config{
				DNSServers: []string{
					"192.168.1.1",
					"8.8.8.8",
					"8.8.4.4",
					"2001:4860:4860::8888",
					"2001:4860:4860::8844",
				},
				SearchDomains: []string{
					".",
				},
			},
			wantErr: false,
		},
		{
			name: "when exec.RunCmd errors",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainManager(suite.ctrl)

				mock.EXPECT().
					RunCmd(mocks.ResolveCommand, []string{"status", mocks.NetworkInterfaceName}).
					Return("", assert.AnError).
					AnyTimes()

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()

			net := dns.NewUbuntuProvider(suite.logger, mock)
			got, err := net.GetResolvConf()

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
func TestUbuntuGetResolvConfPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuGetResolvConfPublicTestSuite))
}
