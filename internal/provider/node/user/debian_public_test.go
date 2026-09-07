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

package user_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/user"
)

const passwdContent = `root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
john:x:1000:1000:John Doe:/home/john:/bin/bash
jane:x:1001:1001:Jane Doe:/home/jane:/bin/zsh
`

const groupContent = `root:x:0:
daemon:x:1:
sudo:x:27:john,jane
docker:x:999:john
developers:x:1001:john,jane,bob
`

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl     *gomock.Controller
	ctx      context.Context
	logger   *slog.Logger
	memFs    avfs.VFS
	mockExec *execmocks.MockManager
	provider *user.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	suite.provider = user.NewDebianProvider(
		suite.logger,
		suite.memFs,
		suite.mockExec,
	)
}

func (suite *DebianPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianPublicTestSuite) writePasswd(
	content string,
) {
	_ = suite.memFs.MkdirAll("/etc", 0o755)

	f, err := suite.memFs.Create("/etc/passwd")
	suite.Require().NoError(err)

	_, err = f.Write([]byte(content))
	suite.Require().NoError(err)
	suite.Require().NoError(f.Close())
}

func (suite *DebianPublicTestSuite) writeGroup(
	content string,
) {
	_ = suite.memFs.MkdirAll("/etc", 0o755)

	f, err := suite.memFs.Create("/etc/group")
	suite.Require().NoError(err)

	_, err = f.Write([]byte(content))
	suite.Require().NoError(err)
	suite.Require().NoError(f.Close())
}

// User tests

func (suite *DebianPublicTestSuite) TestListUsers() {
	tests := []struct {
		name         string
		passwd       string
		setup        func()
		validateFunc func([]user.User, error)
	}{
		{
			name:   "when successful",
			passwd: passwdContent,
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "root"}).
					Return("root", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "root"}).
					Return("root P 01/01/2026 0 99999 7 -1", nil)
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "daemon"}).
					Return("daemon", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "daemon"}).
					Return("daemon L 01/01/2026 0 99999 7 -1", nil)
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "john"}).
					Return("john sudo docker", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "john"}).
					Return("john P 01/01/2026 0 99999 7 -1", nil)
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "jane"}).
					Return("jane sudo", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "jane"}).
					Return("jane L 01/01/2026 0 99999 7 -1", nil)
			},
			validateFunc: func(result []user.User, err error) {
				suite.NoError(err)
				suite.Require().Len(result, 4)

				suite.Equal("root", result[0].Name)
				suite.Equal(0, result[0].UID)
				suite.Equal(0, result[0].GID)
				suite.Equal("/root", result[0].Home)
				suite.Equal("/bin/bash", result[0].Shell)
				suite.Equal([]string{"root"}, result[0].Groups)
				suite.False(result[0].Locked)

				suite.Equal("daemon", result[1].Name)
				suite.Equal(1, result[1].UID)
				suite.True(result[1].Locked)

				suite.Equal("john", result[2].Name)
				suite.Equal(1000, result[2].UID)
				suite.Equal(1000, result[2].GID)
				suite.Equal("/home/john", result[2].Home)
				suite.Equal("/bin/bash", result[2].Shell)
				suite.Equal([]string{"john", "sudo", "docker"}, result[2].Groups)
				suite.False(result[2].Locked)

				suite.Equal("jane", result[3].Name)
				suite.Equal(1001, result[3].UID)
				suite.True(result[3].Locked)
			},
		},
		{
			name:   "when passwd file not found",
			passwd: "",
			setup:  func() {},
			validateFunc: func(result []user.User, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "user:")
			},
		},
		{
			name:   "when only root exists",
			passwd: "root:x:0:0:root:/root:/bin/bash\n",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "root"}).
					Return("root", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "root"}).
					Return("root P 01/01/2026 0 99999 7 -1", nil)
			},
			validateFunc: func(result []user.User, err error) {
				suite.NoError(err)
				suite.Require().Len(result, 1)
				suite.Equal("root", result[0].Name)
				suite.Equal(0, result[0].UID)
			},
		},
		{
			name:   "when passwd has corrupt lines skips them",
			passwd: "# comment line\n\nbaduid:x:notanumber:1001::/home/bad:/bin/sh\nshort:x:1001\nbadgid:x:1002:notanumber::/home/badgid:/bin/sh\nvalid:x:1000:1000:Valid User:/home/valid:/bin/bash\n",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "valid"}).
					Return("valid", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "valid"}).
					Return("valid P 01/01/2026 0 99999 7 -1", nil)
			},
			validateFunc: func(result []user.User, err error) {
				suite.NoError(err)
				suite.Require().Len(result, 1)
				suite.Equal("valid", result[0].Name)
				suite.Equal(1000, result[0].UID)
				suite.Equal(1000, result[0].GID)
			},
		},
		{
			name:   "when read error occurs returns scanner error",
			passwd: "valid:x:1001:1001::/home/valid:/bin/bash\n",
			setup: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				_ = baseFs.WriteFile(
					"/etc/passwd",
					[]byte("valid:x:1001:1001::/home/valid:/bin/bash\n"),
					0o644,
				)

				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnFileRead {
						return errors.New("I/O error")
					}

					return nil
				})
				suite.provider = user.NewDebianProvider(
					suite.logger,
					vfs,
					suite.mockExec,
				)
			},
			validateFunc: func(result []user.User, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "I/O error")
			},
		},
		{
			name:   "when exec error on groups lookup",
			passwd: "testuser:x:1000:1000:Test:/home/testuser:/bin/bash\n",
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "testuser"}).
					Return("", errors.New("id failed"))
			},
			validateFunc: func(result []user.User, err error) {
				suite.NoError(err)
				suite.Require().Len(result, 1)
				suite.Equal("testuser", result[0].Name)
				suite.Nil(result[0].Groups)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.passwd != "" {
				suite.writePasswd(tc.passwd)
			}

			tc.setup()

			result, err := suite.provider.ListUsers(suite.ctx)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestGetUser() {
	tests := []struct {
		name         string
		userName     string
		passwd       string
		setup        func()
		validateFunc func(*user.User, error)
	}{
		{
			name:     "when user found",
			userName: "john",
			passwd:   passwdContent,
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "john"}).
					Return("john sudo docker", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "john"}).
					Return("john P 01/01/2026 0 99999 7 -1", nil)
			},
			validateFunc: func(result *user.User, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("john", result.Name)
				suite.Equal(1000, result.UID)
				suite.Equal(1000, result.GID)
				suite.Equal("/home/john", result.Home)
				suite.Equal("/bin/bash", result.Shell)
				suite.Equal([]string{"john", "sudo", "docker"}, result.Groups)
				suite.False(result.Locked)
			},
		},
		{
			name:     "when user not found",
			userName: "nonexistent",
			passwd:   passwdContent,
			setup:    func() {},
			validateFunc: func(result *user.User, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name:     "when passwd file not found",
			userName: "john",
			passwd:   "",
			setup:    func() {},
			validateFunc: func(result *user.User, err error) {
				suite.Error(err)
				suite.Nil(result)
			},
		},
		{
			name:     "when groups lookup succeeds but passwd status fails",
			userName: "john",
			passwd:   passwdContent,
			setup: func() {
				suite.mockExec.EXPECT().
					RunCmd("id", []string{"-Gn", "john"}).
					Return("john sudo", nil)
				suite.mockExec.EXPECT().
					RunCmd("passwd", []string{"-S", "john"}).
					Return("", errors.New("passwd failed"))
			},
			validateFunc: func(result *user.User, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("john", result.Name)
				suite.Equal([]string{"john", "sudo"}, result.Groups)
				suite.False(result.Locked)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.passwd != "" {
				suite.writePasswd(tc.passwd)
			}

			tc.setup()

			result, err := suite.provider.GetUser(suite.ctx, tc.userName)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestCreateUser() {
	tests := []struct {
		name         string
		opts         user.CreateUserOpts
		setup        func()
		validateFunc func(*user.Result, error)
	}{
		{
			name: "when minimal create succeeds",
			opts: user.CreateUserOpts{
				Name: "newuser",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("useradd", []string{"--create-home", "newuser"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("newuser", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when create with all options",
			opts: user.CreateUserOpts{
				Name:   "newuser",
				UID:    2000,
				GID:    2000,
				Home:   "/opt/newuser",
				Shell:  "/bin/zsh",
				Groups: []string{"sudo", "docker"},
				System: true,
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("useradd", []string{
						"--create-home",
						"-u", "2000",
						"-g", "2000",
						"-d", "/opt/newuser",
						"-s", "/bin/zsh",
						"-G", "sudo,docker",
						"-r",
						"newuser",
					}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("newuser", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when create with password",
			opts: user.CreateUserOpts{
				Name:     "newuser",
				Password: "secret123",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("useradd", []string{"--create-home", "newuser"}).
					Return("", nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sh", []string{"-c", "echo 'newuser:secret123' | chpasswd"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("newuser", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when useradd fails",
			opts: user.CreateUserOpts{
				Name: "newuser",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("useradd", []string{"--create-home", "newuser"}).
					Return("", errors.New("user already exists"))
			},
			validateFunc: func(result *user.Result, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "useradd failed")
			},
		},
		{
			name: "when password set fails after create",
			opts: user.CreateUserOpts{
				Name:     "newuser",
				Password: "secret123",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("useradd", []string{"--create-home", "newuser"}).
					Return("", nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sh", []string{"-c", "echo 'newuser:secret123' | chpasswd"}).
					Return("", errors.New("chpasswd failed"))
			},
			validateFunc: func(result *user.Result, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "set password failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.CreateUser(suite.ctx, tc.opts)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestUpdateUser() {
	lockTrue := true
	lockFalse := false

	tests := []struct {
		name         string
		userName     string
		opts         user.UpdateUserOpts
		setup        func()
		validateFunc func(*user.Result, error)
	}{
		{
			name:     "when shell change",
			userName: "john",
			opts: user.UpdateUserOpts{
				Shell: "/bin/zsh",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("usermod", []string{"-s", "/bin/zsh", "john"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("john", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when groups change",
			userName: "john",
			opts: user.UpdateUserOpts{
				Groups: []string{"sudo", "docker", "admin"},
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("usermod", []string{"-G", "sudo,docker,admin", "john"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when lock user",
			userName: "john",
			opts: user.UpdateUserOpts{
				Lock: &lockTrue,
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("usermod", []string{"-L", "john"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when unlock user",
			userName: "john",
			opts: user.UpdateUserOpts{
				Lock: &lockFalse,
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("usermod", []string{"-U", "john"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when home change with move",
			userName: "john",
			opts: user.UpdateUserOpts{
				Home: "/opt/john",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("usermod", []string{"-d", "/opt/john", "-m", "john"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when no changes specified",
			userName: "john",
			opts:     user.UpdateUserOpts{},
			setup:    func() {},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("john", result.Name)
				suite.False(result.Changed)
			},
		},
		{
			name:     "when usermod fails",
			userName: "john",
			opts: user.UpdateUserOpts{
				Shell: "/bin/zsh",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("usermod", []string{"-s", "/bin/zsh", "john"}).
					Return("", errors.New("usermod error"))
			},
			validateFunc: func(result *user.Result, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "usermod failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.UpdateUser(suite.ctx, tc.userName, tc.opts)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestDeleteUser() {
	tests := []struct {
		name         string
		userName     string
		setup        func()
		validateFunc func(*user.Result, error)
	}{
		{
			name:     "when delete succeeds",
			userName: "john",
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("userdel", []string{"-r", "john"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("john", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when userdel fails",
			userName: "nonexistent",
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("userdel", []string{"-r", "nonexistent"}).
					Return("", errors.New("user does not exist"))
			},
			validateFunc: func(result *user.Result, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "userdel failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.DeleteUser(suite.ctx, tc.userName)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestChangePassword() {
	tests := []struct {
		name         string
		userName     string
		password     string
		setup        func()
		validateFunc func(*user.Result, error)
	}{
		{
			name:     "when password change succeeds",
			userName: "john",
			password: "newpassword",
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sh", []string{"-c", "echo 'john:newpassword' | chpasswd"}).
					Return("", nil)
			},
			validateFunc: func(result *user.Result, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("john", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:     "when chpasswd fails",
			userName: "john",
			password: "newpassword",
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sh", []string{"-c", "echo 'john:newpassword' | chpasswd"}).
					Return("", errors.New("chpasswd error"))
			},
			validateFunc: func(result *user.Result, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "chpasswd failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.ChangePassword(suite.ctx, tc.userName, tc.password)
			tc.validateFunc(result, err)
		})
	}
}

// Group tests

func (suite *DebianPublicTestSuite) TestListGroups() {
	tests := []struct {
		name         string
		groupContent string
		setup        func()
		validateFunc func([]user.Group, error)
	}{
		{
			name:         "when successful",
			groupContent: groupContent,
			setup:        func() {},
			validateFunc: func(result []user.Group, err error) {
				suite.NoError(err)
				suite.Require().Len(result, 5)

				suite.Equal("root", result[0].Name)
				suite.Equal(0, result[0].GID)
				suite.Nil(result[0].Members)

				suite.Equal("sudo", result[2].Name)
				suite.Equal(27, result[2].GID)
				suite.Equal([]string{"john", "jane"}, result[2].Members)

				suite.Equal("developers", result[4].Name)
				suite.Equal(1001, result[4].GID)
				suite.Equal([]string{"john", "jane", "bob"}, result[4].Members)
			},
		},
		{
			name:         "when group has corrupt lines skips them",
			groupContent: "# comment\n\nshort:x\nbadgid:x:notanumber:\nvalid:x:100:john,jane\n",
			setup:        func() {},
			validateFunc: func(result []user.Group, err error) {
				suite.NoError(err)
				suite.Require().Len(result, 1)
				suite.Equal("valid", result[0].Name)
				suite.Equal(100, result[0].GID)
				suite.Equal([]string{"john", "jane"}, result[0].Members)
			},
		},
		{
			name:         "when group file not found",
			groupContent: "",
			setup:        func() {},
			validateFunc: func(result []user.Group, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "group:")
			},
		},
		{
			name:         "when read error occurs returns scanner error",
			groupContent: "root:x:0:\n",
			setup: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				_ = baseFs.WriteFile(
					"/etc/group",
					[]byte("root:x:0:\n"),
					0o644,
				)

				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnFileRead {
						return errors.New("I/O error")
					}

					return nil
				})
				suite.provider = user.NewDebianProvider(
					suite.logger,
					vfs,
					suite.mockExec,
				)
			},
			validateFunc: func(result []user.Group, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "I/O error")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.groupContent != "" {
				suite.writeGroup(tc.groupContent)
			}

			tc.setup()

			result, err := suite.provider.ListGroups(suite.ctx)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestGetGroup() {
	tests := []struct {
		name         string
		groupName    string
		groupContent string
		validateFunc func(*user.Group, error)
	}{
		{
			name:         "when group found",
			groupName:    "sudo",
			groupContent: groupContent,
			validateFunc: func(result *user.Group, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("sudo", result.Name)
				suite.Equal(27, result.GID)
				suite.Equal([]string{"john", "jane"}, result.Members)
			},
		},
		{
			name:         "when group not found",
			groupName:    "nonexistent",
			groupContent: groupContent,
			validateFunc: func(result *user.Group, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name:         "when group file not found",
			groupName:    "sudo",
			groupContent: "",
			validateFunc: func(result *user.Group, err error) {
				suite.Error(err)
				suite.Nil(result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if tc.groupContent != "" {
				suite.writeGroup(tc.groupContent)
			}

			result, err := suite.provider.GetGroup(suite.ctx, tc.groupName)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestCreateGroup() {
	tests := []struct {
		name         string
		opts         user.CreateGroupOpts
		setup        func()
		validateFunc func(*user.GroupResult, error)
	}{
		{
			name: "when minimal create succeeds",
			opts: user.CreateGroupOpts{
				Name: "newgroup",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("groupadd", []string{"newgroup"}).
					Return("", nil)
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("newgroup", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when create with GID",
			opts: user.CreateGroupOpts{
				Name: "newgroup",
				GID:  5000,
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("groupadd", []string{"-g", "5000", "newgroup"}).
					Return("", nil)
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("newgroup", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when create with system flag",
			opts: user.CreateGroupOpts{
				Name:   "sysgroup",
				System: true,
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("groupadd", []string{"-r", "sysgroup"}).
					Return("", nil)
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("sysgroup", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name: "when groupadd fails",
			opts: user.CreateGroupOpts{
				Name: "newgroup",
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("groupadd", []string{"newgroup"}).
					Return("", errors.New("group already exists"))
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "groupadd failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.CreateGroup(suite.ctx, tc.opts)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestUpdateGroup() {
	tests := []struct {
		name         string
		groupName    string
		opts         user.UpdateGroupOpts
		setup        func()
		validateFunc func(*user.GroupResult, error)
	}{
		{
			name:      "when update members succeeds",
			groupName: "developers",
			opts: user.UpdateGroupOpts{
				Members: []string{"john", "jane", "alice"},
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("gpasswd", []string{"-M", "john,jane,alice", "developers"}).
					Return("", nil)
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("developers", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:      "when gpasswd fails",
			groupName: "developers",
			opts: user.UpdateGroupOpts{
				Members: []string{"john"},
			},
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("gpasswd", []string{"-M", "john", "developers"}).
					Return("", errors.New("gpasswd error"))
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "gpasswd failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.UpdateGroup(suite.ctx, tc.groupName, tc.opts)
			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestDeleteGroup() {
	tests := []struct {
		name         string
		groupName    string
		setup        func()
		validateFunc func(*user.GroupResult, error)
	}{
		{
			name:      "when delete succeeds",
			groupName: "developers",
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("groupdel", []string{"developers"}).
					Return("", nil)
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.Equal("developers", result.Name)
				suite.True(result.Changed)
			},
		},
		{
			name:      "when groupdel fails",
			groupName: "nonexistent",
			setup: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("groupdel", []string{"nonexistent"}).
					Return("", errors.New("group does not exist"))
			},
			validateFunc: func(result *user.GroupResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "groupdel failed")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.DeleteGroup(suite.ctx, tc.groupName)
			tc.validateFunc(result, err)
		})
	}
}

func TestDebianPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianPublicTestSuite))
}
