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

	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/network/dns"
)

type UbuntuSetResolvConfPublicTestSuite struct {
	suite.Suite

	appFs          afero.Fs
	logger         *slog.Logger
	resolvConfFile string
}

func (suite *UbuntuSetResolvConfPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.resolvConfFile = "/run/systemd/resolve/resolv.conf"
}

func (suite *UbuntuSetResolvConfPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *UbuntuSetResolvConfPublicTestSuite) TearDownTest() {}

func (suite *UbuntuSetResolvConfPublicTestSuite) TestSetResolvConf() {
	tests := []struct {
		name          string
		content       []byte
		servers       []string
		searchDomains []string
		want          *dns.Config
		wantErr       bool
		wantErrType   error
	}{
		{
			name: "when SetResolvConf Ok",
			content: []byte(`
nameserver 1.1.1.1
nameserver 2.2.2.2
search foo.example.com bar.example.com
`),
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
			content: []byte(`
nameserver 1.1.1.1
nameserver 2.2.2.2
search foo.example.com bar.example.com
`),
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
			content: []byte(`
nameserver 1.1.1.1
nameserver 2.2.2.2
search foo.example.com bar.example.com
`),
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
			name:        "when SetResolvConf missing args errors",
			wantErr:     true,
			wantErrType: fmt.Errorf("no DNS servers or search domains provided; nothing to update"),
		},
		{
			name:    "when SetResolvConf missing resolvConfFile errors",
			content: []byte{},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			want:        &dns.Config{},
			wantErr:     true,
			wantErrType: fmt.Errorf("failed to get current resolv.conf"),
		},
		// {
		// 	name: "when scanner.Err errors",
		// 	// Write a large amount of data to trigger scanner's buffer size limit.
		// 	content: make([]byte, 10*1024*1024),
		// 	want:    &dns.Config{},
		// 	wantErr: true,
		// },
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if len(tc.content) != 0 {
				_ = afero.WriteFile(suite.appFs, suite.resolvConfFile, tc.content, 0o644)
			}

			net := dns.NewUbuntuProvider(suite.appFs, suite.logger)
			err := net.SetResolvConf(tc.servers, tc.searchDomains)

			if tc.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tc.wantErrType.Error())
			} else {
				suite.NoError(err)
				got, err := net.GetResolvConf()
				suite.Equal(tc.want, got)
				suite.NoError(err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuSetResolvConfPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuSetResolvConfPublicTestSuite))
}
