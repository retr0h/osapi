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

// taskServerCmd represents the taskServer command.
var taskServerCmd = &cobra.Command{
	Use:   "server",
	Short: "The server subcommand",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		validateDistribution()

		logger.Info(
			"task server configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.String("server.host", appConfig.Task.Server.Host),
			slog.Int("server.port", appConfig.Task.Server.Port),
			slog.Bool("server.trace", appConfig.Task.Server.Trace),
			slog.Bool("server.no_log", appConfig.Task.Server.NoLog),
			slog.String("server.file_store_dir", appConfig.Task.Server.FileStoreDir),
		)
	},
}

func init() {
	taskCmd.AddCommand(taskServerCmd)

	taskServerCmd.PersistentFlags().
		IntP("port", "", 4222, "Port the server will bind to")
	taskServerCmd.PersistentFlags().
		StringP("host", "", "localhost", "Host bind the server to")
	taskServerCmd.PersistentFlags().
		BoolP("trace", "", false, "Trace enable detailed tracing for debugging")
	taskServerCmd.PersistentFlags().
		BoolP("no-log", "", false, "NoLog enable logging for server events.")
	taskServerCmd.PersistentFlags().
		StringP("file-store-dir", "", "/var/lib/nats/jetstream", "JetStream data will be persisted here")

	_ = viper.BindPFlag("task.server.port", taskServerCmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("task.server.host", taskServerCmd.PersistentFlags().Lookup("host"))
	_ = viper.BindPFlag("task.server.trace", taskServerCmd.PersistentFlags().Lookup("trace"))
	_ = viper.BindPFlag("task.server.no_log", taskServerCmd.PersistentFlags().Lookup("no-log"))
	_ = viper.BindPFlag(
		"task.server.file_store_dir",
		taskServerCmd.PersistentFlags().Lookup("file-store-dir"),
	)
}
