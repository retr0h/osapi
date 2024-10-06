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
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	customerrors "github.com/retr0h/osapi/internal/errors"
	"github.com/retr0h/osapi/internal/task"
)

// GetMessageBySeq retrieves a single message from the specified JetStream
// stream by its sequence number.
func (c *Client) GetMessageBySeq(
	ctx context.Context,
	seq uint64,
) (*MessageItem, error) {
	stream, err := c.JS.Stream(ctx, task.StreamName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving stream: %w", err)
	}

	msg, err := stream.GetMsg(ctx, seq)
	if err != nil {
		var apiErr *jetstream.APIError
		if errors.As(err, &apiErr) {
			if apiErr.Code == 404 {
				return nil, customerrors.NewNotFoundError(fmt.Sprintf("no item found with ID %d", seq))
			}
		}
		return nil, fmt.Errorf("error retrieving message: %w", err)
	}

	item := &MessageItem{
		StreamSeq: msg.Sequence,
		StoredAt:  msg.Time,
		Data:      msg.Data,
	}

	return item, nil
}
