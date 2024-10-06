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

package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/retr0h/osapi/internal/config"
	"github.com/retr0h/osapi/internal/task"
)

// New initialize and configure a new Server instance.
func New(
	appConfig config.Config,
	logger *slog.Logger,
) *Server {
	return &Server{
		logger:    logger,
		appConfig: appConfig,
	}
}

// Start start the embedded NATS server.
func (s *Server) Start() error {
	opts := &server.Options{
		JetStream: true,
		Trace:     s.appConfig.Task.Server.Trace,
		Debug:     s.appConfig.Debug,
		Host:      s.appConfig.Task.Server.Host,
		Port:      s.appConfig.Task.Server.Port,
		NoLog:     s.appConfig.Task.Server.NoLog,
		NoSigs:    true,
	}

	if s.appConfig.Task.Server.FileStoreDir != "" {
		opts.StoreDir = s.appConfig.Task.Server.FileStoreDir
	}

	natsServer, err := server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("error starting NATS server: %w", err)
	}

	go natsServer.Start()

	// Wait for server readiness
	if !natsServer.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("NATS server not ready for connections")
	}

	slogWrapper := &SlogWrapper{
		logger: s.logger,
	}

	natsServer.SetLogger(slogWrapper, true, true)

	s.logger.Info("nats server started successfully")

	if err := s.setupJetStream(); err != nil {
		return fmt.Errorf("error setting up JetStream: %w", err)
	}

	s.logger.Info("jet stream setup completed successfully")

	s.natsServer = natsServer // Store server instance

	return nil
}

// setupJetStream creates the JetStream connection and stream configuration.
func (s *Server) setupJetStream() error {
	nc, err := nats.Connect(fmt.Sprintf("nats://localhost:%d", s.appConfig.Task.Server.Port))
	if err != nil {
		return fmt.Errorf("error connecting to NATS: %w", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("error enabling JetStream: %w", err)
	}

	storage := nats.MemoryStorage
	if s.appConfig.Task.Server.FileStoreDir != "" {
		storage = nats.FileStorage
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     task.StreamName,
		Subjects: []string{task.SubjectName},
		Storage:  storage,
	})
	if err != nil {
		return fmt.Errorf("error creating stream: %w", err)
	}

	_, err = js.AddConsumer(task.StreamName, &nats.ConsumerConfig{
		Durable:    task.ConsumerName,
		AckPolicy:  nats.AckExplicitPolicy,
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	})
	if err != nil {
		return fmt.Errorf("error creating consumer: %w", err)
	}

	return nil
}

// Stop gracefully stops the embedded NATS server.
func (s *Server) Stop() {
	if s.natsServer != nil {
		s.logger.Info("shutting down nats server")
		s.natsServer.Shutdown()
		s.logger.Info("nats server shut down successfully")
	}
}
