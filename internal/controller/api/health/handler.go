// Copyright (c) 2026 John Dewey

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

package health

import (
	"context"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/controller/api"
	gen "github.com/osapi-io/osapi/internal/controller/api/health/gen"
)

// unauthenticatedOperations lists operation IDs that skip auth.
var unauthenticatedOperations = map[string]bool{
	"GetHealth":      true,
	"GetHealthReady": true,
}

// Handler returns health route registration functions.
func Handler(
	logger *slog.Logger,
	checker Checker,
	startTime time.Time,
	version string,
	metrics MetricsProvider,
	subComponents map[string]SubComponentInfo,
	signingKey string,
	customRoles map[string][]string,
) []func(e *echo.Echo) {
	var tokenManager api.TokenValidator = authtoken.New(logger)

	healthHandler := New(logger, checker, startTime, version, metrics, subComponents)
	healthHandler.StartMetricsRefresh(context.Background())

	strictHandler := gen.NewStrictHandler(
		healthHandler,
		[]gen.StrictMiddlewareFunc{
			func(handler gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
				if unauthenticatedOperations[operationID] {
					return handler
				}

				return gen.StrictHandlerFunc(api.ScopeMiddleware(
					api.StrictHandlerFunc(handler),
					tokenManager,
					signingKey,
					string(gen.BearerAuthScopes),
					customRoles,
				))
			},
		},
	)

	return []func(e *echo.Echo){
		func(e *echo.Echo) {
			gen.RegisterHandlers(e, strictHandler)
		},
	}
}
