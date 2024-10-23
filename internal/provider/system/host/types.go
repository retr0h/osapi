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

package host

import (
	"time"
)

// Provider implements the methods to interact with various Host components.
type Provider interface {
	// GetUptime retrieves the system uptime.
	GetUptime() (time.Duration, error)
	// GetHostname retrieves the hostname of the system.
	GetHostname() (string, error)
	// GetOSInfo retrieves information about the operating system, including the
	// distribution name and version. It returns an OSInfo struct containing this
	// data and an error if something goes wrong during the process.
	GetOSInfo() (*OSInfo, error)
}

// OSInfo represents the operating system information.
type OSInfo struct {
	// The name of the Linux distribution (e.g., Ubuntu, CentOS).
	Distribution string
	// The version of the Linux distribution (e.g., 20.04, 8.3).
	Version string
}
