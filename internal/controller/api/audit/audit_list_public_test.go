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

type AuditListPublicTestSuite struct {
	suite.Suite

	appConfig config.Config
	logger    *slog.Logger
	mockCtrl  *gomock.Controller
	mockStore *auditmocks.MockStore
	handler   *auditapi.Audit
	ctx       context.Context
}

func (s *AuditListPublicTestSuite) SetupTest() {
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	s.mockCtrl = gomock.NewController(s.T())
	s.mockStore = auditmocks.NewMockStore(s.mockCtrl)
	s.handler = auditapi.New(s.logger, s.mockStore)
	s.ctx = context.Background()
}

func (s *AuditListPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *AuditListPublicTestSuite) TestGetAuditLogs() {
	limit := 10
	offset := 0

	tests := []struct {
		name         string
		params       gen.GetAuditLogsParams
		setupStore   func()
		validateFunc func(resp gen.GetAuditLogsResponseObject)
	}{
		{
			name:   "returns entries successfully",
			params: gen.GetAuditLogsParams{Limit: &limit, Offset: &offset},
			setupStore: func() {
				s.mockStore.EXPECT().
					List(gomock.Any(), limit, offset).
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
					}, 1, nil)
			},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs200JSONResponse)
				s.True(ok)
				s.Equal(1, r.TotalItems)
				s.Len(r.Items, 1)
				s.Equal("user@example.com", r.Items[0].User)
			},
		},
		{
			name:   "returns entry with trace ID",
			params: gen.GetAuditLogsParams{Limit: &limit, Offset: &offset},
			setupStore: func() {
				s.mockStore.EXPECT().
					List(gomock.Any(), limit, offset).
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
							TraceID:      "4bf92f3577b34da6a3ce929d0e0e4736",
						},
					}, 1, nil)
			},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs200JSONResponse)
				s.True(ok)
				s.Len(r.Items, 1)
				s.Require().NotNil(r.Items[0].TraceId)
				s.Equal("4bf92f3577b34da6a3ce929d0e0e4736", *r.Items[0].TraceId)
			},
		},
		{
			name:   "returns entry with empty trace ID as nil",
			params: gen.GetAuditLogsParams{Limit: &limit, Offset: &offset},
			setupStore: func() {
				s.mockStore.EXPECT().
					List(gomock.Any(), limit, offset).
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
							TraceID:      "",
						},
					}, 1, nil)
			},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs200JSONResponse)
				s.True(ok)
				s.Len(r.Items, 1)
				s.Nil(r.Items[0].TraceId)
			},
		},
		{
			name:   "returns entry with operation ID",
			params: gen.GetAuditLogsParams{Limit: &limit, Offset: &offset},
			setupStore: func() {
				s.mockStore.EXPECT().
					List(gomock.Any(), limit, offset).
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
					}, 1, nil)
			},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs200JSONResponse)
				s.True(ok)
				s.Len(r.Items, 1)
				s.Require().NotNil(r.Items[0].OperationId)
				s.Equal("getNodeHostname", *r.Items[0].OperationId)
			},
		},
		{
			name:   "returns empty list",
			params: gen.GetAuditLogsParams{},
			setupStore: func() {
				s.mockStore.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]auditstore.Entry{}, 0, nil)
			},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs200JSONResponse)
				s.True(ok)
				s.Equal(0, r.TotalItems)
				s.Empty(r.Items)
			},
		},
		{
			name: "returns 400 when limit is zero",
			params: func() gen.GetAuditLogsParams {
				l := 0
				return gen.GetAuditLogsParams{Limit: &l}
			}(),
			setupStore: func() {},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs400JSONResponse)
				s.True(ok)
				s.NotNil(r.Error)
			},
		},
		{
			name: "returns 400 when limit exceeds max",
			params: func() gen.GetAuditLogsParams {
				l := 200
				return gen.GetAuditLogsParams{Limit: &l}
			}(),
			setupStore: func() {},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs400JSONResponse)
				s.True(ok)
				s.NotNil(r.Error)
			},
		},
		{
			name: "returns 400 when offset is negative",
			params: func() gen.GetAuditLogsParams {
				o := -1
				return gen.GetAuditLogsParams{Offset: &o}
			}(),
			setupStore: func() {},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				r, ok := resp.(gen.GetAuditLogs400JSONResponse)
				s.True(ok)
				s.NotNil(r.Error)
			},
		},
		{
			name:   "returns 500 on store error",
			params: gen.GetAuditLogsParams{},
			setupStore: func() {
				s.mockStore.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, 0, fmt.Errorf("store error"))
			},
			validateFunc: func(resp gen.GetAuditLogsResponseObject) {
				_, ok := resp.(gen.GetAuditLogs500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupStore()
			resp, err := s.handler.GetAuditLogs(s.ctx, gen.GetAuditLogsRequestObject{
				Params: tt.params,
			})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *AuditListPublicTestSuite) TestGetAuditLogsValidationHTTP() {
	tests := []struct {
		name         string
		query        string
		setupStore   func(mock *auditmocks.MockStore)
		validateFunc func(rec *httptest.ResponseRecorder)
	}{
		{
			name:  "when valid request returns entries",
			query: "",
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
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
					}, 1, nil)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"total_items":1`)
			},
		},
		{
			name:  "when valid limit and offset params",
			query: "?limit=5&offset=10",
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]auditstore.Entry{}, 0, nil)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"total_items":0`)
			},
		},
		{
			name:       "when limit is zero returns 400",
			query:      "?limit=0",
			setupStore: func(_ *auditmocks.MockStore) {},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
		{
			name:       "when limit exceeds maximum returns 400",
			query:      "?limit=200",
			setupStore: func(_ *auditmocks.MockStore) {},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
		{
			name:       "when offset is negative returns 400",
			query:      "?offset=-1",
			setupStore: func(_ *auditmocks.MockStore) {},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctrl := gomock.NewController(s.T())

			mock := auditmocks.NewMockStore(ctrl)
			tc.setupStore(mock)

			a := api.New(s.appConfig, s.logger)

			auditHandler := newTestAuditHandler(s.logger, mock)
			strictHandler := gen.NewStrictHandler(auditHandler, nil)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/audit"+tc.query,
				nil,
			)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
			ctrl.Finish()
		})
	}
}

const rbacAuditListTestSigningKey = "test-signing-key-for-rbac-integration"

func (s *AuditListPublicTestSuite) TestGetAuditLogsRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupStore   func(mock *auditmocks.MockStore)
		validateFunc func(rec *httptest.ResponseRecorder)
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
					rbacAuditListTestSigningKey,
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
					rbacAuditListTestSigningKey,
					[]string{"admin"},
					"test-user",
					nil,
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupStore: func(mock *auditmocks.MockStore) {
				mock.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]auditstore.Entry{}, 0, nil)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"total_items":0`)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			ctrl := gomock.NewController(s.T())

			mock := auditmocks.NewMockStore(ctrl)
			tc.setupStore(mock)

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacAuditListTestSigningKey,
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
				"/api/audit",
				nil,
			)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
			ctrl.Finish()
		})
	}
}

func TestAuditListPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(AuditListPublicTestSuite))
}
