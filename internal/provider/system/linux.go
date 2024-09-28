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

import (
	"fmt"
	"time"
)

// DefaultLinuxSystem implements the System interface for Linux.
type DefaultLinuxSystem struct{}

// NewDefaultLinuxProvider factory to create a new Linux instance.
func NewDefaultLinuxProvider() *DefaultLinuxSystem {
	return &DefaultLinuxSystem{}
}

// GetUptime retrieves the system uptime.
// It returns the uptime as a time.Duration, and an error if something goes wrong.
func (dls *DefaultLinuxSystem) GetUptime() (time.Duration, error) {
	return 0, fmt.Errorf("GetUptime is not implemented for DefaultLinuxProvider")
}

// GetLocalDiskStats retrieves disk space statistics for local disks only.
// It returns a slice of DiskUsageStats structs, each containing the total, used,
// and free space in bytes for the corresponding local disk.
// An error is returned if somethng goes wrong.
func (dls *DefaultLinuxSystem) GetLocalDiskStats() ([]DiskUsageStats, error) {
	return nil, fmt.Errorf("GetLocalDiskStats is not implemented for DefaultLinuxProvider")
}
