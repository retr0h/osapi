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
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/system"
)

// serverStartCmd represents the serve command.
var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the OSAPI service",
	Long: `Start the  OSAPI service.
`,
	Run: func(_ *cobra.Command, _ []string) {
		logger.Info(
			"server configuration",
			slog.Bool("debug", appConfig.Debug),
			slog.Int("server.port", appConfig.Server.Port),
		)

		// create a type that satisfies the `api.ServerInterface`, which
		// contains an implementation of every operation from the generated code.
		pongHandler := api.New()
		systemHandler := system.New()

		e := echo.New()
		e.HideBanner = true

		e.Use(slogecho.New(logger))
		e.Use(middleware.Recover())
		e.Use(middleware.RequestID())

		api.RegisterHandlers(e, pongHandler)
		system.RegisterHandlers(e, systemHandler)
		e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", appConfig.Server.Port)))
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
}
