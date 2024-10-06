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

	"github.com/retr0h/osapi/internal/task"
)

// CountStreamMessages returns the total number of messages in the specified JetStream stream (queue).
func (c *Client) CountStreamMessages(
	ctx context.Context,
) (int, error) {
	stream, err := c.JS.Stream(ctx, task.StreamName)
	if err != nil {
		return 0, fmt.Errorf("error retrieving stream: %w", err)
	}

	cons, err := stream.Consumer(ctx, task.ConsumerName)
	if err != nil {
		return 0, fmt.Errorf("error retrieving consumer: %w", err)
	}

	consumerInfo, err := cons.Info(ctx)
	if err != nil {
		return 0, fmt.Errorf("error retrieving consumer info: %w", err)
	}

	// NumAckPending is the number of messages that have been delivered but
	// not yet acknowledged.
	numAckPending := consumerInfo.NumAckPending

	// NumPending is the number of messages that match the consumer's
	// filter, but have not been delivered yet.
	numPending := consumerInfo.NumPending

	return numAckPending + int(numPending), nil
}
