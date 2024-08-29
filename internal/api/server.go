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

package api

import (
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"github.com/spf13/afero"

	"github.com/retr0h/osapi/internal/api/ping"             // testing only
	pingGen "github.com/retr0h/osapi/internal/api/ping/gen" // testing only
	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/config"
)

// // registerHandlers initializes and registers all API handlers.
// func registerHandlers(
// 	e *echo.Echo,
// ) {
// 	handlers := []func(e *echo.Echo){
// 		func(e *echo.Echo) {
// 			pingHandler := ping.New()
// 			pingGen.RegisterHandlers(e, pingHandler)
// 		},
// 		func(e *echo.Echo) {
// 			systemHandler := system.New()
// 			systemGen.RegisterHandlers(e, systemHandler)
// 		},
// 		// Add more handler functions as needed
// 	}

// 	for _, register := range handlers {
// 		register(e)
// 	}
// }

// New initializes the Echo server.
func New(
	appFs afero.Fs,
	appConfig config.Config,
	logger *slog.Logger,
) *echo.Echo {
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

	// TODO(retr0h): Determine what to do about this
	// registerHandlers(e)

	pingHandler := ping.New()
	pingGen.RegisterHandlers(e, pingHandler)

	systemHandler := system.New(appFs)
	systemGen.RegisterHandlers(e, systemHandler)

	return e
}
