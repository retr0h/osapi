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

package service_test

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

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/file"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/service"
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

type DebianUnitPublicTestSuite struct {
	suite.Suite

	ctrl            *gomock.Controller
	logger          *slog.Logger
	memFs           avfs.VFS
	mockDeployer    *filemocks.MockDeployer
	mockStateKV     *jobmocks.MockKeyValue
	mockExecManager *execmocks.MockManager
	provider        *service.Debian
}

func (suite *DebianUnitPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockDeployer = filemocks.NewMockDeployer(suite.ctrl)
	suite.mockStateKV = jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockExecManager = execmocks.NewMockManager(suite.ctrl)

	_ = suite.memFs.MkdirAll("/etc/systemd/system", 0o755)

	suite.provider = service.NewDebianProvider(
		suite.logger,
		suite.memFs,
		suite.mockDeployer,
		suite.mockStateKV,
		suite.mockExecManager,
		testHostname,
	)
}

func (suite *DebianUnitPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianUnitPublicTestSuite) TestCreate() {
	tests := []struct {
		name         string
		entry        service.Entry
		setup        func()
		validateFunc func(*service.CreateResult, error)
	}{
		{
			name: "when deploy succeeds and daemon-reload runs",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), file.DeployRequest{
						ObjectName: "myapp-unit",
						Path:       "/etc/systemd/system/osapi-myapp.service",
						Mode:       "0644",
						Metadata:   map[string]string{"source": "custom"},
					}).
					Return(&file.DeployResult{
						Changed: true,
						Path:    "/etc/systemd/system/osapi-myapp.service",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when service unit already managed returns unchanged",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
			},
			validateFunc: func(
				result *service.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("myapp", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name: "when deploy fails",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("deploy error"))
			},
			validateFunc: func(
				result *service.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: create")
			},
		},
		{
			name: "when daemon-reload fails",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", errors.New("exec error"))
			},
			validateFunc: func(
				result *service.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "daemon-reload")
			},
		},
		{
			name: "when name is invalid",
			entry: service.Entry{
				Name:   "",
				Object: "myapp-unit",
			},
			setup: func() {},
			validateFunc: func(
				result *service.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid service name")
			},
		},
		{
			name: "when deploy returns unchanged skips daemon-reload",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: false}, nil)
				// daemon-reload should NOT be called.
			},
			validateFunc: func(
				result *service.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
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

func (suite *DebianUnitPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		entry        service.Entry
		setup        func()
		validateFunc func(*service.UpdateResult, error)
	}{
		{
			name: "when deploy succeeds and daemon-reload runs",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit-v2",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{
						Changed: true,
						Path:    "/etc/systemd/system/osapi-myapp.service",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when service unit does not exist",
			entry: service.Entry{
				Name:   "nonexistent",
				Object: "some-unit",
			},
			setup: func() {},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not managed")
			},
		},
		{
			name: "when deploy fails",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("deploy error"))
			},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: update")
			},
		},
		{
			name: "when deploy returns unchanged skips daemon-reload",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: false}, nil)
				// daemon-reload should NOT be called.
			},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name: "when daemon-reload fails",
			entry: service.Entry{
				Name:   "myapp",
				Object: "myapp-unit",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), gomock.Any()).
					Return(&file.DeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", errors.New("exec error"))
			},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "daemon-reload")
			},
		},
		{
			name: "when name is invalid",
			entry: service.Entry{
				Name:   "bad name!",
				Object: "some-unit",
			},
			setup: func() {},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid service name")
			},
		},
		{
			name: "when object not specified preserves existing from state",
			entry: service.Entry{
				Name: "myapp",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				stateData := managedStateJSON(
					"original-unit",
					"/etc/systemd/system/osapi-myapp.service",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData).AnyTimes()
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockDeployer.EXPECT().
					Deploy(gomock.Any(), file.DeployRequest{
						ObjectName: "original-unit",
						Path:       "/etc/systemd/system/osapi-myapp.service",
						Mode:       "0644",
						Metadata:   map[string]string{"source": "custom"},
					}).
					Return(&file.DeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when object not specified and state lookup fails",
			entry: service.Entry{
				Name: "myapp",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("kv error"))
			},
			validateFunc: func(
				result *service.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "failed to read existing state")
			},
		},
		{
			name: "when object not specified and state returns invalid JSON",
			entry: service.Entry{
				Name: "myapp",
			},
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *service.UpdateResult,
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

func (suite *DebianUnitPublicTestSuite) TestDelete() {
	tests := []struct {
		name         string
		entryName    string
		setup        func()
		validateFunc func(*service.DeleteResult, error)
	}{
		{
			name:      "when undeploy succeeds with stop and disable",
			entryName: "myapp",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"stop", "osapi-myapp.service"}).
					Return("", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"disable", "osapi-myapp.service"}).
					Return("", nil)
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), file.UndeployRequest{
						Path: "/etc/systemd/system/osapi-myapp.service",
					}).
					Return(&file.UndeployResult{
						Changed: true,
						Path:    "/etc/systemd/system/osapi-myapp.service",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:      "when service unit not found returns unchanged",
			entryName: "nonexistent",
			setup:     func() {},
			validateFunc: func(
				result *service.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("nonexistent", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:      "when undeploy fails",
			entryName: "myapp",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"stop", "osapi-myapp.service"}).
					Return("", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"disable", "osapi-myapp.service"}).
					Return("", nil)
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("undeploy error"))
			},
			validateFunc: func(
				result *service.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: delete")
			},
		},
		{
			name:      "when daemon-reload fails",
			entryName: "myapp",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"stop", "osapi-myapp.service"}).
					Return("", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"disable", "osapi-myapp.service"}).
					Return("", nil)
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), gomock.Any()).
					Return(&file.UndeployResult{Changed: true}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", errors.New("exec error"))
			},
			validateFunc: func(
				result *service.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "daemon-reload")
			},
		},
		{
			name:      "when stop and disable fail continues with delete",
			entryName: "myapp",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/systemd/system/osapi-myapp.service",
					[]byte("[Unit]\nDescription=My App"),
					0o644,
				)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"stop", "osapi-myapp.service"}).
					Return("", errors.New("stop error"))
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"disable", "osapi-myapp.service"}).
					Return("", errors.New("disable error"))
				suite.mockDeployer.EXPECT().
					Undeploy(gomock.Any(), file.UndeployRequest{
						Path: "/etc/systemd/system/osapi-myapp.service",
					}).
					Return(&file.UndeployResult{
						Changed: true,
						Path:    "/etc/systemd/system/osapi-myapp.service",
					}, nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"daemon-reload"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("myapp", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:      "when name is invalid",
			entryName: "bad name!",
			setup:     func() {},
			validateFunc: func(
				result *service.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "invalid service name")
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
func TestDebianUnitPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianUnitPublicTestSuite))
}
