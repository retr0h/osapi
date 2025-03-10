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

	"github.com/spf13/afero"

	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/task/client"
)

// New initialize and configure a new queue Worker service.
func New(
	appFs afero.Fs,
	appConfig config.Config,
	logger *slog.Logger,
	cm client.Manager,
) *Worker {
	return &Worker{
		appFs:         appFs,
		logger:        logger,
		appConfig:     appConfig,
		ClientManager: cm,
	}
}

// Start begins the worker process, subscribing to a JetStream stream and
// processing messages until the context is canceled.
func (w *Worker) Start(ctx context.Context) {
	iter, err := w.ClientManager.GetMessageIterator(ctx)
	if err != nil {
		w.logger.Error(
			"error subscribing to stream",
			slog.String("error", err.Error()),
		)
		return
	}

	w.logger.Info("worker started successfully")

	// Run message processing in a separate goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				w.logger.Info("context canceled, stopping worker")
				iter.Stop()
				return

			default:
				// Process the next message
				msg, err := iter.Next()
				if err != nil {
					if err.Error() == "nats: messages iterator closed" {
						w.logger.Info("iterator closed, exiting message loop")
						return
					}
					w.logger.Error(
						"error fetching message",
						slog.String("error", err.Error()),
					)
					continue
				}

				if msg != nil {
					metadata, err := msg.Metadata()
					if err != nil {
						w.logger.Error(
							"error retrieving message metadata",
							slog.String("error", err.Error()),
						)
						return
					}

					seq := metadata.Sequence.Stream
					err = w.reconcile(seq, msg.Data())
					if err != nil {
						w.logger.Error(
							"error reconciling message, not acking",
							slog.String("error", err.Error()),
							slog.Uint64("seq", seq),
						)
					} else {
						// Acknowledge the message if reconcile was successful
						if err := msg.Ack(); err != nil {
							w.logger.Error(
								"error acking message",
								slog.String("error", err.Error()),
								slog.Uint64("seq", seq),
							)
						}
					}
				}
			}
		}
	}()

	// Wait for either context cancellation or message processing completion
	select {
	case <-ctx.Done():
		w.logger.Info("stopping worker")
		iter.Stop()
	case <-done:
		w.logger.Info("worker completed message processing")
	}

	w.logger.Info("worker shut down gracefully")
}
