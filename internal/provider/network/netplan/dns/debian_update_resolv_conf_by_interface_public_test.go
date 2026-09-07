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
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/network/netplan/dns"
)

type DebianUpdateResolvConfByInterfacePublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	logger *slog.Logger
}

func (suite *DebianUpdateResolvConfByInterfacePublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DebianUpdateResolvConfByInterfacePublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianUpdateResolvConfByInterfacePublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianUpdateResolvConfByInterfacePublicTestSuite) TestUpdateResolvConfByInterface() {
	tests := []struct {
		name          string
		setupMock     func() (*execmocks.MockManager, *jobmocks.MockKeyValue)
		servers       []string
		searchDomains []string
		interfaceName string
		validateFunc  func(*dns.UpdateResult, error)
	}{
		{
			name: "when SetResolvConf Ok",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewSetResolvConfMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				// KV Get returns not found (new file).
				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// KV Put succeeds.
				kv.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal(true, result.Changed)
			},
		},
		{
			name: "when SetResolvConf preserves existing servers Ok",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewSetResolvConfPreserveDNSServersMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				kv.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)

				return mock, kv
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal(true, result.Changed)
			},
		},
		{
			name: "when SetResolvConf preserves existing search domains Ok",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewSetResolvConfPreserveDNSDomainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				kv.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal(true, result.Changed)
			},
		},
		{
			name: "when SetResolvConf filters root domain Ok",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewSetResolvConfFiltersRootDNSDomainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				kv.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal(true, result.Changed)
			},
		},
		{
			name: "when SetResolvConf missing args errors",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewPlainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				return mock, kv
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(
					err.Error(),
					"no DNS servers or search domains provided; nothing to update",
				)
			},
		},
		{
			name: "when GetResolvConfByInterface errors",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewPlainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				mock.EXPECT().
					RunCmd(execmocks.ResolveCommand, []string{"status", execmocks.NetworkInterfaceName}).
					Return("", assert.AnError).
					AnyTimes()

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), assert.AnError.Error())
			},
		},
		{
			name: "when netplan generate fails",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewSetResolvConfNetplanGenerateErrorMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				mock.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", errors.New("invalid YAML"))

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "netplan validate failed (file rolled back)")
			},
		},
		{
			name: "when netplan apply fails",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewSetResolvConfNetplanGenerateErrorMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				mock.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				mock.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", errors.New("apply failed"))

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
				"9.9.9.9",
			},
			searchDomains: []string{
				"foo.local",
				"bar.local",
			},
			interfaceName: "wlp0s20f3",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "netplan apply:")
			},
		},
		{
			name: "when interface resolved from facts",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewPlainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				// Read path uses empty interface name, but resolvectl
				// still needs the interface. The handler passes the
				// resolved interface to GetResolvConfByInterface before
				// calling update. For this test the read path errors
				// since we pass empty interface to resolvectl.
				mock.EXPECT().
					RunCmd(execmocks.ResolveCommand, []string{"status", ""}).
					Return("", assert.AnError).
					AnyTimes()

				return mock, kv
			},
			servers: []string{
				"8.8.8.8",
			},
			interfaceName: "",
			validateFunc: func(result *dns.UpdateResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "failed to get current resolvectl configuration")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock, kv := tc.setupMock()
			fs := memfs.New()
			_ = fs.MkdirAll("/etc/netplan", 0o755)

			net := dns.NewDebianProvider(suite.logger, fs, kv, mock, "test-host")
			tc.validateFunc(net.UpdateResolvConfByInterface(
				tc.servers,
				tc.searchDomains,
				tc.interfaceName,
				false,
			))
		})
	}
}

func (suite *DebianUpdateResolvConfByInterfacePublicTestSuite) TestDeleteNetplanConfig() {
	tests := []struct {
		name         string
		setupMock    func() (*execmocks.MockManager, *jobmocks.MockKeyValue)
		setupFS      func(avfs.VFS)
		validateFunc func(any, error)
	}{
		{
			name: "when file exists and remove succeeds",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewPlainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				mock.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				// KV Get for state cleanup — return not found so
				// the cleanup branch is skipped cleanly.
				kv.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				return mock, kv
			},
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc/netplan", 0o755)
				_ = fs.WriteFile(
					"/etc/netplan/osapi-dns.yaml",
					[]byte("network:\n  version: 2\n"),
					0o600,
				)
			},
			validateFunc: func(changed any, err error) {
				suite.NoError(err)
				suite.Equal(true, changed)
			},
		},
		{
			name: "when file does not exist returns not changed",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewPlainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				return mock, kv
			},
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc/netplan", 0o755)
			},
			validateFunc: func(changed any, err error) {
				suite.NoError(err)
				suite.Equal(false, changed)
			},
		},
		{
			name: "when netplan apply fails",
			setupMock: func() (*execmocks.MockManager, *jobmocks.MockKeyValue) {
				mock := execmocks.NewPlainMockManager(suite.ctrl)
				kv := jobmocks.NewMockKeyValue(suite.ctrl)

				mock.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", errors.New("apply failed"))

				return mock, kv
			},
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc/netplan", 0o755)
				_ = fs.WriteFile(
					"/etc/netplan/osapi-dns.yaml",
					[]byte("network:\n  version: 2\n"),
					0o600,
				)
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "dns delete via netplan:")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock, kv := tc.setupMock()
			fs := memfs.New()
			tc.setupFS(fs)

			net := dns.NewDebianProvider(suite.logger, fs, kv, mock, "test-host")
			tc.validateFunc(net.DeleteNetplanConfig("eth0"))
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianUpdateResolvConfByInterfacePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianUpdateResolvConfByInterfacePublicTestSuite))
}
