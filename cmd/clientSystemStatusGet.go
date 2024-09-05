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

package cmd

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
)

// clientSystemStatusGetCmd represents the clientPing command.
var clientSystemStatusGetCmd = &cobra.Command{
	Use:   "status",
	Short: "Status of the server",
	Long: `Obtain the current system status.
`,
	Run: func(_ *cobra.Command, _ []string) {
		resp, err := c.GetSystemStatus()
		if err != nil {
			logFatal("failed to get system status endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				prettyPrintJSON(resp.Body)
				return
			}

			// aggregate disk information
			diskGroups := make([]any, 0, len(*resp.JSON200.Disks))
			for i, disk := range *resp.JSON200.Disks {
				group := slog.Group(strconv.Itoa(i),
					slog.String("Name", disk.Name),
					slog.Int("Total", disk.Total),
					slog.Int("Used", disk.Used),
					slog.Int("Free", disk.Free),
				)
				diskGroups = append(diskGroups, group)
			}

			logger.Info(
				"response",
				slog.Int("code", resp.StatusCode()),
				slog.String("hostname", resp.JSON200.Hostname),
				slog.String("uptime", resp.JSON200.Uptime),
				slog.Int("load.1m", int(resp.JSON200.LoadAverage.N1min)),
				slog.Int("load.5m", int(resp.JSON200.LoadAverage.N5min)),
				slog.Int("load.15m", int(resp.JSON200.LoadAverage.N15min)),
				slog.Int("memory.Total", int(resp.JSON200.Memory.Total)),
				slog.Int("memory.Free", int(resp.JSON200.Memory.Free)),
				slog.Int("memory.Used", int(resp.JSON200.Memory.Used)),
				slog.Group("disks", diskGroups...),
			)
		default:
			if jsonOutput {
				prettyPrintJSON(resp.Body)
				return
			}

			errorMsg := "unknown error"
			if resp.JSON500 != nil {
				errorMsg = resp.JSON500.Error
			}

			logger.Error(
				"error in response",
				slog.Int("code", resp.StatusCode()),
				slog.String("error", errorMsg),
			)
		}
	},
}

func init() {
	clientSystemCmd.AddCommand(clientSystemStatusGetCmd)
}
