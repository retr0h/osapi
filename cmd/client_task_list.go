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

	"github.com/retr0h/osapi/internal/client"
	"github.com/retr0h/osapi/internal/task"
)

// clientTaskListCmd represents the clientTaskList command.
var clientTaskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all actions",
	Long: `List all actions the task runner is processing.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		taskHandler := handler.(client.TaskHandler)
		resp, err := taskHandler.GetTaskList(context.TODO())
		if err != nil {
			logFatal("failed to get task endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"task list",
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("list task response was nil"))
			}

			if resp.JSON200.Items == nil {
				logFatal("failed response", fmt.Errorf("task items in response were nil"))
			}

			totalMessages := safeInt(resp.JSON200.TotalItems)

			messagesData := map[string]interface{}{
				"Total Items": totalMessages,
			}
			printStyledMap(messagesData)

			messageRows := [][]string{}
			for _, message := range *resp.JSON200.Items {
				createdTime := "N/A"
				if message.Created != nil {
					createdTime = message.Created.Format(time.RFC3339)
				}

				protoString := task.SafeMarshalTaskToString(message.Body)
				messageRows = append(messageRows, []string{
					uint64ToSafeString(message.Id),
					createdTime,
					protoString,
				})
			}

			sections := []section{
				{
					Title:   "Messages",
					Headers: []string{"ID", "CREATED", "ACTION"},
					Rows:    messageRows,
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
	clientTaskCmd.AddCommand(clientTaskListCmd)
}
