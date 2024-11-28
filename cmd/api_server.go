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
	"strings"

	"github.com/ggwhite/go-masker/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/retr0h/osapi/internal/config"
)

// apiServerCmd represents the apiServer command.
var apiServerCmd = &cobra.Command{
	Use:   "server",
	Short: "The server subcommand",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		validateDistribution()

		masker := masker.NewMaskerMarshaler()
		maskedConfig, err := masker.Struct(&appConfig)
		if err != nil {
			logFatal("failed to mask config", err)
		}

		maskedAppConfig, ok := maskedConfig.(*config.Config)
		if !ok {
			logFatal("failed to type assert maskedConfig", nil)
		}

		logger.Info(
			"api server configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.Int("api.server.port", appConfig.API.Server.Port),
			slog.String(
				"api.server.security.cors.allow_origins",
				strings.Join(appConfig.API.Server.Security.CORS.AllowOrigins, ","),
			),
			slog.String(
				"api.server.security.signing_key",
				maskedAppConfig.API.Server.Security.SigningKey,
			),
		)
	},
}

func init() {
	apiCmd.AddCommand(apiServerCmd)

	apiServerCmd.PersistentFlags().
		IntP("port", "p", 8080, "Port the server will bind to")

	_ = viper.BindPFlag("api.server.port", apiServerCmd.PersistentFlags().Lookup("port"))
}
