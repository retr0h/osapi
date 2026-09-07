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
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	filemocks "github.com/osapi-io/osapi/internal/provider/file/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/service"

	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
)

type DebianGetPublicTestSuite struct {
	suite.Suite

	ctrl            *gomock.Controller
	mockExecManager *execmocks.MockManager
	provider        *service.Debian
}

func (suite *DebianGetPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	memFs := memfs.New()
	mockDeployer := filemocks.NewMockDeployer(suite.ctrl)
	mockStateKV := jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockExecManager = execmocks.NewMockManager(suite.ctrl)

	suite.provider = service.NewDebianProvider(
		logger,
		memFs,
		mockDeployer,
		mockStateKV,
		suite.mockExecManager,
		testHostname,
	)
}

func (suite *DebianGetPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianGetPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		serviceName  string
		setup        func()
		validateFunc func(*service.Info, error)
	}{
		{
			name:        "when active enabled service with PID returns info",
			serviceName: "nginx.service",
			setup: func() {
				output := "ActiveState=active\nUnitFileState=enabled\nDescription=A high performance web server\nMainPID=1234\n"
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"show",
						"nginx.service",
						"--property=ActiveState,UnitFileState,Description,MainPID",
						"--no-pager",
					}).
					Return(output, nil)
			},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(info)
				suite.Equal("nginx.service", info.Name)
				suite.Equal("active", info.Status)
				suite.True(info.Enabled)
				suite.Equal("A high performance web server", info.Description)
				suite.Equal(1234, info.PID)
			},
		},
		{
			name:        "when inactive disabled service with zero PID returns info",
			serviceName: "cron.service",
			setup: func() {
				output := "ActiveState=inactive\nUnitFileState=disabled\nDescription=Regular background program processing daemon\nMainPID=0\n"
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"show",
						"cron.service",
						"--property=ActiveState,UnitFileState,Description,MainPID",
						"--no-pager",
					}).
					Return(output, nil)
			},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(info)
				suite.Equal("cron.service", info.Name)
				suite.Equal("inactive", info.Status)
				suite.False(info.Enabled)
				suite.Equal("Regular background program processing daemon", info.Description)
				suite.Equal(0, info.PID)
			},
		},
		{
			name:        "when exec fails returns error",
			serviceName: "missing.service",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"show",
						"missing.service",
						"--property=ActiveState,UnitFileState,Description,MainPID",
						"--no-pager",
					}).
					Return("", errors.New("unit not found"))
			},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.Error(err)
				suite.Nil(info)
				suite.Contains(err.Error(), "service: get:")
			},
		},
		{
			name:        "when output has malformed line returns error",
			serviceName: "bad.service",
			setup: func() {
				output := "ActiveState=active\nmalformed-line-no-equals\n"
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"show",
						"bad.service",
						"--property=ActiveState,UnitFileState,Description,MainPID",
						"--no-pager",
					}).
					Return(output, nil)
			},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.Error(err)
				suite.Nil(info)
				suite.Contains(err.Error(), "malformed property line")
			},
		},
		{
			name:        "when output has blank lines they are skipped",
			serviceName: "blanks.service",
			setup: func() {
				output := "ActiveState=active\n\nDescription=test service\n\nMainPID=42\n"
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"show",
						"blanks.service",
						"--property=ActiveState,UnitFileState,Description,MainPID",
						"--no-pager",
					}).
					Return(output, nil)
			},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(info)
				suite.Equal("active", info.Status)
				suite.Equal("test service", info.Description)
				suite.Equal(42, info.PID)
			},
		},
		{
			name:        "when name is invalid returns error",
			serviceName: "bad name!",
			setup:       func() {},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.Error(err)
				suite.Nil(info)
				suite.Contains(err.Error(), "invalid service name")
			},
		},
		{
			name:        "when name is empty returns error",
			serviceName: "",
			setup:       func() {},
			validateFunc: func(
				info *service.Info,
				err error,
			) {
				suite.Error(err)
				suite.Nil(info)
				suite.Contains(err.Error(), "invalid service name: empty")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			info, err := suite.provider.Get(suite.T().Context(), tc.serviceName)

			tc.validateFunc(info, err)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianGetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianGetPublicTestSuite))
}
