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
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/provider/system"
)

// GetSystemStatus (GET /system/status)
func (s System) GetSystemStatus(
	ctx echo.Context,
) error {
	var sm system.Manager = system.New(s.appFs)
	sm.RegisterProviders()

	hostname, err := sm.GetHostname()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: err.Error(),
		})
	}

	uptime, err := sm.GetUptime()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: err.Error(),
		})
	}

	loadAvg, err := sm.GetLoadAverage()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: err.Error(),
		})
	}

	memory, err := sm.GetMemory()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, gen.ErrorResponse{
			Error: err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, gen.SystemStatus{
		Hostname: hostname,
		Uptime:   formatDuration(uptime),
		LoadAverage: gen.LoadAverage{
			N1min:  loadAvg[0],
			N5min:  loadAvg[1],
			N15min: loadAvg[2],
		},
		// Memory: When float64 values are encoded into JSON, large numbers can
		// automatically be formatted in scientific notation, which is typical
		// behavior for JSON encoders to save space and maintain precision.
		Memory: gen.Memory{
			Total: uint64ToInt(memory[0]),
			Free:  uint64ToInt(memory[1]),
			Used:  uint64ToInt(memory[2]),
		},
	})
}

func formatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	// Pluralize days, hours, and minutes
	dayStr := "day"
	if days != 1 {
		dayStr = "days"
	}

	hourStr := "hour"
	if hours != 1 {
		hourStr = "hours"
	}

	minuteStr := "minute"
	if minutes != 1 {
		minuteStr = "minutes"
	}

	// Format the result as a string
	return fmt.Sprintf("%d %s, %d %s, %d %s", days, dayStr, hours, hourStr, minutes, minuteStr)
}

// uint64ToInt convert uint64 to int, with overflow protection.
func uint64ToInt(value uint64) int {
	maxInt := int(^uint(0) >> 1) // maximum value of int based on the architecture
	if value > uint64(maxInt) {  // check for overflow
		return maxInt // return max int to prevent overflow
	}
	return int(value) // conversion within bounds
}
