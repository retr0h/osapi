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
			logger.Info(
				"response",
				slog.Int("code", resp.StatusCode()),
				slog.String("hostname", resp.JSON200.Hostname),
			)
		default:
			logger.Error(
				"error in response",
				slog.Int("code", resp.StatusCode()),
				slog.String("error", resp.JSON500.Error),
			)
		}
	},
}

func init() {
	clientSystemCmd.AddCommand(clientSystemStatusGetCmd)
}
