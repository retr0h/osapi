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
	"fmt"
	"log/slog"
	"net/http"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
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

			diskRows := [][]string{}
			for _, disk := range resp.JSON200.Disks {
				diskRows = append(diskRows, []string{
					disk.Name,
					fmt.Sprintf("%d GB", disk.Total/1024/1024/1024),
					fmt.Sprintf("%d GB", disk.Used/1024/1024/1024),
					fmt.Sprintf("%d GB", disk.Free/1024/1024/1024),
				})
			}

			sections := []section{
				{
					Title:   "Disks:",
					Headers: []string{"DISK NAME", "TOTAL", "USED", "FREE"},
					Rows:    diskRows,
				},
			}

			systemInfo := fmt.Sprintf(
				"\n%s: %s\n%s: %s\n%s: %.2f, %.2f, %.2f\n%s: %d GB used / %d GB total / %d GB free\n",
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Hostname"),
				resp.JSON200.Hostname,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Uptime"),
				resp.JSON200.Uptime,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Load Average (1m, 5m, 15m)"),
				resp.JSON200.LoadAverage.N1min,
				resp.JSON200.LoadAverage.N5min,
				resp.JSON200.LoadAverage.N15min,
				lipgloss.NewStyle().Bold(true).Foreground(purple).Render("Memory"),
				resp.JSON200.Memory.Used/1024/1024/1024,
				resp.JSON200.Memory.Total/1024/1024/1024,
				resp.JSON200.Memory.Free/1024/1024/1024,
			)

			printStyledTable(sections, systemInfo)
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
