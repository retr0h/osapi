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

package system_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system"
)

type UbuntuPublicTestSuite struct {
	suite.Suite
}

func (suite *UbuntuPublicTestSuite) SetupTest() {}

func (suite *UbuntuPublicTestSuite) TearDownTest() {}

func (suite *UbuntuPublicTestSuite) TestUbuntuProvider() {
	tests := []struct {
		name      string
		setupMock func() *system.MockSystem
		fn        func(*system.MockSystem) (interface{}, error)
		want      interface{}
		wantErr   bool
	}{
		{
			name: "when GetHostname Ok",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetHostname()
			},
			want:    "default-hostname",
			wantErr: false,
		},
		{
			name: "when GetHostname errors",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				mock.GetHostnameFunc = func() (string, error) {
					return "", fmt.Errorf("GetHostname error")
				}
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetHostname()
			},
			wantErr: true,
		},
		{
			name: "when GetMemoryStats Ok",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetMemoryStats()
			},
			want: &system.MemoryStats{
				Total:  8388608,
				Free:   4194304,
				Cached: 2097152,
			},
			wantErr: false,
		},
		{
			name: "when GetMemoryStats errors",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				mock.GetMemoryStatsFunc = func() (*system.MemoryStats, error) {
					return nil, fmt.Errorf("GetMemoryStats error")
				}
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetMemoryStats()
			},
			wantErr: true,
		},
		{
			name: "when GetLoadAverageStats Ok",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetLoadAverageStats()
			},
			want:    &system.LoadAverageStats{1.0, 0.5, 0.2},
			wantErr: false,
		},
		{
			name: "when GetLoadAverageStats errors",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				mock.GetLoadAverageStatsFunc = func() (*system.LoadAverageStats, error) {
					return nil, fmt.Errorf("GetLoadAverage error")
				}
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetLoadAverageStats()
			},
			wantErr: true,
		},
		{
			name: "when GetUptime Ok",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetUptime()
			},
			want:    time.Hour * 5,
			wantErr: false,
		},
		{
			name: "when GetUptime errors",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				mock.GetUptimeFunc = func() (time.Duration, error) {
					return 0, fmt.Errorf("GetUptime error")
				}
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetUptime()
			},
			wantErr: true,
		},
		{
			name: "when GetLocalDiskStats Ok",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetLocalDiskStats()
			},
			want: []system.DiskUsageStats{
				{
					Name:  "/dev/disk1",
					Total: 500000000000,
					Used:  250000000000,
					Free:  250000000000,
				},
			},
			wantErr: false,
		},
		{
			name: "when GetLocalDiskStats errors",
			setupMock: func() *system.MockSystem {
				mock := system.NewDefaultMockSystem()
				mock.GetLocalDiskStatsFunc = func() ([]system.DiskUsageStats, error) {
					return nil, fmt.Errorf("GetLocalDiskStats error")
				}
				return mock
			},
			fn: func(m *system.MockSystem) (interface{}, error) {
				return m.GetLocalDiskStats()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			got, err := tc.fn(mock)

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuPublicTestSuite))
}
