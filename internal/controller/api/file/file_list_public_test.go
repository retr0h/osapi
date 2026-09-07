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
	"testing"

	"github.com/nats-io/nats.go"
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

type FileListPublicTestSuite struct {
	suite.Suite

	mockCtrl     *gomock.Controller
	mockObjStore *mocks.MockObjectStoreManager
	handler      *apifile.File
	ctx          context.Context
	appConfig    config.Config
	logger       *slog.Logger
}

func (s *FileListPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockObjStore = mocks.NewMockObjectStoreManager(s.mockCtrl)
	s.handler = apifile.New(slog.Default(), s.mockObjStore, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileListPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *FileListPublicTestSuite) TestGetFiles() {
	tests := []struct {
		name         string
		setupMock    func()
		validateFunc func(resp gen.GetFilesResponseObject)
	}{
		{
			name: "success with user and osapi files",
			setupMock: func() {
				s.mockObjStore.EXPECT().
					List(gomock.Any()).
					Return([]*jetstream.ObjectInfo{
						{
							ObjectMeta: jetstream.ObjectMeta{
								Name: "nginx.conf",
								Headers: nats.Header{
									"Osapi-Content-Type": []string{"raw"},
								},
							},
							Size:   1024,
							Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
						},
						{
							ObjectMeta: jetstream.ObjectMeta{
								Name: "osapi/cron-wrapper.tmpl",
								Headers: nats.Header{
									"Osapi-Content-Type": []string{"template"},
								},
							},
							Size:   512,
							Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetFilesResponseObject) {
				r, ok := resp.(gen.GetFiles200JSONResponse)
				s.True(ok)
				s.Equal(2, r.Total)
				s.Len(r.Files, 2)
				s.Equal("nginx.conf", r.Files[0].Name)
				s.Equal(
					"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					r.Files[0].Sha256,
				)
				s.Equal(1024, r.Files[0].Size)
				s.Equal("raw", r.Files[0].ContentType)
				s.Equal("user", r.Files[0].Source)
				s.Equal("osapi/cron-wrapper.tmpl", r.Files[1].Name)
				s.Equal("template", r.Files[1].ContentType)
				s.Equal("osapi", r.Files[1].Source)
			},
		},
		{
			name: "success with empty store",
			setupMock: func() {
				s.mockObjStore.EXPECT().
					List(gomock.Any()).
					Return([]*jetstream.ObjectInfo{}, nil)
			},
			validateFunc: func(resp gen.GetFilesResponseObject) {
				r, ok := resp.(gen.GetFiles200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Files)
			},
		},
		{
			name: "when ErrNoObjectsFound returns empty list",
			setupMock: func() {
				s.mockObjStore.EXPECT().
					List(gomock.Any()).
					Return(nil, jetstream.ErrNoObjectsFound)
			},
			validateFunc: func(resp gen.GetFilesResponseObject) {
				r, ok := resp.(gen.GetFiles200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Files)
			},
		},
		{
			name: "filters deleted objects",
			setupMock: func() {
				s.mockObjStore.EXPECT().
					List(gomock.Any()).
					Return([]*jetstream.ObjectInfo{
						{
							ObjectMeta: jetstream.ObjectMeta{
								Name: "active.conf",
								Headers: nats.Header{
									"Osapi-Content-Type": []string{"raw"},
								},
							},
							Size:   100,
							Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
						},
						{
							ObjectMeta: jetstream.ObjectMeta{Name: "deleted.conf"},
							Size:       200,
							Digest:     "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
							Deleted:    true,
						},
					}, nil)
			},
			validateFunc: func(resp gen.GetFilesResponseObject) {
				r, ok := resp.(gen.GetFiles200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Len(r.Files, 1)
				s.Equal("active.conf", r.Files[0].Name)
			},
		},
		{
			name: "object store error",
			setupMock: func() {
				s.mockObjStore.EXPECT().
					List(gomock.Any()).
					Return(nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetFilesResponseObject) {
				_, ok := resp.(gen.GetFiles500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetFiles(s.ctx, gen.GetFilesRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileListPublicTestSuite) TestGetFilesHTTP() {
	tests := []struct {
		name         string
		setupMock    func() *mocks.MockObjectStoreManager
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when list Ok",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					List(gomock.Any()).
					Return([]*jetstream.ObjectInfo{
						{
							ObjectMeta: jetstream.ObjectMeta{
								Name: "nginx.conf",
								Headers: nats.Header{
									"Osapi-Content-Type": []string{"raw"},
								},
							},
							Size:   1024,
							Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
						},
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"files"`)
				s.Contains(rec.Body.String(), `"nginx.conf"`)
				s.Contains(rec.Body.String(), `"total":1`)
				s.Contains(rec.Body.String(), `"content_type":"raw"`)
			},
		},
		{
			name: "when object store error",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					List(gomock.Any()).
					Return(nil, assert.AnError)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "failed to list files")
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

			req := httptest.NewRequest(http.MethodGet, "/api/file", nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacListTestSigningKey = "test-signing-key-for-file-list-rbac"

func (s *FileListPublicTestSuite) TestGetFilesRBACHTTP() {
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
					rbacListTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"node:read"},
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
			name: "when valid token with file:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacListTestSigningKey,
					[]string{"admin"},
					"test-user",
					[]string{"file:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					List(gomock.Any()).
					Return([]*jetstream.ObjectInfo{}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"files"`)
				s.Contains(rec.Body.String(), `"total":0`)
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
							SigningKey: rbacListTestSigningKey,
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

			req := httptest.NewRequest(http.MethodGet, "/api/file", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileListPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileListPublicTestSuite))
}
