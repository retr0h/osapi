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

// serverCmd represents the server command.
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "The server subcommand",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		validateDistribution()

		logger.Info(
			"server configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.Int("server.port", appConfig.Server.Port),
			slog.Any(
				"server.security.cors.allow_origins",
				appConfig.Server.Security.CORS.AllowOrigins,
			),
			slog.String("database.driver_name", appConfig.Database.DriverName),
			slog.String("database.data_source_name", appConfig.Database.DataSourceName),
			slog.Int("database.max_open_conns", appConfig.Database.MaxOpenConns),
			slog.Int("database.max_idle_conns", appConfig.Database.MaxIdleConns),
		)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	registerDatabaseFlags(serverCmd)

	serverCmd.PersistentFlags().
		IntP("port", "p", 8080, "Port the server will bind to")

	_ = viper.BindPFlag("server.port", serverCmd.PersistentFlags().Lookup("port"))
}
