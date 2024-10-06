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

	customerrors "github.com/retr0h/osapi/internal/errors"
	"github.com/retr0h/osapi/internal/task"
)

// DeleteMessageBySeq deletes a single message from the specified JetStream
// stream by its sequence number.
func (c *Client) DeleteMessageBySeq(
	ctx context.Context,
	seq uint64,
) error {
	_, err := c.GetMessageBySeq(ctx, seq)
	if err != nil {
		var notFoundErr *customerrors.NotFoundError
		if errors.As(err, &notFoundErr) {
			// Return custom MessageNotFoundError if message does not exist
			return customerrors.NewNotFoundError(fmt.Sprintf("no item found with ID %d", seq))
		}
		return fmt.Errorf("failed to retrieve message: %w", err)
	}

	stream, err := c.JS.Stream(ctx, task.StreamName)
	if err != nil {
		return fmt.Errorf("error retrieving stream: %w", err)
	}

	return stream.DeleteMsg(ctx, seq)
}
