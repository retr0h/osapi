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
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
)

// UbuntuSystem implements the System interface for Ubuntu.
type UbuntuSystem struct{}

// NewUbuntuProvider factory to create a new Ubuntu instance.
func NewUbuntuProvider() *UbuntuSystem {
	return &UbuntuSystem{}
}

// GetUptime retrieves the system uptime.
// It returns the uptime as a time.Duration, and an error if something goes wrong.
func (us *UbuntuSystem) GetUptime() (time.Duration, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return 0, err
	}
	return time.Duration(hostInfo.Uptime) * time.Second, nil
}

// GetLocalDiskStats retrieves disk space statistics for local disks only.
// It returns a slice of DiskUsageStats structs, each containing the total, used,
// and free space in bytes for the corresponding local disk.
// It gracefully skips partitions where a permission error occurs (e.g., for mounts
// that the user cannot access without root privileges), and continues processing
// the remaining partitions.
// If a non-permission-related error occurs, the function returns an error.
func (us *UbuntuSystem) GetLocalDiskStats() ([]DiskUsageStats, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	diskSpaces := make([]DiskUsageStats, 0, len(partitions))
	for _, partition := range partitions {
		// Skip non-local devices, network-mounted partitions, Docker, and Kubernetes mounts
		if partition.Device == "" || partition.Fstype == "" || !isLocalPartition(partition) {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			fmt.Println(err)
			fmt.Println(err)
			fmt.Println(err)
			fmt.Println(err)
			if isPermissionError(err) {
				fmt.Printf(
					"Skipping partition %s due to permission error: %v\n",
					partition.Mountpoint,
					err,
				)
				continue // Skip this partition if we encounter a permission error
			}
			return nil, fmt.Errorf("failed to get disk usage for %s: %w", partition.Mountpoint, err)
		}

		diskSpaces = append(diskSpaces, DiskUsageStats{
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

// // isDockerOrKubernetesMount checks if the mount point is used by Docker, Kubernetes, or other virtual filesystems.
// func isDockerOrKubernetesMount(partition disk.PartitionStat) bool {
// 	// Filter by filesystem type
// 	dockerK8sFsTypes := map[string]bool{
// 		"tmpfs":    true, // Common in Kubernetes for ephemeral storage
// 		"overlay":  true, // Used by Docker
// 		"aufs":     true, // Used by Docker
// 		"devtmpfs": true, // Device filesystems
// 		"proc":     true, // Process filesystems
// 		"sysfs":    true, // System filesystems
// 	}

// 	// If the filesystem type matches a known Docker/Kubernetes type, skip it
// 	if dockerK8sFsTypes[partition.Fstype] {
// 		return true
// 	}

// 	// Also filter by mount paths for Docker and Kubernetes
// 	if strings.HasPrefix(partition.Mountpoint, "/var/lib/docker") || strings.HasPrefix(partition.Mountpoint, "/var/lib/kubelet") {
// 		return true
// 	}

// 	return false
// }
