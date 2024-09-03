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

	"github.com/spf13/afero"
)

// NewOSLoadProvider factory to create a new instance.
func NewOSLoadProvider(
	appFs afero.Fs,
) *OSLoadProvider {
	return &OSLoadProvider{
		appFs: appFs,
	}
}

// GetLoadAverage retrieves the system's load average for 1, 5, and 15 minutes.
//
// Cross-platform within Unix-like systems: This works on Linux, BSD, and other
// Unix-like operating systems that support getloadavg(), making it
// distro-independent as long as the OS is Unix-based.
//
// About /proc/loadavg:
//   - This file contains data related to the system's load average over the
//     past 1, 5, and 15 minutes, along with additional statistics about the
//     running processes and the last process ID used.
//
// See `proc(5)` manual page for further information.
//
// Mocking:
//   - Afero is used for file system abstraction, which allows for easier testing and mocking
//     of file reads. In the future, other methods for simulating system calls or commands
//     might be considered for more comprehensive testing scenarios.
func (p *OSLoadProvider) GetLoadAverage() ([3]float32, error) {
	const loadAvgFile = "/proc/loadavg"

	data, err := afero.ReadFile(p.appFs, loadAvgFile)
	if err != nil {
		return [3]float32{}, fmt.Errorf("error reading /proc/loadavg: %w", err)
	}

	// Split the contents by spaces
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return [3]float32{}, fmt.Errorf("unexpected format in /proc/loadavg")
	}

	// Parse the first three fields as load averages (float64)
	load1, err := strconv.ParseFloat(fields[0], 32)
	if err != nil {
		return [3]float32{}, fmt.Errorf("error parsing 1-minute load average: %w", err)
	}

	load5, err := strconv.ParseFloat(fields[1], 32)
	if err != nil {
		return [3]float32{}, fmt.Errorf("error parsing 5-minute load average: %w", err)
	}

	load15, err := strconv.ParseFloat(fields[2], 32)
	if err != nil {
		return [3]float32{}, fmt.Errorf("error parsing 15-minute load average: %w", err)
	}

	// Convert float64 values to float32 and return them
	return [3]float32{float32(load1), float32(load5), float32(load15)}, nil
}
