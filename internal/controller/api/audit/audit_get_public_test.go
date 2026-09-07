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

package audit_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	auditstore "github.com/osapi-io/osapi/internal/audit"
	auditmocks "github.com/osapi-io/osapi/internal/audit/mocks"
	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	auditapi "github.com/osapi-io/osapi/internal/controller/api/audit"
	"github.com/osapi-io/osapi/internal/controller/api/audit/gen"
)

type AuditGetPublicTestSuite struct {
	suite.Suite

	appConfig config.Config
	logger    *slog.Logger
	mockCtrl  *gomock.Controller
	mockStore *auditmocks.MockStore
	handler   *auditapi.Audit
	ctx       context.Context
}

func (s *AuditGetPublicTestSuite) SetupTest() {
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	s.mockCtrl = gomock.NewController(s.T())
	s.mockStore = auditmocks.NewMockStore(s.mockCtrl)
	s.handler = auditapi.New(s.logger, s.mockStore)
	s.ctx = context.Background()
}

func (s *AuditGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AuditGetPublicTestSuite) TestGetAuditLogByID() {
	testID := uuid.New()

	tests := []struct {
		name         string
		id           openapi_types.UUID
		setupStore   func()
		validateFunc func(resp gen.GetAuditLogByIDResponseObject)
	}{
		{
			name: "returns entry successfully",
			id:   testID,
			setupStore: func() {
				s.mockStore.EXPECT().
					Get(gomock.Any(), testID.String()).
					Return(&auditstore.Entry{
						ID:           testID.String(),
						Timestamp:    time.Now(),
						User:         "user@example.com",
						Roles:        []string{"admin"},
						Method:       "GET",
						Path:         "/api/node/hostname",
						SourceIP:     "127.0.0.1",
						ResponseCode: 200,
						DurationMs:   42,
					}, nil)
			},
			validateFunc: func(resp gen.GetAuditLogByIDResponseObject) {
				r, ok := resp.(gen.GetAuditLogByID200JSONResponse)
				s.True(ok)
				s.Equal("user@example.com", r.Entry.User)
			},
		},
		{
			name: "returns 404 when not found",
			id:   testID,
			setupStore: func() {
				s.mockStore.EXPECT().
					Get(gomock.Any(), testID.String()).
					Return(nil, fmt.Errorf("get audit entry: not found"))
			},
			validateFunc: func(resp gen.GetAuditLogByIDResponseObject) {
				_, ok := resp.(gen.GetAuditLogByID404JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "returns 500 on store error",
			id:   testID,
			setupStore: func() {
				s.mockStore.EXPECT().
					Get(gomock.Any(), testID.String()).
					Return(nil, fmt.Errorf("connection error"))
			},
			validateFunc: func(resp gen.GetAuditLogByIDResponseObject) {
				_, ok := resp.(gen.GetAuditLogByID500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupStore()
			resp, err := s.handler.GetAuditLogByID(s.ctx, gen.GetAuditLogByIDRequestObject{
				Id: tt.id,
			})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AuditGetPublicTestSuite) TestGetAuditLogByIDHTTP() {
	tests := []struct {
		name         string
		path         string
		setupStore   func(mock *auditmocks.MockStore)
		wantCode     int
		wantContains []string
	}{
		{
			name: "when valid UUID returns entry",
			path: "/api/audit/550e8400-e29b-41d4-a716-446655440000",
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					Get(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(&auditstore.Entry{
						ID:           "550e8400-e29b-41d4-a716-446655440000",
						Timestamp:    time.Now(),
						User:         "user@example.com",
						Roles:        []string{"admin"},
						Method:       "GET",
						Path:         "/api/node/hostname",
						SourceIP:     "127.0.0.1",
						ResponseCode: 200,
						DurationMs:   42,
					}, nil)
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"user":"user@example.com"`},
		},
		{
			name:         "when invalid UUID returns 400",
			path:         "/api/audit/not-a-uuid",
			setupStore:   func(_ *auditmocks.MockStore) {},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()

			mock := auditmocks.NewMockStore(ctrl)
			tc.setupStore(mock)

			a := api.New(s.appConfig, s.logger)

			auditHandler := newTestAuditHandler(s.logger, mock)
			strictHandler := gen.NewStrictHandler(auditHandler, nil)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodGet,
				tc.path,
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacAuditGetTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *AuditGetPublicTestSuite) TestGetAuditLogByIDRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupStore   func(mock *auditmocks.MockStore)
		wantCode     int
		wantContains []string
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupStore:   func(_ *auditmocks.MockStore) {},
			wantCode:     http.StatusUnauthorized,
			wantContains: []string{"Bearer token required"},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAuditGetTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"job:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupStore:   func(_ *auditmocks.MockStore) {},
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with audit:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAuditGetTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					Get(gomock.Any(), "550e8400-e29b-41d4-a716-446655440000").
					Return(&auditstore.Entry{
						ID:           "550e8400-e29b-41d4-a716-446655440000",
						Timestamp:    time.Now(),
						User:         "user@example.com",
						Roles:        []string{"admin"},
						Method:       "GET",
						Path:         "/api/node/hostname",
						SourceIP:     "127.0.0.1",
						ResponseCode: 200,
						DurationMs:   42,
					}, nil)
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"user":"user@example.com"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctrl := gomock.NewController(s.T())
			defer ctrl.Finish()

			mock := auditmocks.NewMockStore(ctrl)
			tc.setupStore(mock)

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacAuditGetTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := auditapi.Handler(
				s.logger,
				mock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/audit/550e8400-e29b-41d4-a716-446655440000",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

func TestAuditGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuditGetPublicTestSuite))
}
