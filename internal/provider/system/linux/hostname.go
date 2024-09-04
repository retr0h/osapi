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

package linux

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"
)

// NewOSHostnameProvider factory to create a new instance.
func NewOSHostnameProvider(
	appFs afero.Fs,
) *OSHostnameProvider {
	return &OSHostnameProvider{
		appFs: appFs,
	}
}

// GetHostname reads the system's hostname from /proc/sys/kernel/hostname.
// It is cross-compatible with all Linux distributions.
//
// Cross-platform considerations: This function is specifically designed for Linux
// systems where /proc/sys/kernel/hostname is available. For non-Linux Unix-like
// systems (e.g., BSD or macOS), the file location and method to retrieve the
// hostname may differ. On these systems, alternative approaches or system-specific
// files should be used to obtain the hostname.
//
// About /proc/sys/kernel/hostname:
//   - This file contains the current hostname of the system as recognized by the kernel.
//   - It reflects the system's runtime hostname, which can be set using commands like
//     `hostname` or by writing directly to this file.
//   - This file is available on all Linux distributions as part of the procfs, making it
//     a reliable and cross-distro way to fetch the system's hostname.
//   - Modifications to this file directly impact the kernel's idea of the system hostname.
//
// See `proc(5)` manual page for further information.
//
// Mocking:
//   - Afero is used for file system abstraction, which allows for easier testing and mocking
//     of file reads. In the future, other methods for simulating system calls or commands
//     might be considered for more comprehensive testing scenarios.
func (p *OSHostnameProvider) GetHostname() (string, error) {
	const hostnameFile = "/proc/sys/kernel/hostname"

	// Read the hostname file
	data, err := afero.ReadFile(p.appFs, hostnameFile)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", hostnameFile, err)
	}

	// Trim any surrounding whitespace or newline characters
	return strings.TrimSpace(string(data)), nil
}