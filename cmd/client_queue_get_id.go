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

	"github.com/retr0h/osapi/internal/task"
)

var messageID string

// clientQueueGetIDCmd represents the clientQueueGetID command.
var clientQueueGetIDCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a messge from the queue",
	Long: `Gets a message item from the queue for viewing.
`,
	Run: func(_ *cobra.Command, _ []string) {
		errorMsg := "unknown error"
		resp, err := handler.GetQueueByID(context.TODO(), messageID)
		if err != nil {
			logFatal("failed to get queue endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"queue get",
					slog.String("message_id", messageID),
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("get queue response was nil"))
			}

			queueData := map[string]interface{}{
				"ID":       safeString(resp.JSON200.Id),
				"Created":  safeTime(resp.JSON200.Created),
				"Updated":  safeTime(resp.JSON200.Updated),
				"Timeout":  safeTime(resp.JSON200.Timeout),
				"Received": safeInt(resp.JSON200.Received),
			}
			printStyledMap(queueData)

			protoString := task.SafeMarshalTaskToString(resp.JSON200.Body)
			itemRows := [][]string{}
			itemRows = append(itemRows, []string{
				protoString,
			})

			sections := []section{
				{
					Title:   "Task",
					Headers: []string{"ACTION"},
					Rows:    itemRows,
				},
			}
			printStyledTable(sections)

		case http.StatusNotFound:
			if resp.JSON404 != nil {
				errorMsg = resp.JSON404.Error
			}

			logger.Error(
				"not found",
				slog.Int("code", resp.StatusCode()),
				slog.String("response", errorMsg),
			)
		default:
			logger.Error(
				"error in response",
				slog.Int("code", resp.StatusCode()),
				slog.String("response", errorMsg),
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
