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

package netplan_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan"
)

// netplanStatusJSON is a trimmed version of real netplan status output used
// across all status tests.
const netplanStatusJSON = `{
  "netplan-global-state": {"online": true},
  "lo": {
    "index": 1,
    "adminstate": "UP",
    "operstate": "UNKNOWN",
    "type": "ethernet",
    "macaddress": "00:00:00:00:00:00",
    "addresses": [
      {"127.0.0.1": {"prefix": 8}},
      {"::1": {"prefix": 128}}
    ],
    "routes": [
      {"to": "127.0.0.0/8", "family": 2, "type": "local", "scope": "host", "protocol": "kernel", "table": "local"}
    ]
  },
  "wlp0s20f3": {
    "index": 3,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "wifi",
    "backend": "networkd",
    "id": "wlp0s20f3",
    "macaddress": "b0:a4:60:17:cb:90",
    "vendor": "Intel Corporation",
    "addresses": [
      {"192.168.0.241": {"prefix": 16, "flags": ["dhcp"]}},
      {"2600:6c50:4900:c740:b2a4:60ff:fe17:cb90": {"prefix": 64}},
      {"fe80::b2a4:60ff:fe17:cb90": {"prefix": 64, "flags": ["link"]}}
    ],
    "dns_addresses": ["192.168.0.247"],
    "routes": [
      {"to": "default", "family": 2, "via": "192.168.0.1", "metric": 600, "type": "unicast", "scope": "global", "protocol": "dhcp", "table": "main"},
      {"to": "192.168.0.0/16", "family": 2, "metric": 600, "type": "unicast", "scope": "link", "protocol": "kernel", "table": "main"},
      {"to": "192.168.0.241", "family": 2, "type": "local", "scope": "host", "protocol": "kernel", "table": "local"},
      {"to": "192.168.255.255", "family": 2, "type": "broadcast", "scope": "link", "protocol": "kernel", "table": "local"},
      {"to": "2600:6c50:4900:c740::", "family": 10, "type": "anycast", "scope": "global", "protocol": "kernel", "table": "local"},
      {"to": "ff00::/8", "family": 10, "metric": 256, "type": "multicast", "scope": "global", "protocol": "kernel", "table": "local"}
    ]
  },
  "cni0": {
    "index": 5,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "bridge",
    "macaddress": "06:12:3b:bd:0e:d9",
    "addresses": [
      {"10.42.0.1": {"prefix": 24}},
      {"fe80::4c5f:e5ff:fe41:e658": {"prefix": 64, "flags": ["link"]}}
    ],
    "routes": [
      {"to": "10.42.0.0/24", "family": 2, "type": "unicast", "scope": "link", "protocol": "kernel", "table": "main"}
    ],
    "interfaces": ["veth38cc21af"]
  }
}`

type StatusPublicTestSuite struct {
	suite.Suite

	ctrl     *gomock.Controller
	mockExec *execmocks.MockManager
}

func (suite *StatusPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)
}

func (suite *StatusPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *StatusPublicTestSuite) TearDownSubTest() {}

func (suite *StatusPublicTestSuite) TestGetStatus() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func(netplan.Status, error)
	}{
		{
			name: "when netplan status returns valid JSON",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusJSON, nil)
			},
			validateFunc: func(result netplan.Status, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(result)

				// Global state is skipped.
				_, hasGlobal := result["netplan-global-state"]
				suite.False(hasGlobal)

				// lo is parsed.
				lo, ok := result["lo"]
				suite.True(ok)
				suite.Equal(1, lo.Index)
				suite.Equal("00:00:00:00:00:00", lo.MACAddress)

				// wlp0s20f3 is parsed with all fields.
				wifi, ok := result["wlp0s20f3"]
				suite.True(ok)
				suite.Equal(3, wifi.Index)
				suite.Equal("UP", wifi.AdminState)
				suite.Equal("UP", wifi.OperState)
				suite.Equal("wifi", wifi.Type)
				suite.Equal("networkd", wifi.Backend)
				suite.Equal("b0:a4:60:17:cb:90", wifi.MACAddress)
				suite.Equal("Intel Corporation", wifi.Vendor)
				suite.Require().NotEmpty(wifi.Addresses)
				suite.Require().NotEmpty(wifi.Routes)
				suite.Require().NotEmpty(wifi.DNSAddresses)
				suite.Equal("192.168.0.247", wifi.DNSAddresses[0])

				// cni0 has interfaces list.
				cni, ok := result["cni0"]
				suite.True(ok)
				suite.Equal("bridge", cni.Type)
				suite.Require().Len(cni.Interfaces, 1)
				suite.Equal("veth38cc21af", cni.Interfaces[0])
			},
		},
		{
			name: "when netplan command fails",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return("", errors.New("command not found"))
			},
			validateFunc: func(result netplan.Status, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "netplan status:")
			},
		},
		{
			name: "when output is invalid JSON",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return("not valid json", nil)
			},
			validateFunc: func(result netplan.Status, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "parse netplan status:")
			},
		},
		{
			name: "when output contains only global state",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(`{"netplan-global-state": {"online": true}}`, nil)
			},
			validateFunc: func(result netplan.Status, err error) {
				suite.Require().NoError(err)
				suite.Empty(result)
			},
		},
		{
			name: "when interface entry is invalid JSON object",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(`{"netplan-global-state": {"online": true}, "eth0": "not-an-object"}`, nil)
			},
			validateFunc: func(result netplan.Status, err error) {
				suite.Require().NoError(err)
				// Invalid entry is skipped.
				suite.Empty(result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := netplan.GetStatus(suite.mockExec)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *StatusPublicTestSuite) TestInterfaceStatusIPv4() {
	tests := []struct {
		name         string
		iface        netplan.InterfaceStatus
		validateFunc func(string)
	}{
		{
			name: "when IPv4 address exists",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"192.168.0.241": {Prefix: 16, Flags: []string{"dhcp"}}},
				},
			},
			validateFunc: func(result string) {
				suite.Equal("192.168.0.241", result)
			},
		},
		{
			name: "when only link-local IPv4",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"169.254.1.1": {Prefix: 16}},
				},
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when only IPv6 addresses",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"2600:6c50:4900:c740:b2a4:60ff:fe17:cb90": {Prefix: 64}},
				},
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when no addresses",
			iface: netplan.InterfaceStatus{
				Addresses: nil,
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when invalid address string",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"not-an-ip": {Prefix: 24}},
				},
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when link-local skipped and regular IPv4 follows",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"fe80::1": {Prefix: 64, Flags: []string{"link"}}},
					{"10.0.0.5": {Prefix: 24}},
				},
			},
			validateFunc: func(result string) {
				suite.Equal("10.0.0.5", result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := tc.iface.IPv4()

			tc.validateFunc(result)
		})
	}
}

func (suite *StatusPublicTestSuite) TestInterfaceStatusIPv6() {
	tests := []struct {
		name         string
		iface        netplan.InterfaceStatus
		validateFunc func(string)
	}{
		{
			name: "when global IPv6 exists",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"2600:6c50:4900:c740:b2a4:60ff:fe17:cb90": {Prefix: 64}},
				},
			},
			validateFunc: func(result string) {
				suite.Equal("2600:6c50:4900:c740:b2a4:60ff:fe17:cb90", result)
			},
		},
		{
			name: "when only link-local IPv6",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"fe80::b2a4:60ff:fe17:cb90": {Prefix: 64, Flags: []string{"link"}}},
				},
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when only IPv4 addresses",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"192.168.0.1": {Prefix: 24}},
				},
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when invalid address string",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"not-an-ip": {Prefix: 24}},
				},
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
		{
			name: "when no addresses",
			iface: netplan.InterfaceStatus{
				Addresses: nil,
			},
			validateFunc: func(result string) {
				suite.Empty(result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := tc.iface.IPv6()

			tc.validateFunc(result)
		})
	}
}

func (suite *StatusPublicTestSuite) TestInterfaceStatusIsDHCP() {
	tests := []struct {
		name         string
		iface        netplan.InterfaceStatus
		validateFunc func(bool)
	}{
		{
			name: "when address has dhcp flag",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"192.168.0.241": {Prefix: 16, Flags: []string{"dhcp"}}},
				},
			},
			validateFunc: func(result bool) {
				suite.True(result)
			},
		},
		{
			name: "when no dhcp flag",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"10.42.0.1": {Prefix: 24}},
				},
			},
			validateFunc: func(result bool) {
				suite.False(result)
			},
		},
		{
			name: "when no addresses",
			iface: netplan.InterfaceStatus{
				Addresses: nil,
			},
			validateFunc: func(result bool) {
				suite.False(result)
			},
		},
		{
			name: "when address has link flag only",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"fe80::1": {Prefix: 64, Flags: []string{"link"}}},
				},
			},
			validateFunc: func(result bool) {
				suite.False(result)
			},
		},
		{
			name: "when DHCP flag case insensitive",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"192.168.0.1": {Prefix: 24, Flags: []string{"DHCP"}}},
				},
			},
			validateFunc: func(result bool) {
				suite.True(result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := tc.iface.IsDHCP()

			tc.validateFunc(result)
		})
	}
}

func (suite *StatusPublicTestSuite) TestInterfaceStatusHasDefaultRoute() {
	tests := []struct {
		name         string
		iface        netplan.InterfaceStatus
		validateFunc func(bool)
	}{
		{
			name: "when has default route",
			iface: netplan.InterfaceStatus{
				Routes: []netplan.RouteStatus{
					{To: "default", Via: "192.168.0.1", Protocol: "dhcp"},
				},
			},
			validateFunc: func(result bool) {
				suite.True(result)
			},
		},
		{
			name: "when no default route",
			iface: netplan.InterfaceStatus{
				Routes: []netplan.RouteStatus{
					{To: "10.42.0.0/24", Protocol: "kernel"},
				},
			},
			validateFunc: func(result bool) {
				suite.False(result)
			},
		},
		{
			name: "when no routes",
			iface: netplan.InterfaceStatus{
				Routes: nil,
			},
			validateFunc: func(result bool) {
				suite.False(result)
			},
		},
		{
			name: "when default route among many",
			iface: netplan.InterfaceStatus{
				Routes: []netplan.RouteStatus{
					{To: "192.168.0.0/16", Protocol: "kernel"},
					{To: "default", Via: "192.168.0.1", Protocol: "dhcp"},
					{To: "10.0.0.0/8", Via: "192.168.0.1"},
				},
			},
			validateFunc: func(result bool) {
				suite.True(result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := tc.iface.HasDefaultRoute()

			tc.validateFunc(result)
		})
	}
}

func (suite *StatusPublicTestSuite) TestInterfaceStatusAddressFamily() {
	tests := []struct {
		name         string
		iface        netplan.InterfaceStatus
		validateFunc func(string)
	}{
		{
			name: "when first non-link-local is IPv4",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"192.168.0.241": {Prefix: 16}},
				},
			},
			validateFunc: func(result string) {
				suite.Equal("inet", result)
			},
		},
		{
			name: "when first non-link-local is IPv6",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"2600:6c50:4900:c740::1": {Prefix: 64}},
				},
			},
			validateFunc: func(result string) {
				suite.Equal("inet6", result)
			},
		},
		{
			name: "when only link-local addresses defaults to inet",
			iface: netplan.InterfaceStatus{
				Addresses: []map[string]netplan.AddressInfo{
					{"fe80::1": {Prefix: 64, Flags: []string{"link"}}},
				},
			},
			validateFunc: func(result string) {
				suite.Equal("inet", result)
			},
		},
		{
			name: "when no addresses defaults to inet",
			iface: netplan.InterfaceStatus{
				Addresses: nil,
			},
			validateFunc: func(result string) {
				suite.Equal("inet", result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := tc.iface.AddressFamily()

			tc.validateFunc(result)
		})
	}
}

func (suite *StatusPublicTestSuite) TestSectionForInterface() {
	tests := []struct {
		name         string
		setup        func()
		ifaceName    string
		validateFunc func(string)
	}{
		{
			name: "when interface found returns correct section",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusJSON, nil)
			},
			ifaceName: "wlp0s20f3",
			validateFunc: func(result string) {
				suite.Equal("wifis", result)
			},
		},
		{
			name: "when interface found returns bridges section",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusJSON, nil)
			},
			ifaceName: "cni0",
			validateFunc: func(result string) {
				suite.Equal("bridges", result)
			},
		},
		{
			name: "when interface not found falls back to ethernets",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusJSON, nil)
			},
			ifaceName: "nonexistent0",
			validateFunc: func(result string) {
				suite.Equal("ethernets", result)
			},
		},
		{
			name: "when exec error falls back to ethernets",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return("", errors.New("command not found"))
			},
			ifaceName: "eth0",
			validateFunc: func(result string) {
				suite.Equal("ethernets", result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result := netplan.SectionForInterface(suite.mockExec, tc.ifaceName)

			tc.validateFunc(result)
		})
	}
}

func (suite *StatusPublicTestSuite) TestSectionForType() {
	tests := []struct {
		name         string
		ifaceType    string
		validateFunc func(string)
	}{
		{
			name:      "when type is wifi returns wifis",
			ifaceType: "wifi",
			validateFunc: func(result string) {
				suite.Equal("wifis", result)
			},
		},
		{
			name:      "when type is bridge returns bridges",
			ifaceType: "bridge",
			validateFunc: func(result string) {
				suite.Equal("bridges", result)
			},
		},
		{
			name:      "when type is bond returns bonds",
			ifaceType: "bond",
			validateFunc: func(result string) {
				suite.Equal("bonds", result)
			},
		},
		{
			name:      "when type is tunnel returns tunnels",
			ifaceType: "tunnel",
			validateFunc: func(result string) {
				suite.Equal("tunnels", result)
			},
		},
		{
			name:      "when type is vxlan returns tunnels",
			ifaceType: "vxlan",
			validateFunc: func(result string) {
				suite.Equal("tunnels", result)
			},
		},
		{
			name:      "when type is empty returns ethernets",
			ifaceType: "",
			validateFunc: func(result string) {
				suite.Equal("ethernets", result)
			},
		},
		{
			name:      "when type is ethernet returns ethernets",
			ifaceType: "ethernet",
			validateFunc: func(result string) {
				suite.Equal("ethernets", result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := netplan.SectionForType(tc.ifaceType)

			tc.validateFunc(result)
		})
	}
}

func TestStatusPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(StatusPublicTestSuite))
}
