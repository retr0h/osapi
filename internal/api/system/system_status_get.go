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
	"context"
	"fmt"
	"time"

	"github.com/retr0h/osapi/internal/api/system/gen"
)

// GetSystemStatus get the system status API endpoint.
func (s *System) GetSystemStatus(
	_ context.Context,
	_ gen.GetSystemStatusRequestObject,
) (gen.GetSystemStatusResponseObject, error) {
	hostname, err := s.HostProvider.GetHostname()
	if err != nil {
		errMsg := err.Error()
		return gen.GetSystemStatus500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	uptime, err := s.HostProvider.GetUptime()
	if err != nil {
		errMsg := err.Error()
		return gen.GetSystemStatus500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	loadAvgStats, err := s.LoadProvider.GetAverageStats()
	if err != nil {
		errMsg := err.Error()
		return gen.GetSystemStatus500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	memStats, err := s.MemProvider.GetStats()
	if err != nil {
		errMsg := err.Error()
		return gen.GetSystemStatus500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	osInfo, err := s.HostProvider.GetOSInfo()
	if err != nil {
		errMsg := err.Error()
		return gen.GetSystemStatus500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	diskStats, err := s.DiskProvider.GetLocalUsageStats()
	if err != nil {
		errMsg := err.Error()
		return gen.GetSystemStatus500JSONResponse{
			Error: &errMsg,
		}, nil
	}

	// Convert []systemDiskUsageStats to []gen.Disks
	disks := make([]gen.DiskResponse, 0, len(diskStats))
	for _, d := range diskStats {
		disk := gen.DiskResponse{
			Name:  d.Name,
			Total: uint64ToInt(d.Total),
			Used:  uint64ToInt(d.Used),
			Free:  uint64ToInt(d.Free),
		}
		disks = append(disks, disk)
	}

	return gen.GetSystemStatus200JSONResponse{
		Hostname: hostname,
		Uptime:   formatDuration(uptime),
		LoadAverage: gen.LoadAverageResponse{
			N1min:  loadAvgStats.Load1,
			N5min:  loadAvgStats.Load5,
			N15min: loadAvgStats.Load15,
		},
		OsInfo: gen.OSInfoResponse{
			Distribution: osInfo.Distribution,
			Version:      osInfo.Version,
		},
		// Memory: When float64 values are encoded into JSON, large numbers can
		// automatically be formatted in scientific notation, which is typical
		// behavior for JSON encoders to save space and maintain precision.
		Memory: gen.MemoryResponse{
			Total: uint64ToInt(memStats.Total),
			Free:  uint64ToInt(memStats.Free),
			Used:  uint64ToInt(memStats.Cached),
		},
		Disks: disks,
	}, nil
}

func formatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	// Pluralize days, hours, and minutes
	dayStr := "day"
	if days != 1 {
		dayStr = "days"
	}

	hourStr := "hour"
	if hours != 1 {
		hourStr = "hours"
	}

	minuteStr := "minute"
	if minutes != 1 {
		minuteStr = "minutes"
	}

	// Format the result as a string
	return fmt.Sprintf("%d %s, %d %s, %d %s", days, dayStr, hours, hourStr, minutes, minuteStr)
}

// uint64ToInt convert uint64 to int, with overflow protection.
func uint64ToInt(value uint64) int {
	maxInt := int(^uint(0) >> 1) // maximum value of int based on the architecture
	if value > uint64(maxInt) {  // check for overflow
		return maxInt // return max int to prevent overflow
	}
	return int(value) // conversion within bounds
}
