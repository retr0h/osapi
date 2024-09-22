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
// Provider implements the methods to interact with various system-level
// components.

package queue

import (
	"context"

	"github.com/maragudk/goqite"
)

// Manager responsible for Queue operations.
type Manager interface {
	// SetupSchema setup the schema for use with the queue.
	SetupSchema(ctx context.Context) error
	// SetupQueue sets up the queue with the specified name.
	SetupQueue()
	// SetQueue allows setting a custom Queue.
	SetQueue(mp MessageProcessor)

	// Get a message from the queue, during which time it's not available to
	Get(
		ctx context.Context,
	) (*goqite.Message, error)
	// GetAll the message in the queue.
	GetAll(ctx context.Context, limit int, offset int) ([]Item, error)
	// GetByID fetches a single item from the database by its ID.
	GetByID(ctx context.Context, messageID string) (*Item, error)
	// Put the message into the queue.
	Put(
		ctx context.Context,
		data []byte,
	) error
	// Delete the message from the queue, so it doesn't get redelivered.
	DeleteByID(
		ctx context.Context,
		messageID string,
	) error
	// Count counts the number of rows in the table.
	Count(ctx context.Context) (int, error)
}
