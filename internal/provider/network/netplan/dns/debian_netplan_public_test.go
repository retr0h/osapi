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

package dns_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
)

type DebianNetplanPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
}

func (suite *DebianNetplanPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DebianNetplanPublicTestSuite) TestGenerateDNSNetplanYAML() {
	tests := []struct {
		name          string
		interfaceName string
		servers       []string
		searchDomains []string
		validateFunc  func(got []byte)
	}{
		{
			name:          "when servers and search domains are provided",
			interfaceName: "eth0",
			servers:       []string{"8.8.8.8", "8.8.4.4"},
			searchDomains: []string{"example.com", "local.lan"},
			validateFunc: func(got []byte) {
				content := string(got)
				suite.Contains(content, "addresses:")
				suite.Contains(content, "- 8.8.8.8")
				suite.Contains(content, "- 8.8.4.4")
				suite.Contains(content, "search:")
				suite.Contains(content, "- example.com")
				suite.Contains(content, "- local.lan")
			},
		},
		{
			name:          "when only servers are provided",
			interfaceName: "eth0",
			servers:       []string{"1.1.1.1"},
			searchDomains: nil,
			validateFunc: func(got []byte) {
				content := string(got)
				suite.Contains(content, "addresses:")
				suite.Contains(content, "- 1.1.1.1")
				suite.NotContains(content, "search:")
			},
		},
		{
			name:          "when only search domains are provided",
			interfaceName: "eth0",
			servers:       nil,
			searchDomains: []string{"example.com"},
			validateFunc: func(got []byte) {
				content := string(got)
				suite.NotContains(content, "addresses:")
				suite.Contains(content, "search:")
				suite.Contains(content, "- example.com")
			},
		},
		{
			name:          "when interface name appears in output",
			interfaceName: "wlp0s20f3",
			servers:       []string{"8.8.8.8"},
			searchDomains: nil,
			validateFunc: func(got []byte) {
				suite.Contains(string(got), "wlp0s20f3:")
			},
		},
		{
			name:          "when multiple servers each appear on own line",
			interfaceName: "eth0",
			servers:       []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			searchDomains: nil,
			validateFunc: func(got []byte) {
				content := string(got)
				suite.Contains(content, "- 192.168.1.1\n")
				suite.Contains(content, "- 192.168.1.2\n")
				suite.Contains(content, "- 192.168.1.3\n")
			},
		},
		{
			name:          "when IPv6 servers are provided",
			interfaceName: "eth0",
			servers:       []string{"2001:4860:4860::8888", "2001:4860:4860::8844"},
			searchDomains: nil,
			validateFunc: func(got []byte) {
				content := string(got)
				suite.Contains(content, "- 2001:4860:4860::8888")
				suite.Contains(content, "- 2001:4860:4860::8844")
			},
		},
		{
			name:          "when overrideDHCP is true adds DHCP overrides",
			interfaceName: "eth0",
			servers:       []string{"8.8.8.8"},
			searchDomains: nil,
			validateFunc: func(got []byte) {
				content := string(got)
				suite.Contains(content, "dhcp4-overrides:")
				suite.Contains(content, "use-dns: false")
				suite.Contains(content, "dhcp6-overrides:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			overrideDHCP := tc.name == "when overrideDHCP is true adds DHCP overrides"
			got := dns.ExportGenerateDNSNetplanYAML(
				tc.interfaceName,
				"ethernets",
				tc.servers,
				tc.searchDomains,
				overrideDHCP,
			)

			tc.validateFunc(got)
		})
	}
}

func (suite *DebianNetplanPublicTestSuite) TestDNSNetplanPath() {
	tests := []struct {
		name         string
		validateFunc func(got string)
	}{
		{
			name: "when path is returned",
			validateFunc: func(got string) {
				suite.Equal("/etc/netplan/osapi-dns.yaml", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := dns.ExportDNSNetplanPath()

			tc.validateFunc(got)
		})
	}
}

func (suite *DebianNetplanPublicTestSuite) TestResolvePrimaryInterface() {
	tests := []struct {
		name          string
		interfaceName string
		setupFacts    func(p *dns.Debian)
		want          string
	}{
		{
			name:          "when explicit interface name is provided",
			interfaceName: "enp3s0",
			setupFacts:    func(_ *dns.Debian) {},
			want:          "enp3s0",
		},
		{
			name:          "when empty interface and facts has primary_interface",
			interfaceName: "",
			setupFacts: func(p *dns.Debian) {
				p.SetFactsFunc(func() map[string]any {
					return map[string]any{
						"primary_interface": "ens3",
					}
				})
			},
			want: "ens3",
		},
		{
			name:          "when empty interface and no facts",
			interfaceName: "",
			setupFacts:    func(_ *dns.Debian) {},
			want:          "eth0",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctrl := gomock.NewController(suite.T())
			mock := mocks.NewPlainMockManager(ctrl)

			p := dns.NewDebianProvider(suite.logger, memfs.New(), nil, mock, "test-host")
			tc.setupFacts(p)

			got := p.ExportResolvePrimaryInterface(tc.interfaceName)

			suite.Equal(tc.want, got)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianNetplanPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianNetplanPublicTestSuite))
}
