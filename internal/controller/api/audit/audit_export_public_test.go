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

type AuditExportPublicTestSuite struct {
	suite.Suite

	appConfig config.Config
	logger    *slog.Logger
	mockCtrl  *gomock.Controller
	mockStore *auditmocks.MockStore
	handler   *auditapi.Audit
	ctx       context.Context
}

func (s *AuditExportPublicTestSuite) SetupTest() {
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	s.mockCtrl = gomock.NewController(s.T())
	s.mockStore = auditmocks.NewMockStore(s.mockCtrl)
	s.handler = auditapi.New(s.logger, s.mockStore)
	s.ctx = context.Background()
}

func (s *AuditExportPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AuditExportPublicTestSuite) TestGetAuditExport() {
	tests := []struct {
		name         string
		setupStore   func()
		validateFunc func(resp gen.GetAuditExportResponseObject)
	}{
		{
			name: "returns entries successfully",
			setupStore: func() {
				s.mockStore.EXPECT().
					ListAll(gomock.Any()).
					Return([]auditstore.Entry{
						{
							ID:           "550e8400-e29b-41d4-a716-446655440000",
							Timestamp:    time.Now(),
							User:         "user@example.com",
							Roles:        []string{"admin"},
							Method:       "GET",
							Path:         "/api/node/hostname",
							SourceIP:     "127.0.0.1",
							ResponseCode: 200,
							DurationMs:   42,
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetAuditExportResponseObject) {
				r, ok := resp.(gen.GetAuditExport200JSONResponse)
				s.True(ok)
				s.Equal(1, r.TotalItems)
				s.Len(r.Items, 1)
				s.Equal("user@example.com", r.Items[0].User)
			},
		},
		{
			name: "returns empty list",
			setupStore: func() {
				s.mockStore.EXPECT().
					ListAll(gomock.Any()).
					Return([]auditstore.Entry{}, nil)
			},
			validateFunc: func(resp gen.GetAuditExportResponseObject) {
				r, ok := resp.(gen.GetAuditExport200JSONResponse)
				s.True(ok)
				s.Equal(0, r.TotalItems)
				s.Empty(r.Items)
			},
		},
		{
			name: "returns entry with operation ID",
			setupStore: func() {
				s.mockStore.EXPECT().
					ListAll(gomock.Any()).
					Return([]auditstore.Entry{
						{
							ID:           "550e8400-e29b-41d4-a716-446655440000",
							Timestamp:    time.Now(),
							User:         "user@example.com",
							Roles:        []string{"admin"},
							Method:       "GET",
							Path:         "/api/node/hostname",
							OperationID:  "getNodeHostname",
							SourceIP:     "127.0.0.1",
							ResponseCode: 200,
							DurationMs:   42,
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetAuditExportResponseObject) {
				r, ok := resp.(gen.GetAuditExport200JSONResponse)
				s.True(ok)
				s.Len(r.Items, 1)
				s.Require().NotNil(r.Items[0].OperationId)
				s.Equal("getNodeHostname", *r.Items[0].OperationId)
			},
		},
		{
			name: "returns 500 on store error",
			setupStore: func() {
				s.mockStore.EXPECT().
					ListAll(gomock.Any()).
					Return(nil, fmt.Errorf("store error"))
			},
			validateFunc: func(resp gen.GetAuditExportResponseObject) {
				_, ok := resp.(gen.GetAuditExport500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupStore()
			resp, err := s.handler.GetAuditExport(s.ctx, gen.GetAuditExportRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AuditExportPublicTestSuite) TestGetAuditExportHTTP() {
	tests := []struct {
		name         string
		setupStore   func(mock *auditmocks.MockStore)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when valid request returns entries",
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					ListAll(gomock.Any()).
					Return([]auditstore.Entry{
						{
							ID:           "550e8400-e29b-41d4-a716-446655440000",
							Timestamp:    time.Now(),
							User:         "user@example.com",
							Roles:        []string{"admin"},
							Method:       "GET",
							Path:         "/api/node/hostname",
							SourceIP:     "127.0.0.1",
							ResponseCode: 200,
							DurationMs:   42,
						},
					}, nil)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "total_items")
			},
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
				"/api/audit/export",
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacAuditExportTestSigningKey = "test-signing-key-for-rbac-export"

func (s *AuditExportPublicTestSuite) TestGetAuditExportRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupStore   func(mock *auditmocks.MockStore)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupStore: func(_ *auditmocks.MockStore) {},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusUnauthorized, rec.Code)
				s.Contains(rec.Body.String(), "Bearer token required")
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAuditExportTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"job:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupStore: func(_ *auditmocks.MockStore) {},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusForbidden, rec.Code)
				s.Contains(rec.Body.String(), "Insufficient permissions")
			},
		},
		{
			name: "when valid token with audit:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacAuditExportTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					ListAll(gomock.Any()).
					Return([]auditstore.Entry{}, nil)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), "total_items")
			},
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
							SigningKey: rbacAuditExportTestSigningKey,
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
				"/api/audit/export",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestAuditExportPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuditExportPublicTestSuite))
}
