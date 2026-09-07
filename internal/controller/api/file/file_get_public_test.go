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

type FileGetPublicTestSuite struct {
	suite.Suite

	mockCtrl     *gomock.Controller
	mockObjStore *mocks.MockObjectStoreManager
	handler      *apifile.File
	ctx          context.Context
	appConfig    config.Config
	logger       *slog.Logger
}

func (s *FileGetPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockObjStore = mocks.NewMockObjectStoreManager(s.mockCtrl)
	s.handler = apifile.New(slog.Default(), s.mockObjStore, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileGetPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *FileGetPublicTestSuite) TestGetFileByName() {
	tests := []struct {
		name         string
		request      gen.GetFileByNameRequestObject
		setupMock    func()
		validateFunc func(resp gen.GetFileByNameResponseObject)
	}{
		{
			name:    "success",
			request: gen.GetFileByNameRequestObject{Name: "nginx.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{
							Name: "nginx.conf",
							Headers: nats.Header{
								"Osapi-Content-Type": []string{"raw"},
							},
						},
						Size:   1024,
						Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
					}, nil)
			},
			validateFunc: func(resp gen.GetFileByNameResponseObject) {
				r, ok := resp.(gen.GetFileByName200JSONResponse)
				s.True(ok)
				s.Equal("nginx.conf", r.Name)
				s.Equal(
					"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					r.Sha256,
				)
				s.Equal(1024, r.Size)
				s.Equal("raw", r.ContentType)
			},
		},
		{
			name:    "validation error name too long",
			request: gen.GetFileByNameRequestObject{Name: strings.Repeat("a", 256)},
			setupMock: func() {
				// No mock calls expected; validation rejects before reaching obj store.
			},
			validateFunc: func(resp gen.GetFileByNameResponseObject) {
				_, ok := resp.(gen.GetFileByName400JSONResponse)
				s.True(ok)
			},
		},
		{
			name:    "not found",
			request: gen.GetFileByNameRequestObject{Name: "missing.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "missing.conf").
					Return(nil, jetstream.ErrObjectNotFound)
			},
			validateFunc: func(resp gen.GetFileByNameResponseObject) {
				r, ok := resp.(gen.GetFileByName404JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "file not found")
			},
		},
		{
			name:    "object store error",
			request: gen.GetFileByNameRequestObject{Name: "nginx.conf"},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)
			},
			validateFunc: func(resp gen.GetFileByNameResponseObject) {
				_, ok := resp.(gen.GetFileByName500JSONResponse)
				s.True(ok)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.GetFileByName(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileGetPublicTestSuite) TestGetFileByNameValidationHTTP() {
	tests := []struct {
		name         string
		path         string
		setupMock    func() *mocks.MockObjectStoreManager
		wantCode     int
		wantContains []string
	}{
		{
			name: "when name too long returns 400",
			path: "/api/file/" + strings.Repeat("a", 256),
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			wantCode:     http.StatusBadRequest,
			wantContains: []string{`"error"`},
		},
		{
			name: "when get Ok",
			path: "/api/file/nginx.conf",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{
							Name: "nginx.conf",
							Headers: nats.Header{
								"Osapi-Content-Type": []string{"raw"},
							},
						},
						Size:   1024,
						Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
					}, nil)
				return mock
			},
			wantCode: http.StatusOK,
			wantContains: []string{
				`"name":"nginx.conf"`,
				`"sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`,
				`"size":1024`,
				`"content_type":"raw"`,
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
			wantCode:     http.StatusNotFound,
			wantContains: []string{"file not found"},
		},
		{
			name: "when object store error",
			path: "/api/file/nginx.conf",
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)
				return mock
			},
			wantCode:     http.StatusInternalServerError,
			wantContains: []string{"failed to get file info"},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			objMock := tc.setupMock()

			fileHandler := apifile.New(s.logger, objMock, nil)
			strictHandler := gen.NewStrictHandler(fileHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			s.Equal(tc.wantCode, rec.Code)
			for _, str := range tc.wantContains {
				s.Contains(rec.Body.String(), str)
			}
		})
	}
}

const rbacGetTestSigningKey = "test-signing-key-for-file-get-rbac"

func (s *FileGetPublicTestSuite) TestGetFileByNameRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupMock    func() *mocks.MockObjectStoreManager
		wantCode     int
		wantContains []string
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			wantCode:     http.StatusUnauthorized,
			wantContains: []string{"Bearer token required"},
		},
		{
			name: "when insufficient permissions returns 403",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacGetTestSigningKey,
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
			wantCode:     http.StatusForbidden,
			wantContains: []string{"Insufficient permissions"},
		},
		{
			name: "when valid token with file:read returns 200",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacGetTestSigningKey,
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
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{
							Name: "nginx.conf",
							Headers: nats.Header{
								"Osapi-Content-Type": []string{"raw"},
							},
						},
						Size:   1024,
						Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
					}, nil)
				return mock
			},
			wantCode:     http.StatusOK,
			wantContains: []string{`"name":"nginx.conf"`, `"sha256"`},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			objMock := tc.setupMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacGetTestSigningKey,
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

			req := httptest.NewRequest(http.MethodGet, "/api/file/nginx.conf", nil)
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

func TestFileGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileGetPublicTestSuite))
}
