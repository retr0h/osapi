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

package system

// LoadAverageStats represents the system load averages over 1, 5, and 15 minutes.
type LoadAverageStats struct {
	// Load average over the last 1 minute
	Load1 float32
	// Load average over the last 5 minutes
	Load5 float32
	// Load average over the last 15 minutes
	Load15 float32
}

// DiskUsageStats holds information about disk space usage.
type DiskUsageStats struct {
	// Disk identifier, e.g., "/dev/sda1"
	Name string
	// Total disk space in bytes
	Total uint64
	// Used disk space in bytes
	Used uint64
	// Free disk space in bytes
	Free uint64
}
