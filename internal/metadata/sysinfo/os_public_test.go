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

	"github.com/retr0h/osapi/internal/metadata/sysinfo"
)

type OSPublicTestSuite struct {
	suite.Suite

	appFs         afero.Fs
	si            *sysinfo.SysInfo
	osReleaseFile string
}

func (suite *OSPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.si = sysinfo.New(suite.appFs)
	suite.osReleaseFile = "/etc/os-release"
}

func (suite *OSPublicTestSuite) SetupSubTest() {
	// initializes a new afero.Fs in the table tests
	suite.SetupTest()
}

func (suite *OSPublicTestSuite) TestGetOSInfo() {
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
		{
			name: "when distro is fedora",
			content: []byte(`
NAME=Fedora
VERSION_ID=32`),
			want: struct {
				distribution string
				version      string
			}{
				distribution: "fedora",
				version:      "32",
			},
		},
		{
			name: "when cannot parse osReleaseFile",
			content: []byte(`
NAME^"invalid"`),
			want: struct {
				distribution string
				version      string
			}{
				distribution: "",
				version:      "",
			},
		},
		{
			name:    "when missing osReleaseFile",
			content: []byte{},
			want: struct {
				distribution string
				version      string
			}{
				distribution: "",
				version:      "",
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if len(tc.content) != 0 {
				_ = afero.WriteFile(suite.appFs, suite.osReleaseFile, tc.content, 0o644)
			}

			got := suite.si.GetOSInfo()

			assert.Equal(suite.T(), tc.want.distribution, got.Distribution)
			assert.Equal(suite.T(), tc.want.version, got.Version)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestOSPublicTestSuite(t *testing.T) {
	suite.Run(t, new(OSPublicTestSuite))
}
