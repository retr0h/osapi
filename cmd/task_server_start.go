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
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/task/server"
)

// taskServerStartCmd represents the taskServerStart command.
var taskServerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Long: `Start the embedded NATS server for task management.
The NATS server is embedded, which means it is run as part of the application
itself, removing the need for a separately deployed server. This setup ensures
a seamless integration for managing real-time communication and message distribution.
`,
	Run: func(_ *cobra.Command, _ []string) {
		var sm server.Manager = server.New(appConfig, logger)
		err := sm.Start()
		if err != nil {
			logFatal("failed to start server", err)
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		sm.Stop()
	},
}

func init() {
	taskServerCmd.AddCommand(taskServerStartCmd)
}
