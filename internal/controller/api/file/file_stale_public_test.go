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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
	"github.com/osapi-io/osapi/internal/job"
	jobMocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type FileStalePublicTestSuite struct {
	suite.Suite

	mockCtrl     *gomock.Controller
	mockObjStore *mocks.MockObjectStoreManager
	mockStateKV  *mocks.MockStateKeyValue
	handler      *apifile.File
	ctx          context.Context
	appConfig    config.Config
	logger       *slog.Logger
}

func (s *FileStalePublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockObjStore = mocks.NewMockObjectStoreManager(s.mockCtrl)
	s.mockStateKV = mocks.NewMockStateKeyValue(s.mockCtrl)
	s.handler = apifile.New(slog.Default(), s.mockObjStore, s.mockStateKV)
	s.ctx = context.Background()
	s.appConfig = config.Config{}
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *FileStalePublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *FileStalePublicTestSuite) TestGetFileStale() {
	contentOld := []byte("old content")
	contentNew := []byte("new content")
	oldSHA := sha256Hex(contentOld)
	newSHA := sha256Hex(contentNew)

	// hostname "web-01" + "." + sha256 of path = key
	pathHash := sha256Hex([]byte("/etc/systemd/system/osapi-hello.service"))
	stateKey := "web-01." + pathHash

	tests := []struct {
		name         string
		setupMock    func()
		setupHandler func() *apifile.File
		validateFunc func(resp gen.GetFileStaleResponseObject)
	}{
		{
			name: "when state KV is nil returns 500",
			setupMock: func() {
				// No mocks needed — stateKV is nil.
			},
			setupHandler: func() *apifile.File {
				return apifile.New(slog.Default(), s.mockObjStore, nil)
			},
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale500JSONResponse)
				s.True(ok)
				s.Require().NotNil(r.Error)
				s.Contains(*r.Error, "file state KV not available")
			},
		},
		{
			name: "when no keys found returns empty list",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Stale)
			},
		},
		{
			name: "when Keys errors returns 500",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return(nil, assert.AnError)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				_, ok := resp.(gen.GetFileStale500JSONResponse)
				s.True(ok)
			},
		},
		{
			name: "when SHA mismatch returns stale entry",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{stateKey}, nil)

				state := job.FileState{
					ObjectName: "hello-echo",
					Path:       "/etc/systemd/system/osapi-hello.service",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateJSON, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateJSON)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), stateKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "hello-echo").
					Return(contentNew, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Require().Len(r.Stale, 1)
				s.Equal("hello-echo", r.Stale[0].ObjectName)
				s.Equal("web-01", r.Stale[0].Hostname)
				s.Equal("service", r.Stale[0].Provider)
				s.Equal("/etc/systemd/system/osapi-hello.service", r.Stale[0].Path)
				s.Equal(oldSHA, r.Stale[0].DeployedSha)
				s.Equal(newSHA, r.Stale[0].CurrentSha)
				s.Equal("2026-04-01T18:00:00Z", r.Stale[0].DeployedAt)
			},
		},
		{
			name: "when SHA matches returns empty list",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{stateKey}, nil)

				state := job.FileState{
					ObjectName: "hello-echo",
					Path:       "/etc/systemd/system/osapi-hello.service",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateJSON, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateJSON)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), stateKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "hello-echo").
					Return(contentOld, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Stale)
			},
		},
		{
			name: "when entry is undeployed skips it",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{stateKey}, nil)

				state := job.FileState{
					ObjectName:   "hello-echo",
					Path:         "/etc/systemd/system/osapi-hello.service",
					SHA256:       oldSHA,
					DeployedAt:   "2026-04-01T18:00:00Z",
					UndeployedAt: "2026-04-01T19:00:00Z",
				}
				stateJSON, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateJSON)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), stateKey).
					Return(entry, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Stale)
			},
		},
		{
			name: "when object deleted returns stale with empty current_sha",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{stateKey}, nil)

				state := job.FileState{
					ObjectName: "hello-echo",
					Path:       "/etc/systemd/system/osapi-hello.service",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateJSON, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateJSON)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), stateKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "hello-echo").
					Return(nil, assert.AnError)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Require().Len(r.Stale, 1)
				s.Equal("service", r.Stale[0].Provider)
				s.Equal("", r.Stale[0].CurrentSha)
				s.Equal(oldSHA, r.Stale[0].DeployedSha)
			},
		},
		{
			name: "when Get entry fails skips and continues",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{stateKey}, nil)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), stateKey).
					Return(nil, assert.AnError)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Stale)
			},
		},
		{
			name: "when unmarshal fails skips and continues",
			setupMock: func() {
				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{stateKey}, nil)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return([]byte("not-json"))

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), stateKey).
					Return(entry, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(0, r.Total)
				s.Empty(r.Stale)
			},
		},
		{
			name: "when certificate path returns certificate provider",
			setupMock: func() {
				certPathHash := sha256Hex(
					[]byte("/usr/local/share/ca-certificates/osapi-my-ca.crt"),
				)
				certKey := "web-02." + certPathHash

				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{certKey}, nil)

				state := job.FileState{
					ObjectName: "my-ca",
					Path:       "/usr/local/share/ca-certificates/osapi-my-ca.crt",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateBytes)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), certKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "my-ca").
					Return(contentNew, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Require().Len(r.Stale, 1)
				s.Equal("certificate", r.Stale[0].Provider)
			},
		},
		{
			name: "when cron path returns cron provider",
			setupMock: func() {
				cronPathHash := sha256Hex([]byte("/etc/cron.d/osapi-backup"))
				cronKey := "web-03." + cronPathHash

				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{cronKey}, nil)

				state := job.FileState{
					ObjectName: "backup-cron",
					Path:       "/etc/cron.d/osapi-backup",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateBytes)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), cronKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "backup-cron").
					Return(contentNew, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Require().Len(r.Stale, 1)
				s.Equal("cron", r.Stale[0].Provider)
			},
		},
		{
			name: "when unknown path returns file provider",
			setupMock: func() {
				filePathHash := sha256Hex([]byte("/tmp/app.conf"))
				fileKey := "web-04." + filePathHash

				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{fileKey}, nil)

				state := job.FileState{
					ObjectName: "app-conf",
					Path:       "/tmp/app.conf",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateBytes)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), fileKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "app-conf").
					Return(contentNew, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Require().Len(r.Stale, 1)
				s.Equal("file", r.Stale[0].Provider)
			},
		},
		{
			name: "when key is too short extractHostname returns full key",
			setupMock: func() {
				shortKey := "short"

				s.mockStateKV.EXPECT().
					Keys(gomock.Any()).
					Return([]string{shortKey}, nil)

				state := job.FileState{
					ObjectName: "test",
					Path:       "/tmp/test",
					SHA256:     oldSHA,
					DeployedAt: "2026-04-01T18:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				entry := jobMocks.NewMockKeyValueEntry(s.mockCtrl)
				entry.EXPECT().Value().Return(stateBytes)

				s.mockStateKV.EXPECT().
					Get(gomock.Any(), shortKey).
					Return(entry, nil)

				s.mockObjStore.EXPECT().
					GetBytes(gomock.Any(), "test").
					Return(contentNew, nil)
			},
			setupHandler: func() *apifile.File { return s.handler },
			validateFunc: func(resp gen.GetFileStaleResponseObject) {
				r, ok := resp.(gen.GetFileStale200JSONResponse)
				s.True(ok)
				s.Equal(1, r.Total)
				s.Require().Len(r.Stale, 1)
				s.Equal("short", r.Stale[0].Hostname)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.setupMock()
			handler := tt.setupHandler()

			resp, err := handler.GetFileStale(s.ctx, gen.GetFileStaleRequestObject{})
			s.NoError(err)
			tt.validateFunc(resp)
		})
	}
}

func (s *FileStalePublicTestSuite) TestGetFileStaleHTTP() {
	tests := []struct {
		name         string
		setupMock    func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when empty state KV returns 200 with empty list",
			setupMock: func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue) {
				objMock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				kvMock := mocks.NewMockStateKeyValue(s.mockCtrl)
				kvMock.EXPECT().
					Keys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
				return objMock, kvMock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"stale":[]`)
				s.Contains(rec.Body.String(), `"total":0`)
			},
		},
		{
			name: "when state KV nil returns 500",
			setupMock: func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue) {
				objMock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				return objMock, nil
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusInternalServerError, rec.Code)
				s.Contains(rec.Body.String(), "file state KV not available")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			objMock, kvMock := tc.setupMock()

			var stateKV apifile.StateKeyValue
			if kvMock != nil {
				stateKV = kvMock
			}

			fileHandler := apifile.New(s.logger, objMock, stateKV)
			strictHandler := gen.NewStrictHandler(fileHandler, nil)

			a := api.New(s.appConfig, s.logger)
			gen.RegisterHandlers(a.Echo, strictHandler)

			req := httptest.NewRequest(http.MethodGet, "/api/file/stale", nil)
			rec := httptest.NewRecorder()

			a.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

const rbacStaleTestSigningKey = "test-signing-key-for-file-stale-rbac"

func (s *FileStalePublicTestSuite) TestGetFileStaleRBACHTTP() {
	tokenManager := authtoken.New(s.logger)

	tests := []struct {
		name         string
		setupAuth    func(req *http.Request)
		setupMock    func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "when no token returns 401",
			setupAuth: func(_ *http.Request) {
				// No auth header set.
			},
			setupMock: func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue) {
				return mocks.NewMockObjectStoreManager(s.mockCtrl),
					mocks.NewMockStateKeyValue(s.mockCtrl)
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
					rbacStaleTestSigningKey,
					[]string{"read"},
					"test-user",
					[]string{"node:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMock: func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue) {
				return mocks.NewMockObjectStoreManager(s.mockCtrl),
					mocks.NewMockStateKeyValue(s.mockCtrl)
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
					rbacStaleTestSigningKey,
					[]string{"admin"},
					"test-user",
					[]string{"file:read"},
				)
				s.Require().NoError(err)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			},
			setupMock: func() (*mocks.MockObjectStoreManager, *mocks.MockStateKeyValue) {
				objMock := mocks.NewMockObjectStoreManager(s.mockCtrl)
				kvMock := mocks.NewMockStateKeyValue(s.mockCtrl)
				kvMock.EXPECT().
					Keys(gomock.Any()).
					Return(nil, jetstream.ErrNoKeysFound)
				return objMock, kvMock
			},
			validateFunc: func(rec *httptest.ResponseRecorder) {
				s.Equal(http.StatusOK, rec.Code)
				s.Contains(rec.Body.String(), `"stale"`)
				s.Contains(rec.Body.String(), `"total":0`)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			objMock, kvMock := tc.setupMock()

			appConfig := config.Config{
				Controller: config.Controller{
					API: config.APIServer{
						Security: config.ServerSecurity{
							SigningKey: rbacStaleTestSigningKey,
						},
					},
				},
			}

			server := api.New(appConfig, s.logger)
			handlers := apifile.Handler(
				s.logger,
				objMock,
				kvMock,
				appConfig.Controller.API.Security.SigningKey,
				nil,
			)
			server.RegisterHandlers(handlers)

			req := httptest.NewRequest(http.MethodGet, "/api/file/stale", nil)
			tc.setupAuth(req)
			rec := httptest.NewRecorder()

			server.Echo.ServeHTTP(rec, req)

			tc.validateFunc(rec)
		})
	}
}

func TestFileStalePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FileStalePublicTestSuite))
}

// sha256Hex returns the hex-encoded SHA-256 digest of data.
func sha256Hex(
	data []byte,
) string {
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}
