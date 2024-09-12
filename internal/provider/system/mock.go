//go:build test
// +build test

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
)

// Mock is a mock implementation of the System interface for testing.
type Mock struct {
	GetHostnameFunc         func() (string, error)
	GetMemoryStatsFunc      func() (*MemoryStats, error)
	GetLoadAverageStatsFunc func() (*LoadAverageStats, error)
	GetUptimeFunc           func() (time.Duration, error)
	GetLocalDiskStatsFunc   func() ([]DiskUsageStats, error)
}

// NewDefaultMock creates a Mock with default return values.
func NewDefaultMock() *Mock {
	return &Mock{
		GetHostnameFunc: func() (string, error) {
			return "default-hostname", nil
		},
		GetMemoryStatsFunc: func() (*MemoryStats, error) {
			return &MemoryStats{
				Total:  8388608,
				Free:   4194304,
				Cached: 2097152,
			}, nil
		},
		GetLoadAverageStatsFunc: func() (*LoadAverageStats, error) {
			return &LoadAverageStats{1.0, 0.5, 0.2}, nil
		},
		GetUptimeFunc: func() (time.Duration, error) {
			return time.Hour * 5, nil
		},
		GetLocalDiskStatsFunc: func() ([]DiskUsageStats, error) {
			return []DiskUsageStats{
				{
					Name:  "/dev/disk1",
					Total: 500000000000,
					Used:  250000000000,
					Free:  250000000000,
				},
			}, nil
		},
	}
}

// GetHostname mocked for tests.
func (m *Mock) GetHostname() (string, error) {
	return m.GetHostnameFunc()
}

// GetMemoryStats mocked for tests.
func (m *Mock) GetMemoryStats() (*MemoryStats, error) {
	return m.GetMemoryStatsFunc()
}

// GetLoadAverageStats mocked for tests.
func (m *Mock) GetLoadAverageStats() (*LoadAverageStats, error) {
	return m.GetLoadAverageStatsFunc()
}

// GetUptime mocked for tests.
func (m *Mock) GetUptime() (time.Duration, error) {
	return m.GetUptimeFunc()
}

// GetLocalDiskStats mocked for tests.
func (m *Mock) GetLocalDiskStats() ([]DiskUsageStats, error) {
	return m.GetLocalDiskStatsFunc()
}
