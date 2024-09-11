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

package queue

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/maragudk/goqite"
	_ "github.com/mattn/go-sqlite3" //lint:ignore

	"github.com/retr0h/osapi/internal/config"
)

// New factory to create a new instance.
func New(
	logger *slog.Logger,
	appConfig config.Config,
	db *sql.DB,
) *Queue {
	return &Queue{
		logger:    logger,
		appConfig: appConfig,
		DB:        db,
	}
}

// SetupQueue sets up the queue with the specified name.
func (q *Queue) SetupQueue() error {
	q.Queue = goqite.New(goqite.NewOpts{
		DB:   q.DB,
		Name: "osapi_jobs",
	})
	return nil
}

// Get a message from the queue, during which time it's not available to
// other consumers (until the message timeout has passed).
func (q *Queue) Get(
	ctx context.Context,
) (*goqite.Message, error) {
	m, err := q.Queue.Receive(ctx)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Put the message into the queue.
func (q *Queue) Put(
	ctx context.Context,
	data []byte,
) error {
	err := q.Queue.Send(ctx, goqite.Message{
		Body: data,
	})
	if err != nil {
		return err
	}

	return nil
}

// Delete the message from the queue, so it doesn't get redelivered.
func (q *Queue) Delete(
	ctx context.Context,
	msgID goqite.ID,
) error {
	return q.Queue.Delete(ctx, msgID)
}
