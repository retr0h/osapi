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

package process

import (
	"fmt"
	"time"

	"github.com/osapi-io/osapi/internal/job"
)

// ConditionThresholds holds thresholds for process-level conditions.
type ConditionThresholds struct {
	// MemoryPressureBytes is the RSS threshold in bytes.
	MemoryPressureBytes int64
	// HighCPUPercent is the CPU usage threshold as a percentage.
	HighCPUPercent float64
}

// EvaluateProcessConditions evaluates process-level conditions against
// thresholds and returns the condition list. Tracks transition times
// using the previous condition state.
func EvaluateProcessConditions(
	metrics *Metrics,
	thresholds ConditionThresholds,
	prev []job.Condition,
) []job.Condition {
	if metrics == nil {
		return nil
	}

	var conditions []job.Condition

	// ProcessMemoryPressure
	if thresholds.MemoryPressureBytes > 0 {
		memStatus := metrics.RSSBytes > thresholds.MemoryPressureBytes
		reason := ""
		if memStatus {
			reason = fmt.Sprintf(
				"process RSS %d bytes exceeds threshold %d bytes",
				metrics.RSSBytes,
				thresholds.MemoryPressureBytes,
			)
		}

		conditions = append(conditions, job.Condition{
			Type:               "ProcessMemoryPressure",
			Status:             memStatus,
			Reason:             reason,
			LastTransitionTime: transitionTime("ProcessMemoryPressure", memStatus, prev),
		})
	}

	// ProcessHighCPU
	if thresholds.HighCPUPercent > 0 {
		cpuStatus := metrics.CPUPercent > thresholds.HighCPUPercent
		reason := ""
		if cpuStatus {
			reason = fmt.Sprintf(
				"process CPU %.1f%% exceeds threshold %.1f%%",
				metrics.CPUPercent,
				thresholds.HighCPUPercent,
			)
		}

		conditions = append(conditions, job.Condition{
			Type:               "ProcessHighCPU",
			Status:             cpuStatus,
			Reason:             reason,
			LastTransitionTime: transitionTime("ProcessHighCPU", cpuStatus, prev),
		})
	}

	return conditions
}

// transitionTime returns the previous LastTransitionTime if the status
// hasn't changed, otherwise returns now.
func transitionTime(
	condType string,
	newStatus bool,
	prev []job.Condition,
) time.Time {
	for i := range prev {
		if prev[i].Type == condType && prev[i].Status == newStatus {
			return prev[i].LastTransitionTime
		}
	}

	return time.Now()
}
