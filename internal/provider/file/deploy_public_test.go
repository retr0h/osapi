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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/file"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
)

type DeployPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
	ctx    context.Context
}

func (suite *DeployPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.ctx = context.Background()
}

func (suite *DeployPublicTestSuite) TearDownSubTest() {
	file.ResetMarshalJSON()
}

func (suite *DeployPublicTestSuite) TearDownTest() {}

func (suite *DeployPublicTestSuite) TestDeploy() {
	fileContent := []byte("server { listen 80; }")
	existingSHA := computeTestSHA256(fileContent)
	differentContent := []byte("server { listen 443; }")
	differentSHA := computeTestSHA256(differentContent)

	tests := []struct {
		name         string
		setupFunc    func()
		setupMock    func(*gomock.Controller, *filemocks.MockObjectStore, *jobmocks.MockKeyValue, *avfs.VFS)
		req          file.DeployRequest
		want         *file.DeployResult
		wantErr      bool
		wantErrMsg   string
		validateFunc func(avfs.VFS)
	}{
		{
			name: "when marshal state fails returns error",
			setupFunc: func() {
				file.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, fmt.Errorf("marshal failure")
				})
			},
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return([]byte("server { listen 80; }"), nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
			},
			wantErr:    true,
			wantErrMsg: "failed to marshal file state",
		},
		{
			name: "when deploy succeeds (new file)",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				Mode:        "0644",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
			validateFunc: func(appFs avfs.VFS) {
				data, err := appFs.ReadFile("/etc/nginx/nginx.conf")
				suite.Require().NoError(err)
				suite.Equal(fileContent, data)
			},
		},
		{
			name: "when deploy succeeds (changed content)",
			setupMock: func(
				ctrl *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				existingState := job.FileState{
					SHA256: differentSHA,
					Path:   "/etc/nginx/nginx.conf",
				}
				stateBytes, _ := json.Marshal(existingState)

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				Mode:        "0644",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
			validateFunc: func(appFs avfs.VFS) {
				data, err := appFs.ReadFile("/etc/nginx/nginx.conf")
				suite.Require().NoError(err)
				suite.Equal(fileContent, data)
			},
		},
		{
			name: "when deploy skips (unchanged)",
			setupMock: func(
				ctrl *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				appFs *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				_ = (*appFs).MkdirAll("/etc/nginx", 0o755)
				_ = (*appFs).WriteFile("/etc/nginx/nginx.conf", fileContent, 0o644)

				existingState := job.FileState{
					SHA256: existingSHA,
					Path:   "/etc/nginx/nginx.conf",
				}
				stateBytes, _ := json.Marshal(existingState)

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: false,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
		},
		{
			name: "when file is deleted but state exists redeploys",
			setupMock: func(
				ctrl *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				existingState := job.FileState{
					SHA256: existingSHA,
					Path:   "/etc/nginx/nginx.conf",
				}
				stateBytes, _ := json.Marshal(existingState)

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				Mode:        "0644",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
			validateFunc: func(appFs avfs.VFS) {
				data, err := appFs.ReadFile("/etc/nginx/nginx.conf")
				suite.Require().NoError(err)
				suite.Equal(fileContent, data)
			},
		},
		{
			name: "when Object Store get fails",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				_ *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
			req: file.DeployRequest{
				ObjectName:  "missing.conf",
				Path:        "/etc/missing.conf",
				ContentType: "raw",
			},
			wantErr:    true,
			wantErrMsg: "failed to get object",
		},
		{
			name: "when content type is template",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return([]byte("server {{ .Vars.host }}"), nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "template",
				Vars:        map[string]any{"host": "10.0.0.1"},
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  computeTestSHA256([]byte("server 10.0.0.1")),
				Path:    "/etc/nginx/nginx.conf",
			},
			validateFunc: func(appFs avfs.VFS) {
				data, err := appFs.ReadFile("/etc/nginx/nginx.conf")
				suite.Require().NoError(err)
				suite.Equal("server 10.0.0.1", string(data))
			},
		},
		{
			name: "when content type is empty resolves template from object header",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return([]byte("server {{ .Vars.host }}"), nil)

				mockObj.EXPECT().
					GetInfo(gomock.Any(), gomock.Any()).
					Return(&jetstream.ObjectInfo{
						ObjectMeta: jetstream.ObjectMeta{
							Name: "nginx.conf",
							Headers: nats.Header{
								"Osapi-Content-Type": []string{"template"},
							},
						},
					}, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName: "nginx.conf",
				Path:       "/etc/nginx/nginx.conf",
				Vars:       map[string]any{"host": "10.0.0.1"},
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  computeTestSHA256([]byte("server 10.0.0.1")),
				Path:    "/etc/nginx/nginx.conf",
			},
			validateFunc: func(appFs avfs.VFS) {
				data, err := appFs.ReadFile("/etc/nginx/nginx.conf")
				suite.Require().NoError(err)
				suite.Equal("server 10.0.0.1", string(data))
			},
		},
		{
			name: "when content type is empty and GetInfo fails defaults to raw",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockObj.EXPECT().
					GetInfo(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("info error"))

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName: "nginx.conf",
				Path:       "/etc/nginx/nginx.conf",
				Mode:       "0644",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
		},
		{
			name: "when file write fails",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				appFs *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				// Use failfs to block OpenFile (which WriteFile calls internally).
				vfs := failfs.New(memfs.New())
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnOpenFile {
						return errors.New("write failed")
					}

					return nil
				})
				*appFs = vfs
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
			},
			wantErr:    true,
			wantErrMsg: "failed to write file",
		},
		{
			name: "when mkdir fails",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				appFs *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				vfs := failfs.New(memfs.New())
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnMkdirAll {
						return errors.New("mkdir failed")
					}

					return nil
				})
				*appFs = vfs
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
			},
			wantErr:    true,
			wantErrMsg: "failed to create directory",
		},
		{
			name: "when state KV put fails",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), assert.AnError)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
			},
			wantErr:    true,
			wantErrMsg: "failed to update file state",
		},
		{
			name: "when state KV has corrupt data proceeds to deploy",
			setupMock: func(
				ctrl *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
		},
		{
			name: "when mode is invalid defaults to 0644",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "nginx.conf",
				Path:        "/etc/nginx/nginx.conf",
				Mode:        "not-octal",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/etc/nginx/nginx.conf",
			},
			validateFunc: func(appFs avfs.VFS) {
				info, err := appFs.Stat("/etc/nginx/nginx.conf")
				suite.Require().NoError(err)
				suite.Equal(os.FileMode(0o644), info.Mode())
			},
		},
		{
			name: "when mode is set",
			setupMock: func(
				_ *gomock.Controller,
				mockObj *filemocks.MockObjectStore,
				mockKV *jobmocks.MockKeyValue,
				_ *avfs.VFS,
			) {
				mockObj.EXPECT().
					GetBytes(gomock.Any(), gomock.Any()).
					Return(fileContent, nil)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)

				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.DeployRequest{
				ObjectName:  "script.sh",
				Path:        "/usr/local/bin/script.sh",
				Mode:        "0755",
				ContentType: "raw",
			},
			want: &file.DeployResult{
				Changed: true,
				SHA256:  existingSHA,
				Path:    "/usr/local/bin/script.sh",
			},
			validateFunc: func(appFs avfs.VFS) {
				info, err := appFs.Stat("/usr/local/bin/script.sh")
				suite.Require().NoError(err)
				suite.Equal(os.FileMode(0o755), info.Mode())
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			ctrl := gomock.NewController(suite.T())
			defer ctrl.Finish()

			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			var appFs avfs.VFS = memfs.New()
			mockKV := jobmocks.NewMockKeyValue(ctrl)
			mockObj := filemocks.NewMockObjectStore(ctrl)

			if tc.setupMock != nil {
				tc.setupMock(ctrl, mockObj, mockKV, &appFs)
			}

			provider := file.New(
				suite.logger,
				appFs,
				mockObj,
				mockKV,
				"test-host",
			)

			got, err := provider.Deploy(suite.ctx, tc.req)

			if tc.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tc.wantErrMsg)
				suite.Nil(got)
			} else {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(tc.want, got)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(appFs)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDeployPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DeployPublicTestSuite))
}
