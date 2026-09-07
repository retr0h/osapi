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
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/file"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/certificate"
)

const testHostname = "test-host"

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl            *gomock.Controller
	logger          *slog.Logger
	memFs           avfs.VFS
	mockDeployer    *filemocks.MockDeployer
	mockStateKV     *jobmocks.MockKeyValue
	mockExecManager *execmocks.MockManager
	provider        *certificate.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
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

func (suite *DebianPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianPublicTestSuite) TestCreate() {
	tests := []struct {
		name         string
		entry        certificate.Entry
		setup        func()
		validateFunc func(*certificate.CreateResult, error)
	}{
		{
			name: "when deploy succeeds and update-ca-certificates runs",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), file.DeployRequest{
						ObjectName: "my-ca-cert",
						Path:       "/usr/local/share/ca-certificates/osapi-my-ca.crt",
						Mode:       "0644",
						Metadata:   map[string]string{"source": "custom"},
					}).
					Return(&file.DeployResult{
						Changed: true,
						Path:    "/usr/local/share/ca-certificates/osapi-my-ca.crt",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", nil)
			},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("my-ca", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when certificate already managed returns unchanged",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
			},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("my-ca", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name: "when deploy fails",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("deploy error"))
			},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "create certificate")
			},
		},
		{
			name: "when update-ca-certificates fails",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", errors.New("exec error"))
			},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "update-ca-certificates")
			},
		},
		{
			name: "when name is empty",
			entry: certificate.Entry{
				Name:   "",
				Object: "my-ca-cert",
			},
			setup: func() {},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid certificate name")
			},
		},
		{
			name: "when name has special chars",
			entry: certificate.Entry{
				Name:   "bad name!",
				Object: "my-ca-cert",
			},
			setup: func() {},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid certificate name")
			},
		},
		{
			name: "when deploy returns unchanged",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: false}, nil)
				// update-ca-certificates should NOT be called.
			},
			validateFunc: func(
				result *certificate.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("my-ca", result.Name)
				suite.False(result.Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Create(context.Background(), tc.entry)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		entry        certificate.Entry
		setup        func()
		validateFunc func(*certificate.UpdateResult, error)
	}{
		{
			name: "when deploy succeeds",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert-v2",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{
						Changed: true,
						Path:    "/usr/local/share/ca-certificates/osapi-my-ca.crt",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", nil)
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("my-ca", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when certificate does not exist",
			entry: certificate.Entry{
				Name:   "nonexistent",
				Object: "some-cert",
			},
			setup: func() {},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "does not exist")
			},
		},
		{
			name: "when deploy fails",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("deploy error"))
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "update certificate")
			},
		},
		{
			name: "when deploy returns unchanged",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: false}, nil)
				// update-ca-certificates should NOT be called.
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("my-ca", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name: "when update-ca-certificates fails",
			entry: certificate.Entry{
				Name:   "my-ca",
				Object: "my-ca-cert",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", errors.New("exec error"))
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "update-ca-certificates")
			},
		},
		{
			name: "when name is invalid",
			entry: certificate.Entry{
				Name:   "bad name",
				Object: "some-cert",
			},
			setup: func() {},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid certificate name")
			},
		},
		{
			name: "when object not specified preserves existing",
			entry: certificate.Entry{
				Name: "my-ca",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				stateData := managedStateJSON(
					"original-cert",
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData).AnyTimes()
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), file.DeployRequest{
						ObjectName: "original-cert",
						Path:       "/usr/local/share/ca-certificates/osapi-my-ca.crt",
						Mode:       "0644",
						Metadata:   map[string]string{"source": "custom"},
					}).
					Return(&file.DeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", nil)
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("my-ca", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when object not specified and state lookup fails",
			entry: certificate.Entry{
				Name: "my-ca",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("kv error"))
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "failed to read existing state")
			},
		},
		{
			name: "when object not specified and state returns invalid JSON",
			entry: certificate.Entry{
				Name: "my-ca",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *certificate.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "failed to read existing state")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Update(context.Background(), tc.entry)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestDelete() {
	tests := []struct {
		name         string
		entryName    string
		setup        func()
		validateFunc func(*certificate.DeleteResult, error)
	}{
		{
			name:      "when undeploy succeeds",
			entryName: "my-ca",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), file.UndeployRequest{
						Path: "/usr/local/share/ca-certificates/osapi-my-ca.crt",
					}).
					Return(&file.UndeployResult{
						Changed: true,
						Path:    "/usr/local/share/ca-certificates/osapi-my-ca.crt",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", nil)
			},
			validateFunc: func(
				result *certificate.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("my-ca", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:      "when certificate not found",
			entryName: "nonexistent",
			setup:     func() {},
			validateFunc: func(
				result *certificate.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("nonexistent", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:      "when undeploy fails",
			entryName: "my-ca",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("undeploy error"))
			},
			validateFunc: func(
				result *certificate.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "delete certificate")
			},
		},
		{
			name:      "when update-ca-certificates fails",
			entryName: "my-ca",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/usr/local/share/ca-certificates/osapi-my-ca.crt",
					[]byte("existing cert"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), gomock.Any()).
					Return(&file.UndeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("update-ca-certificates", nil).
					Return("", errors.New("exec error"))
			},
			validateFunc: func(
				result *certificate.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "update-ca-certificates")
			},
		},
		{
			name:      "when name is invalid",
			entryName: "bad name",
			setup:     func() {},
			validateFunc: func(
				result *certificate.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid certificate name")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Delete(context.Background(), tc.entryName)

			tc.validateFunc(result, err)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianPublicTestSuite))
}
