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

package disk_test

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	sysDisk "github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/disk"
)

type DebianGetLocalUsageStatsPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
}

func (suite *DebianGetLocalUsageStatsPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DebianGetLocalUsageStatsPublicTestSuite) TearDownTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DebianGetLocalUsageStatsPublicTestSuite) TestGetLocalUsageStats() {
	tests := []struct {
		name         string
		setupMock    func(*disk.Debian)
		validateFunc func(any, error)
	}{
		{
			name: "when GetLocalUsageStats Ok",
			setupMock: func(u *disk.Debian) {
				u.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
					return []sysDisk.PartitionStat{
						{
							Mountpoint: "/dev/disk1",
							Device:     "/dev/disk1",
							Fstype:     "ext4",
						},
						{
							Mountpoint: "/dev/disk2",
							Device:     "/dev/disk2",
							Fstype:     "xfs",
						},
						{
							Mountpoint: "/network",
							Device:     "network",
							Fstype:     "nfs",
						},
						{
							Mountpoint: "/docker",
							Device:     "docker",
							Fstype:     "overlay",
						},
						{
							Mountpoint: "/empty",
							Device:     "",
							Fstype:     "",
						},
						{
							Mountpoint: "/restricted",
							Device:     "/restricted",
							Fstype:     "ext4",
						}, // Permission denied.
					}, nil
				}
				u.UsageFn = func(path string) (*sysDisk.UsageStat, error) {
					switch path {
					case "/dev/disk1":
						return &sysDisk.UsageStat{
							Path:  "/dev/disk1",
							Total: 500000000000,
							Used:  250000000000,
							Free:  250000000000,
						}, nil
					case "/dev/disk2":
						return &sysDisk.UsageStat{
							Path:  "/dev/disk2",
							Total: 1000000000000,
							Used:  750000000000,
							Free:  250000000000,
						}, nil
					case "/restricted":
						return nil, fmt.Errorf("permission denied")
					default:
						return nil, fmt.Errorf("partition not found")
					}
				}
			},
			validateFunc: func(got any, err error) {
				suite.NoError(err)
				suite.NotNil(got)
				suite.Equal([]disk.Result{
					{
						Name:  "/dev/disk1",
						Total: 500000000000,
						Used:  250000000000,
						Free:  250000000000,
					},
					{
						Name:  "/dev/disk2",
						Total: 1000000000000,
						Used:  750000000000,
						Free:  250000000000,
					},
				}, got)
			},
		},
		{
			name: "when disk.Partitions errors",
			setupMock: func(u *disk.Debian) {
				u.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
					return nil, assert.AnError
				}
			},
			validateFunc: func(got any, err error) {
				suite.Error(err)
				suite.ErrorContains(err, assert.AnError.Error())
				suite.Nil(got)
			},
		},
		{
			name: "when disk.Usage errors",
			setupMock: func(u *disk.Debian) {
				u.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
					return []sysDisk.PartitionStat{
						{
							Mountpoint: "/dev/disk1",
							Device:     "/dev/disk1",
							Fstype:     "ext4",
						},
					}, nil
				}
				u.UsageFn = func(_ string) (*sysDisk.UsageStat, error) {
					return nil, assert.AnError
				}
			},
			validateFunc: func(got any, err error) {
				suite.Error(err)
				suite.ErrorContains(err, assert.AnError.Error())
				suite.Nil(got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			debian := disk.NewDebianProvider(suite.logger)

			if tc.setupMock != nil {
				tc.setupMock(debian)
			}

			tc.validateFunc(debian.GetLocalUsageStats())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianGetLocalUsageStatsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianGetLocalUsageStatsPublicTestSuite))
}
