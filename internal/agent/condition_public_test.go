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

package agent_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/agent"
	"github.com/osapi-io/osapi/internal/job"
	"github.com/osapi-io/osapi/internal/provider/node/disk"
	"github.com/osapi-io/osapi/internal/provider/node/load"
	"github.com/osapi-io/osapi/internal/provider/node/mem"
)

type ConditionPublicTestSuite struct {
	suite.Suite
}

func (s *ConditionPublicTestSuite) TestFindPrevCondition() {
	tests := []struct {
		name         string
		condType     string
		prev         []job.Condition
		validateFunc func(*job.Condition)
	}{
		{
			name:     "when condition type is found returns pointer",
			condType: job.ConditionMemoryPressure,
			prev: []job.Condition{
				{
					Type:   job.ConditionMemoryPressure,
					Status: true,
					Reason: "high",
				},
				{
					Type:   job.ConditionHighLoad,
					Status: false,
				},
			},
			validateFunc: func(c *job.Condition) {
				s.Require().NotNil(c)
				s.Equal(job.ConditionMemoryPressure, c.Type)
				s.True(c.Status)
				s.Equal("high", c.Reason)
			},
		},
		{
			name:     "when condition type is not found returns nil",
			condType: job.ConditionDiskPressure,
			prev: []job.Condition{
				{
					Type:   job.ConditionMemoryPressure,
					Status: true,
				},
			},
			validateFunc: func(c *job.Condition) {
				s.Nil(c)
			},
		},
		{
			name:     "when prev is empty returns nil",
			condType: job.ConditionHighLoad,
			prev:     []job.Condition{},
			validateFunc: func(c *job.Condition) {
				s.Nil(c)
			},
		},
		{
			name:     "when prev is nil returns nil",
			condType: job.ConditionHighLoad,
			prev:     nil,
			validateFunc: func(c *job.Condition) {
				s.Nil(c)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportFindPrevCondition(tt.condType, tt.prev)
			tt.validateFunc(result)
		})
	}
}

func (s *ConditionPublicTestSuite) TestTransitionTime() {
	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		condType     string
		newStatus    bool
		prev         []job.Condition
		validateFunc func(time.Time)
	}{
		{
			name:      "when matching prev has same status preserves transition time",
			condType:  job.ConditionHighLoad,
			newStatus: true,
			prev: []job.Condition{
				{
					Type:               job.ConditionHighLoad,
					Status:             true,
					LastTransitionTime: fixedTime,
				},
			},
			validateFunc: func(t time.Time) {
				s.Equal(fixedTime, t)
			},
		},
		{
			name:      "when matching prev has different status returns now",
			condType:  job.ConditionHighLoad,
			newStatus: true,
			prev: []job.Condition{
				{
					Type:               job.ConditionHighLoad,
					Status:             false,
					LastTransitionTime: fixedTime,
				},
			},
			validateFunc: func(t time.Time) {
				s.NotEqual(fixedTime, t)
				s.WithinDuration(time.Now(), t, 2*time.Second)
			},
		},
		{
			name:      "when no matching prev returns now",
			condType:  job.ConditionDiskPressure,
			newStatus: true,
			prev: []job.Condition{
				{
					Type:               job.ConditionHighLoad,
					Status:             true,
					LastTransitionTime: fixedTime,
				},
			},
			validateFunc: func(t time.Time) {
				s.WithinDuration(time.Now(), t, 2*time.Second)
			},
		},
		{
			name:      "when prev is empty returns now",
			condType:  job.ConditionHighLoad,
			newStatus: false,
			prev:      []job.Condition{},
			validateFunc: func(t time.Time) {
				s.WithinDuration(time.Now(), t, 2*time.Second)
			},
		},
		{
			name:      "when prev is nil returns now",
			condType:  job.ConditionHighLoad,
			newStatus: false,
			prev:      nil,
			validateFunc: func(t time.Time) {
				s.WithinDuration(time.Now(), t, 2*time.Second)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportTransitionTime(tt.condType, tt.newStatus, tt.prev)
			tt.validateFunc(result)
		})
	}
}

func (s *ConditionPublicTestSuite) TestEvaluateMemoryPressure() {
	tests := []struct {
		name         string
		stats        *mem.Result
		threshold    int
		prev         []job.Condition
		validateFunc func(job.Condition)
	}{
		{
			name: "when usage above threshold returns true with reason",
			stats: &mem.Result{
				Total:     8 * 1024 * 1024 * 1024, // 8 GB
				Available: 1 * 1024 * 1024 * 1024, // 1 GB available = 87.5% used
			},
			threshold: 80,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionMemoryPressure, c.Type)
				s.True(c.Status)
				s.Contains(c.Reason, "memory")
				s.Contains(c.Reason, "88%")
				s.Contains(c.Reason, "GB")
			},
		},
		{
			name: "when usage below threshold returns false",
			stats: &mem.Result{
				Total:     8 * 1024 * 1024 * 1024, // 8 GB
				Available: 6 * 1024 * 1024 * 1024, // 6 GB available = 25% used
			},
			threshold: 80,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionMemoryPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name:      "when stats is nil returns false",
			stats:     nil,
			threshold: 80,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionMemoryPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name: "when total is zero returns false",
			stats: &mem.Result{
				Total:     0,
				Available: 0,
			},
			threshold: 80,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionMemoryPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name: "when usage exactly at threshold returns false",
			stats: &mem.Result{
				Total:     100,
				Available: 20, // 80% used, threshold is 80 (> not >=)
			},
			threshold: 80,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionMemoryPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportEvaluateMemoryPressure(tt.stats, tt.threshold, tt.prev)
			tt.validateFunc(result)
		})
	}
}

func (s *ConditionPublicTestSuite) TestEvaluateHighLoad() {
	tests := []struct {
		name         string
		loadAvg      *load.Result
		cpuCount     int
		multiplier   float64
		prev         []job.Condition
		validateFunc func(job.Condition)
	}{
		{
			name: "when load above threshold returns true with reason",
			loadAvg: &load.Result{
				Load1:  8.5,
				Load5:  7.0,
				Load15: 6.0,
			},
			cpuCount:   4,
			multiplier: 2.0, // threshold = 8.0
			prev:       nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionHighLoad, c.Type)
				s.True(c.Status)
				s.Contains(c.Reason, "load 8.50")
				s.Contains(c.Reason, "threshold 8.00")
				s.Contains(c.Reason, "4 CPUs")
			},
		},
		{
			name: "when load below threshold returns false",
			loadAvg: &load.Result{
				Load1:  2.0,
				Load5:  1.5,
				Load15: 1.0,
			},
			cpuCount:   4,
			multiplier: 2.0, // threshold = 8.0
			prev:       nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionHighLoad, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name:       "when load is nil returns false",
			loadAvg:    nil,
			cpuCount:   4,
			multiplier: 2.0,
			prev:       nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionHighLoad, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name: "when cpu count is zero returns false",
			loadAvg: &load.Result{
				Load1:  8.5,
				Load5:  7.0,
				Load15: 6.0,
			},
			cpuCount:   0,
			multiplier: 2.0,
			prev:       nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionHighLoad, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name: "when load exactly at threshold returns false",
			loadAvg: &load.Result{
				Load1:  8.0,
				Load5:  5.0,
				Load15: 3.0,
			},
			cpuCount:   4,
			multiplier: 2.0, // threshold = 8.0, Load1 = 8.0 (not >)
			prev:       nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionHighLoad, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportEvaluateHighLoad(tt.loadAvg, tt.cpuCount, tt.multiplier, tt.prev)
			tt.validateFunc(result)
		})
	}
}

func (s *ConditionPublicTestSuite) TestEvaluateDiskPressure() {
	tests := []struct {
		name         string
		disks        []disk.Result
		threshold    int
		prev         []job.Condition
		validateFunc func(job.Condition)
	}{
		{
			name: "when one disk above threshold returns true",
			disks: []disk.Result{
				{
					Name:  "/dev/sda1",
					Total: 100 * 1024 * 1024 * 1024, // 100 GB
					Used:  95 * 1024 * 1024 * 1024,  // 95 GB = 95%
					Free:  5 * 1024 * 1024 * 1024,
				},
			},
			threshold: 90,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionDiskPressure, c.Type)
				s.True(c.Status)
				s.Contains(c.Reason, "/dev/sda1")
				s.Contains(c.Reason, "95%")
				s.Contains(c.Reason, "GB")
			},
		},
		{
			name: "when all disks below threshold returns false",
			disks: []disk.Result{
				{
					Name:  "/dev/sda1",
					Total: 100 * 1024 * 1024 * 1024,
					Used:  50 * 1024 * 1024 * 1024, // 50%
					Free:  50 * 1024 * 1024 * 1024,
				},
				{
					Name:  "/dev/sdb1",
					Total: 200 * 1024 * 1024 * 1024,
					Used:  60 * 1024 * 1024 * 1024, // 30%
					Free:  140 * 1024 * 1024 * 1024,
				},
			},
			threshold: 90,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionDiskPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name:      "when disks is nil returns false",
			disks:     nil,
			threshold: 90,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionDiskPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name:      "when disks is empty returns false",
			disks:     []disk.Result{},
			threshold: 90,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionDiskPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name: "when disk total is zero skips it",
			disks: []disk.Result{
				{
					Name:  "/dev/sda1",
					Total: 0,
					Used:  0,
					Free:  0,
				},
			},
			threshold: 90,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionDiskPressure, c.Type)
				s.False(c.Status)
				s.Empty(c.Reason)
			},
		},
		{
			name: "when second disk is above threshold reports it",
			disks: []disk.Result{
				{
					Name:  "/dev/sda1",
					Total: 100 * 1024 * 1024 * 1024,
					Used:  50 * 1024 * 1024 * 1024, // 50%
					Free:  50 * 1024 * 1024 * 1024,
				},
				{
					Name:  "/dev/sdb1",
					Total: 200 * 1024 * 1024 * 1024,
					Used:  195 * 1024 * 1024 * 1024, // 97.5%
					Free:  5 * 1024 * 1024 * 1024,
				},
			},
			threshold: 90,
			prev:      nil,
			validateFunc: func(c job.Condition) {
				s.Equal(job.ConditionDiskPressure, c.Type)
				s.True(c.Status)
				s.Contains(c.Reason, "/dev/sdb1")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := agent.ExportEvaluateDiskPressure(tt.disks, tt.threshold, tt.prev)
			tt.validateFunc(result)
		})
	}
}

func (s *ConditionPublicTestSuite) TestLastTransitionTimeTracking() {
	fixedPast := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		evalFunc     func([]job.Condition) job.Condition
		prev         []job.Condition
		validateFunc func(job.Condition)
	}{
		{
			name: "when status flips from false to true transition time updates",
			evalFunc: func(prev []job.Condition) job.Condition {
				return agent.ExportEvaluateMemoryPressure(
					&mem.Result{
						Total:     100,
						Available: 10, // 90% used
					},
					80,
					prev,
				)
			},
			prev: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             false,
					LastTransitionTime: fixedPast,
				},
			},
			validateFunc: func(c job.Condition) {
				s.True(c.Status)
				s.NotEqual(fixedPast, c.LastTransitionTime)
				s.WithinDuration(time.Now(), c.LastTransitionTime, 2*time.Second)
			},
		},
		{
			name: "when status stays true transition time is preserved",
			evalFunc: func(prev []job.Condition) job.Condition {
				return agent.ExportEvaluateMemoryPressure(
					&mem.Result{
						Total:     100,
						Available: 10, // 90% used
					},
					80,
					prev,
				)
			},
			prev: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             true,
					LastTransitionTime: fixedPast,
				},
			},
			validateFunc: func(c job.Condition) {
				s.True(c.Status)
				s.Equal(fixedPast, c.LastTransitionTime)
			},
		},
		{
			name: "when status flips from true to false transition time updates",
			evalFunc: func(prev []job.Condition) job.Condition {
				return agent.ExportEvaluateMemoryPressure(
					&mem.Result{
						Total:     100,
						Available: 80, // 20% used
					},
					80,
					prev,
				)
			},
			prev: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             true,
					LastTransitionTime: fixedPast,
				},
			},
			validateFunc: func(c job.Condition) {
				s.False(c.Status)
				s.NotEqual(fixedPast, c.LastTransitionTime)
				s.WithinDuration(time.Now(), c.LastTransitionTime, 2*time.Second)
			},
		},
		{
			name: "when status stays false transition time is preserved",
			evalFunc: func(prev []job.Condition) job.Condition {
				return agent.ExportEvaluateMemoryPressure(
					&mem.Result{
						Total:     100,
						Available: 80, // 20% used
					},
					80,
					prev,
				)
			},
			prev: []job.Condition{
				{
					Type:               job.ConditionMemoryPressure,
					Status:             false,
					LastTransitionTime: fixedPast,
				},
			},
			validateFunc: func(c job.Condition) {
				s.False(c.Status)
				s.Equal(fixedPast, c.LastTransitionTime)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := tt.evalFunc(tt.prev)
			tt.validateFunc(result)
		})
	}
}

func TestConditionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConditionPublicTestSuite))
}
