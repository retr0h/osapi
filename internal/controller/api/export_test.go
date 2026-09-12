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

package api

import (
	"log/slog"

	"github.com/labstack/echo/v5"

	"github.com/osapi-io/osapi/internal/audit"
	"github.com/osapi-io/osapi/internal/authtoken"
)

// ExportAuditMiddleware exposes the private auditMiddleware for testing.
func ExportAuditMiddleware(
	store audit.Store,
	logger *slog.Logger,
) echo.MiddlewareFunc {
	return auditMiddleware(store, logger)
}

// ExportScopeMiddleware exposes ScopeMiddleware for testing.
func ExportScopeMiddleware(
	next StrictHandlerFunc,
	tokenManager *authtoken.Token,
	signingKey string,
	contextKey string,
	customRoles map[string][]string,
) StrictHandlerFunc {
	return ScopeMiddleware(next, tokenManager, signingKey, contextKey, customRoles)
}
