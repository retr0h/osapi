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

package process_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/telemetry/process"
)

type ConditionPublicTestSuite struct {
	suite.Suite
}

func (s *ConditionPublicTestSuite) TestEvaluateProcessConditions() {
	tests := []struct {
		name         string
		metrics      *process.Metrics
		thresholds   process.ConditionThresholds
		prev         []job.Condition
		validateFunc func(got []job.Condition)
	}{
		{
			name:    "returns nil when metrics is nil",
			metrics: nil,
			thresholds: process.ConditionThresholds{
				MemoryPressureBytes: 100,
				HighCPUPercent:      50,
			},
			prev: nil,
			validateFunc: func(got []job.Condition) {
				s.Nil(got)
			},
		},
		{
			name: "returns empty when both thresholds are zero",
			metrics: &process.Metrics{
				RSSBytes:   1024 * 1024 * 512,
				CPUPercent: 95.0,
			},
			thresholds: process.ConditionThresholds{},
			prev:       nil,
			validateFunc: func(got []job.Condition) {
				s.Empty(got)
			},
		},
		{
			name: "memory pressure false when RSS is below threshold",
			metrics: &process.Metrics{
				RSSBytes:   50,
				CPUPercent: 0,
			},
			thresholds: process.ConditionThresholds{MemoryPressureBytes: 100},
			prev:       nil,
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 1)
				s.Equal("ProcessMemoryPressure", got[0].Type)
				s.False(got[0].Status)
				s.Empty(got[0].Reason)
			},
		},
		{
			name: "memory pressure true when RSS exceeds threshold",
			metrics: &process.Metrics{
				RSSBytes:   200,
				CPUPercent: 0,
			},
			thresholds: process.ConditionThresholds{MemoryPressureBytes: 100},
			prev:       nil,
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 1)
				s.Equal("ProcessMemoryPressure", got[0].Type)
				s.True(got[0].Status)
				s.NotEmpty(got[0].Reason)
			},
		},
		{
			name: "CPU pressure false when CPU is below threshold",
			metrics: &process.Metrics{
				RSSBytes:   0,
				CPUPercent: 30.0,
			},
			thresholds: process.ConditionThresholds{HighCPUPercent: 80.0},
			prev:       nil,
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 1)
				s.Equal("ProcessHighCPU", got[0].Type)
				s.False(got[0].Status)
				s.Empty(got[0].Reason)
			},
		},
		{
			name: "CPU pressure true when CPU exceeds threshold",
			metrics: &process.Metrics{
				RSSBytes:   0,
				CPUPercent: 90.0,
			},
			thresholds: process.ConditionThresholds{HighCPUPercent: 80.0},
			prev:       nil,
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 1)
				s.Equal("ProcessHighCPU", got[0].Type)
				s.True(got[0].Status)
				s.NotEmpty(got[0].Reason)
			},
		},
		{
			name: "both conditions active when both thresholds exceeded",
			metrics: &process.Metrics{
				RSSBytes:   500,
				CPUPercent: 90.0,
			},
			thresholds: process.ConditionThresholds{
				MemoryPressureBytes: 100,
				HighCPUPercent:      80.0,
			},
			prev: nil,
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 2)
				s.Equal("ProcessMemoryPressure", got[0].Type)
				s.True(got[0].Status)
				s.Equal("ProcessHighCPU", got[1].Type)
				s.True(got[1].Status)
			},
		},
		{
			name: "transition time preserved when status unchanged",
			metrics: &process.Metrics{
				RSSBytes:   200,
				CPUPercent: 0,
			},
			thresholds: process.ConditionThresholds{MemoryPressureBytes: 100},
			prev: []job.Condition{
				{
					Type:               "ProcessMemoryPressure",
					Status:             true,
					LastTransitionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 1)
				s.Equal("ProcessMemoryPressure", got[0].Type)
				s.True(got[0].Status)
				s.Equal(
					time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					got[0].LastTransitionTime,
				)
			},
		},
		{
			name: "transition time updated when status changes",
			metrics: &process.Metrics{
				RSSBytes:   50,
				CPUPercent: 0,
			},
			thresholds: process.ConditionThresholds{MemoryPressureBytes: 100},
			prev: []job.Condition{
				{
					Type:               "ProcessMemoryPressure",
					Status:             true, // was true, now false
					LastTransitionTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			validateFunc: func(got []job.Condition) {
				s.Require().Len(got, 1)
				s.False(got[0].Status)
				// Transition time must be after the old one.
				s.True(
					got[0].LastTransitionTime.After(
						time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					),
				)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := process.EvaluateProcessConditions(tt.metrics, tt.thresholds, tt.prev)
			tt.validateFunc(got)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestConditionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConditionPublicTestSuite))
}
