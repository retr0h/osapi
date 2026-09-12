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
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/controller/api/common/gen"
)

// Context key constants for injecting user identity into handlers.
const (
	ContextKeySubject = "auth.subject"
	ContextKeyRoles   = "auth.roles"
)

// TokenValidator parses and validates JWT tokens.
type TokenValidator interface {
	Validate(
		tokenString string,
		signingKey string,
	) (*authtoken.CustomClaims, error)
}

// ScopeMiddleware validates JWT tokens and checks for required permissions.
func ScopeMiddleware(
	handler StrictHandlerFunc,
	tokenManager TokenValidator,
	signingKey string,
	contextKey string,
	customRoles map[string][]string,
) StrictHandlerFunc {
	return StrictHandlerFunc(
		func(ctx *echo.Context, request interface{}) (response interface{}, err error) {
			authHeader := ctx.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				errMsg := "Bearer token required"
				return nil, ctx.JSON(http.StatusUnauthorized, gen.ErrorResponse{
					Error: &errMsg,
				})
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := tokenManager.Validate(tokenString, signingKey)
			if err != nil {
				errMsg := "Invalid token: " + err.Error()
				return nil, ctx.JSON(http.StatusUnauthorized, gen.ErrorResponse{
					Error: &errMsg,
				})
			}

			// Inject user identity into context for handlers and audit logging.
			ctx.Set(ContextKeySubject, claims.Subject)
			ctx.Set(ContextKeyRoles, claims.Roles)

			requiredScopes, ok := ctx.Get(contextKey).([]string)
			if !ok || len(requiredScopes) == 0 {
				return handler(ctx, request)
			}

			resolved := authtoken.ResolvePermissions(
				claims.Roles,
				claims.Permissions,
				customRoles,
			)

			for _, required := range requiredScopes {
				if authtoken.HasPermission(resolved, required) {
					return handler(ctx, request)
				}
			}

			errMsg := fmt.Sprintf(
				"Insufficient permissions. Required: %v, resolved: %v",
				requiredScopes,
				resolved,
			)
			return nil, ctx.JSON(http.StatusForbidden, gen.ErrorResponse{
				Error: &errMsg,
			})
		},
	)
}
