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

	durableConsumer, err := stream.Consumer(ctx, task.ConsumerName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving consumer: %w", err)
	}

	consumerInfo, err := durableConsumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving consumer info: %w", err)
	}

	lastAckedSeq := int(consumerInfo.Delivered.Stream) - consumerInfo.NumAckPending

	ephemeralConsumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		OptStartSeq:   uint64(lastAckedSeq + 1),
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy, // Use correct policy
	})
	if err != nil {
		return nil, fmt.Errorf("error creating ephemeral consumer: %w", err)
	}

	var messageItems []MessageItem

	numAckPending := consumerInfo.NumAckPending
	numPending := consumerInfo.NumPending

	pendingMessages := numAckPending + int(numPending)

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
