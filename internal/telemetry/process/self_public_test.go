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

package process_test

import (
	"errors"
	"os"
	"testing"

	gopsutil "github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/telemetry/process"
)

type ProcessPublicTestSuite struct {
	suite.Suite
}

func (suite *ProcessPublicTestSuite) SetupTest() {}

func (suite *ProcessPublicTestSuite) TearDownTest() {}

func (suite *ProcessPublicTestSuite) TestGetMetrics() {
	tests := []struct {
		name         string
		validateFunc func(got *process.Metrics, err error)
	}{
		{
			name: "returns non-nil metrics",
			validateFunc: func(got *process.Metrics, err error) {
				suite.NoError(err)
				suite.NotNil(got)
			},
		},
		{
			name: "goroutines is positive",
			validateFunc: func(got *process.Metrics, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Greater(got.Goroutines, 0)
			},
		},
		{
			name: "rss is positive",
			validateFunc: func(got *process.Metrics, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Greater(got.RSSBytes, int64(0))
			},
		},
		{
			name: "cpu is non-negative",
			validateFunc: func(got *process.Metrics, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.GreaterOrEqual(got.CPUPercent, float64(0))
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			p := process.New()

			got, err := p.GetMetrics()

			tc.validateFunc(got, err)
		})
	}
}

func (suite *ProcessPublicTestSuite) TestGetMetricsWithInjection() {
	tests := []struct {
		name         string
		pid          int32
		setup        func()
		teardown     func()
		validateFunc func(got *process.Metrics, err error)
	}{
		{
			name: "returns metrics for current process using real functions",
			pid:  int32(os.Getpid()),
			validateFunc: func(got *process.Metrics, err error) {
				suite.NoError(err)
				suite.NotNil(got)
				suite.GreaterOrEqual(got.CPUPercent, float64(0))
				suite.Greater(got.RSSBytes, int64(0))
				suite.Greater(got.Goroutines, 0)
			},
		},
		{
			name: "returns error for invalid pid",
			pid:  -99999,
			validateFunc: func(got *process.Metrics, err error) {
				suite.Nil(got)
				suite.Error(err)
				suite.Contains(err.Error(), "get process")
			},
		},
		{
			name: "returns error when CPUPercent fails",
			pid:  0,
			setup: func() {
				process.SetNewProcessFn(func(_ int32) (*gopsutil.Process, error) {
					return &gopsutil.Process{}, nil
				})
				process.SetCPUPercentFn(func(_ *gopsutil.Process) (float64, error) {
					return 0, errors.New("cpu error")
				})
			},
			teardown: func() {
				process.ResetNewProcessFn()
				process.ResetCPUPercentFn()
			},
			validateFunc: func(got *process.Metrics, err error) {
				suite.Nil(got)
				suite.Error(err)
				suite.Contains(err.Error(), "get cpu percent")
			},
		},
		{
			name: "returns error when MemoryInfo fails",
			pid:  0,
			setup: func() {
				process.SetNewProcessFn(func(_ int32) (*gopsutil.Process, error) {
					return &gopsutil.Process{}, nil
				})
				process.SetCPUPercentFn(func(_ *gopsutil.Process) (float64, error) {
					return 1.5, nil
				})
				process.SetMemoryInfoFn(func(_ *gopsutil.Process) (uint64, error) {
					return 0, errors.New("memory error")
				})
			},
			teardown: func() {
				process.ResetNewProcessFn()
				process.ResetCPUPercentFn()
				process.ResetMemoryInfoFn()
			},
			validateFunc: func(got *process.Metrics, err error) {
				suite.Nil(got)
				suite.Error(err)
				suite.Contains(err.Error(), "get memory info")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.setup != nil {
				tc.setup()
			}
			if tc.teardown != nil {
				defer tc.teardown()
			}

			p := process.ExportNewProviderWithPID(tc.pid)
			got, err := p.GetMetrics()
			tc.validateFunc(got, err)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestProcessPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ProcessPublicTestSuite))
}
