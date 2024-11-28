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

package cmd

import (
	"os"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

var supportedVersions = []struct {
	Distribution string
	Version      string
}{
	{"ubuntu", "20.04"},
	{"ubuntu", "22.04"},
	{"ubuntu", "24.04"},
}

// IsLinuxVersionSupported checks if the given distribution and version are supported.
func IsLinuxVersionSupported(
	distro string,
	version string,
) bool {
	// Convert both distro and version to lowercase to make the check case-insensitive
	distro = strings.ToLower(distro)

	for _, supported := range supportedVersions {
		if strings.ToLower(supported.Distribution) == distro && supported.Version == version {
			return true
		}
	}
	return false
}

// validateDistribution checks if the CLI is being run on the correct Linux distribution.
func validateDistribution() {
	info, err := host.Info()
	if err != nil {
		logFatal("failed to get host info", err)
	}

	if os.Getenv("IGNORE_LINUX") != "" {
		return
	}

	if !IsLinuxVersionSupported(info.Platform, info.PlatformVersion) {
		logFatal(
			"distro not supported",
			nil,
			"distro",
			info.Platform,
			"version",
			info.PlatformVersion,
		)
	}
}
