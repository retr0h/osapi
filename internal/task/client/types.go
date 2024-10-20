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
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/retr0h/osapi/internal/config"
)

// Client implementation of the Client's task operations.
type Client struct {
	logger      *slog.Logger
	appConfig   config.Config
	JS          jetstream.JetStream
	ConnectFn   func(url string, options ...nats.Option) (*nats.Conn, error)
	JetStreamFn func(*nats.Conn) (jetstream.JetStream, error)
}

// MessageItem represents a JetStream message with metadata and content.
type MessageItem struct {
	// Stream sequence number.
	StreamSeq uint64 `json:"stream_sequence"`
	// Timestamp when the message was stored in the stream.
	StoredAt time.Time `json:"stored_at"`
	// Data the message (raw data).
	Data []byte `json:"body"`
}
