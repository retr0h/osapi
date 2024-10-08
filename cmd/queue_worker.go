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
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// queueWorkerCmd represents the queueWorker command.
var queueWorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "The worker subcommand",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		logger.Info(
			"queue configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.String("database.driver_name", appConfig.Database.DriverName),
			slog.String("database.data_source_name", appConfig.Database.DataSourceName),
			slog.Int("database.max_open_conns", appConfig.Database.MaxOpenConns),
			slog.Int("database.max_idle_conns", appConfig.Database.MaxIdleConns),
			slog.Int("queue.poll_interval.seconds", appConfig.Queue.PollInterval.Seconds),
		)
	},
}

func init() {
	queueCmd.AddCommand(queueWorkerCmd)
	registerDatabaseFlags(queueWorkerCmd)

	queueWorkerCmd.PersistentFlags().
		IntP("poll-interval-seconds", "p", 60, "The interval (in seconds) between polling operations")

	_ = viper.BindPFlag("server.port", queueWorkerCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag(
		"queue.poll_interval.seconds",
		queueWorkerCmd.PersistentFlags().Lookup("poll-interval-seconds"),
	)
}
