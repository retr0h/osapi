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
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"k8s.io/utils/ptr"

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

type FileUploadPublicTestSuite struct {
	suite.Suite

	mockCtrl     *gomock.Controller
	mockObjStore *mocks.MockObjectStoreManager
	handler      *apifile.File
	ctx          context.Context
	appConfig    config.Config
	logger       *slog.Logger
}

func (s *FileUploadPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockObjStore = mocks.NewMockObjectStoreManager(s.mockCtrl)
	s.handler = apifile.New(slog.Default(), s.mockObjStore, nil)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileUploadPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func makeMultipartReader(
	name string,
	contentType string,
	data []byte,
) *multipart.Reader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if name != "" {
		_ = writer.WriteField("name", name)
	}
	if contentType != "" {
		_ = writer.WriteField("content_type", contentType)
	}
	if data != nil {
		part, _ := writer.CreateFormFile("file", "upload")
		_, _ = part.Write(data)
	}

	_ = writer.Close()

	return multipart.NewReader(body, writer.Boundary())
}

// makeBrokenMultipartReader returns a multipart.Reader that delivers a valid
// "name" part first, then produces a non-EOF error on the next NextPart() call.
// This triggers the non-EOF error path (lines 126-129) in parseMultipart.
func makeBrokenMultipartReader() *multipart.Reader {
	boundary := "testboundary"

	// Build a valid first part (name field) followed by a corrupt second part
	// header that will cause NextPart() to return a non-EOF error.
	raw := "--" + boundary + "\r\n" +
		"Content-Disposition: form-data; name=\"name\"\r\n\r\n" +
		"test.conf\r\n" +
		"--" + boundary + "\r\n" +
		"Malformed-No-Blank-Line\r\n"

	return multipart.NewReader(bytes.NewReader([]byte(raw)), boundary)
}

// makeMultipartBody builds a multipart body and returns the body and
// content-type header for HTTP tests.
func makeMultipartBody(
	name string,
	contentType string,
	data []byte,
) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if name != "" {
		_ = writer.WriteField("name", name)
	}
	if contentType != "" {
		_ = writer.WriteField("content_type", contentType)
	}
	if data != nil {
		part, _ := writer.CreateFormFile("file", "upload")
		_, _ = part.Write(data)
	}

	_ = writer.Close()

	return body, writer.FormDataContentType()
}

func (s *FileUploadPublicTestSuite) TestPostFile() {
	fileContent := []byte("server { listen 80; }")

	tests := []struct {
		name         string
		request      gen.PostFileRequestObject
		setupMock    func()
		validateFunc func(resp gen.PostFileResponseObject)
	}{
		{
			name: "when new file",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("nginx.conf", "raw", fileContent),
			},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)

				s.mockObjStore.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						meta jetstream.ObjectMeta,
						_ io.Reader,
					) (*jetstream.ObjectInfo, error) {
						s.Equal("nginx.conf", meta.Name)
						s.Equal("raw", meta.Headers.Get("Osapi-Content-Type"))
						return &jetstream.ObjectInfo{
							ObjectMeta: meta,
							Size:       uint64(len(fileContent)),
						}, nil
					})
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile201JSONResponse)
				s.True(ok)
				s.Equal("nginx.conf", r.Name)
				s.Equal(len(fileContent), r.Size)
				s.NotEmpty(r.Sha256)
				s.True(r.Changed)
				s.Equal("raw", r.ContentType)
			},
		},
		{
			name: "when template file",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("tmpl.conf", "template", fileContent),
			},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "tmpl.conf").
					Return(nil, assert.AnError)

				s.mockObjStore.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(
						_ context.Context,
						meta jetstream.ObjectMeta,
						_ io.Reader,
					) (*jetstream.ObjectInfo, error) {
						s.Equal("template", meta.Headers.Get("Osapi-Content-Type"))
						return &jetstream.ObjectInfo{
							ObjectMeta: meta,
							Size:       uint64(len(fileContent)),
						}, nil
					})
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile201JSONResponse)
				s.True(ok)
				s.Equal("tmpl.conf", r.Name)
				s.True(r.Changed)
				s.Equal("template", r.ContentType)
			},
		},
		{
			name: "when content_type defaults to raw",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("f.txt", "", fileContent),
			},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "f.txt").
					Return(nil, assert.AnError)

				s.mockObjStore.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "f.txt"},
						Size:       uint64(len(fileContent)),
					}, nil)
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile201JSONResponse)
				s.True(ok)
				s.Equal("raw", r.ContentType)
			},
		},
		{
			name: "when unchanged content",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("nginx.conf", "raw", fileContent),
			},
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
						Size:   uint64(len(fileContent)),
						Digest: "SHA-256=udwh0KiTQXw0wAbA6MMre9G3vJSOnF4MeW7eBweZr0g=",
					}, nil)
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile201JSONResponse)
				s.True(ok)
				s.Equal("nginx.conf", r.Name)
				s.Equal(len(fileContent), r.Size)
				s.False(r.Changed)
				s.Equal("raw", r.ContentType)
			},
		},
		{
			name: "when different content without force returns 409",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("nginx.conf", "raw", fileContent),
			},
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
						Size:   100,
						Digest: "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
					}, nil)
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile409JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "already exists with different content")
			},
		},
		{
			name: "when force upload bypasses digest check",
			request: gen.PostFileRequestObject{
				Params: gen.PostFileParams{Force: ptr.To(true)},
				Body:   makeMultipartReader("nginx.conf", "raw", fileContent),
			},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       uint64(len(fileContent)),
					}, nil)
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile201JSONResponse)
				s.True(ok)
				s.Equal("nginx.conf", r.Name)
				s.True(r.Changed)
			},
		},
		{
			name: "when force upload same content still writes",
			request: gen.PostFileRequestObject{
				Params: gen.PostFileParams{Force: ptr.To(true)},
				Body:   makeMultipartReader("nginx.conf", "raw", fileContent),
			},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       uint64(len(fileContent)),
					}, nil)
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile201JSONResponse)
				s.True(ok)
				s.True(r.Changed)
			},
		},
		{
			name: "when multipart read error returns 400",
			request: gen.PostFileRequestObject{
				Body: makeBrokenMultipartReader(),
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "failed to read multipart")
			},
		},
		{
			name: "validation error empty name",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("", "raw", fileContent),
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "name is required")
			},
		},
		{
			name: "validation error name too long",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader(strings.Repeat("a", 256), "raw", fileContent),
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "name is required and must be 1-255 characters")
			},
		},
		{
			name: "validation error empty file",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("test.txt", "raw", nil),
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "file is required")
			},
		},
		{
			name: "validation error invalid content_type",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("test.txt", "invalid", fileContent),
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile400JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "content_type must be raw or template")
			},
		},
		{
			name: "object store error",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("nginx.conf", "raw", fileContent),
			},
			setupMock: func() {
				s.mockObjStore.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)

				s.mockObjStore.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
			validateFunc: func(resp gen.PostFileResponseObject) {
				_, ok := resp.(gen.PostFile500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when uploading system file returns 403",
			request: gen.PostFileRequestObject{
				Body: makeMultipartReader("osapi/cron-template.tmpl", "raw", fileContent),
			},
			setupMock: func() {},
			validateFunc: func(resp gen.PostFileResponseObject) {
				r, ok := resp.(gen.PostFile403JSONResponse)
				s.True(ok)
				s.Contains(*r.Error, "cannot overwrite protected osapi file")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()

			resp, err := s.handler.PostFile(s.ctx, tt.request)
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileUploadPublicTestSuite) TestPostFileValidationHTTP() {
	fileContent := []byte("server { listen 80; }")

	tests := []struct {
		name         string
		path         string
		buildBody    func() (*bytes.Buffer, string)
		setupMock    func() *mocks.MockObjectStoreManager
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when upload Ok",
			buildBody: func() (*bytes.Buffer, string) {
				return makeMultipartBody("nginx.conf", "raw", fileContent)
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)
				mock.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       uint64(len(fileContent)),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusCreated, rec.Code)
				s.Contains(rec.Body.String(), "name")
				s.Contains(rec.Body.String(), "nginx.conf")
				s.Contains(rec.Body.String(), "sha256")
				s.Contains(rec.Body.String(), "size")
				s.Contains(rec.Body.String(), "changed")
				s.Contains(rec.Body.String(), "content_type")
				s.Contains(rec.Body.String(), "raw")
			},
		},
		{
			name: "when validation error",
			buildBody: func() (*bytes.Buffer, string) {
				return makeMultipartBody("", "raw", fileContent)
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "name is required")
			},
		},
		{
			name: "when different content without force returns 409",
			buildBody: func() (*bytes.Buffer, string) {
				return makeMultipartBody("nginx.conf", "raw", fileContent)
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       100,
						Digest:     "SHA-256=47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusConflict, rec.Code)
				s.Contains(rec.Body.String(), "already exists with different content")
			},
		},
		{
			name: "when force upload bypasses digest check",
			path: "/api/file?force=true",
			buildBody: func() (*bytes.Buffer, string) {
				return makeMultipartBody("nginx.conf", "raw", fileContent)
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       uint64(len(fileContent)),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusCreated, rec.Code)
				s.Contains(rec.Body.String(), "changed")
			},
		},
		{
			name: "when invalid force param returns 400",
			path: "/api/file?force=notabool",
			buildBody: func() (*bytes.Buffer, string) {
				return makeMultipartBody("nginx.conf", "raw", fileContent)
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				return mocks.NewMockObjectStoreManager(s.mockCtrl)
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusBadRequest, rec.Code)
				s.Contains(rec.Body.String(), "Invalid format for parameter force")
			},
		},
		{
			name: "when object store error",
			buildBody: func() (*bytes.Buffer, string) {
				return makeMultipartBody("nginx.conf", "raw", fileContent)
			},
			setupMock: func() *mocks.MockObjectStoreManager {
				mock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				mock.EXPECT().
					GetInfo(gomock.Any(), "nginx.conf").
					Return(nil, assert.AnError)
				mock.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "failed to store file")
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

			body, ct := tc.buildBody()

			path := tc.path
			if path == "" {
				path = "/api/file"
			}

			req := httptest.NewRequest(
				http.MethodPost,
				path,
				body,
			)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacUploadTestSigningKey = "test-signing-key-for-file-upload-rbac"

func (s *FileUploadPublicTestSuite) TestPostFileRBACHTTP() {
	tokenManager := authtoken.New(s.logger)
	fileContent := []byte("server { listen 80; }")

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
					rbacUploadTestSigningKey,
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
			name: "when valid token with file:write returns 201",
			setupAuth: func(req *http.Request) {
				token, err := tokenManager.Generate(
					rbacUploadTestSigningKey,
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
					Return(nil, assert.AnError)
				mock.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{Name: "nginx.conf"},
						Size:       uint64(len(fileContent)),
					}, nil)
				return mock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusCreated, rec.Code)
				s.Contains(rec.Body.String(), "name")
				s.Contains(rec.Body.String(), "nginx.conf")
				s.Contains(rec.Body.String(), "sha256")
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
							SigningKey: rbacUploadTestSigningKey,
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

			body, ct := makeMultipartBody("nginx.conf", "raw", fileContent)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/file",
				body,
			)
			req.Header.Set("Content-Type", ct)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileUploadPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileUploadPublicTestSuite))
}
