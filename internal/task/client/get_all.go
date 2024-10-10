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
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// GetAllPaginatedMessages retrieves messages from the specified JetStream
// stream in a paginated manner. This function allows you to skip a given
// number of messages (offset) and fetch a limited number of messages (limit)
// at a time. The messages are not removed from the stream and can be accessed
// again.
//
// NOTE(retr0h): JetStream does not have a native method for pagination, so
// this approach is slightly hacky. It simulates pagination by manually skipping
// messages and then fetching the desired number.
func (c *Client) GetAllPaginatedMessages(
	ctx context.Context,
	streamName string,
	limit int,
	offset int,
) ([]MessageItem, error) {
	// Get the total number of messages using the existing CountStreamMessages function
	totalMessages, err := c.CountStreamMessages(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("error counting stream messages: %w", err)
	}

	// Validate offset against totalMessages
	if offset >= totalMessages {
		return nil, fmt.Errorf("offset exceeds total number of messages")
	}

	// Adjust limit if it exceeds the available messages
	if limit+offset > totalMessages {
		limit = totalMessages - offset
	}

	stream, err := c.js.Stream(ctx, streamName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving stream: %w", err)
	}

	cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy: jetstream.AckExplicitPolicy,
		Durable:   "foo",
	})

	// cons, _ := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
	// 	InactiveThreshold: 10 * time.Millisecond,
	// })

	var messageItems []MessageItem

	// Skip messages to implement "offset"
	skipped := 0
	for skipped < offset {
		fetchResult, err := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
		if err != nil {
			return nil, fmt.Errorf("error fetching message: %w", err)
		}

		// Just consume and skip the message
		for range fetchResult.Messages() {
			skipped++
		}
	}

	// Fetch messages to implement "limit"
	fetched := 0
	for fetched < limit {
		fetchResult, err := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
		if err != nil {
			return nil, fmt.Errorf("error fetching message: %w", err)
		}

		for msg := range fetchResult.Messages() {
			meta, err := msg.Metadata()
			if err != nil {
				return nil, fmt.Errorf("error retrieving metadata: %w", err)
			}

			item := MessageItem{
				StreamSeq: meta.Sequence.Stream,
				StoredAt:  meta.Timestamp,
				Data:      msg.Data(),
			}

			messageItems = append(messageItems, item)
			fetched++
		}

		if fetched >= limit {
			break
		}
	}

	return messageItems, nil
}
