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
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/queue"
)

func worker(ctx context.Context, qm queue.Manager) {
	checkInterval := time.Duration(appConfig.Queue.PollInterval.Seconds) * time.Second

	for {
		// Check if the context is done before processing the queue
		select {
		case <-ctx.Done():
			logger.Info("worker shutting down")
			return
		case <-time.After(checkInterval):
			logger.Info("checking items in the queue")

			// Retrieve item from the queue
			m, err := qm.Get(ctx)
			if err != nil {
				logger.Error(
					"error getting queue item",
					slog.String("error", err.Error()),
				)
				continue
			}

			// No messages found, retry
			if m == nil {
				logger.Info("no queue items found")
				continue
			}

			// Process message
			logger.Info(
				"processing item",
				slog.Any("id", m.ID),
			)

			// TODO(retr0h): Implement
			// Simulate message processing
			time.Sleep(5 * time.Second)

			// Delete the message from the queue
			err = qm.DeleteByID(ctx, string(m.ID))
			if err != nil {
				logger.Error(
					"error deleting item",
					slog.Any("id", m.ID),
					slog.String("error", err.Error()),
				)
				continue
			}

			logger.Info(
				"item processed successfully",
				slog.Any("id", m.ID),
			)
		}
	}
}

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

		// Create a cancellable context
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup

		// Listen for OS signals to gracefully cancel the worker
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, qm)
		}()

		go func() {
			<-signalChan
			logger.Info("received interrupt signal, shutting down gracefully")
			cancel()
		}()

		// Wait for the worker to finish
		wg.Wait()
	},
}

func init() {
	queueWorkerCmd.AddCommand(queueWorkerStartCmd)
}
