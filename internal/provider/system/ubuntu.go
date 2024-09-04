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
	"time"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
)

// UbuntuSystem implements the System interface for Ubuntu.
type UbuntuSystem struct{}

// NewUbuntuProvider factory to create a new Ubuntu instance.
func NewUbuntuProvider() *UbuntuSystem {
	return &UbuntuSystem{}
}

// GetHostname retrieves the hostname of the system.
// It returns the hostname as a string and an error if something goes wrong.
func (us *UbuntuSystem) GetHostname() (string, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return "", err
	}
	return hostInfo.Hostname, nil
}

// GetMemoryStats retrieves memory statistics of the system.
// It returns a MemoryStats struct with total, free, and cached memory in
// bytes, and an error if something goes wrong.
func (us *UbuntuSystem) GetMemoryStats() (*MemoryStats, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return &MemoryStats{}, err
	}

	return &MemoryStats{
		Total:  memInfo.Total,
		Free:   memInfo.Free,
		Cached: memInfo.Cached,
	}, nil
}

// GetLoadAverageStats returns the system's load averages over 1, 5, and 15 minutes.
// It returns a LoadAverageStats struct with load over 1, 5, and 15 minutes,
// and an error if something goes wrong.
func (us *UbuntuSystem) GetLoadAverageStats() (*LoadAverageStats, error) {
	avg, err := load.Avg()
	if err != nil {
		return nil, err
	}
	return &LoadAverageStats{
		Load1:  float32(avg.Load1),
		Load5:  float32(avg.Load5),
		Load15: float32(avg.Load15),
	}, nil
}

// GetUptime retrieves the system uptime.
// It returns the uptime as a time.Duration and an error if something goes wrong.
func (us *UbuntuSystem) GetUptime() (time.Duration, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return 0, err
	}
	return time.Duration(hostInfo.Uptime) * time.Second, nil
}
