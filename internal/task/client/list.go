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

	"github.com/retr0h/osapi/internal/task"
)

// ListUndeliveredMessages retrieves a list of undelivered messages from the
// JetStream consumer.
//
// The ephemeral consumer is created specifically to "peek" into the stream and fetch undelivered
// messages without acknowledging them, thus allowing other consumers to still process the
// messages normally. This method is useful for scenarios where you want visibility into pending
// messages but do not want to consume them in the traditional sense.
//
// NOTE(retr0h): This function does not implement pagination as the queue is
// not expected to grow large enough to warrant it. In normal operation, if
// the queue does become large, it indicates an operational issue that requires
// further investigation.
func (c *Client) ListUndeliveredMessages(
	ctx context.Context,
) ([]MessageItem, error) {
	stream, err := c.JS.Stream(ctx, task.StreamName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving stream: %w", err)
	}

	// Retrieve durable consumer info
	durableConsumer, err := stream.Consumer(ctx, task.ConsumerName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving consumer: %w", err)
	}

	consumerInfo, err := durableConsumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving consumer info: %w", err)
	}

	lastAckedSeq := int(consumerInfo.Delivered.Stream) - consumerInfo.NumAckPending

	// Create an ephemeral consumer starting from the last acknowledged sequence
	ephemeralConsumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		OptStartSeq:   uint64(lastAckedSeq + 1),
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy, // Use correct policy

	})
	if err != nil {
		return nil, fmt.Errorf("error creating ephemeral consumer: %w", err)
	}

	// Fetch messages from the ephemeral consumer
	var messageItems []MessageItem
	// pendingMessages := consumerInfo.NumPending
	pendingMessages := 5
	for i := 0; i < int(pendingMessages); i++ {
		fetchResult, err := ephemeralConsumer.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
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
		}
	}

	return messageItems, nil
}

// stream, err := c.JS.Stream(ctx, task.StreamName)
// if err != nil {
// 	return nil, fmt.Errorf("error retrieving stream: %w", err)
// }

// // 	// Create an ephemeral consumer tied to a consumer group (queue group)
// // 	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
// // 		AckPolicy:     jetstream.AckExplicitPolicy,
// // 		FilterSubject: "tasks", // Filter messages for the "tasks" subject
// // 	})
// // 	if err != nil {
// // 		return nil, fmt.Errorf("error creating ephemeral consumer: %w", err)
// // 	}

// cons, err := stream.Consumer(ctx, task.ConsumerName)
// if err != nil {
// 	return nil, fmt.Errorf("error retrieving consumer: %w", err)
// }

// // consumerInfo, err := cons.Info(ctx)
// // if err != nil {
// // 	return nil, fmt.Errorf("error retrieving consumer info: %w", err)
// // }

// // Fetch undelivered messages
// consumerInfo, err := cons.Info(ctx)
// if err != nil {
// 	return nil, fmt.Errorf("error retrieving consumer info: %w", err)
// }

// var messageItems []MessageItem

// pendingMessages := consumerInfo.NumPending
// for i := 0; i < int(pendingMessages); i++ {
// 	fetchResult, err := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
// 	if err != nil {
// 		return nil, fmt.Errorf("error fetching message: %w", err)
// 	}

// 	for msg := range fetchResult.Messages() {
// 		meta, err := msg.Metadata()
// 		if err != nil {
// 			return nil, fmt.Errorf("error retrieving metadata: %w", err)
// 		}

// 		item := MessageItem{
// 			StreamSeq: meta.Sequence.Stream,
// 			StoredAt:  meta.Timestamp,
// 			Data:      msg.Data(),
// 		}

// 		messageItems = append(messageItems, item)
// 	}
// }

// return messageItems, nil
// }

// // // Retrieve the consumer.
// // stream, err := c.js.Stream(ctx, task.StreamName)
// // if err != nil {
// // 	return nil, fmt.Errorf("error retrieving stream: %w", err)
// // }

// // cons, err := stream.Consumer(ctx, task.ConsumerName)
// // if err != nil {
// // 	return nil, fmt.Errorf("error retrieving consumer: %w", err)
// // }

// // // Get consumer info which includes the pending message count.
// // consumerInfo, err := cons.Info(ctx)
// // if err != nil {
// // 	return nil, fmt.Errorf("error retrieving consumer info: %w", err)
// // }

// // var messageItems []MessageItem

// // // Fetch pending messages count.
// // pendingMessages := consumerInfo.NumPending
// // for i := 0; i < int(pendingMessages); i++ {
// // 	// Fetch each undelivered message without acknowledging it.
// // 	fetchResult, err := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
// // 	if err != nil {
// // 		return nil, fmt.Errorf("error fetching message: %w", err)
// // 	}

// // 	for msg := range fetchResult.Messages() {
// // 		meta, err := msg.Metadata()
// // 		if err != nil {
// // 			return nil, fmt.Errorf("error retrieving metadata: %w", err)
// // 		}

// // 		// Add message details to the list.
// // 		messageItems = append(messageItems, MessageItem{
// // 			StreamSeq: meta.Sequence.Stream,
// // 			StoredAt:  meta.Timestamp,
// // 			Data:      msg.Data(),
// // 		})
// // 	}
// // }

// // return messageItems, nil
// // }

// // // Get the total number of messages using the existing CountStreamMessages function
// // totalMessages, err := c.CountStreamMessages(ctx)
// // if err != nil {
// // 	return nil, fmt.Errorf("error counting stream messages: %w", err)
// // }

// // // Validate offset against totalMessages
// // if offset >= totalMessages {
// // 	return nil, fmt.Errorf("offset exceeds total number of messages")
// // }

// // // Adjust limit if it exceeds the available messages
// // if limit+offset > totalMessages {
// // 	limit = totalMessages - offset
// // }

// // stream, err := c.js.Stream(ctx, task.StreamName)
// // if err != nil {
// // 	return nil, fmt.Errorf("error retrieving stream: %w", err)
// // }

// // cons, err := stream.Consumer(ctx, task.ConsumerName)
// // if err != nil {
// // 	return nil, fmt.Errorf("error retrieving consumer: %w", err)
// // }

// // var messageItems []MessageItem

// // // Skip messages to implement "offset"
// // skipped := 0
// // for skipped < offset {
// // 	fetchResult, err := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
// // 	if err != nil {
// // 		return nil, fmt.Errorf("error fetching message: %w", err)
// // 	}

// // 	// Just consume and skip the message
// // 	for range fetchResult.Messages() {
// // 		skipped++
// // 	}
// // }

// // // Fetch messages to implement "limit"
// // fetched := 0
// // for fetched < limit {
// // 	fetchResult, err := cons.Fetch(1, jetstream.FetchMaxWait(100*time.Millisecond))
// // 	if err != nil {
// // 		return nil, fmt.Errorf("error fetching message: %w", err)
// // 	}

// // 	for msg := range fetchResult.Messages() {
// // 		meta, err := msg.Metadata()
// // 		if err != nil {
// // 			return nil, fmt.Errorf("error retrieving metadata: %w", err)
// // 		}

// // 		item := MessageItem{
// // 			StreamSeq: meta.Sequence.Stream,
// // 			StoredAt:  meta.Timestamp,
// // 			Data:      msg.Data(),
// // 		}

// // 		messageItems = append(messageItems, item)
// // 		fetched++
// // 	}

// // 	if fetched >= limit {
// // 		break
// // 	}
// // }

// // return messageItems, nil
// // }
