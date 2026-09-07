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

package certificate_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/certificate"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
)

// managedStateJSON returns a JSON-encoded FileState with no UndeployedAt,
// indicating the file is actively managed by osapi.
func managedStateJSON(
	objectName string,
	path string,
) []byte {
	state := job.FileState{
		ObjectName: objectName,
		Path:       path,
		SHA256:     "abc123",
		DeployedAt: "2026-01-01T00:00:00Z",
		Metadata:   map[string]string{"source": "custom"},
	}

	b, _ := json.Marshal(state)

	return b
}

type DebianListPublicTestSuite struct {
	suite.Suite

	ctrl            *gomock.Controller
	logger          *slog.Logger
	memFs           avfs.VFS
	mockDeployer    *filemocks.MockDeployer
	mockStateKV     *jobmocks.MockKeyValue
	mockExecManager *execmocks.MockManager
	provider        *certificate.Debian
}

func (suite *DebianListPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockDeployer = filemocks.NewMockDeployer(suite.ctrl)
	suite.mockStateKV = jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockExecManager = execmocks.NewMockManager(suite.ctrl)

	_ = suite.memFs.MkdirAll("/usr/share/ca-certificates", 0o755)
	_ = suite.memFs.MkdirAll("/usr/local/share/ca-certificates", 0o755)

	suite.provider = certificate.NewDebianProvider(
		suite.logger,
		suite.memFs,
		suite.mockDeployer,
		suite.mockStateKV,
		suite.mockExecManager,
		testHostname,
	)
}

func (suite *DebianListPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianListPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func([]certificate.Entry, error)
	}{
		{
			name: "when system and custom certs exist",
			setup: func() {
				// System certs.
				_ = suite.memFs.MkdirAll("/usr/share/ca-certificates/mozilla", 0o755)
				_ = suite.memFs.WriteFile(
					"/usr/share/ca-certificates/mozilla/DigiCert.crt",
					[]byte("cert"),
					0o644,
				)
				_ = suite.memFs.WriteFile(
					"/usr/share/ca-certificates/mozilla/LetsEncrypt.crt",
					[]byte("cert"),
					0o644,
				)
				// Custom managed cert.
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("cert"),
					0o644,
				)
				stateData := managedStateJSON(
					"my-ca-cert",
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData).AnyTimes()
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Len(entries, 3)
				// Custom certs first.
				suite.Equal("my-ca", entries[0].Name)
				suite.Equal("custom", entries[0].Source)
				// System certs after.
				suite.Equal("mozilla/DigiCert", entries[1].Name)
				suite.Equal("system", entries[1].Source)
				suite.Equal("mozilla/LetsEncrypt", entries[2].Name)
				suite.Equal("system", entries[2].Source)
			},
		},
		{
			name: "when system dir read fails",
			setup: func() {
				badFs := memfs.New()
				_ = badFs.MkdirAll("/usr/local/share/ca-certificates", 0o755)
				// No system dir = WalkDir fails.
				suite.provider = certificate.NewDebianProvider(
					suite.logger,
					badFs,
					suite.mockDeployer,
					suite.mockStateKV,
					suite.mockExecManager,
					testHostname,
				)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.Error(err)
				suite.Nil(entries)
				suite.Contains(err.Error(), "list certificates")
			},
		},
		{
			name: "when directories are empty",
			setup: func() {
				// Both dirs exist but have no files.
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom cert without managed prefix is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/manual-ca.crt",
					[]byte("cert"),
					0o644,
				)
				// No stateKV calls expected.
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom cert has invalid JSON state is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-bad-json.crt",
					[]byte("cert"),
					0o644,
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom cert has undeployed state is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-old-ca.crt",
					[]byte("cert"),
					0o644,
				)
				undeployedState, _ := json.Marshal(job.FileState{
					ObjectName:   "old-ca-cert",
					Path:         "/usr/local/share/ca-certificates/osapi-old-ca.crt",
					SHA256:       "abc123",
					DeployedAt:   "2026-01-01T00:00:00Z",
					UndeployedAt: "2026-02-01T00:00:00Z",
				})
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(undeployedState)
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom cert without file state is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-unknown.crt",
					[]byte("cert"),
					0o644,
				)
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom dir has subdirectory it is skipped",
			setup: func() {
				_ = suite.memFs.MkdirAll(
					"/usr/local/share/ca-certificates/subdir",
					0o755,
				)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom cert has no crt suffix it is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.pem",
					[]byte("cert"),
					0o644,
				)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when system dir has non-crt file it is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/share/ca-certificates/README",
					[]byte("readme"),
					0o644,
				)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when custom dir read fails gracefully returns system certs only",
			setup: func() {
				// Create FS with only system dir, no custom dir.
				noCustomFs := memfs.New()
				_ = noCustomFs.MkdirAll("/usr/share/ca-certificates", 0o755)
				_ = noCustomFs.WriteFile(
					"/usr/share/ca-certificates/test.crt",
					[]byte("cert"),
					0o644,
				)
				suite.provider = certificate.NewDebianProvider(
					suite.logger,
					noCustomFs,
					suite.mockDeployer,
					suite.mockStateKV,
					suite.mockExecManager,
					testHostname,
				)
			},
			validateFunc: func(
				entries []certificate.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Len(entries, 1)
				suite.Equal("test", entries[0].Name)
				suite.Equal("system", entries[0].Source)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			entries, err := suite.provider.List(context.Background())

			tc.validateFunc(entries, err)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianListPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianListPublicTestSuite))
}
