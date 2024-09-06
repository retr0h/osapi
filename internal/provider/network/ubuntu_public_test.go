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

package network_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/network"
)

type UbuntuPublicTestSuite struct {
	suite.Suite

	appFs          afero.Fs
	resolvConfFile string
}

func (suite *UbuntuPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.resolvConfFile = "/run/systemd/resolve/resolv.conf"
}

func (suite *UbuntuPublicTestSuite) SetupSubTest() {
	// initializes a new afero.Fs in the table tests
	suite.SetupTest()
}

func (suite *UbuntuPublicTestSuite) TearDownTest() {}

func (suite *UbuntuPublicTestSuite) TestGetResolvConf() {
	tests := []struct {
		name    string
		content []byte
		want    *network.DNSConfig
		wantErr bool
	}{
		{
			name: "when GetResolvConf Ok",
			content: []byte(`
nameserver 192.168.1.1
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 2001:4860:4860::8888
nameserver 2001:4860:4860::8844
search example.com local.lan
options edns0`),
			want: &network.DNSConfig{
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
			name:    "when GetResolvConf missing resolvConfFile",
			content: []byte{},
			want:    &network.DNSConfig{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if len(tc.content) != 0 {
				_ = afero.WriteFile(suite.appFs, suite.resolvConfFile, tc.content, 0o644)
			}

			net := network.NewUbuntuProvider(suite.appFs)
			got, err := net.GetResolvConf()

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuPublicTestSuite))
}
