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

package mocks

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	// NetworkInterfaceName specifies the name of the network interface used for testing.
	NetworkInterfaceName = "wlp0s20f3"

	// ResolveCommand represents the `resolvectl` command used for resolving network settings.
	ResolveCommand = "resolvectl"
)

// NewPlainManager creates a Mock without defaults.
func NewPlainManager(ctrl *gomock.Controller) *MockManager {
	return NewMockManager(ctrl)
}

// NewGetResolvConfManager creates a DNS Mock for GetResolvConf.
func NewGetResolvConfManager(ctrl *gomock.Controller) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 192.168.1.1 8.8.8.8 8.8.4.4 2001:4860:4860::8888 2001:4860:4860::8844
DNS Domain: example.com local.lan`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)

	return mock
}

// NewGetResolvConfManagerNoDNSDomain creates a DNS Mock for GetResolvConf with no
// DNS Domain.
func NewGetResolvConfManagerNoDNSDomain(ctrl *gomock.Controller) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 192.168.1.1 8.8.8.8 8.8.4.4 2001:4860:4860::8888 2001:4860:4860::8844`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)

	return mock
}

// NewSetResolvConfManager creates a DNS Mock for SetResolvConf.
func NewSetResolvConfManager(ctrl *gomock.Controller) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 8.8.8.8
DNS Servers: 8.8.8.8 9.9.9.9
DNS Domain: foo.local bar.local`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockRunCmdDNS(mock, []string{"8.8.8.8", "9.9.9.9"}, nil)
	mockRunCmdDomain(mock, []string{"foo.local", "bar.local"}, nil)

	return mock
}

// NewSetResolvConfManagerPreserveDNSServers creates a DNS Mock for SetResolvConf
// with existing DNS Servers.
func NewSetResolvConfManagerPreserveDNSServers(ctrl *gomock.Controller) *MockManager {
	initialOutput := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: example.com local.lan `

	subsequentOutput := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 1.1.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: foo.local bar.local`

	mock := NewMockManager(ctrl)

	gomock.InOrder(
		mock.EXPECT().
			RunCmd(ResolveCommand, []string{"status", NetworkInterfaceName}).
			Return(initialOutput, nil),

		mock.EXPECT().
			RunCmd(ResolveCommand, []string{"status", NetworkInterfaceName}).
			Return(subsequentOutput, nil),
	)

	mockRunCmdDNS(mock, []string{"1.1.1.1", "2.2.2.2"}, nil)
	mockRunCmdDomain(mock, []string{"foo.local", "bar.local"}, nil)

	return mock
}

// NewSetResolvConfManagerPreserveDNSDomain creates a DNS Mock for SetResolvConf
// with existing DNS Domain.
func NewSetResolvConfManagerPreserveDNSDomain(ctrl *gomock.Controller) *MockManager {
	initialOutput := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: foo.example.com bar.example.com`

	subsequentOutput := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 1.1.1.1
DNS Servers: 8.8.8.8 9.9.9.9
DNS Domain: foo.example.com bar.example.com`

	mock := NewMockManager(ctrl)

	gomock.InOrder(
		mock.EXPECT().
			RunCmd(ResolveCommand, []string{"status", NetworkInterfaceName}).
			Return(initialOutput, nil),

		mock.EXPECT().
			RunCmd(ResolveCommand, []string{"status", NetworkInterfaceName}).
			Return(subsequentOutput, nil),
	)

	mockRunCmdDNS(mock, []string{"8.8.8.8", "9.9.9.9"}, nil)
	mockRunCmdDomain(mock, []string{"foo.example.com", "bar.example.com"}, nil)

	return mock
}

// NewSetResolvConfManagerSetDNSDomainError creates a DNS Mock for SetResolvConf
// when exec.RunCmd errors setting DNS Domain.
func NewSetResolvConfManagerSetDNSDomainError(ctrl *gomock.Controller) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 8.8.8.8
DNS Servers: 8.8.8.8 9.9.9.9
DNS Domain: foo.local bar.local`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockRunCmdDNS(mock, []string{"8.8.8.8", "9.9.9.9"}, nil)
	mockRunCmdDomain(mock, []string{"foo.local", "bar.local"}, assert.AnError)

	return mock
}

// NewSetResolvConfManagerSetDNSServersError creates a DNS Mock for SetResolvConf
// when exec.RunCmd errors setting DNS Servers.
func NewSetResolvConfManagerSetDNSServersError(ctrl *gomock.Controller) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 8.8.8.8
DNS Servers: 8.8.8.8 9.9.9.9
DNS Domain: foo.local bar.local`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockRunCmdDNS(mock, []string{"8.8.8.8", "9.9.9.9"}, assert.AnError)

	return mock
}

// mockRunCmdStatus sets up a mock for the "status" RunCmd call.
func mockRunCmdStatus(mock *MockManager, output string) {
	mock.EXPECT().
		RunCmd(ResolveCommand, []string{"status", NetworkInterfaceName}).
		Return(output, nil).
		AnyTimes()
}

// mockRunCmdDNS sets up a mock for the "dns" RunCmd call.
func mockRunCmdDNS(mock *MockManager, dnsServers []string, err error) {
	mock.EXPECT().
		RunCmd(ResolveCommand, append([]string{"dns", NetworkInterfaceName}, dnsServers...)).
		Return("", err).
		AnyTimes()
}

// mockRunCmdDomain sets up a mock for the "domain" RunCmd call.
func mockRunCmdDomain(mock *MockManager, domains []string, err error) {
	mock.EXPECT().
		RunCmd(ResolveCommand, append([]string{"domain", NetworkInterfaceName}, domains...)).
		Return("", err).
		AnyTimes()
}
