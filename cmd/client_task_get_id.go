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
	"github.com/retr0h/osapi/internal/task"
)

// clientTaskGetIDCmd represents the clientTaskGetID command.
var clientTaskGetIDCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a task action",
	Long: `Gets a task action for viewing.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		messageID, _ := cmd.Flags().GetUint64("message-id")

		taskHandler := handler.(client.TaskHandler)
		resp, err := taskHandler.GetTaskByID(context.TODO(), messageID)
		if err != nil {
			logFatal("failed to get task endpoint", err)
		}

		errorMsg := "unknown error"
		switch resp.StatusCode() {
		case http.StatusOK:
			if jsonOutput {
				logger.Info(
					"task get",
					slog.Uint64("message_id", messageID),
					slog.Int("code", resp.StatusCode()),
					slog.String("response", string(resp.Body)),
				)
				return
			}

			if resp.JSON200 == nil {
				logFatal("failed response", fmt.Errorf("get task response was nil"))
			}

			taskData := map[string]interface{}{
				"ID":      uint64ToSafeString(resp.JSON200.Id),
				"Created": safeTime(resp.JSON200.Created),
			}
			printStyledMap(taskData)

			protoString := task.SafeMarshalTaskToString(resp.JSON200.Body)
			messageRows := [][]string{}
			messageRows = append(messageRows, []string{
				protoString,
			})

			sections := []section{
				{
					Title:   "Task",
					Headers: []string{"ACTION"},
					Rows:    messageRows,
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
	clientTaskCmd.AddCommand(clientTaskGetIDCmd)

	clientTaskGetIDCmd.PersistentFlags().
		Uint64P("message-id", "", 0, "The message ID of the message to delete")

	_ = clientTaskGetIDCmd.MarkPersistentFlagRequired("message-id")
}
