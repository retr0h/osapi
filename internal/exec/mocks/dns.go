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

// Package mocks provides mock implementations for testing.
package mocks

import (
	"go.uber.org/mock/gomock"
)

const (
	// NetworkInterfaceName specifies the name of the network interface used for testing.
	NetworkInterfaceName = "wlp0s20f3"

	// ResolveCommand represents the `resolvectl` command used for resolving network settings.
	ResolveCommand = "resolvectl"

	// NetplanCommand represents the `netplan` command used for applying network configuration.
	NetplanCommand = "netplan"
)

// NewPlainMockManager creates a Mock without defaults.
func NewPlainMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	return NewMockManager(ctrl)
}

// NewDefaultMockManager creates a Mock with defaults.
func NewDefaultMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	mock := NewPlainMockManager(ctrl)

	// Add common expectations here if needed
	return mock
}

// NewGetResolvConfMockManager creates a DNS Mock for GetResolvConf.
func NewGetResolvConfMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 192.168.1.1 8.8.8.8 8.8.4.4 2001:4860:4860::8888 2001:4860:4860::8844
DNS Domain: example.com local.lan
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)

	return mock
}

// NewGetResolvConfNoDNSDomainMockManager creates a DNS Mock for GetResolvConf with no
// DNS Domain.
func NewGetResolvConfNoDNSDomainMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 192.168.1.1 8.8.8.8 8.8.4.4 2001:4860:4860::8888 2001:4860:4860::8844
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)

	return mock
}

// NewSetResolvConfMockManager creates a DNS Mock for UpdateResolvConfByInterface
// with new servers and domains that differ from the existing config.
func NewSetResolvConfMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 1.1.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: old.local
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockNetplanStatus(mock)
	mockNetplanApply(mock, nil, nil)

	return mock
}

// NewSetResolvConfPreserveDNSServersMockManager creates a DNS Mock for
// UpdateResolvConfByInterface with existing DNS Servers preserved.
func NewSetResolvConfPreserveDNSServersMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: example.com local.lan
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockNetplanStatus(mock)
	mockNetplanApply(mock, nil, nil)

	return mock
}

// NewSetResolvConfPreserveDNSDomainMockManager creates a DNS Mock for
// UpdateResolvConfByInterface with existing DNS Domain preserved.
func NewSetResolvConfPreserveDNSDomainMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: foo.example.com bar.example.com
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockNetplanStatus(mock)
	mockNetplanApply(mock, nil, nil)

	return mock
}

// NewSetResolvConfFiltersRootDNSDomainMockManager creates a DNS Mock for
// UpdateResolvConfByInterface with no DNS Domain (only root ".").
func NewSetResolvConfFiltersRootDNSDomainMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
DNS Servers: 1.1.1.1 2.2.2.2
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockNetplanStatus(mock)
	mockNetplanApply(mock, nil, nil)

	return mock
}

// NewSetResolvConfNetplanGenerateErrorMockManager creates a DNS Mock for
// UpdateResolvConfByInterface when `netplan generate` fails.
func NewSetResolvConfNetplanGenerateErrorMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	// Initial state must differ from desired so the update proceeds.
	output := `
Current Scopes: DNS
Protocols: +DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 1.1.1.1
DNS Servers: 1.1.1.1 2.2.2.2
DNS Domain: old.local
`

	mock := NewMockManager(ctrl)

	mockRunCmdStatus(mock, output)
	mockNetplanStatus(mock)

	return mock
}

// NewSetResolvConfSetDNSDomainErrorMockManager creates a DNS Mock for
// UpdateResolvConfByInterface when the write path fails. Kept for
// backwards compatibility with existing test names.
func NewSetResolvConfSetDNSDomainErrorMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	return NewSetResolvConfNetplanGenerateErrorMockManager(ctrl)
}

// NewSetResolvConfSetDNSServersErrorMockManager creates a DNS Mock for
// UpdateResolvConfByInterface when the write path fails. Kept for
// backwards compatibility with existing test names.
func NewSetResolvConfSetDNSServersErrorMockManager(
	ctrl *gomock.Controller,
) *MockManager {
	return NewSetResolvConfNetplanGenerateErrorMockManager(ctrl)
}

// mockRunCmdStatus sets up a mock for the "status" RunCmd call.
func mockRunCmdStatus(
	mock *MockManager,
	output string,
) {
	mock.EXPECT().
		RunCmd(ResolveCommand, []string{"status", NetworkInterfaceName}).
		Return(output, nil).
		AnyTimes()
}

// mockNetplanStatus sets up a mock for `netplan status --format json`.
func mockNetplanStatus(
	mock *MockManager,
) {
	statusJSON := `{"` + NetworkInterfaceName + `": {"type": "wifi", "macaddress": "b0:a4:60:17:cb:90"}}`
	mock.EXPECT().
		RunCmd(NetplanCommand, []string{"status", "--format", "json"}).
		Return(statusJSON, nil).
		AnyTimes()
}

// mockNetplanApply sets up mocks for `netplan generate` and `netplan apply`.
func mockNetplanApply(
	mock *MockManager,
	genErr error,
	applyErr error,
) {
	mock.EXPECT().
		RunPrivilegedCmd(NetplanCommand, []string{"generate"}).
		Return("", genErr).
		AnyTimes()

	if genErr == nil {
		mock.EXPECT().
			RunPrivilegedCmd(NetplanCommand, []string{"apply"}).
			Return("", applyErr).
			AnyTimes()
	}
}
