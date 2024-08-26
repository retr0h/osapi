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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"github.com/spf13/cobra"

	"github.com/retr0h/osapi/internal/api/ping"             // testing only
	pingGen "github.com/retr0h/osapi/internal/api/ping/gen" // testing only
	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
)

// registerHandlers initializes and registers all API handlers.
func registerHandlers(
	e *echo.Echo,
) {
	handlers := []func(e *echo.Echo){
		func(e *echo.Echo) {
			pingHandler := ping.New()
			pingGen.RegisterHandlers(e, pingHandler)
		},
		func(e *echo.Echo) {
			systemHandler := system.New()
			systemGen.RegisterHandlers(e, systemHandler)
		},
		// Add more handler functions as needed
	}

	for _, register := range handlers {
		register(e)
	}
}

// serverStartCmd represents the serve command.
var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the OSAPI service",
	Long: `Start the  OSAPI service.
`,
	Run: func(_ *cobra.Command, _ []string) {
		e := echo.New()
		e.HideBanner = true

		// Initialize CORS configuration
		corsConfig := middleware.CORSConfig{}

		allowOrigins := appConfig.Server.Security.CORS.AllowOrigins
		if len(allowOrigins) > 0 {
			corsConfig.AllowOrigins = allowOrigins
		}

		e.Use(slogecho.New(logger))
		e.Use(middleware.Recover())
		e.Use(middleware.RequestID())
		e.Use(middleware.CORSWithConfig(corsConfig))

		registerHandlers(e)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		// Start server
		go func() {
			// e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", appConfig.Server.Port)))
			if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
				e.Logger.Fatal("shutting down the server")
			}
		}()

		// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Shutdown(ctx); err != nil {
			e.Logger.Fatal(err)
		}

		fmt.Println("Interrupt signal received. Exiting...")
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
}
