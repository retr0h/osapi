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
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// queueClientGetCmd represents the queueClientGet command.
var queueClientGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a messge from the queue",
	Long: `Gets a message item from the queue for viewing.
`,
	Run: func(_ *cobra.Command, _ []string) {
		item, err := qm.GetByID(context.Background(), messageID)
		if err != nil {
			logFatal("failed to get message from the queue", err)
		}

		if item != nil {
			queueMsg := fmt.Sprintf(
				"\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %d\n",
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("ID"),
				item.ID,
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Created"),
				item.Created.Format(time.RFC3339),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Updated"),
				item.Updated.Format(time.RFC3339),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Timeout"),
				item.Timeout.Format(time.RFC3339),
				lipgloss.NewStyle().
					Bold(true).
					Foreground(purple).
					Render("Received"),
				item.Received,
			)

			itemRows := [][]string{}
			itemRows = append(itemRows, []string{
				string(item.Body),
			})

			sections := []section{
				{
					Headers: []string{"Body"},
					Rows:    itemRows,
				},
			}

			printStyledTable(sections, queueMsg)
		}
	},
}

func init() {
	queueClientCmd.AddCommand(queueClientGetCmd)

	queueClientGetCmd.PersistentFlags().
		StringVarP(&messageID, "message-id", "m", "", "The message ID of the queue item to fetch")
	_ = queueClientGetCmd.MarkPersistentFlagRequired("message-id")
}
