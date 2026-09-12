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
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/file"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
)

type UndeployPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
	ctx    context.Context
}

func (suite *UndeployPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.ctx = context.Background()
}

func (suite *UndeployPublicTestSuite) TearDownSubTest() {
	file.ResetMarshalJSON()
}

func (suite *UndeployPublicTestSuite) TearDownTest() {}

func (suite *UndeployPublicTestSuite) TestUndeploy() {
	tests := []struct {
		name         string
		setupFunc    func()
		setupMock    func(*gomock.Controller, *jobmocks.MockKeyValue, avfs.VFS)
		req          file.UndeployRequest
		want         *file.UndeployResult
		wantErr      bool
		wantErrMsg   string
		useFailFs    bool
		validateFunc func(avfs.VFS)
	}{
		{
			name: "when marshal state fails still returns changed",
			setupFunc: func() {
				file.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, fmt.Errorf("marshal failure")
				})
			},
			setupMock: func(
				ctrl *gomock.Controller,
				mockKV *jobmocks.MockKeyValue,
				appFs avfs.VFS,
			) {
				_ = appFs.MkdirAll("/etc/cron.d", 0o755)
				_ = appFs.WriteFile("/etc/cron.d/backup", []byte("content"), 0o644)

				stateJSON, _ := json.Marshal(job.FileState{
					ObjectName: "backup-script",
					Path:       "/etc/cron.d/backup",
					SHA256:     "abc123",
					DeployedAt: "2026-03-22T00:00:00Z",
				})

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return(stateJSON)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.UndeployRequest{Path: "/etc/cron.d/backup"},
			want: &file.UndeployResult{
				Changed: true,
				Path:    "/etc/cron.d/backup",
			},
		},
		{
			name: "when file exists on disk with state",
			setupMock: func(
				ctrl *gomock.Controller,
				mockKV *jobmocks.MockKeyValue,
				appFs avfs.VFS,
			) {
				_ = appFs.MkdirAll("/etc/cron.d", 0o755)
				_ = appFs.WriteFile("/etc/cron.d/backup", []byte("content"), 0o644)

				stateJSON, _ := json.Marshal(job.FileState{
					ObjectName: "backup-script",
					Path:       "/etc/cron.d/backup",
					SHA256:     "abc123",
					Mode:       "0644",
					DeployedAt: "2026-03-22T00:00:00Z",
				})

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return(stateJSON)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				mockKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			req: file.UndeployRequest{Path: "/etc/cron.d/backup"},
			want: &file.UndeployResult{
				Changed: true,
				Path:    "/etc/cron.d/backup",
			},
			validateFunc: func(appFs avfs.VFS) {
				_, err := appFs.Stat("/etc/cron.d/backup")
				suite.True(err != nil, "file should be removed from disk")
			},
		},
		{
			name: "when file does not exist on disk",
			setupMock: func(
				_ *gomock.Controller,
				_ *jobmocks.MockKeyValue,
				_ avfs.VFS,
			) {
			},
			req: file.UndeployRequest{Path: "/etc/cron.d/nonexistent"},
			want: &file.UndeployResult{
				Changed: false,
				Path:    "/etc/cron.d/nonexistent",
			},
		},
		{
			name: "when file exists but no state entry",
			setupMock: func(
				_ *gomock.Controller,
				mockKV *jobmocks.MockKeyValue,
				appFs avfs.VFS,
			) {
				_ = appFs.MkdirAll("/etc/cron.d", 0o755)
				_ = appFs.WriteFile("/etc/cron.d/orphan", []byte("content"), 0o644)

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("key not found"))
			},
			req: file.UndeployRequest{Path: "/etc/cron.d/orphan"},
			want: &file.UndeployResult{
				Changed: true,
				Path:    "/etc/cron.d/orphan",
			},
			validateFunc: func(appFs avfs.VFS) {
				_, err := appFs.Stat("/etc/cron.d/orphan")
				suite.True(err != nil, "file should be removed from disk")
			},
		},
		{
			name: "when fs remove fails returns error",
			setupMock: func(
				_ *gomock.Controller,
				_ *jobmocks.MockKeyValue,
				appFs avfs.VFS,
			) {
				_ = appFs.MkdirAll("/etc/cron.d", 0o755)
				_ = appFs.WriteFile("/etc/cron.d/locked", []byte("content"), 0o644)
			},
			req:        file.UndeployRequest{Path: "/etc/cron.d/locked"},
			wantErr:    true,
			wantErrMsg: "failed to remove file",
			useFailFs:  true,
		},
		{
			name: "when file exists but state entry value is invalid JSON",
			setupMock: func(
				ctrl *gomock.Controller,
				mockKV *jobmocks.MockKeyValue,
				appFs avfs.VFS,
			) {
				_ = appFs.MkdirAll("/etc/cron.d", 0o755)
				_ = appFs.WriteFile("/etc/cron.d/corrupt", []byte("content"), 0o644)

				mockEntry := jobmocks.NewMockKeyValueEntry(ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-valid-json"))

				mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.UndeployRequest{Path: "/etc/cron.d/corrupt"},
			want: &file.UndeployResult{
				Changed: true,
				Path:    "/etc/cron.d/corrupt",
			},
			validateFunc: func(appFs avfs.VFS) {
				_, err := appFs.Stat("/etc/cron.d/corrupt")
				suite.True(err != nil, "file should be removed from disk")
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ctrl := gomock.NewController(suite.T())
			defer ctrl.Finish()

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			baseFs := memfs.New()
			mockObj := filemocks.NewMockObjectStore(ctrl)
			mockKV := jobmocks.NewMockKeyValue(ctrl)

			tt.setupMock(ctrl, mockKV, baseFs)

			var providerFs avfs.VFS = baseFs
			if tt.useFailFs {
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnRemove {
						return errors.New("remove failed")
					}

					return nil
				})
				providerFs = vfs
			}

			provider := file.New(suite.logger, providerFs, mockObj, mockKV, "test-host")

			got, err := provider.Undeploy(suite.ctx, tt.req)

			if tt.wantErr {
				suite.Error(err)
				suite.ErrorContains(err, tt.wantErrMsg)
				suite.Nil(got)
			} else {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(tt.want.Changed, got.Changed)
				suite.Equal(tt.want.Path, got.Path)

				if tt.validateFunc != nil {
					tt.validateFunc(baseFs)
				}
			}
		})
	}
}

func TestUndeployPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UndeployPublicTestSuite))
}
