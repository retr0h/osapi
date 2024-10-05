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
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/client"
)

// clientSystemStatusGetCmd represents the clientSystemStatusGet command.
var clientSystemStatusGetCmd = &cobra.Command{
	Use:   "status",
	Short: "Status of the server",
	Long: `Obtain the current system status.
`,
	Run: func(_ *cobra.Command, _ []string) {
		systemHandler := handler.(client.SystemHandler)
		resp, err := systemHandler.GetSystemStatus(context.TODO())
		if err != nil {
			logFatal("failed to get system status endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"system status",
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("system data response was nil"))
			}

			systemData := map[string]interface{}{
				"Load Average (1m, 5m, 15m)": fmt.Sprintf(
					"%.2f, %.2f, %.2f",
					resp.JSON200.LoadAverage.N1min,
					resp.JSON200.LoadAverage.N5min,
					resp.JSON200.LoadAverage.N15min,
				),
				"Memory": fmt.Sprintf(
					"%d GB used / %d GB total / %d GB free",
					resp.JSON200.Memory.Used/1024/1024/1024,
					resp.JSON200.Memory.Total/1024/1024/1024,
					resp.JSON200.Memory.Free/1024/1024/1024,
				),
				"OS": fmt.Sprintf(
					"%s %s",
					resp.JSON200.OsInfo.Distribution,
					resp.JSON200.OsInfo.Version,
				),
			}
			printStyledMap(systemData)

			diskRows := [][]string{}

			if resp.JSON200.Disks != nil {
				for _, disk := range resp.JSON200.Disks {
					diskRows = append(diskRows, []string{
						disk.Name,
						fmt.Sprintf("%d GB", disk.Total/1024/1024/1024),
						fmt.Sprintf("%d GB", disk.Used/1024/1024/1024),
						fmt.Sprintf("%d GB", disk.Free/1024/1024/1024),
					})
				}
			}

			sections := []section{
				{
					Title:   "Disks",
					Headers: []string{"DISK NAME", "TOTAL", "USED", "FREE"},
					Rows:    diskRows,
				},
			}
			printStyledTable(sections)
		default:
			errorMsg := "unknown error"
			if resp.JSON500 != nil {
				errorMsg = resp.JSON500.Error
			}

			logger.Error(
				"error in response",
				slog.Int("code", resp.StatusCode()),
				slog.String("response", errorMsg),
			)
		}
	},
}

func init() {
	clientSystemCmd.AddCommand(clientSystemStatusGetCmd)
}
