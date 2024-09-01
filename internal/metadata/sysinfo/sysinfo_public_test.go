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

package sysinfo_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/metadata"
	"github.com/retr0h/osapi/internal/metadata/sysinfo"
)

type SysInfoPublicTestSuite struct {
	suite.Suite

	appFs         afero.Fs
	sim           metadata.SysInfoManager
	osReleaseFile string
}

func (suite *SysInfoPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.sim = sysinfo.New(suite.appFs)
	suite.osReleaseFile = "/etc/os-release"
}

func (suite *SysInfoPublicTestSuite) TestGetSysInfoOk() {
	tests := []struct {
		name    string
		content []byte
		want    struct {
			distribution string
			version      string
		}
	}{
		{
			name: "when distro is ubuntu",
			content: []byte(`
NAME=Ubuntu
VERSION_ID=22.04`),
			want: struct {
				distribution string
				version      string
			}{
				distribution: "ubuntu",
				version:      "22.04",
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			_ = afero.WriteFile(suite.appFs, suite.osReleaseFile, tc.content, 0o644)

			got := suite.sim.GetSysInfo()

			assert.Equal(suite.T(), tc.want.distribution, got.OS.Distribution)
			assert.Equal(suite.T(), tc.want.version, got.OS.Version)
		})
	}
}

func (suite *SysInfoPublicTestSuite) TestGetSysInfoReturnsEmptyWhenGetOSInfoErrors() {
	got := suite.sim.GetSysInfo()

	assert.Empty(suite.T(), got.OS.Distribution)
	assert.Empty(suite.T(), got.OS.Version)
}

func (suite *SysInfoPublicTestSuite) TestIsLinuxVersionSupported() {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "Ubuntu",
			version: "22.04",
			want:    true,
		},
		{
			name:    "UbUntU",
			version: "22.04",
			want:    true,
		},
		{
			name:    "Ubuntu",
			version: "20.04",
			want:    true,
		},
		{
			name:    "Fedora",
			version: "32",
			want:    false,
		},
		{
			name:    "unsupported",
			version: "version",
			want:    false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := suite.sim.IsLinuxVersionSupported(tc.name, tc.version)
			assert.Equal(suite.T(), tc.want, got)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestSysInfoPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SysInfoPublicTestSuite))
}
