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

package client

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"
)

// Manager responsible for Client operations.
type Manager interface {
	// Connect establishes the connection to the NATS server and JetStream context.
	// This method returns an error if there are any issues during connection.
	Connect() error
	//// Connect(...jetstream.JetStreamOpt) error
	// CountStreamMessages returns the total number of messages in the specified JetStream stream (queue).
	CountStreamMessages(
		ctx context.Context,
	) (int, error)
	// DeleteMessageBySeq deletes a single message from the specified JetStream
	// stream by its sequence number.
	DeleteMessageBySeq(
		ctx context.Context,
		seq uint64,
	) error
	// GetMessageBySeq retrieves a single message from the specified JetStream
	// stream by its sequence number.
	GetMessageBySeq(
		ctx context.Context,
		seq uint64,
	) (*MessageItem, error)
	// ListUndeliveredMessages retrieves a list of undelivered messages from the
	// JetStream consumer.
	ListUndeliveredMessages(
		ctx context.Context,
	) ([]MessageItem, error)
	// PublishToStream publishes a message to the specified JetStream stream using the existing JetStream context.
	PublishToStream(
		ctx context.Context,
		data []byte,
	) (uint64, error)
	// GetMessageIterator retrieves the message iterator using a JetStream consumer.
	GetMessageIterator(
		ctx context.Context,
	) (jetstream.MessagesContext, error)
}
