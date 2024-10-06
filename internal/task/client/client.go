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
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/retr0h/osapi/internal/config"
)

// New initialize and configure a new Client instance.
func New(
	appConfig config.Config,
	logger *slog.Logger,
) *Client {
	return &Client{
		logger:    logger,
		appConfig: appConfig,
	}
}

// Connect establishes the connection to the NATS server and JetStream context.
// This method returns an error if there are any issues during connection.
func (c *Client) Connect() error {
	nc, err := nats.Connect(fmt.Sprintf("nats://localhost:%d", c.appConfig.Task.Server.Port))
	if err != nil {
		return fmt.Errorf("error connecting to NATS: %w", err)
	}
	c.nc = nc

	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("error creating JetStream context: %w", err)
	}
	c.JS = js

	return nil
}
