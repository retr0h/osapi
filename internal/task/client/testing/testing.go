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
	"errors"
	"log/slog"
	"os"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	natsclient "github.com/osapi-io/nats-client/pkg/client"
	"github.com/osapi-io/nats-server/pkg/server"

	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/task"
	"github.com/retr0h/osapi/internal/task/client"
)

var (
	appConfig = config.Config{
		Task: config.Task{
			Server: config.TaskServer{
				Host: "0.0.0.0",
				Port: 4222,
			},
		},
	}
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
)

// NewServer initializes and returns a new Server instance.
func NewServer() *server.Server {
	opts := &server.Options{
		Options: &natsserver.Options{
			Host:      appConfig.Task.Server.Host,
			Port:      appConfig.Task.Server.Port,
			JetStream: true,
			Debug:     true,
			Trace:     true,
			StoreDir:  "",
			NoSigs:    true,
			NoLog:     false,
		},
		ReadyTimeout: 5 * time.Second,
	}

	return server.New(logger, opts)
}

// SetupStream configures a NATS JetStream stream and consumers for testing.
func SetupStream() error {
	jsOpts := &natsclient.ClientOptions{
		Host: appConfig.Task.Server.Host,
		Port: appConfig.Task.Server.Port,
		Auth: natsclient.AuthOptions{
			AuthType: natsclient.NoAuth,
		},
	}

	js, err := natsclient.NewJetStreamContext(jsOpts)
	if err != nil {
		return err
	}

	streamOpts := &natsclient.StreamConfig{
		StreamConfig: &nats.StreamConfig{
			Name:     task.StreamName,
			Subjects: []string{task.SubjectName},
			Storage:  nats.MemoryStorage,
		},
		Consumers: []*natsclient.ConsumerConfig{
			{
				ConsumerConfig: &nats.ConsumerConfig{
					Durable:    task.ConsumerName,
					AckPolicy:  nats.AckExplicitPolicy,
					MaxDeliver: 5,
					AckWait:    30 * time.Second,
				},
			},
		},
	}

	c := natsclient.New(logger)
	if err := c.SetupJetStream(js, streamOpts); err != nil &&
		!errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		return err
	}

	return nil
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
