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
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/osapi-io/osapi/internal/audit"
	"github.com/osapi-io/osapi/internal/config"
)

// Server implementation of the Server's API operations.
type Server struct {
	Echo          *echo.Echo
	logger        *slog.Logger
	appConfig     config.Config
	auditStore    audit.Store
	meterProvider *sdkmetric.MeterProvider

	// stop cancels the context StartConfig.Start is listening on. v5 has no
	// Shutdown method: cancelling the context is how the server is asked to
	// drain.
	stop context.CancelFunc
}

// StrictHandlerFunc is the shape oapi-codegen's strict handlers take under
// Echo v5. Each generated package declares its own identical type, so a helper
// shared across packages cannot name any one of them. Declaring it once here
// gives the shared middleware a type to sit on; call sites convert to and from
// their own package's version, which Go permits because the underlying
// signatures match.
type StrictHandlerFunc func(ctx *echo.Context, request any) (any, error)

// Option is a functional option for configuring the Server.
type Option func(*Server)
