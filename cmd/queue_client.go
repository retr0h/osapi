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

// queueClientCmd represents the queue command.
var queueClientCmd = &cobra.Command{
	Use:   "client",
	Short: "The client subcommand",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		logger.Info(
			"queue configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.String("queue.database.driver_name", appConfig.Database.DriverName),
			slog.String("queue.database.data_source_name", appConfig.Database.DataSourceName),
			slog.Int("queue.database.max_open_conns", appConfig.Database.MaxOpenConns),
			slog.Int("queue.database.max_idle_conns", appConfig.Database.MaxIdleConns),
		)
	},
}

func init() {
	queueCmd.AddCommand(queueClientCmd)

	queueClientCmd.PersistentFlags().
		StringP("driver-name", "t", "sqlite", "Name of the database driver to use")
	queueClientCmd.PersistentFlags().
		StringP("dsn", "s", ":memory:?_journal=WAL&_timeout=5000&_fk=true", "The data source name (DSN) for the database connection")
	queueClientCmd.PersistentFlags().
		IntP("max-open-conns", "o", 1, "The maximum number of open connections to the database")
	queueClientCmd.PersistentFlags().
		IntP("max-idle-conns", "i", 1, "The maximum number of idle connections in the pool")

	queueClientCmd.PersistentFlags().
		StringP("url", "u", "http://0.0.0.0:8080", "URL the client will connect to")
	queueClientCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Enable JSON output")

	_ = viper.BindPFlag(
		"queue.database.driver_name",
		queueClientCmd.PersistentFlags().Lookup("driver-name"),
	)
	_ = viper.BindPFlag(
		"queue.database.data_source_name",
		queueClientCmd.PersistentFlags().Lookup("dsn"),
	)
	_ = viper.BindPFlag(
		"queue.database.max_open_conns",
		queueClientCmd.PersistentFlags().Lookup("max-open-conns"),
	)
	_ = viper.BindPFlag(
		"queue.database.max_idle_conns",
		queueClientCmd.PersistentFlags().Lookup("max-idle-conns"),
	)
	_ = viper.BindPFlag("client.url", clientCmd.PersistentFlags().Lookup("url"))
}
