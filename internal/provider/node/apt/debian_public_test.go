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

package apt_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execMocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/apt"
)

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl        *gomock.Controller
	mockExec    *execMocks.MockManager
	logger      *slog.Logger
	provider    *apt.Debian
	dpkgFormat  string
	dpkgOutput  string
	aptUpOutput string
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockExec = execMocks.NewMockManager(suite.ctrl)
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.provider = apt.NewDebianProvider(suite.logger, suite.mockExec)
	suite.dpkgFormat = "${Package}\t${Version}\t${binary:Summary}\t${db:Status-Abbrev}\t${Installed-Size}\n"
	suite.dpkgOutput = "vim\t2:9.0.1378-2\tVi IMproved - enhanced vi editor\tii \t3826\n" +
		"curl\t7.88.1-10+deb12u5\tcommand line tool for transferring data with URL syntax\tii \t513\n" +
		"removed-pkg\t1.0-1\tA removed package\trc \t100\n"
	suite.aptUpOutput = "Listing... Done\n" +
		"vim/stable 2:9.0.1378-3 amd64 [upgradable from: 2:9.0.1378-2]\n" +
		"curl/stable 7.88.1-10+deb12u6 amd64 [upgradable from: 7.88.1-10+deb12u5]\n"
}

func (suite *DebianPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setupMock    func()
		wantErr      bool
		errContains  string
		validateFunc func([]apt.Package)
	}{
		{
			name: "when list succeeds",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat}).
					Return(suite.dpkgOutput, nil)
			},
			validateFunc: func(pkgs []apt.Package) {
				suite.Require().Len(pkgs, 2)
				suite.Equal("vim", pkgs[0].Name)
				suite.Equal("2:9.0.1378-2", pkgs[0].Version)
				suite.Equal("Vi IMproved - enhanced vi editor", pkgs[0].Description)
				suite.Equal("installed", pkgs[0].Status)
				suite.Equal(int64(3826*1024), pkgs[0].Size)
				suite.Equal("curl", pkgs[1].Name)
				suite.Equal(int64(513*1024), pkgs[1].Size)
			},
		},
		{
			name: "when exec error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat}).
					Return("", fmt.Errorf("exec failed"))
			},
			wantErr:     true,
			errContains: "package: list:",
		},
		{
			name: "when empty output",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat}).
					Return("", nil)
			},
			validateFunc: func(pkgs []apt.Package) {
				suite.Require().Empty(pkgs)
			},
		},
		{
			name: "when malformed lines are skipped",
			setupMock: func() {
				output := "only-two\tfields\n" +
					"vim\t2:9.0.1378-2\tVi IMproved\tii \t3826\n"
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat}).
					Return(output, nil)
			},
			validateFunc: func(pkgs []apt.Package) {
				suite.Require().Len(pkgs, 1)
				suite.Equal("vim", pkgs[0].Name)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.List(context.Background())

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.errContains)

				return
			}

			suite.Require().NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		pkgName      string
		setupMock    func()
		wantErr      bool
		errContains  string
		validateFunc func(*apt.Package)
	}{
		{
			name:    "when get succeeds",
			pkgName: "vim",
			setupMock: func() {
				output := "vim\t2:9.0.1378-2\tVi IMproved - enhanced vi editor\tii \t3826\n"
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat, "vim"}).
					Return(output, nil)
			},
			validateFunc: func(pkg *apt.Package) {
				suite.Require().NotNil(pkg)
				suite.Equal("vim", pkg.Name)
				suite.Equal("2:9.0.1378-2", pkg.Version)
				suite.Equal("installed", pkg.Status)
				suite.Equal(int64(3826*1024), pkg.Size)
			},
		},
		{
			name:    "when package not installed",
			pkgName: "nonexistent",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat, "nonexistent"}).
					Return("", fmt.Errorf("no packages found matching nonexistent"))
			},
			wantErr:     true,
			errContains: "package: get \"nonexistent\":",
		},
		{
			name:    "when package in removed state",
			pkgName: "removed-pkg",
			setupMock: func() {
				output := "removed-pkg\t1.0-1\tA removed package\trc \t100\n"
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat, "removed-pkg"}).
					Return(output, nil)
			},
			wantErr:     true,
			errContains: "package: get \"removed-pkg\": not found",
		},
		{
			name:    "when exec error",
			pkgName: "vim",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("dpkg-query", []string{"-W", "-f", suite.dpkgFormat, "vim"}).
					Return("", fmt.Errorf("exec failed"))
			},
			wantErr:     true,
			errContains: "package: get \"vim\":",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Get(context.Background(), tc.pkgName)

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.errContains)

				return
			}

			suite.Require().NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestInstall() {
	tests := []struct {
		name         string
		pkgName      string
		setupMock    func()
		wantErr      bool
		errContains  string
		validateFunc func(*apt.Result)
	}{
		{
			name:    "when install succeeds",
			pkgName: "vim",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("apt-get", []string{"install", "-y", "vim"}).
					Return("", nil)
			},
			validateFunc: func(result *apt.Result) {
				suite.Require().NotNil(result)
				suite.Equal("vim", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:    "when exec error",
			pkgName: "badpkg",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("apt-get", []string{"install", "-y", "badpkg"}).
					Return("", fmt.Errorf("E: Unable to locate package badpkg"))
			},
			wantErr:     true,
			errContains: "package: install \"badpkg\":",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Install(context.Background(), tc.pkgName)

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.errContains)

				return
			}

			suite.Require().NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestRemove() {
	tests := []struct {
		name         string
		pkgName      string
		setupMock    func()
		wantErr      bool
		errContains  string
		validateFunc func(*apt.Result)
	}{
		{
			name:    "when remove succeeds",
			pkgName: "vim",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("apt-get", []string{"remove", "-y", "vim"}).
					Return("", nil)
			},
			validateFunc: func(result *apt.Result) {
				suite.Require().NotNil(result)
				suite.Equal("vim", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:    "when exec error",
			pkgName: "vim",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("apt-get", []string{"remove", "-y", "vim"}).
					Return("", fmt.Errorf("permission denied"))
			},
			wantErr:     true,
			errContains: "package: remove \"vim\":",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Remove(context.Background(), tc.pkgName)

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.errContains)

				return
			}

			suite.Require().NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		setupMock    func()
		wantErr      bool
		errContains  string
		validateFunc func(*apt.Result)
	}{
		{
			name: "when update succeeds",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("apt-get", []string{"update"}).
					Return("", nil)
			},
			validateFunc: func(result *apt.Result) {
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
		{
			name: "when exec error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("apt-get", []string{"update"}).
					Return("", fmt.Errorf("permission denied"))
			},
			wantErr:     true,
			errContains: "package: update:",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Update(context.Background())

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.errContains)

				return
			}

			suite.Require().NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestListUpdates() {
	tests := []struct {
		name         string
		setupMock    func()
		wantErr      bool
		errContains  string
		validateFunc func([]apt.Update)
	}{
		{
			name: "when list updates succeeds",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("apt", []string{"list", "--upgradable"}).
					Return(suite.aptUpOutput, nil)
			},
			validateFunc: func(updates []apt.Update) {
				suite.Require().Len(updates, 2)
				suite.Equal("vim", updates[0].Name)
				suite.Equal("2:9.0.1378-3", updates[0].NewVersion)
				suite.Equal("2:9.0.1378-2", updates[0].CurrentVersion)
				suite.Equal("curl", updates[1].Name)
				suite.Equal("7.88.1-10+deb12u6", updates[1].NewVersion)
				suite.Equal("7.88.1-10+deb12u5", updates[1].CurrentVersion)
			},
		},
		{
			name: "when no updates available",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("apt", []string{"list", "--upgradable"}).
					Return("Listing... Done\n", nil)
			},
			validateFunc: func(updates []apt.Update) {
				suite.Require().Empty(updates)
			},
		},
		{
			name: "when malformed lines are skipped",
			setupMock: func() {
				output := "Listing... Done\n" +
					"no-slash-here\n" +
					"short/line version\n" +
					"vim/stable 2:9.0.1378-3 amd64 [upgradable from: 2:9.0.1378-2]\n"
				suite.mockExec.EXPECT().
					RunCmd("apt", []string{"list", "--upgradable"}).
					Return(output, nil)
			},
			validateFunc: func(updates []apt.Update) {
				suite.Require().Len(updates, 1)
				suite.Equal("vim", updates[0].Name)
			},
		},
		{
			name: "when exec error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("apt", []string{"list", "--upgradable"}).
					Return("", fmt.Errorf("exec failed"))
			},
			wantErr:     true,
			errContains: "package: list updates:",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.ListUpdates(context.Background())

			if tc.wantErr {
				suite.Require().Error(err)
				suite.Contains(err.Error(), tc.errContains)

				return
			}

			suite.Require().NoError(err)
			tc.validateFunc(got)
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
