// Copyright (c) 2026 John Dewey

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
	"syscall"
	"testing"

	sysDisk "github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/disk"
)

type DarwinGetLocalUsageStatsPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
}

func (suite *DarwinGetLocalUsageStatsPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DarwinGetLocalUsageStatsPublicTestSuite) TearDownTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *DarwinGetLocalUsageStatsPublicTestSuite) TestGetLocalUsageStats() {
	tests := []struct {
		name         string
		setupMock    func(*disk.Darwin)
		validateFunc func(any, error)
	}{
		{
			name: "when GetLocalUsageStats Ok",
			setupMock: func(d *disk.Darwin) {
				d.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
					return []sysDisk.PartitionStat{
						{
							Mountpoint: "/",
							Device:     "disk1s1",
							Fstype:     "apfs",
						},
						{
							Mountpoint: "/Volumes/Data",
							Device:     "disk1s2",
							Fstype:     "apfs",
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
							Fstype:     "apfs",
						}, // Permission denied.
					}, nil
				}
				d.UsageFn = func(path string) (*sysDisk.UsageStat, error) {
					switch path {
					case "/":
						return &sysDisk.UsageStat{
							Path:  "/",
							Total: 500000000000,
							Used:  250000000000,
							Free:  250000000000,
						}, nil
					case "/Volumes/Data":
						return &sysDisk.UsageStat{
							Path:  "/Volumes/Data",
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
						Name:  "/",
						Total: 500000000000,
						Used:  250000000000,
						Free:  250000000000,
					},
					{
						Name:  "/Volumes/Data",
						Total: 1000000000000,
						Used:  750000000000,
						Free:  250000000000,
					},
				}, got)
			},
		},
		{
			name: "when disk.Usage returns EACCES PathError",
			setupMock: func(d *disk.Darwin) {
				d.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
					return []sysDisk.PartitionStat{
						{
							Mountpoint: "/",
							Device:     "disk1s1",
							Fstype:     "apfs",
						},
						{
							Mountpoint: "/restricted",
							Device:     "/restricted",
							Fstype:     "apfs",
						},
					}, nil
				}
				d.UsageFn = func(path string) (*sysDisk.UsageStat, error) {
					switch path {
					case "/":
						return &sysDisk.UsageStat{
							Path:  "/",
							Total: 500000000000,
							Used:  250000000000,
							Free:  250000000000,
						}, nil
					case "/restricted":
						return nil, &os.PathError{
							Op:   "stat",
							Path: "/restricted",
							Err:  syscall.EACCES,
						}
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
						Name:  "/",
						Total: 500000000000,
						Used:  250000000000,
						Free:  250000000000,
					},
				}, got)
			},
		},
		{
			name: "when disk.Partitions errors",
			setupMock: func(d *disk.Darwin) {
				d.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
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
			setupMock: func(d *disk.Darwin) {
				d.PartitionsFn = func(_ bool) ([]sysDisk.PartitionStat, error) {
					return []sysDisk.PartitionStat{
						{
							Mountpoint: "/",
							Device:     "disk1s1",
							Fstype:     "apfs",
						},
					}, nil
				}
				d.UsageFn = func(_ string) (*sysDisk.UsageStat, error) {
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
			darwin := disk.NewDarwinProvider(suite.logger)

			if tc.setupMock != nil {
				tc.setupMock(darwin)
			}

			tc.validateFunc(darwin.GetLocalUsageStats())
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDarwinGetLocalUsageStatsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DarwinGetLocalUsageStatsPublicTestSuite))
}
