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

package file_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/authtoken"
	"github.com/osapi-io/osapi/internal/config"
	"github.com/osapi-io/osapi/internal/controller/api"
	apifile "github.com/osapi-io/osapi/internal/controller/api/file"
	"github.com/osapi-io/osapi/internal/controller/api/file/gen"
	"github.com/osapi-io/osapi/internal/controller/api/file/mocks"
)

type FileDeletePublicTestSuite struct {
	suite.Suite

	mockCtrl     *gomock.Controller
	mockObjStore *mocks.MockObjectStoreManager
	handler      *apifile.File
	ctx          context.Context
	appConfig    config.Config
	logger       *slog.Logger
}

func (s *FileDeletePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockObjStore = mocks.NewMockObjectStoreManager(s.mockCtrl)
	s.handler = apifile.New(slog.Default(), s.mockObjStore, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileDeletePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *FileDeletePublicTestSuite) TestDeleteFileByName() {
	tests := []struct {
		name         string
		request      gen.DeleteFileByNameRequestObject
		setupMock    func()
		validateFunc func(resp gen.DeleteFileByNameResponseObject)
	}{
		{
			name:    "success",
			request: gen.DeleteFileByNameRequestObject{Name: "nginx.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       1024,
					}, nil)
				s.mockObjStore.EXPECT().
					Delete(gomock.Any(), "nginx.conf").
					Return(nil)
			},
			validateFunc: func(resp gen.DeleteFileByNameResponseObject) {
				r, ok := resp.(gen.DeleteFileByName200JSONResponse)
				s.True(ok)
				s.Equal("nginx.conf", r.Name)
				s.True(r.Deleted)
			},
		},
		{
			name:    "validation error name too long",
			request: gen.DeleteFileByNameRequestObject{Name: strings.Repeat("a", 256)},
			setupMock: func() {
				// No mock calls expected; validation rejects before reaching obj store.
			},
			validateFunc: func(resp gen.DeleteFileByNameResponseObject) {
				_, ok := resp.(gen.DeleteFileByName400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "when deleting system file returns 403",
			request: gen.DeleteFileByNameRequestObject{Name: "osapi/cron-template.tmpl"},
			setupMock: func() {
				// No mock calls expected; protected file check rejects before reaching obj store.
			},
			validateFunc: func(resp gen.DeleteFileByNameResponseObject) {
				r, ok := resp.(gen.DeleteFileByName403JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "cannot delete protected osapi file")
			},
		},
		{
			name:    "not found",
			request: gen.DeleteFileByNameRequestObject{Name: "missing.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "missing.conf").
					Return(nil, jetstream.ErrObjectNotFound)
			},
			validateFunc: func(resp gen.DeleteFileByNameResponseObject) {
				r, ok := resp.(gen.DeleteFileByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "file not found")
			},
		},
		{
			name:    "get info error",
			request: gen.DeleteFileByNameRequestObject{Name: "nginx.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)
			},
			validateFunc: func(resp gen.DeleteFileByNameResponseObject) {
				_, ok := resp.(gen.DeleteFileByName500JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "delete error",
			request: gen.DeleteFileByNameRequestObject{Name: "nginx.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       1024,
					}, nil)
				s.mockObjStore.EXPECT().
					Delete(gomock.Any(), "nginx.conf").
					Return(assert.AnError)
			},
			validateFunc: func(resp gen.DeleteFileByNameResponseObject) {
				_, ok := resp.(gen.DeleteFileByName500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.DeleteFileByName(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileDeletePublicTestSuite) TestDeleteFileByNameValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupMock    func() *mocks.MockObjectStoreManager
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when name too long returns 400",
			path: "/api/file/" + strings.Repeat("a", 256),
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), `"error"`)
			},
		},
		{
			name: "when delete Ok",
			path: "/api/file/nginx.conf",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       1024,
					}, nil)
				mock.EXPECT().
					Delete(gomock.Any(), "nginx.conf").
					Return(nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"name":"nginx.conf"`)
				s.Contains(rec.Body.String(), `"deleted":true`)
			},
		},
		{
			name: "when not found",
			path: "/api/file/missing.conf",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "missing.conf").
					Return(nil, jetstream.ErrObjectNotFound)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusNotFound, rec.Code)
				s.Contains(rec.Body.String(), "file not found")
			},
		},
		{
			name: "when delete error",
			path: "/api/file/nginx.conf",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       1024,
					}, nil)
				mock.EXPECT().
					Delete(gomock.Any(), "nginx.conf").
					Return(assert.AnError)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "failed to delete file")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			objMock := tc.setupMock()

			fileHandler := apifile.New(s.logger, objMock, nil)
			strictHandler := gen.NewStrictHandler(fileHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacDeleteTestSigningKey = "test-signing-key-for-file-delete-rbac"

func (s *FileDeletePublicTestSuite) TestDeleteFileByNameRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupMock    func() *mocks.MockObjectStoreManager
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusUnauthorized, rec.Code)
				s.Contains(rec.Body.String(), "Bearer token required")
			},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacDeleteTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"file:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusForbidden, rec.Code)
				s.Contains(rec.Body.String(), "Insufficient permissions")
			},
		},
		{
			name: "when valid token with file:write returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacDeleteTestSigningKey,
					[]string{"admin"},
					"test-user",
					[]string{"file:write"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       1024,
					}, nil)
				mock.EXPECT().
					Delete(gomock.Any(), "nginx.conf").
					Return(nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"name":"nginx.conf"`)
				s.Contains(rec.Body.String(), `"deleted":true`)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			objMock := tc.setupMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacDeleteTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apifile.Handler(
				s.logger,
				objMock,
				nil,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodDelete, "/api/file/nginx.conf", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileDeletePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileDeletePublicTestSuite))
}
