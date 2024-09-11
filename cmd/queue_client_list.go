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

var (
	pageNumber int
	pageSize   int
)

// queueClientListCmd represents the queueClientList command.
var queueClientListCmd = &cobra.Command{
	Use:   "list",
	Short: "List messges in the queue",
	Long: `List all messages in the queue for viewing.
`,
	Run: func(_ *cobra.Command, _ []string) {
		offset := (pageNumber - 1) * pageSize

		items, err := qm.GetAll(context.Background(), pageSize, offset)
		if err != nil {
			logFatal("failed to get message from the queue", err)
		}

		itemRows := [][]string{}
		for _, item := range items {
			itemRows = append(itemRows, []string{
				item.ID,
				item.Created.Format(time.RFC3339),
				item.Body,
			})
		}

		sections := []section{
			{
				Title:   "Items:",
				Headers: []string{"ID", "Created", "Body"},
				Rows:    itemRows,
			},
		}

		totalItems, err := qm.Count(context.Background())
		if err != nil {
			logFatal("failed to get queue count", err)
		}

		totalPages := calculateTotalPages(totalItems, pageSize)

		itemsInfo := fmt.Sprintf(
			"\n%s: %d\n%s: %d\n",
			lipgloss.NewStyle().Bold(true).Foreground(purple).Render("Total Items"),
			totalItems,
			lipgloss.NewStyle().Bold(true).Foreground(purple).Render("Total Pages"),
			totalPages,
		)

		printStyledTable(sections, itemsInfo)
	},
}

func init() {
	queueClientCmd.AddCommand(queueClientListCmd)

	queueClientListCmd.PersistentFlags().
		IntVarP(&pageSize, "limit", "l", 10, "Number of items to fetch per page")
	queueClientListCmd.PersistentFlags().
		IntVarP(&pageNumber, "page", "p", 1, "Page number to fetch")
}

func calculateTotalPages(totalItems, pageSize int) int {
	if pageSize == 0 {
		return 0 // Avoid division by zero
	}
	return (totalItems + pageSize - 1) / pageSize
}
