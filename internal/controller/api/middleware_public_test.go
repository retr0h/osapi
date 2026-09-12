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

package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/controller/api"
)

const testSigningKey = "test-signing-key-for-middleware"

type MiddlewarePublicTestSuite struct {
	suite.Suite

	tokenManager *authtoken.Token
}

func (s *MiddlewarePublicTestSuite) SetupSuite() {
	logger := slog.Default()
	s.tokenManager = authtoken.New(logger)
}

func (s *MiddlewarePublicTestSuite) generateToken(
	roles []string,
) string {
	token, err := s.tokenManager.Generate(testSigningKey, roles, "test-subject", nil)
	s.Require().NoError(err)

	return token
}

func (s *MiddlewarePublicTestSuite) generateTokenWithPerms(
	roles []string,
	permissions []string,
) string {
	token, err := s.tokenManager.Generate(testSigningKey, roles, "test-subject", permissions)
	s.Require().NoError(err)

	return token
}

func (s *MiddlewarePublicTestSuite) TestScopeMiddleware() {
	contextKey := "BearerAuthScopes"

	tests := []struct {
		name            string
		authHeader      string
		requiredScopes  []string
		customRoles     map[string][]string
		expectedStatus  int
		setupContextKey bool
		validateFunc    func(bool, *httptest.ResponseRecorder)
	}{
		{
			name:           "no auth header returns 401",
			authHeader:     "",
			requiredScopes: []string{"node:read"},
			expectedStatus: http.StatusUnauthorized,
			validateFunc: func(called bool, rec *httptest.ResponseRecorder) {
				s.False(called)
				s.Equal(http.StatusUnauthorized, rec.Code)
			},
			setupContextKey: true,
		},
		{
			name:           "non-bearer auth header returns 401",
			authHeader:     "Basic dXNlcjpwYXNz",
			requiredScopes: []string{"node:read"},
			expectedStatus: http.StatusUnauthorized,
			validateFunc: func(called bool, rec *httptest.ResponseRecorder) {
				s.False(called)
				s.Equal(http.StatusUnauthorized, rec.Code)
			},
			setupContextKey: true,
		},
		{
			name:           "invalid token returns 401",
			authHeader:     "Bearer invalid-token-string",
			requiredScopes: []string{"node:read"},
			expectedStatus: http.StatusUnauthorized,
			validateFunc: func(called bool, rec *httptest.ResponseRecorder) {
				s.False(called)
				s.Equal(http.StatusUnauthorized, rec.Code)
			},
			setupContextKey: true,
		},
		{
			name:           "admin role has node:read",
			authHeader:     "", // set dynamically
			requiredScopes: []string{"node:read"},
			expectedStatus: http.StatusOK,
			validateFunc: func(called bool, _ *httptest.ResponseRecorder) {
				s.True(called)
			},
			setupContextKey: true,
		},
		{
			name:           "read role has node:read",
			authHeader:     "", // set dynamically
			requiredScopes: []string{"node:read"},
			expectedStatus: http.StatusOK,
			validateFunc: func(called bool, _ *httptest.ResponseRecorder) {
				s.True(called)
			},
			setupContextKey: true,
		},
		{
			name:           "read role lacks network:write returns 403",
			authHeader:     "", // set dynamically
			requiredScopes: []string{"network:write"},
			expectedStatus: http.StatusForbidden,
			validateFunc: func(called bool, rec *httptest.ResponseRecorder) {
				s.False(called)
				s.Equal(http.StatusForbidden, rec.Code)
			},
			setupContextKey: true,
		},
		{
			name:           "valid token with no required scopes calls handler",
			authHeader:     "", // set dynamically
			requiredScopes: nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(called bool, _ *httptest.ResponseRecorder) {
				s.True(called)
			},
			setupContextKey: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			handlerCalled := false
			testHandler := api.StrictHandlerFunc(
				func(_ *echo.Context, _ interface{}) (interface{}, error) {
					handlerCalled = true
					return "ok", nil
				},
			)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			authHeader := tt.authHeader
			if authHeader == "" && tt.expectedStatus != http.StatusUnauthorized {
				if tt.expectedStatus == http.StatusForbidden {
					authHeader = "Bearer " + s.generateToken([]string{"read"})
				} else {
					authHeader = "Bearer " + s.generateToken([]string{"admin"})
				}
			}
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			ctx := e.NewContext(req, rec)

			if tt.setupContextKey && tt.requiredScopes != nil {
				ctx.Set(contextKey, tt.requiredScopes)
			}

			wrapped := api.ExportScopeMiddleware(
				testHandler,
				s.tokenManager,
				testSigningKey,
				contextKey,
				tt.customRoles,
			)
			_, _ = wrapped(ctx, nil)

			tt.validateFunc(handlerCalled, rec)
		})
	}
}

func (s *MiddlewarePublicTestSuite) TestScopeMiddlewareCustomRoles() {
	contextKey := "BearerAuthScopes"

	tests := []struct {
		name           string
		tokenRoles     []string
		customRoles    map[string][]string
		requiredScope  string
		expectedStatus int
		validateFunc   func(bool, *httptest.ResponseRecorder)
	}{
		{
			name:       "custom role grants access",
			tokenRoles: []string{"ops"},
			customRoles: map[string][]string{
				"ops": {"node:read", "health:read"},
			},
			requiredScope:  "node:read",
			expectedStatus: http.StatusOK,
			validateFunc: func(called bool, _ *httptest.ResponseRecorder) {
				s.True(called)
			},
		},
		{
			name:       "custom role lacks permission",
			tokenRoles: []string{"ops"},
			customRoles: map[string][]string{
				"ops": {"health:read"},
			},
			requiredScope:  "node:read",
			expectedStatus: http.StatusForbidden,
			validateFunc: func(called bool, rec *httptest.ResponseRecorder) {
				s.False(called)
				s.Equal(http.StatusForbidden, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			handlerCalled := false
			testHandler := api.StrictHandlerFunc(
				func(_ *echo.Context, _ interface{}) (interface{}, error) {
					handlerCalled = true
					return "ok", nil
				},
			)

			token, err := s.tokenManager.Generate(
				testSigningKey,
				[]string{"admin"},
				"test-subject",
				nil,
			)
			s.Require().NoError(err)

			// Override custom roles to shadow "admin".
			customRoles := map[string][]string{
				"admin": tt.customRoles[tt.tokenRoles[0]],
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			ctx := e.NewContext(req, rec)
			ctx.Set(contextKey, []string{tt.requiredScope})

			wrapped := api.ExportScopeMiddleware(
				testHandler,
				s.tokenManager,
				testSigningKey,
				contextKey,
				customRoles,
			)
			_, _ = wrapped(ctx, nil)

			tt.validateFunc(handlerCalled, rec)
		})
	}
}

func (s *MiddlewarePublicTestSuite) TestScopeMiddlewareDirectPermissions() {
	contextKey := "BearerAuthScopes"

	tests := []struct {
		name           string
		permissions    []string
		requiredScope  string
		expectedStatus int
		validateFunc   func(bool, *httptest.ResponseRecorder)
	}{
		{
			name:           "direct permission grants access",
			permissions:    []string{"node:read"},
			requiredScope:  "node:read",
			expectedStatus: http.StatusOK,
			validateFunc: func(called bool, _ *httptest.ResponseRecorder) {
				s.True(called)
			},
		},
		{
			name:           "direct permission restricts to only listed",
			permissions:    []string{"health:read"},
			requiredScope:  "node:read",
			expectedStatus: http.StatusForbidden,
			validateFunc: func(called bool, rec *httptest.ResponseRecorder) {
				s.False(called)
				s.Equal(http.StatusForbidden, rec.Code)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			handlerCalled := false
			testHandler := api.StrictHandlerFunc(
				func(_ *echo.Context, _ interface{}) (interface{}, error) {
					handlerCalled = true
					return "ok", nil
				},
			)

			token := s.generateTokenWithPerms([]string{"admin"}, tt.permissions)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			ctx := e.NewContext(req, rec)
			ctx.Set(contextKey, []string{tt.requiredScope})

			wrapped := api.ExportScopeMiddleware(
				testHandler,
				s.tokenManager,
				testSigningKey,
				contextKey,
				nil,
			)
			_, _ = wrapped(ctx, nil)

			tt.validateFunc(handlerCalled, rec)
		})
	}
}

func (s *MiddlewarePublicTestSuite) TestScopeMiddlewareInjectsIdentity() {
	contextKey := "BearerAuthScopes"

	tests := []struct {
		name           string
		tokenRoles     []string
		authHeader     string
		requiredScopes []string
		validateFunc   func(bool, string, []string)
	}{
		{
			name:           "admin token injects subject and roles",
			tokenRoles:     []string{"admin"},
			requiredScopes: []string{"node:read"},
			validateFunc: func(called bool, subject string, roles []string) {
				s.True(called)
				s.Equal("test-subject", subject)
				s.Equal([]string{"admin"}, roles)
			},
		},
		{
			name:           "read token injects subject and read role",
			tokenRoles:     []string{"read"},
			requiredScopes: []string{"node:read"},
			validateFunc: func(called bool, subject string, roles []string) {
				s.True(called)
				s.Equal("test-subject", subject)
				s.Equal([]string{"read"}, roles)
			},
		},
		{
			name:           "invalid token does not inject identity",
			authHeader:     "Bearer invalid-token-string",
			requiredScopes: []string{"node:read"},
			validateFunc: func(called bool, _ string, _ []string) {
				s.False(called)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			handlerCalled := false
			var capturedSubject string
			var capturedRoles []string

			testHandler := api.StrictHandlerFunc(
				func(ctx *echo.Context, _ interface{}) (interface{}, error) {
					handlerCalled = true
					capturedSubject, _ = ctx.Get(api.ContextKeySubject).(string)
					capturedRoles, _ = ctx.Get(api.ContextKeyRoles).([]string)
					return "ok", nil
				},
			)

			authHeader := tt.authHeader
			if authHeader == "" && len(tt.tokenRoles) > 0 {
				authHeader = "Bearer " + s.generateToken(tt.tokenRoles)
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}
			rec := httptest.NewRecorder()

			ctx := e.NewContext(req, rec)
			if tt.requiredScopes != nil {
				ctx.Set(contextKey, tt.requiredScopes)
			}

			wrapped := api.ExportScopeMiddleware(
				testHandler,
				s.tokenManager,
				testSigningKey,
				contextKey,
				nil,
			)
			_, _ = wrapped(ctx, nil)

			tt.validateFunc(handlerCalled, capturedSubject, capturedRoles)
		})
	}
}

func TestMiddlewarePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MiddlewarePublicTestSuite))
}
