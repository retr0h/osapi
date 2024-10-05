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

package worker

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/queue"
)

// New initialize and configure a new queue Worker service.
func New(
	appConfig config.Config,
	logger *slog.Logger,
	qm queue.Manager,
) *Worker {
	return &Worker{
		logger:    logger,
		appConfig: appConfig,
		qm:        qm,
	}
}

// start runs the worker that checks the queue and processes tasks.
func (w *Worker) start(ctx context.Context) {
	checkInterval := time.Duration(w.appConfig.Queue.PollInterval.Seconds) * time.Second

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker shutting down")
			return
		case <-time.After(checkInterval):
			w.logger.Info("checking items in the queue")

			m, err := w.qm.Get(ctx)
			if err != nil {
				w.logger.Error("error getting queue item", slog.String("error", err.Error()))
				continue
			}

			if m == nil {
				w.logger.Info("no queue items found")
				continue
			}

			w.logger.Info("processing item", slog.Any("id", m.ID))

			// Simulate processing
			time.Sleep(5 * time.Second)

			err = w.qm.DeleteByID(ctx, string(m.ID))
			if err != nil {
				w.logger.Error("error deleting item", slog.Any("id", m.ID), slog.String("error", err.Error()))
				continue
			}

			w.logger.Info("item processed successfully", slog.Any("id", m.ID))
		}
	}
}

// Run starts the worker and listens for signals to shut down gracefully.
func (w *Worker) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Handle OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		w.start(ctx)
	}()

	go func() {
		<-signalChan
		w.logger.Info("received interrupt signal, shutting down gracefully")
		cancel()
	}()

	wg.Wait()
}
