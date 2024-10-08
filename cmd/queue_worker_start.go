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

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/queue"
	"github.com/retr0h/osapi/internal/worker"
)

// queueWorkerStartCmd represents the queueWorkerStart command.
var queueWorkerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Long: `Start the queue worker.
It continuously checks the queue every N seconds and processes any tasks.
`,
	Run: func(_ *cobra.Command, _ []string) {
		db, err := queue.OpenDB(appConfig)
		if err != nil {
			logFatal("failed to open database", err)
		}

		var qm queue.Manager = queue.New(logger, appConfig, db)

		err = qm.SetupSchema(context.Background())
		if err != nil {
			logFatal("failed to set up database schema", err)
		}

		qm.SetupQueue()

		var workerServer worker.ServerManager = worker.New(appFs, appConfig, logger, qm)
		workerServer.Run()
	},
}

func init() {
	queueWorkerCmd.AddCommand(queueWorkerStartCmd)
}
