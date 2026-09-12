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

package iface_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/iface"
)

const testHostname = "test-host"

// netplanStatusTwoIfaces is a netplan status JSON fixture with two
// interfaces: eth0 (DHCP, default route) and eth1 (static, no default).
const netplanStatusTwoIfaces = `{
  "netplan-global-state": {"online": true},
  "lo": {
    "index": 1,
    "adminstate": "UP",
    "operstate": "UNKNOWN",
    "type": "ethernet",
    "macaddress": "00:00:00:00:00:00",
    "addresses": [{"127.0.0.1": {"prefix": 8}}],
    "routes": []
  },
  "eth0": {
    "index": 2,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "ethernet",
    "macaddress": "aa:bb:cc:dd:ee:f0",
    "addresses": [
      {"10.0.0.5": {"prefix": 24, "flags": ["dhcp"]}}
    ],
    "routes": [
      {"to": "default", "via": "10.0.0.1", "family": 2, "metric": 100, "type": "unicast", "scope": "global", "protocol": "dhcp", "table": "main"}
    ]
  },
  "eth1": {
    "index": 3,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "ethernet",
    "macaddress": "aa:bb:cc:dd:ee:f1",
    "addresses": [
      {"10.0.1.5": {"prefix": 24}}
    ],
    "routes": [
      {"to": "10.0.1.0/24", "family": 2, "type": "unicast", "scope": "link", "protocol": "kernel", "table": "main"}
    ]
  }
}`

// netplanStatusSingleIface is a netplan status with one interface.
const netplanStatusSingleIface = `{
  "netplan-global-state": {"online": true},
  "eth0": {
    "index": 2,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "ethernet",
    "macaddress": "aa:bb:cc:dd:ee:f0",
    "addresses": [
      {"10.0.0.5": {"prefix": 24}}
    ],
    "routes": []
  }
}`

// netplanStatusEmpty has only global state and loopback.
const netplanStatusEmpty = `{
  "netplan-global-state": {"online": true},
  "lo": {
    "index": 1,
    "adminstate": "UP",
    "operstate": "UNKNOWN",
    "type": "ethernet",
    "macaddress": "00:00:00:00:00:00",
    "addresses": [{"127.0.0.1": {"prefix": 8}}],
    "routes": []
  }
}`

// netplanStatusSameIndexIfaces has two non-loopback interfaces that share
// the same index, exercising the sort fallback to alphabetical name order.
// Interfaces are listed in JSON with "zeth" before "aeth" so the sort has
// to reorder them for the alphabetical fallback to be observable.
const netplanStatusSameIndexIfaces = `{
  "netplan-global-state": {"online": true},
  "zeth": {
    "index": 2,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "ethernet",
    "macaddress": "aa:bb:cc:dd:ee:f2",
    "addresses": [
      {"10.0.2.5": {"prefix": 24}}
    ],
    "routes": []
  },
  "aeth": {
    "index": 2,
    "adminstate": "UP",
    "operstate": "UP",
    "type": "ethernet",
    "macaddress": "aa:bb:cc:dd:ee:f3",
    "addresses": [
      {"10.0.3.5": {"prefix": 24}}
    ],
    "routes": []
  }
}`

type InterfacePublicTestSuite struct {
	suite.Suite

	ctrl        *gomock.Controller
	ctx         context.Context
	logger      *slog.Logger
	memFs       avfs.VFS
	mockStateKV *jobmocks.MockKeyValue
	mockExec    *execmocks.MockManager
	provider    *iface.Debian
}

func (suite *InterfacePublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockStateKV = jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	_ = suite.memFs.MkdirAll("/etc/netplan", 0o755)

	suite.provider = iface.NewDebianProvider(
		suite.logger,
		suite.memFs,
		suite.mockStateKV,
		suite.mockExec,
		testHostname,
	)
}

func (suite *InterfacePublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *InterfacePublicTestSuite) TearDownSubTest() {}

func (suite *InterfacePublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func([]iface.InterfaceEntry, error)
	}{
		{
			name: "when interfaces exist",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusTwoIfaces, nil)
			},
			validateFunc: func(result []iface.InterfaceEntry, err error) {
				suite.Require().NoError(err)
				// lo is filtered, so 2 interfaces.
				suite.Require().Len(result, 2)

				// Sorted by index: eth0 (index 2) first, eth1 (index 3) second.
				suite.Equal("eth0", result[0].Name)
				suite.Equal("10.0.0.5", result[0].IPv4)
				suite.Equal("aa:bb:cc:dd:ee:f0", result[0].MAC)
				suite.True(result[0].Primary)
				suite.Require().NotNil(result[0].DHCP4)
				suite.True(*result[0].DHCP4)

				suite.Equal("eth1", result[1].Name)
				suite.Equal("10.0.1.5", result[1].IPv4)
				suite.False(result[1].Primary)
				suite.Require().NotNil(result[1].DHCP4)
				suite.False(*result[1].DHCP4)
			},
		},
		{
			name: "when single interface exists",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusSingleIface, nil)
			},
			validateFunc: func(result []iface.InterfaceEntry, err error) {
				suite.Require().NoError(err)
				suite.Require().Len(result, 1)

				suite.Equal("eth0", result[0].Name)
			},
		},
		{
			name: "when only loopback exists",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusEmpty, nil)
			},
			validateFunc: func(result []iface.InterfaceEntry, err error) {
				suite.Require().NoError(err)
				suite.Empty(result)
			},
		},
		{
			name: "when netplan status fails",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return("", errors.New("command not found"))
			},
			validateFunc: func(result []iface.InterfaceEntry, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "interface list:")
			},
		},
		{
			name: "when two interfaces share the same index, sort falls back to alphabetical order",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusSameIndexIfaces, nil)
			},
			validateFunc: func(result []iface.InterfaceEntry, err error) {
				suite.Require().NoError(err)
				suite.Require().Len(result, 2)

				// aeth should come before zeth by name despite
				// both having index 2.
				suite.Equal("aeth", result[0].Name)
				suite.Equal("zeth", result[1].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.List(suite.ctx)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *InterfacePublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		interfaceNm  string
		setup        func()
		validateFunc func(*iface.InterfaceEntry, error)
	}{
		{
			name:        "when interface found",
			interfaceNm: "eth0",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusTwoIfaces, nil)
			},
			validateFunc: func(result *iface.InterfaceEntry, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("eth0", result.Name)
				suite.Equal("10.0.0.5", result.IPv4)
				suite.Equal("aa:bb:cc:dd:ee:f0", result.MAC)
				suite.True(result.Primary)
				suite.Require().NotNil(result.DHCP4)
				suite.True(*result.DHCP4)
			},
		},
		{
			name:        "when interface not found",
			interfaceNm: "eth99",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(netplanStatusSingleIface, nil)
			},
			validateFunc: func(result *iface.InterfaceEntry, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name:        "when name is empty",
			interfaceNm: "",
			setup:       func() {},
			validateFunc: func(result *iface.InterfaceEntry, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "name must not be empty")
			},
		},
		{
			name:        "when netplan status fails",
			interfaceNm: "eth0",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return("", errors.New("command not found"))
			},
			validateFunc: func(result *iface.InterfaceEntry, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "interface get:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Get(suite.ctx, tc.interfaceNm)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *InterfacePublicTestSuite) TestCreate() {
	dhcp4True := true

	tests := []struct {
		name         string
		entry        iface.InterfaceEntry
		setup        func()
		validateFunc func(*iface.InterfaceResult, error)
	}{
		{
			name: "when new interface deploys successfully",
			entry: iface.InterfaceEntry{
				Name:      "eth0",
				DHCP4:     &dhcp4True,
				Addresses: []string{"10.0.0.5/24"},
				Gateway4:  "10.0.0.1",
				MTU:       1500,
			},
			setup: func() {
				// SectionForInterface calls netplan status.
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(`{"eth0": {"type": "ethernet"}}`, nil).
					AnyTimes()

				// KV Get returns not found (new file).
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// netplan generate succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				// netplan apply succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				// KV Put succeeds.
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("eth0", result.Name)
				suite.True(result.Changed)

				// Verify file was written.
				data, readErr := suite.memFs.ReadFile(
					"/etc/netplan/osapi-eth0.yaml",
				)
				suite.Require().NoError(readErr)
				suite.Contains(string(data), "eth0:")
				suite.Contains(string(data), "dhcp4: true")
				suite.Contains(string(data), "10.0.0.5/24")
				suite.Contains(string(data), "gateway4: 10.0.0.1")
				suite.Contains(string(data), "mtu: 1500")
			},
		},
		{
			name: "when interface already managed returns unchanged",
			entry: iface.InterfaceEntry{
				Name:  "eth0",
				DHCP4: &dhcp4True,
			},
			setup: func() {
				// File already exists on disk.
				_ = suite.memFs.WriteFile(
					"/etc/netplan/osapi-eth0.yaml",
					[]byte("existing"),
					0o644,
				)
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().NoError(err)
				suite.NotNil(result)
				suite.Equal("eth0", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name: "when name is empty",
			entry: iface.InterfaceEntry{
				Name: "",
			},
			setup: func() {},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "name must not be empty")
			},
		},
		{
			name: "when name has invalid characters",
			entry: iface.InterfaceEntry{
				Name: "eth 0!",
			},
			setup: func() {},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid characters")
			},
		},
		{
			name: "when ApplyConfig fails",
			entry: iface.InterfaceEntry{
				Name:  "eth0",
				DHCP4: &dhcp4True,
			},
			setup: func() {
				// SectionForInterface calls netplan status.
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(`{"eth0": {"type": "ethernet"}}`, nil).
					AnyTimes()

				// KV Get returns not found.
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// netplan generate fails.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", errors.New("invalid YAML"))
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "interface create:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Create(suite.ctx, tc.entry)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *InterfacePublicTestSuite) TestUpdate() {
	dhcp4False := false

	tests := []struct {
		name         string
		entry        iface.InterfaceEntry
		setup        func()
		validateFunc func(*iface.InterfaceResult, error)
	}{
		{
			name: "when update succeeds",
			entry: iface.InterfaceEntry{
				Name:      "eth0",
				DHCP4:     &dhcp4False,
				Addresses: []string{"10.0.0.10/24"},
			},
			setup: func() {
				// Managed file exists on disk.
				_ = suite.memFs.WriteFile(
					"/etc/netplan/osapi-eth0.yaml",
					[]byte("old content"),
					0o644,
				)

				// SectionForInterface calls netplan status.
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(`{"eth0": {"type": "ethernet"}}`, nil).
					AnyTimes()

				// KV Get returns not found (different SHA).
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// netplan generate succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				// netplan apply succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				// KV Put succeeds.
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("eth0", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when interface not managed",
			entry: iface.InterfaceEntry{
				Name:  "eth0",
				DHCP4: &dhcp4False,
			},
			setup: func() {
				// No file on disk.
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not managed")
			},
		},
		{
			name: "when name is empty",
			entry: iface.InterfaceEntry{
				Name: "",
			},
			setup: func() {},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "name must not be empty")
			},
		},
		{
			name: "when ApplyConfig fails",
			entry: iface.InterfaceEntry{
				Name:  "eth0",
				DHCP4: &dhcp4False,
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/netplan/osapi-eth0.yaml",
					[]byte("old"),
					0o644,
				)

				// SectionForInterface calls netplan status.
				suite.mockExec.EXPECT().
					RunCmd("netplan", []string{"status", "--format", "json"}).
					Return(`{"eth0": {"type": "ethernet"}}`, nil).
					AnyTimes()

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", errors.New("validation error"))
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "interface update:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Update(suite.ctx, tc.entry)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *InterfacePublicTestSuite) TestDelete() {
	tests := []struct {
		name         string
		interfaceNm  string
		setup        func()
		validateFunc func(*iface.InterfaceResult, error)
	}{
		{
			name:        "when file exists and removal succeeds",
			interfaceNm: "eth0",
			setup: func() {
				// Create the managed file on disk.
				_ = suite.memFs.WriteFile(
					"/etc/netplan/osapi-eth0.yaml",
					[]byte("network:\n"),
					0o644,
				)

				// netplan apply succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				// KV state exists for undeploy marking.
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("eth0", result.Name)
				suite.True(result.Changed)

				// Verify file was removed.
				_, statErr := suite.memFs.Stat(
					"/etc/netplan/osapi-eth0.yaml",
				)
				suite.Error(statErr)
			},
		},
		{
			name:        "when interface not managed returns unchanged",
			interfaceNm: "eth0",
			setup:       func() {},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("eth0", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when name is empty",
			interfaceNm: "",
			setup:       func() {},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "name must not be empty")
			},
		},
		{
			name:        "when RemoveConfig fails",
			interfaceNm: "eth0",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/netplan/osapi-eth0.yaml",
					[]byte("network:\n"),
					0o644,
				)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", errors.New("apply failed"))
			},
			validateFunc: func(result *iface.InterfaceResult, err error) {
				suite.Require().Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "interface delete:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Delete(suite.ctx, tc.interfaceNm)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *InterfacePublicTestSuite) TestGenerateInterfaceYAML() {
	dhcp4True := true
	dhcp4False := false
	dhcp6True := true
	wolTrue := true

	tests := []struct {
		name         string
		entry        iface.InterfaceEntry
		validateFunc func(string)
	}{
		{
			name: "when all fields set",
			entry: iface.InterfaceEntry{
				Name:       "eth0",
				DHCP4:      &dhcp4False,
				DHCP6:      &dhcp6True,
				Addresses:  []string{"10.0.0.5/24", "10.0.0.6/24"},
				Gateway4:   "10.0.0.1",
				Gateway6:   "fe80::1",
				MTU:        9000,
				MACAddress: "aa:bb:cc:dd:ee:ff",
				WakeOnLAN:  &wolTrue,
			},
			validateFunc: func(result string) {
				suite.Contains(result, "network:")
				suite.Contains(result, "  version: 2")
				suite.Contains(result, "  ethernets:")
				suite.Contains(result, "    eth0:")
				suite.Contains(result, "      dhcp4: false")
				suite.Contains(result, "      dhcp6: true")
				suite.Contains(result, "      addresses:")
				suite.Contains(result, "        - 10.0.0.5/24")
				suite.Contains(result, "        - 10.0.0.6/24")
				suite.Contains(result, "      gateway4: 10.0.0.1")
				suite.Contains(result, "      gateway6: fe80::1")
				suite.Contains(result, "      mtu: 9000")
				suite.Contains(result, "      macaddress: aa:bb:cc:dd:ee:ff")
				suite.Contains(result, "      wakeonlan: true")
			},
		},
		{
			name: "when only DHCP4 set",
			entry: iface.InterfaceEntry{
				Name:  "eth0",
				DHCP4: &dhcp4True,
			},
			validateFunc: func(result string) {
				suite.Contains(result, "network:")
				suite.Contains(result, "    eth0:")
				suite.Contains(result, "      dhcp4: true")
				suite.NotContains(result, "addresses:")
				suite.NotContains(result, "gateway4:")
				suite.NotContains(result, "gateway6:")
				suite.NotContains(result, "mtu:")
				suite.NotContains(result, "macaddress:")
				suite.NotContains(result, "wakeonlan:")
			},
		},
		{
			name: "when only addresses set",
			entry: iface.InterfaceEntry{
				Name:      "ens3",
				Addresses: []string{"192.168.1.10/24"},
			},
			validateFunc: func(result string) {
				suite.Contains(result, "    ens3:")
				suite.Contains(result, "      addresses:")
				suite.Contains(result, "        - 192.168.1.10/24")
				suite.NotContains(result, "dhcp4:")
				suite.NotContains(result, "gateway4:")
				suite.NotContains(result, "mtu:")
			},
		},
		{
			name: "when IPv6 gateway set",
			entry: iface.InterfaceEntry{
				Name:      "eth0",
				Addresses: []string{"fd00::5/64"},
				Gateway6:  "fd00::1",
			},
			validateFunc: func(result string) {
				suite.Contains(result, "      gateway6: fd00::1")
				suite.Contains(result, "        - fd00::5/64")
				suite.NotContains(result, "gateway4:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := iface.GenerateInterfaceYAML(tc.entry, "ethernets")

			tc.validateFunc(string(result))
		})
	}
}

func TestInterfacePublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(InterfacePublicTestSuite))
}
