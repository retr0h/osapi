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

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// clientPingCmd represents the clientPing command.
var clientPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the server",
	Long: `Interact with the server by issuing a ping.
`,
	Run: func(_ *cobra.Command, _ []string) {
		resp, err := handler.GetPing(context.TODO())
		if err != nil {
			logFatal("failed to get ping endpoint", err)
		}

		if jsonOutput {
			logger.Info(
				"ping",
				slog.String("response", string(resp.Body)),
			)
			return
		}

		sections := []section{{}}
		pingInfo := fmt.Sprintf(
			"\n%s: %s\n",
			lipgloss.NewStyle().Bold(true).Foreground(purple).Render("Ping"), resp.JSON200.Ping,
		)

		printStyledTable(sections, pingInfo)
	},
}

func init() {
	clientCmd.AddCommand(clientPingCmd)
}