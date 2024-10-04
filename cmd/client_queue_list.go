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
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/task"
)

// clientQueueListCmd represents the clientQueueList command.
var clientQueueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List messges in the queue",
	Long: `List all messages in the queue for viewing.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		pageSize, _ := cmd.Flags().GetInt("limit")
		pageNumber, _ := cmd.Flags().GetInt("page")
		offset := (pageNumber - 1) * pageSize

		resp, err := handler.GetQueueAll(context.TODO(), pageSize, offset)
		if err != nil {
			logFatal("failed to get queue endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"queue list",
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("list queue response was nil"))
			}

			if resp.JSON200.Items == nil {
				logFatal("failed response", fmt.Errorf("items in response were nil"))
			}

			totalItems := safeInt(resp.JSON200.TotalItems)
			totalPages := calculateTotalPages(totalItems, pageSize)

			itemsData := map[string]interface{}{
				"Total Items": totalItems,
				"Total Pages": totalPages,
			}
			printStyledMap(itemsData)

			itemRows := [][]string{}
			for _, item := range *resp.JSON200.Items {
				createdTime := "N/A"
				if item.Created != nil {
					createdTime = item.Created.Format(time.RFC3339)
				}

				protoString := task.SafeMarshalTaskToString(item.Body)
				itemRows = append(itemRows, []string{
					safeString(item.Id),
					createdTime,
					protoString,
				})
			}

			sections := []section{
				{
					Title:   "Items",
					Headers: []string{"ID", "CREATED", "ACTION"},
					Rows:    itemRows,
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
	clientQueueCmd.AddCommand(clientQueueListCmd)

	clientQueueListCmd.PersistentFlags().
		IntP("limit", "l", 10, "Number of items to fetch per page")
	clientQueueListCmd.PersistentFlags().
		IntP("page", "p", 1, "Page number to fetch")
}

func calculateTotalPages(totalItems, pageSize int) int {
	if pageSize == 0 {
		return 0 // Avoid division by zero
	}
	return (totalItems + pageSize - 1) / pageSize
}
