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

// NewOSMemoryProvider factory to create a new instance.
func NewOSMemoryProvider(
	appFs afero.Fs,
) *OSMemoryProvider {
	return &OSMemoryProvider{
		appFs: appFs,
	}
}

// GetMemory retrieves the system's memory statistics from /proc/meminfo.
//
// Cross-platform within Unix-like systems: This function works on Linux-based systems
// where /proc/meminfo is available. It is designed to be distro-independent as long
// as the system follows the typical /proc/meminfo format for reporting memory statistics.
//
// About /proc/meminfo:
//   - This file contains various memory-related statistics, including total memory,
//     free memory, and cached memory. The function extracts relevant fields to compute
//     the total, free, and used memory in bytes.
//
// Memory Calculation:
//   - Total memory is obtained from the MemTotal field.
//   - Free memory is obtained from the MemFree field.
//   - Used memory is calculated as Total - Free - Cached.
//
// Mocking:
//   - Afero is used for file system abstraction, which allows for easier testing and mocking
//     of file reads. In the future, other methods for simulating system calls or commands
//     might be considered for more comprehensive testing scenarios.
func (p *OSMemoryProvider) GetMemory() ([]uint64, error) {
	const memInfoFile = "/proc/meminfo"

	data, err := afero.ReadFile(p.appFs, memInfoFile)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", memInfoFile, err)
	}

	var total, free, cached uint64

	// Parse /proc/meminfo content
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Convert the value from kB to bytes (multiply by 1024)
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		value *= 1024 // Convert kB to bytes

		switch fields[0] {
		case "MemTotal:":
			total = value
		case "MemFree:":
			free = value
		case "Cached:":
			cached = value
		}
	}

	used := calculateUsed(total, free, cached)

	return []uint64{total, free, used}, nil
}

// calculateUsed Calculate used memory as total - free - cached.
func calculateUsed(total, free, cached uint64) uint64 {
	// Check to prevent underflow
	if total >= free+cached {
		return total - free - cached
	}

	// Handle the potential underflow case
	return 0
}
