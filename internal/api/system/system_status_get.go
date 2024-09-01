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
	"github.com/retr0h/osapi/internal/manager/system"
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

	return ctx.JSON(http.StatusOK, gen.SystemStatus{
		Hostname: hostname,
		Uptime:   formatDuration(uptime),
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
