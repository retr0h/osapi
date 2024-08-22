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

	"github.com/retr0h/osapi/internal/client"
)

var c *client.Client

// clientCmd represents the client command.
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "The client subcommand",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		logger.Info(
			"client configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.String("client.url", appConfig.Client.URL),
		)

		cwr, err := client.NewClientWithResponses(appConfig)
		if err != nil {
			logFatal("failed to create http client", err)
		}

		c = client.New(
			logger,
			appConfig,
			cwr,
		)
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.PersistentFlags().
		StringP("url", "l", "http://0.0.0.0:8080", "URL the client will connect to")

	_ = viper.BindPFlag("client.url", clientCmd.PersistentFlags().Lookup("url"))
}
