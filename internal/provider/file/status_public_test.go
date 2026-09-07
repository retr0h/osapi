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
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/file"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
)

type StatusPublicTestSuite struct {
	suite.Suite

	ctrl    *gomock.Controller
	logger  *slog.Logger
	ctx     context.Context
	appFs   avfs.VFS
	mockKV  *jobmocks.MockKeyValue
	mockObj *filemocks.MockObjectStore
}

func (suite *StatusPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.ctx = context.Background()
	suite.appFs = memfs.New()
	suite.mockKV = jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockObj = filemocks.NewMockObjectStore(suite.ctrl)
}

func (suite *StatusPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *StatusPublicTestSuite) TestStatus() {
	fileContent := []byte("server { listen 80; }")
	fileSHA := computeTestSHA256(fileContent)
	driftedContent := []byte("server { listen 443; }")
	driftedSHA := computeTestSHA256(driftedContent)

	tests := []struct {
		name         string
		setupMock    func()
		req          file.StatusRequest
		validateFunc func(any, error)
	}{
		{
			name: "when file in sync",
			setupMock: func() {
				_ = suite.appFs.MkdirAll("/etc/nginx", 0o755)
				_ = suite.appFs.WriteFile("/etc/nginx/nginx.conf", fileContent, 0o644)

				existingState := job.FileState{
					SHA256: fileSHA,
					Path:   "/etc/nginx/nginx.conf",
				}
				stateBytes, _ := json.Marshal(existingState)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.StatusRequest{
				Path: "/etc/nginx/nginx.conf",
			},
			validateFunc: func(got any, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(&file.StatusResult{
					Path:   "/etc/nginx/nginx.conf",
					Status: "in-sync",
					SHA256: fileSHA,
				}, got)
			},
		},
		{
			name: "when file drifted",
			setupMock: func() {
				_ = suite.appFs.MkdirAll("/etc/nginx", 0o755)
				_ = suite.appFs.WriteFile("/etc/nginx/nginx.conf", driftedContent, 0o644)

				existingState := job.FileState{
					SHA256: fileSHA,
					Path:   "/etc/nginx/nginx.conf",
				}
				stateBytes, _ := json.Marshal(existingState)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.StatusRequest{
				Path: "/etc/nginx/nginx.conf",
			},
			validateFunc: func(got any, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(&file.StatusResult{
					Path:   "/etc/nginx/nginx.conf",
					Status: "drifted",
					SHA256: driftedSHA,
				}, got)
			},
		},
		{
			name: "when file missing on disk",
			setupMock: func() {
				existingState := job.FileState{
					SHA256: fileSHA,
					Path:   "/etc/nginx/nginx.conf",
				}
				stateBytes, _ := json.Marshal(existingState)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.StatusRequest{
				Path: "/etc/nginx/nginx.conf",
			},
			validateFunc: func(got any, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(&file.StatusResult{
					Path:   "/etc/nginx/nginx.conf",
					Status: "missing",
				}, got)
			},
		},
		{
			name: "when state entry has invalid JSON",
			setupMock: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))

				suite.mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			req: file.StatusRequest{
				Path: "/etc/nginx/nginx.conf",
			},
			validateFunc: func(got any, err error) {
				suite.Error(err)
				suite.ErrorContains(err, "failed to parse file state")
				suite.Nil(got)
			},
		},
		{
			name: "when no state entry",
			setupMock: func() {
				suite.mockKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, assert.AnError)
			},
			req: file.StatusRequest{
				Path: "/etc/nginx/nginx.conf",
			},
			validateFunc: func(got any, err error) {
				suite.NoError(err)
				suite.Require().NotNil(got)
				suite.Equal(&file.StatusResult{
					Path:   "/etc/nginx/nginx.conf",
					Status: "missing",
				}, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Reset filesystem for each test case.
			suite.appFs = memfs.New()

			if tc.setupMock != nil {
				tc.setupMock()
			}

			provider := file.New(
				suite.logger,
				suite.appFs,
				suite.mockObj,
				suite.mockKV,
				"test-host",
			)

			tc.validateFunc(provider.Status(suite.ctx, tc.req))
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestStatusPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(StatusPublicTestSuite))
}
