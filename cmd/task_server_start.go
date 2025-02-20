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

package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/osapi-io/nats-client/pkg/client"
	"github.com/osapi-io/nats-server/pkg/server"
	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/task"
)

// ServerManager responsible for Server operations.
type ServerManager interface {
	// Start starts the server.
	Start() error
	// Stop gracefully stops the embedded NATS server.
	Stop()
}

// ClientManager responsible for Client operations.
type ClientManager interface {
	// SetupJetStream configures JetStream streams and consumers.
	SetupJetStream(js nats.JetStreamContext, streamConfigs ...*client.StreamConfig) error
}

// taskServerStartCmd represents the taskServerStart command.
var taskServerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server",
	Long: `Start the embedded NATS server for task management.
The NATS server is embedded, which means it is run as part of the application
itself, removing the need for a separately deployed server. This setup ensures
a seamless integration for managing real-time communication and message distribution.
`,
	Run: func(_ *cobra.Command, _ []string) {
		opts := &server.Options{
			Options: &natsserver.Options{
				Host:      appConfig.Task.Server.Host,
				Port:      appConfig.Task.Server.Port,
				JetStream: true,
				Debug:     appConfig.Debug,
				Trace:     appConfig.Task.Server.Trace,
				StoreDir:  appConfig.Task.Server.FileStoreDir,
				NoSigs:    true,
				NoLog:     appConfig.Task.Server.NoLog,
			},
			ReadyTimeout: 5 * time.Second,
		}

		var sm ServerManager = server.New(logger, opts)
		err := sm.Start()
		if err != nil {
			logger.Error("failed to start server", "error", err)
			os.Exit(1)
		}

		jsOpts := &client.ClientOptions{
			Host: appConfig.Task.Server.Host,
			Port: appConfig.Task.Server.Port,
			Auth: client.AuthOptions{
				AuthType: client.NoAuth,
			},
		}

		js, err := client.NewJetStreamContext(jsOpts)
		if err != nil {
			logger.Error("failed to create jetstream context", "error", err)
			os.Exit(1)
		}

		streamOpts := &client.StreamConfig{
			StreamConfig: &nats.StreamConfig{
				Name:     task.StreamName,
				Subjects: []string{task.SubjectName},
				Storage:  nats.FileStorage,
				Replicas: 1,
			},
			Consumers: []*client.ConsumerConfig{
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

		var cm ClientManager = client.New(logger)
		if err := cm.SetupJetStream(js, streamOpts); err != nil {
			logger.Error("failed setting up jetstream", "error", err)
			os.Exit(1)
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		sm.Stop()
	},
}

func init() {
	taskServerCmd.AddCommand(taskServerStartCmd)
}
