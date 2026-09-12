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

const testHostname = "test-host"

type DebianListPublicTestSuite struct {
	suite.Suite

	ctrl            *gomock.Controller
	mockExecManager *execmocks.MockManager
	provider        *service.Debian
}

func (suite *DebianListPublicTestSuite) SetupTest() {
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

func (suite *DebianListPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianListPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func([]service.Info, error)
	}{
		{
			name: "when services exist returns merged list",
			setup: func() {
				unitsJSON := `[
					{"unit":"nginx.service","load":"loaded","active":"active","sub":"running","description":"A high performance web server"},
					{"unit":"ssh.service","load":"loaded","active":"active","sub":"running","description":"OpenBSD Secure Shell server"}
				]`
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-units",
						"--type=service",
						"--all",
						"--no-pager",
						"--output=json",
					}).
					Return(unitsJSON, nil)

				unitFilesJSON := `[
					{"unit_file":"nginx.service","state":"enabled","preset":"enabled"},
					{"unit_file":"ssh.service","state":"disabled","preset":"enabled"}
				]`
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-unit-files",
						"--type=service",
						"--no-pager",
						"--output=json",
					}).
					Return(unitFilesJSON, nil)
			},
			validateFunc: func(
				infos []service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.Len(infos, 2)

				suite.Equal("nginx.service", infos[0].Name)
				suite.Equal("active", infos[0].Status)
				suite.True(infos[0].Enabled)
				suite.Equal("A high performance web server", infos[0].Description)

				suite.Equal("ssh.service", infos[1].Name)
				suite.Equal("active", infos[1].Status)
				suite.False(infos[1].Enabled)
				suite.Equal("OpenBSD Secure Shell server", infos[1].Description)
			},
		},
		{
			name: "when list-units exec fails returns error",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-units",
						"--type=service",
						"--all",
						"--no-pager",
						"--output=json",
					}).
					Return("", errors.New("command failed"))
			},
			validateFunc: func(
				infos []service.Info,
				err error,
			) {
				suite.Error(err)
				suite.Nil(infos)
				suite.Contains(err.Error(), "service: list:")
			},
		},
		{
			name: "when list-unit-files exec fails returns services with enabled false",
			setup: func() {
				unitsJSON := `[
					{"unit":"nginx.service","load":"loaded","active":"active","sub":"running","description":"nginx"}
				]`
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-units",
						"--type=service",
						"--all",
						"--no-pager",
						"--output=json",
					}).
					Return(unitsJSON, nil)

				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-unit-files",
						"--type=service",
						"--no-pager",
						"--output=json",
					}).
					Return("", errors.New("command failed"))
			},
			validateFunc: func(
				infos []service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.Len(infos, 1)
				suite.Equal("nginx.service", infos[0].Name)
				suite.False(infos[0].Enabled)
			},
		},
		{
			name: "when no services exist returns empty list",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-units",
						"--type=service",
						"--all",
						"--no-pager",
						"--output=json",
					}).
					Return("[]", nil)

				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-unit-files",
						"--type=service",
						"--no-pager",
						"--output=json",
					}).
					Return("[]", nil)
			},
			validateFunc: func(
				infos []service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(infos)
			},
		},
		{
			name: "when list-units returns malformed JSON returns error",
			setup: func() {
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-units",
						"--type=service",
						"--all",
						"--no-pager",
						"--output=json",
					}).
					Return("not-json", nil)
			},
			validateFunc: func(
				infos []service.Info,
				err error,
			) {
				suite.Error(err)
				suite.Nil(infos)
				suite.Contains(err.Error(), "service: list: parse units:")
			},
		},
		{
			name: "when list-unit-files returns malformed JSON returns services with enabled false",
			setup: func() {
				unitsJSON := `[
					{"unit":"cron.service","load":"loaded","active":"inactive","sub":"dead","description":"cron daemon"}
				]`
				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-units",
						"--type=service",
						"--all",
						"--no-pager",
						"--output=json",
					}).
					Return(unitsJSON, nil)

				suite.mockExecManager.EXPECT().
					RunCmd("systemctl", []string{
						"list-unit-files",
						"--type=service",
						"--no-pager",
						"--output=json",
					}).
					Return("not-json", nil)
			},
			validateFunc: func(
				infos []service.Info,
				err error,
			) {
				suite.NoError(err)
				suite.Len(infos, 1)
				suite.Equal("cron.service", infos[0].Name)
				suite.Equal("inactive", infos[0].Status)
				suite.False(infos[0].Enabled)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			infos, err := suite.provider.List(suite.T().Context())

			tc.validateFunc(infos, err)
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
