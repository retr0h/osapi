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

var messageID string

// clientQueueGetIDCmd represents the clientQueueGetID command.
var clientQueueGetIDCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a messge from the queue",
	Long: `Gets a message item from the queue for viewing.
`,
	Run: func(_ *cobra.Command, _ []string) {
		resp, err := handler.GetQueueID(messageID)
		if err != nil {
			logFatal("failed to get queue endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				prettyPrintJSON(resp.Body)
				return
			}

			queueMsg := fmt.Sprintf(
				"\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %d\n",
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("ID"),
				safeString(resp.JSON200.Id),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Created"),
				safeTime(resp.JSON200.Created),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Updated"),
				safeTime(resp.JSON200.Updated),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Timeout"),
				safeTime(resp.JSON200.Timeout),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Received"),
				safeInt(resp.JSON200.Received),
			)

			itemRows := [][]string{}
			itemRows = append(itemRows, []string{
				safeString(resp.JSON200.Body),
			})

			sections := []section{
				{
					Headers: []string{"Body"},
					Rows:    itemRows,
				},
			}

			printStyledTable(sections, queueMsg)

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
	clientQueueCmd.AddCommand(clientQueueGetIDCmd)

	clientQueueGetIDCmd.PersistentFlags().
		StringVarP(&messageID, "message-id", "m", "", "The message ID of the queue item to fetch")
	_ = clientQueueGetIDCmd.MarkPersistentFlagRequired("message-id")
}
