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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// NewOSUptimeProvider factory to create a new instance.
func NewOSUptimeProvider(
	appFs afero.Fs,
) *OSUptimeProvider {
	return &OSUptimeProvider{
		appFs: appFs,
	}
}

// GetUptime returns the system uptime as a duration. It is cross-compatible
// with all Linux distributions.
//
// The /proc/uptime file contains two floating-point numbers:
//  1. The first number represents the total number of seconds the system has
//     been up since the last boot.
//  2. The second number indicates the total time (in seconds) that all CPUs
//     have spent idle, summed up across all CPUs.
//
// These values are calculated by the Linux kernel and updated in real-time
// to reflect the current state of the system.
//
// According to the `proc(5)` manual page:
func (p *OSUptimeProvider) GetUptime() (time.Duration, error) {
	const uptimePath = "/proc/uptime"

	// Read the contents of /proc/uptime
	data, err := afero.ReadFile(p.appFs, uptimePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read uptime: %w", err)
	}

	// Split the data to get the uptime in seconds
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("unexpected format of /proc/uptime")
	}

	// Parse the uptime string into a float
	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uptime seconds: %w", err)
	}

	// Convert the seconds to a time.Duration
	uptime := time.Duration(uptimeSeconds) * time.Second

	return uptime, nil
}
