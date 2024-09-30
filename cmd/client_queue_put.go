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
	"encoding/base64"
	"log/slog"
	"net/http"

	"github.com/spf13/cobra"
)

var messageBody string

// clientQueuePutCmd represents the clientQueuePut command.
var clientQueuePutCmd = &cobra.Command{
	Use:   "put",
	Short: "Put a messge into the queue",
	Long: `Puts a base64 encoded message into the queue for processing by the task runner.
`,
	Run: func(_ *cobra.Command, _ []string) {
		errorMsg := "unknown error"

		_, err := base64.StdEncoding.DecodeString(messageBody)
		if err != nil {
			logFatal("invalid Base64-encoded message body", err)
		}

		resp, err := handler.PostQueue(context.TODO(), messageBody)
		if err != nil {
			logFatal("failed to post queue endpoint", err)
		}

		switch resp.StatusCode() {
		case http.StatusCreated:
			logger.Info(
				"queue put",
				slog.String("message_body", messageBody),
				slog.String("response", string(resp.Body)),
				slog.String("status", "ok"),
			)

		case http.StatusBadRequest:
			if resp.JSON400 != nil {
				errorMsg = resp.JSON400.Error
			}

			logger.Error(
				"bad request",
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
	clientQueueCmd.AddCommand(clientQueuePutCmd)

	clientQueuePutCmd.PersistentFlags().
		StringVarP(&messageBody, "message-body", "b", "", "The Base64-encoded message body of the queue item to add")
	_ = clientQueuePutCmd.MarkPersistentFlagRequired("message-body")
}