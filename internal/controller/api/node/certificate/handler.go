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

package certificate

import (
	"log/slog"

	"github.com/labstack/echo/v5"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/controller/api"
	gen "github.com/osapi-io/osapi/internal/controller/api/node/certificate/gen"
	"github.com/osapi-io/osapi/internal/job/client"
)

// Handler returns Certificate route registration functions.
func Handler(
	logger *slog.Logger,
	jobClient client.JobClient,
	signingKey string,
	customRoles map[string][]string,
) []func(e *echo.Echo) {
	var tokenManager api.TokenValidator = authtoken.New(logger)

	certificateHandler := New(logger, jobClient)

	strictHandler := gen.NewStrictHandler(
		certificateHandler,
		[]gen.StrictMiddlewareFunc{
			func(handler gen.StrictHandlerFunc, _ string) gen.StrictHandlerFunc {
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
