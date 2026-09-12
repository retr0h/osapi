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

package facts_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	"github.com/osapi-io/osapi/internal/controller/api/facts"
	"github.com/osapi-io/osapi/internal/controller/api/facts/gen"
	factskeys "github.com/osapi-io/osapi/internal/facts"
)

const testSigningKey = "test-signing-key-for-facts"

type FactsKeysGetPublicTestSuite struct {
	suite.Suite

	logger  *slog.Logger
	handler *facts.Facts
}

func (s *FactsKeysGetPublicTestSuite) SetupTest() {
	s.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	s.handler = facts.New(s.logger)
}

func (s *FactsKeysGetPublicTestSuite) TestGetFactKeys() {
	tests := []struct {
		name         string
		validateFunc func(gen.GetFactKeysResponseObject)
	}{
		{
			name: "when successful returns all built-in keys",
			validateFunc: func(resp gen.GetFactKeysResponseObject) {
				jsonResp, ok := resp.(gen.GetFactKeys200JSONResponse)
				s.True(ok, "expected 200 response")

				builtIn := factskeys.BuiltInKeys()
				s.Len(jsonResp.Keys, len(builtIn))

				keys := make([]string, len(jsonResp.Keys))
				for i, entry := range jsonResp.Keys {
					keys[i] = entry.Key
				}
				for _, key := range builtIn {
					s.Contains(keys, key)
				}
			},
		},
		{
			name: "when all entries have descriptions",
			validateFunc: func(resp gen.GetFactKeysResponseObject) {
				jsonResp := resp.(gen.GetFactKeys200JSONResponse)
				for _, entry := range jsonResp.Keys {
					s.NotNil(entry.Description, "key %s should have a description", entry.Key)
					s.NotEmpty(
						*entry.Description,
						"key %s description should not be empty",
						entry.Key,
					)
				}
			},
		},
		{
			name: "when all entries are marked as builtin",
			validateFunc: func(resp gen.GetFactKeysResponseObject) {
				jsonResp := resp.(gen.GetFactKeys200JSONResponse)
				for _, entry := range jsonResp.Keys {
					s.NotNil(entry.Builtin, "key %s should have builtin field", entry.Key)
					s.True(*entry.Builtin, "key %s should be marked as builtin", entry.Key)
				}
			},
		},
		{
			name: "when contains expected specific keys",
			validateFunc: func(resp gen.GetFactKeysResponseObject) {
				jsonResp := resp.(gen.GetFactKeys200JSONResponse)
				keyMap := make(map[string]gen.FactKeyEntry)
				for _, entry := range jsonResp.Keys {
					keyMap[entry.Key] = entry
				}
				s.Contains(keyMap, factskeys.KeyInterfacePrimary)
				s.Contains(keyMap, factskeys.KeyHostname)
				s.Contains(keyMap, factskeys.KeyArch)
				s.Contains(keyMap, factskeys.KeyKernel)
				s.Contains(keyMap, factskeys.KeyFQDN)
				s.Contains(keyMap, factskeys.KeyContainerized)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp, err := s.handler.GetFactKeys(
				context.Background(),
				gen.GetFactKeysRequestObject{},
			)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FactsKeysGetPublicTestSuite) TestGetFactKeysRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		validateFunc func(int)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusUnauthorized, got)
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					testSigningKey,
					[]string{"write"},
					"test-user",
					[]string{"file:write"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusForbidden, got)
			},
		},
		{
			name: "when valid token with agent:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					testSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"agent:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusOK, got)
			},
		},
		{
			name: "when admin role returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					testSigningKey,
					[]string{"admin"},
					"test-admin",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			validateFunc: func(got int) {
				s.Equal(http.StatusOK, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: testSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := facts.Handler(s.logger, appConfig.Controller.API.Security.SigningKey, nil)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/facts/keys", nil)
			tt.setupAuth(req)
			rec := httptest.NewRecorder()
			server.Echo.ServeHTTP(rec, req)

			tt.validateFunc(rec.Code)
		})
	}
}

func TestFactsKeysGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FactsKeysGetPublicTestSuite))
}
