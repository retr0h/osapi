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

	"github.com/retr0h/osapi/internal/provider/system/disk"
)

type UbuntuGetLocalUsageStatsPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
}

func (suite *UbuntuGetLocalUsageStatsPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *UbuntuGetLocalUsageStatsPublicTestSuite) TearDownTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *UbuntuGetLocalUsageStatsPublicTestSuite) TestGetLocalUsageStats() {
	tests := []struct {
		name        string
		setupMock   func(*disk.Ubuntu)
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetLocalUsageStats Ok",
			setupMock: func(u *disk.Ubuntu) {
				u.PartitionsFunc = func(all bool) ([]sysDisk.PartitionStat, error) {
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
				u.UsageFunc = func(path string) (*sysDisk.UsageStat, error) {
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
			want: []disk.UsageStats{
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
			},

			wantErr: false,
		},
		{
			name: "when disk.Partitions errors",
			setupMock: func(u *disk.Ubuntu) {
				u.PartitionsFunc = func(all bool) ([]sysDisk.PartitionStat, error) {
					return nil, assert.AnError
				}
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when disk.Usage errors",
			setupMock: func(u *disk.Ubuntu) {
				u.PartitionsFunc = func(all bool) ([]sysDisk.PartitionStat, error) {
					return []sysDisk.PartitionStat{
						{
							Mountpoint: "/dev/disk1",
							Device:     "/dev/disk1",
							Fstype:     "ext4",
						},
					}, nil
				}
				u.UsageFunc = func(path string) (*sysDisk.UsageStat, error) {
					return nil, assert.AnError
				}
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ubuntu := disk.NewUbuntuProvider(suite.logger)

			if tc.setupMock != nil {
				tc.setupMock(ubuntu)
			}

			got, err := ubuntu.GetLocalUsageStats()

			if tc.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tc.wantErrType.Error())
				suite.Nil(got)
			} else {
				suite.NoError(err)
				suite.NotNil(got)
				suite.Equal(tc.want, got)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuGetLocalUsageStatsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuGetLocalUsageStatsPublicTestSuite))
}
