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

package authtoken_test

import (
	"encoding/base64"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/authtoken"
)

type AuthTokenPublicTestSuite struct {
	suite.Suite

	token      *authtoken.Token
	signingKey string
}

func (s *AuthTokenPublicTestSuite) SetupTest() {
	s.token = authtoken.New(slog.Default())
	s.signingKey = "test-signing-key-for-jwt-operations"
}

func (s *AuthTokenPublicTestSuite) TestNew() {
	t := authtoken.New(slog.Default())
	s.NotNil(t)
}

func (s *AuthTokenPublicTestSuite) TestGenerateAllowedRoles() {
	roles := authtoken.GenerateAllowedRoles(authtoken.RoleHierarchy)

	s.Len(roles, 3)
	s.ElementsMatch([]string{"admin", "write", "read"}, roles)
}

func (s *AuthTokenPublicTestSuite) TestGenerate() {
	tokenString, err := s.token.Generate(s.signingKey, []string{"admin"}, "test-subject", nil)

	s.NoError(err)
	s.NotEmpty(tokenString)
}

func (s *AuthTokenPublicTestSuite) TestValidate() {
	tests := []struct {
		name         string
		tokenFunc    func() string
		signingKey   string
		expectError  bool
		errContains  string
		validateFunc func(*authtoken.CustomClaims)
	}{
		{
			name: "valid token",
			tokenFunc: func() string {
				t, _ := s.token.Generate(s.signingKey, []string{"admin"}, "test-subject", nil)
				return t
			},
			signingKey:  s.signingKey,
			expectError: false,
			validateFunc: func(claims *authtoken.CustomClaims) {
				s.Equal([]string{"admin"}, claims.Roles)
				s.Equal("test-subject", claims.Subject)
				s.Equal("osapi", claims.Issuer)
			},
		},
		{
			name: "wrong signing key",
			tokenFunc: func() string {
				t, _ := s.token.Generate(s.signingKey, []string{"read"}, "test-subject", nil)
				return t
			},
			signingKey:  "wrong-key",
			expectError: true,
			errContains: "signature is invalid",
		},
		{
			name: "malformed token",
			tokenFunc: func() string {
				return "not-a-valid-jwt-token"
			},
			signingKey:  s.signingKey,
			expectError: true,
			errContains: "invalid number of segments",
		},
		{
			name: "empty token",
			tokenFunc: func() string {
				return ""
			},
			signingKey:  s.signingKey,
			expectError: true,
			errContains: "invalid number of segments",
		},
		{
			name: "unexpected signing method",
			tokenFunc: func() string {
				header := base64.RawURLEncoding.EncodeToString(
					[]byte(`{"alg":"none","typ":"JWT"}`),
				)
				payload := base64.RawURLEncoding.EncodeToString(
					[]byte(`{"roles":["admin"]}`),
				)
				return header + "." + payload + "."
			},
			signingKey:  s.signingKey,
			expectError: true,
			errContains: "unexpected signing method",
		},
		{
			name: "claims fail struct validation",
			tokenFunc: func() string {
				claims := authtoken.CustomClaims{
					Roles: []string{"invalid_role"},
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    "osapi",
						IssuedAt:  jwt.NewNumericDate(time.Now()),
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
						Subject:   "test",
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				t, _ := token.SignedString([]byte(s.signingKey))
				return t
			},
			signingKey:  s.signingKey,
			expectError: true,
			errContains: "Roles",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tokenString := tt.tokenFunc()

			claims, err := s.token.Validate(tokenString, tt.signingKey)

			if tt.expectError {
				s.Error(err)
				s.Nil(claims)
				if tt.errContains != "" {
					s.Contains(err.Error(), tt.errContains)
				}
			} else {
				s.NoError(err)
				s.NotNil(claims)
				tt.validateFunc(claims)
			}
		})
	}
}

func (s *AuthTokenPublicTestSuite) TestGenerateAndValidateRoundTrip() {
	tests := []struct {
		name    string
		roles   []string
		subject string
	}{
		{
			name:    "admin role round trip",
			roles:   []string{"admin"},
			subject: "admin-user",
		},
		{
			name:    "multiple roles round trip",
			roles:   []string{"read", "write"},
			subject: "rw-user",
		},
		{
			name:    "read only round trip",
			roles:   []string{"read"},
			subject: "reader",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tokenString, err := s.token.Generate(s.signingKey, tt.roles, tt.subject, nil)
			s.NoError(err)
			s.NotEmpty(tokenString)

			claims, err := s.token.Validate(tokenString, s.signingKey)
			s.NoError(err)
			s.NotNil(claims)
			s.Equal(tt.roles, claims.Roles)
			s.Equal(tt.subject, claims.Subject)
		})
	}
}

func TestAuthTokenPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuthTokenPublicTestSuite))
}
