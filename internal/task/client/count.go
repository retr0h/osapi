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
)

// CountStreamMessages returns the total number of messages in the specified JetStream stream (queue).
func (c *Client) CountStreamMessages(
	ctx context.Context,
) (int, error) {
	stream, err := c.js.Stream(ctx, StreamName)
	if err != nil {
		return 0, fmt.Errorf("error retrieving stream: %w", err)
	}

	// cons, _ := stream.Consumer(ctx, "foo")

	// info, _ := cons.Info(ctx)
	// fmt.Println(info)

	streamInfo, err := stream.Info(ctx)
	if err != nil {
		return 0, fmt.Errorf("error retrieving stream info: %w", err)
	}

	return int(streamInfo.State.Msgs), nil
}
