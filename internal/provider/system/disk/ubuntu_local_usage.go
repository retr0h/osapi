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

package disk

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v4/disk"
)

// GetLocalUsageStats retrieves disk space statistics for local disks only.
// It returns a slice of UsageStats structs, each containing the total, used,
// and free space in bytes for the corresponding local disk.
// It gracefully skips partitions where a permission error occurs (e.g., for mounts
// that the user cannot access without root privileges), and continues processing
// the remaining partitions.
// If a non-permission-related error occurs, the function returns an error.
func (u *Ubuntu) GetLocalUsageStats() ([]UsageStats, error) {
	partitions, err := u.PartitionsFn(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	diskSpaces := make([]UsageStats, 0, len(partitions))
	for _, partition := range partitions {
		// Skip non-local devices, network-mounted partitions, Docker, and Kubernetes mounts
		if partition.Device == "" || partition.Fstype == "" || !isLocalPartition(partition) {
			continue
		}

		usage, err := u.UsageFn(partition.Mountpoint)
		if err != nil {
			if isPermissionError(err) {
				u.logger.Warn(
					"skipping partiion due to permission error",
					slog.String("mount", partition.Mountpoint),
					slog.String("error", err.Error()),
				)
				continue // Skip this partition if we encounter a permission error
			}
			return nil, fmt.Errorf("failed to get disk usage for %s: %w", partition.Mountpoint, err)
		}

		diskSpaces = append(diskSpaces, UsageStats{
			Total: usage.Total,
			Used:  usage.Used,
			Free:  usage.Free,
			Name:  partition.Mountpoint,
		})
	}

	return diskSpaces, nil
}

// isPermissionError checks if an error is related to permission issues.
// Note: This function checks for syscall.EACCES, which represents "permission
// denied" errors at the system call level, as well as the string "permission
// denied" in the error message.
// This is not an ideal approach. We resort to this due to limitations in the
// gopsutil library, which does not provide a more explicit way of classifying
// permission-related errors.
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}

	// Check if the error is related to EACCES or permission denied.
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == syscall.EACCES {
			return true
		}
	}

	// Additionally, some errors might be plain "permission denied" strings
	if strings.Contains(err.Error(), "permission denied") {
		return true
	}

	return false
}

// isLocalPartition determines if a partition is a local disk (not network or special filesystems).
func isLocalPartition(partition disk.PartitionStat) bool {
	// Add conditions to filter only local filesystems, e.g., ext4, xfs, etc.
	localFileSystems := map[string]bool{
		"ext4":  true,
		"xfs":   true,
		"btrfs": true,
	}

	return localFileSystems[partition.Fstype]
}
