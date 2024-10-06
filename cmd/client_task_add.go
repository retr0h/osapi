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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/client"
)

// clientTaskAddCmd represents the clientTaskAdd command.
var clientTaskAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an action to the task runner",
	Long: `Adds an action for processing by the task runner.
`,
	Run: func(cmd *cobra.Command, _ []string) {
		filePath, _ := cmd.Flags().GetString("proto-file")

		file, err := os.Open(filePath)
		if err != nil {
			logFatal("failed to open file", err)
		}
		defer func() { _ = file.Close() }()

		fileContents, err := io.ReadAll(file)
		if err != nil {
			logFatal("failed to read file", err)
		}

		encodedBody := base64.StdEncoding.EncodeToString([]byte(fileContents))
		taskHandler := handler.(client.TaskHandler)
		resp, err := taskHandler.PostTask(context.TODO(), encodedBody)
		if err != nil {
			logFatal("failed to add task endpoint", err)
		}

		errorMsg := "unknown error"
		switch resp.StatusCode() {
		case http.StatusCreated:
			if jsonOutput {
				logger.Info(
					"task add",
					slog.String("proto_file", filePath),
					slog.String("response", string(resp.Body)),
					slog.String("status", "ok"),
				)
				return
			}

			if resp.JSON201 == nil {
				logFatal("failed response", fmt.Errorf("get task response was nil"))
			}

			taskData := map[string]interface{}{
				"ID": uint64ToSafeString(resp.JSON201.Id),
			}
			printStyledMap(taskData)

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
	clientTaskCmd.AddCommand(clientTaskAddCmd)

	clientTaskAddCmd.PersistentFlags().
		StringP("proto-file", "", "", "Path to the file containing the binary protobuf data")
}
