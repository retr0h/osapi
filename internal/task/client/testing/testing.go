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

package testing

import (
	"log/slog"
	"os"

	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/task/client"
	"github.com/retr0h/osapi/internal/task/server"
)

var (
	appConfig = config.Config{
		Task: config.Task{
			Server: config.TaskServer{
				Port: 4222,
			},
		},
	}
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
)

// NewServer initializes and returns a new Server instance.
func NewServer() *server.Server {
	return server.New(appConfig, logger)
}

// NewClient initializes and returns a new Client instance.
func NewClient() (*client.Client, error) {
	c := client.New(appConfig, logger)
	err := c.Connect()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// NewClientWithoutConnect initializes and returns a new Client instance without calling Connect.
func NewClientWithoutConnect() *client.Client {
	return client.New(appConfig, logger)
}
