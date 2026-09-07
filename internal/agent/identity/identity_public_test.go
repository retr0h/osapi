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

package identity_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/agent/identity"
)

// defaultGetMachineID uses the injectable osPlatform variable, so all
// platform branches (linux, darwin, unsupported) can be tested on any
// host by swapping osPlatform via SetOSPlatform/ResetOSPlatform.

type GetMachineIDFromFSPublicTestSuite struct {
	suite.Suite
}

func (suite *GetMachineIDFromFSPublicTestSuite) TestGetMachineIDFromFS() {
	tests := []struct {
		name         string
		setupFS      func(fs avfs.VFS)
		validateFunc func(any, error)
	}{
		{
			name: "when valid machine-id file exists",
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc", 0o755)
				_ = fs.WriteFile("/etc/machine-id", []byte("abc123def456\n"), 0o444)
			},
			validateFunc: func(got any, err error) {
				require.NoError(suite.T(), err)
				assert.Equal(suite.T(), "abc123def456", got)
			},
		},
		{
			name: "when machine-id has leading and trailing whitespace",
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc", 0o755)
				_ = fs.WriteFile("/etc/machine-id", []byte("  abc123def456  \n"), 0o444)
			},
			validateFunc: func(got any, err error) {
				require.NoError(suite.T(), err)
				assert.Equal(suite.T(), "abc123def456", got)
			},
		},
		{
			name: "when machine-id file does not exist",
			setupFS: func(_ avfs.VFS) {
				// no file created
			},
			validateFunc: func(got any, err error) {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), "read machine-id")
				assert.Empty(suite.T(), got)
			},
		},
		{
			name: "when machine-id file is empty",
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc", 0o755)
				_ = fs.WriteFile("/etc/machine-id", []byte(""), 0o444)
			},
			validateFunc: func(got any, err error) {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), "empty machine-id")
				assert.Empty(suite.T(), got)
			},
		},
		{
			name: "when machine-id file contains only whitespace",
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc", 0o755)
				_ = fs.WriteFile("/etc/machine-id", []byte("   \n\t\n"), 0o444)
			},
			validateFunc: func(got any, err error) {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), "empty machine-id")
				assert.Empty(suite.T(), got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			tc.setupFS(fs)

			tc.validateFunc(identity.GetMachineIDFromFS(fs))
		})
	}
}

func TestGetMachineIDFromFSPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GetMachineIDFromFSPublicTestSuite))
}

type GetDarwinMachineIDPublicTestSuite struct {
	suite.Suite
}

func (suite *GetDarwinMachineIDPublicTestSuite) TearDownSubTest() {
	identity.ResetIoregFn()
}

func (suite *GetDarwinMachineIDPublicTestSuite) TestGetDarwinMachineID() {
	tests := []struct {
		name         string
		ioregOutput  string
		ioregErr     error
		validateFunc func(any, error)
	}{
		{
			name: "when ioreg returns valid UUID",
			ioregOutput: `+-o Root  <class IORegistryEntry, id 0x100000100, retain 22>
    {
      "IOPlatformUUID" = "12345678-ABCD-EFGH-IJKL-123456789ABC"
    }
`,
			validateFunc: func(got any, err error) {
				require.NoError(suite.T(), err)
				assert.Equal(suite.T(), "12345678-ABCD-EFGH-IJKL-123456789ABC", got)
			},
		},
		{
			name:     "when ioreg command fails",
			ioregErr: fmt.Errorf("command not found"),
			validateFunc: func(got any, err error) {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), "run ioreg")
				assert.Empty(suite.T(), got)
			},
		},
		{
			name: "when ioreg output has no UUID",
			ioregOutput: `+-o Root  <class IORegistryEntry, id 0x100000100, retain 22>
    {
      "SomeOtherKey" = "value"
    }
`,
			validateFunc: func(got any, err error) {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), "IOPlatformUUID not found")
				assert.Empty(suite.T(), got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			identity.SetIoregFn(func() (string, error) {
				return tc.ioregOutput, tc.ioregErr
			})

			tc.validateFunc(identity.GetDarwinMachineID())
		})
	}
}

func TestGetDarwinMachineIDPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GetDarwinMachineIDPublicTestSuite))
}

type GetIdentityPublicTestSuite struct {
	suite.Suite
}

func (suite *GetIdentityPublicTestSuite) TearDownSubTest() {
	identity.ResetGetMachineIDFn()
}

func (suite *GetIdentityPublicTestSuite) TestGetIdentity() {
	tests := []struct {
		name            string
		configHostname  string
		machineIDFn     func(avfs.VFS) (string, error)
		wantMachineID   string
		wantHostname    string
		wantErr         bool
		wantContains    string
		hostnameNonZero bool
	}{
		{
			name:           "when config hostname is provided and machine-id succeeds",
			configHostname: "my-host",
			machineIDFn: func(_ avfs.VFS) (string, error) {
				return "abc123", nil
			},
			wantMachineID: "abc123",
			wantHostname:  "my-host",
		},
		{
			name:           "when config hostname is empty falls back to system hostname",
			configHostname: "",
			machineIDFn: func(_ avfs.VFS) (string, error) {
				return "abc123", nil
			},
			wantMachineID:   "abc123",
			hostnameNonZero: true,
		},
		{
			name:           "when machine-id resolution fails",
			configHostname: "my-host",
			machineIDFn: func(_ avfs.VFS) (string, error) {
				return "", fmt.Errorf("read machine-id: file not found")
			},
			wantErr:      true,
			wantContains: "machine-id",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()

			if tc.machineIDFn != nil {
				identity.SetGetMachineIDFn(tc.machineIDFn)
			}

			got, err := identity.GetIdentity(fs, tc.configHostname)

			if tc.wantErr {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantContains)
				assert.Nil(suite.T(), got)
			} else {
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), got)
				assert.Equal(suite.T(), tc.wantMachineID, got.MachineID)
				if tc.hostnameNonZero {
					assert.NotEmpty(suite.T(), got.Hostname)
				} else {
					assert.Equal(suite.T(), tc.wantHostname, got.Hostname)
				}
			}
		})
	}
}

func TestGetIdentityPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(GetIdentityPublicTestSuite))
}

// PlatformPublicTestSuite tests the real platform-specific functions.
// These run the actual OS commands so they only pass on their target platform.
type PlatformPublicTestSuite struct {
	suite.Suite
}

func (suite *PlatformPublicTestSuite) TearDownSubTest() {
	identity.ResetExecCommandFn()
	identity.ResetIoregFn()
	identity.ResetOSPlatform()
}

func (suite *PlatformPublicTestSuite) TestDefaultIoregFn() {
	tests := []struct {
		name         string
		setupFn      func()
		skipUnless   string
		validateFunc func(any, error)
	}{
		{
			name:       "when ioreg command succeeds on macOS",
			skipUnless: "darwin",
			validateFunc: func(out any, err error) {
				require.NoError(suite.T(), err)
				assert.Contains(suite.T(), out, "IOPlatformUUID")
			},
		},
		{
			name: "when exec command fails returns error",
			setupFn: func() {
				identity.SetExecCommandFn(func() ([]byte, error) {
					return nil, fmt.Errorf("exec: command not found")
				})
			},
			validateFunc: func(out any, err error) {
				require.Error(suite.T(), err)
				assert.Empty(suite.T(), out)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.skipUnless != "" && runtime.GOOS != tc.skipUnless {
				suite.T().Skipf("only runs on %s", tc.skipUnless)
			}

			if tc.setupFn != nil {
				tc.setupFn()
			}

			tc.validateFunc(identity.ExportDefaultIoregFn())
		})
	}
}

func (suite *PlatformPublicTestSuite) TestDefaultGetMachineID() {
	tests := []struct {
		name         string
		platform     string
		setupFS      func(fs avfs.VFS)
		setupFn      func()
		validateFunc func(any, error)
	}{
		{
			name:     "when platform is darwin returns machine ID",
			platform: "darwin",
			setupFS:  func(_ avfs.VFS) {},
			setupFn: func() {
				identity.SetIoregFn(func() (string, error) {
					return `"IOPlatformUUID" = "FAKE-UUID-1234"`, nil
				})
			},
			validateFunc: func(got any, err error) {
				require.NoError(suite.T(), err)
				assert.NotEmpty(suite.T(), got)
			},
		},
		{
			name:     "when platform is linux reads /etc/machine-id",
			platform: "linux",
			setupFS: func(fs avfs.VFS) {
				_ = fs.MkdirAll("/etc", 0o755)
				_ = fs.WriteFile("/etc/machine-id", []byte("abc123\n"), 0o444)
			},
			setupFn: func() {},
			validateFunc: func(got any, err error) {
				require.NoError(suite.T(), err)
				assert.NotEmpty(suite.T(), got)
			},
		},
		{
			name:     "when platform is unsupported returns error",
			platform: "windows",
			setupFS:  func(_ avfs.VFS) {},
			setupFn:  func() {},
			validateFunc: func(_ any, err error) {
				require.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), "unsupported platform: windows")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fs := memfs.New()
			tc.setupFS(fs)
			identity.SetOSPlatform(tc.platform)
			if tc.setupFn != nil {
				tc.setupFn()
			}

			tc.validateFunc(identity.ExportDefaultGetMachineID(fs))
		})
	}
}

func TestPlatformPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlatformPublicTestSuite))
}
