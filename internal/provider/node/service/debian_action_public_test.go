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

type DebianActionPublicTestSuite struct {
	suite.Suite

	ctrl            *gomock.Controller
	mockExecManager *execmocks.MockManager
	provider        *service.Debian
}

func (suite *DebianActionPublicTestSuite) SetupTest() {
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

func (suite *DebianActionPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianActionPublicTestSuite) TestStart() {
	tests := []struct {
		name         string
		serviceName  string
		setup        func()
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name:        "when service is inactive starts it",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-active", "osapi-nginx.service"}).
					Return("inactive\n", errors.New("exit status 3"))
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"start", "osapi-nginx.service"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:        "when service is already active returns changed false",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-active", "osapi-nginx.service"}).
					Return("active\n", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when start command fails returns error",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-active", "osapi-nginx.service"}).
					Return("inactive\n", errors.New("exit status 3"))
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"start", "osapi-nginx.service"}).
					Return("", errors.New("start failed"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: start:")
			},
		},
		{
			name:        "when name is invalid returns error",
			serviceName: "bad name!",
			setup:       func() {},
			validateFunc: func(
				result *service.ActionResult,
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

			result, err := suite.provider.Start(suite.T().Context(), tc.serviceName)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianActionPublicTestSuite) TestStop() {
	tests := []struct {
		name         string
		serviceName  string
		setup        func()
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name:        "when service is active stops it",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-active", "osapi-nginx.service"}).
					Return("active\n", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"stop", "osapi-nginx.service"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:        "when service is already stopped returns changed false",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-active", "osapi-nginx.service"}).
					Return("inactive\n", errors.New("exit status 3"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when stop command fails returns error",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-active", "osapi-nginx.service"}).
					Return("active\n", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"stop", "osapi-nginx.service"}).
					Return("", errors.New("stop failed"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: stop:")
			},
		},
		{
			name:        "when name is invalid returns error",
			serviceName: "bad name!",
			setup:       func() {},
			validateFunc: func(
				result *service.ActionResult,
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

			result, err := suite.provider.Stop(suite.T().Context(), tc.serviceName)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianActionPublicTestSuite) TestRestart() {
	tests := []struct {
		name         string
		serviceName  string
		setup        func()
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name:        "when restart succeeds returns changed true",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"restart", "osapi-nginx.service"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:        "when restart command fails returns error",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"restart", "osapi-nginx.service"}).
					Return("", errors.New("restart failed"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: restart:")
			},
		},
		{
			name:        "when name is invalid returns error",
			serviceName: "bad name!",
			setup:       func() {},
			validateFunc: func(
				result *service.ActionResult,
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

			result, err := suite.provider.Restart(suite.T().Context(), tc.serviceName)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianActionPublicTestSuite) TestEnable() {
	tests := []struct {
		name         string
		serviceName  string
		setup        func()
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name:        "when service is disabled enables it",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-enabled", "osapi-nginx.service"}).
					Return("disabled\n", errors.New("exit status 1"))
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"enable", "osapi-nginx.service"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:        "when service is already enabled returns changed false",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-enabled", "osapi-nginx.service"}).
					Return("enabled\n", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when enable command fails returns error",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-enabled", "osapi-nginx.service"}).
					Return("disabled\n", errors.New("exit status 1"))
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"enable", "osapi-nginx.service"}).
					Return("", errors.New("enable failed"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: enable:")
			},
		},
		{
			name:        "when name is invalid returns error",
			serviceName: "bad name!",
			setup:       func() {},
			validateFunc: func(
				result *service.ActionResult,
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

			result, err := suite.provider.Enable(suite.T().Context(), tc.serviceName)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianActionPublicTestSuite) TestDisable() {
	tests := []struct {
		name         string
		serviceName  string
		setup        func()
		validateFunc func(*service.ActionResult, error)
	}{
		{
			name:        "when service is enabled disables it",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-enabled", "osapi-nginx.service"}).
					Return("enabled\n", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"disable", "osapi-nginx.service"}).
					Return("", nil)
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:        "when service is already disabled returns changed false",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-enabled", "osapi-nginx.service"}).
					Return("disabled\n", errors.New("exit status 1"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("nginx", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when disable command fails returns error",
			serviceName: "nginx",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{"is-enabled", "osapi-nginx.service"}).
					Return("enabled\n", nil)
				suite.mockExecManager.EXPECT().
					RunPrivilegedCmd("systemctl", []string{"disable", "osapi-nginx.service"}).
					Return("", errors.New("disable failed"))
			},
			validateFunc: func(
				result *service.ActionResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "service: disable:")
			},
		},
		{
			name:        "when name is invalid returns error",
			serviceName: "bad name!",
			setup:       func() {},
			validateFunc: func(
				result *service.ActionResult,
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

			result, err := suite.provider.Disable(suite.T().Context(), tc.serviceName)

			tc.validateFunc(result, err)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianActionPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianActionPublicTestSuite))
}
