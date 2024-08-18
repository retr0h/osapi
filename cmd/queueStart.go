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
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/maragudk/goqite"
	_ "github.com/mattn/go-sqlite3" //lint:ignore
	"github.com/spf13/cobra"
)

// queueStartCmd represents the queueStart command.
var queueStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the task runner service",
	Long:  `Start the  task runner service.`,
	Run: func(_ *cobra.Command, _ []string) {
		logger.Info(
			"queue configuration",
			slog.Bool("debug", appConfig.Debug),
		)

		db, err := sql.Open("sqlite3", ":memory:?_journal=WAL&_timeout=5000&_fk=true")
		if err != nil {
			log.Fatalln(err)
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if err := goqite.Setup(context.Background(), db); err != nil {
			log.Fatalln(err)
		}

		// Create a new queue named "jobs".
		// You can also customize the message redelivery timeout and maximum receive count,
		// but here, we use the defaults.
		q := goqite.New(goqite.NewOpts{
			DB:   db,
			Name: "jobs",
		})

		// Send a message to the queue.
		// Note that the body is an arbitrary byte slice, so you can decide
		// what kind of payload you have. You can also set a message delay.
		err = q.Send(context.Background(), goqite.Message{
			Body: []byte("yo"),
		})
		if err != nil {
			log.Fatalln(err)
		}

		// Receive a message from the queue, during which time it's not available to
		// other consumers (until the message timeout has passed).
		m, err := q.Receive(context.Background())
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println(string(m.Body))

		// If you need more time for processing the message, you can extend
		// the message timeout as many times as you want.
		if err := q.Extend(context.Background(), m.ID, time.Second); err != nil {
			log.Fatalln(err)
		}

		// Make sure to delete the message, so it doesn't get redelivered.
		if err := q.Delete(context.Background(), m.ID); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	queueCmd.AddCommand(queueStartCmd)
}
