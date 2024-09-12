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
	// "context"
	// "log/slog"

	"github.com/spf13/cobra"
)

// queueClientDeleteCmd represents the queueClientDelete command.
var queueClientDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a messge from the queue",
	Long: `Deletes a message item from the queue.
`,
	Run: func(_ *cobra.Command, _ []string) {
		// err := qm.DeleteByID(context.Background(), messageID)
		// if err != nil {
		// 	logFatal("failed to get message from the queue", err)
		// }

		// logger.Info(
		// 	"queue delete",
		// 	slog.String("messageID", messageID),
		// 	slog.String("status", "ok"),
		// )
	},
}

func init() {
	queueClientCmd.AddCommand(queueClientDeleteCmd)

	queueClientDeleteCmd.PersistentFlags().
		StringVarP(&messageID, "message-id", "m", "", "The message ID of the queue item to delete")
	_ = queueClientDeleteCmd.MarkPersistentFlagRequired("message-id")
}
