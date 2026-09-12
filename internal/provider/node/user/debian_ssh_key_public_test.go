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
	"fmt"
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

const (
	// Valid ed25519 key line for testing.
	testKey1Line = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHRlc3RrZXkxZGF0YQ== user@host"
	testKey1FP   = "SHA256:fs7cRe+Lieb9g9TQ7a4HbYTDyVWnO8tXg6D9H2cAWIY"

	// Valid RSA key line for testing.
	testKey2Line = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQC7 admin@server"
	testKey2FP   = "SHA256:BgjHH5Pzls0x0ceexhHl0tFm6EBSFKWukOczrQrdl9Y"

	testPasswdSSH = `root:x:0:0:root:/root:/bin/bash
testuser:x:1000:1000:Test:/home/testuser:/bin/bash
`
)

type DebianSSHKeyPublicTestSuite struct {
	suite.Suite

	ctrl     *gomock.Controller
	ctx      context.Context
	logger   *slog.Logger
	memFs    avfs.VFS
	mockExec *execmocks.MockManager
	provider *user.Debian
}

func (suite *DebianSSHKeyPublicTestSuite) SetupTest() {
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

func (suite *DebianSSHKeyPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianSSHKeyPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianSSHKeyPublicTestSuite) writePasswd(
	content string,
) {
	_ = suite.memFs.MkdirAll("/etc", 0o755)

	f, err := suite.memFs.Create("/etc/passwd")
	suite.Require().NoError(err)

	_, err = f.Write([]byte(content))
	suite.Require().NoError(err)
	suite.Require().NoError(f.Close())
}

func (suite *DebianSSHKeyPublicTestSuite) writeAuthorizedKeys(
	_ string,
	homeDir string,
	content string,
) {
	sshDir := homeDir + "/.ssh"
	_ = suite.memFs.MkdirAll(sshDir, 0o700)

	f, err := suite.memFs.Create(sshDir + "/authorized_keys")
	suite.Require().NoError(err)

	_, err = f.Write([]byte(content))
	suite.Require().NoError(err)
	suite.Require().NoError(f.Close())
}

func (suite *DebianSSHKeyPublicTestSuite) readFile(
	path string,
) string {
	content, err := suite.memFs.ReadFile(path)
	suite.Require().NoError(err)

	return string(content)
}

// newFailFSProvider creates a provider backed by failfs wrapping memfs.
// The caller must write /etc/passwd and any other files to baseFs before
// setting the fail function.
func (suite *DebianSSHKeyPublicTestSuite) newFailFSProvider(
	baseFs avfs.VFS,
	ff failfs.FailFunc,
) *user.Debian {
	ffs := failfs.New(baseFs)
	_ = ffs.SetFailFunc(ff)

	return user.NewDebianProvider(
		suite.logger,
		ffs,
		suite.mockExec,
	)
}

func (suite *DebianSSHKeyPublicTestSuite) TestListKeys() {
	tests := []struct {
		name         string
		username     string
		passwd       string
		skipPasswd   bool
		setupFS      func()
		validateFunc func([]user.SSHKey, error)
	}{
		{
			name:     "when successful with two keys",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS: func() {
				suite.writeAuthorizedKeys(
					"testuser",
					"/home/testuser",
					testKey1Line+"\n"+testKey2Line+"\n",
				)
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Require().Len(keys, 2)

				suite.Equal("ssh-ed25519", keys[0].Type)
				suite.Equal(testKey1FP, keys[0].Fingerprint)
				suite.Equal("user@host", keys[0].Comment)
				suite.Empty(keys[0].RawLine)

				suite.Equal("ssh-rsa", keys[1].Type)
				suite.Equal(testKey2FP, keys[1].Fingerprint)
				suite.Equal("admin@server", keys[1].Comment)
				suite.Empty(keys[1].RawLine)
			},
		},
		{
			name:     "when user not found in passwd",
			username: "nonexistent",
			passwd:   testPasswdSSH,
			setupFS:  func() {},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.Error(err)
				suite.Nil(keys)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name:       "when passwd file missing",
			username:   "testuser",
			skipPasswd: true,
			setupFS:    func() {},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.Error(err)
				suite.Nil(keys)
				suite.Contains(err.Error(), "ssh key: list")
			},
		},
		{
			name:     "when no authorized_keys file",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS:  func() {},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Empty(keys)
			},
		},
		{
			name:     "when authorized_keys is empty",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS: func() {
				suite.writeAuthorizedKeys("testuser", "/home/testuser", "")
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Empty(keys)
			},
		},
		{
			name:     "when comment lines and blank lines are skipped",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS: func() {
				content := "# This is a comment\n\n" + testKey1Line + "\n\n# Another comment\n"
				suite.writeAuthorizedKeys("testuser", "/home/testuser", content)
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Require().Len(keys, 1)
				suite.Equal(testKey1FP, keys[0].Fingerprint)
			},
		},
		{
			name:     "when malformed line with only one field is skipped",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS: func() {
				content := "onlyonefield\n" + testKey1Line + "\n"
				suite.writeAuthorizedKeys("testuser", "/home/testuser", content)
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Require().Len(keys, 1)
				suite.Equal(testKey1FP, keys[0].Fingerprint)
			},
		},
		{
			name:     "when invalid base64 in key data is skipped",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS: func() {
				content := "ssh-rsa !!!invalid-base64!!! bad@key\n" + testKey1Line + "\n"
				suite.writeAuthorizedKeys("testuser", "/home/testuser", content)
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Require().Len(keys, 1)
				suite.Equal(testKey1FP, keys[0].Fingerprint)
			},
		},
		{
			name:       "when read file fails with non-NotExist error",
			username:   "testuser",
			passwd:     testPasswdSSH,
			skipPasswd: true,
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()
				sshDir := "/home/testuser/.ssh"
				_ = baseFs.MkdirAll(sshDir, 0o700)
				af, _ := baseFs.Create(sshDir + "/authorized_keys")
				_, _ = af.Write([]byte(testKey1Line + "\n"))
				_ = af.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnReadFile {
							return fmt.Errorf("injected I/O error")
						}

						return nil
					})
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.Error(err)
				suite.Nil(keys)
				suite.Contains(err.Error(), "ssh key: list: read")
			},
		},
		{
			name:       "when passwd read fails mid-scan",
			username:   "testuser",
			skipPasswd: true,
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte("other:x:1000:1000:Other:/home/other:/bin/bash\n"))
				_ = f.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnFileRead {
							return fmt.Errorf("injected read error")
						}

						return nil
					})
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.Error(err)
				suite.Nil(keys)
				suite.Contains(err.Error(), "ssh key: list")
			},
		},
		{
			name:     "when passwd has comments and malformed lines",
			username: "testuser",
			passwd:   "# comment\n\nshort:line\ntestuser:x:1000:1000:Test:/home/testuser:/bin/bash\n",
			setupFS: func() {
				suite.writeAuthorizedKeys(
					"testuser",
					"/home/testuser",
					testKey1Line+"\n",
				)
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Require().Len(keys, 1)
				suite.Equal(testKey1FP, keys[0].Fingerprint)
			},
		},
		{
			name:     "when key has no comment",
			username: "testuser",
			passwd:   testPasswdSSH,
			setupFS: func() {
				content := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHRlc3RrZXkxZGF0YQ==\n"
				suite.writeAuthorizedKeys("testuser", "/home/testuser", content)
			},
			validateFunc: func(keys []user.SSHKey, err error) {
				suite.NoError(err)
				suite.Require().Len(keys, 1)
				suite.Equal("ssh-ed25519", keys[0].Type)
				suite.Equal(testKey1FP, keys[0].Fingerprint)
				suite.Empty(keys[0].Comment)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if !tc.skipPasswd {
				suite.writePasswd(tc.passwd)
			}
			tc.setupFS()

			result, err := suite.provider.ListKeys(suite.ctx, tc.username)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianSSHKeyPublicTestSuite) TestAddKey() {
	tests := []struct {
		name         string
		username     string
		passwd       string
		skipPasswd   bool
		key          user.SSHKey
		setupFS      func()
		setupMock    func()
		validateFunc func(*user.SSHKeyResult, error)
	}{
		{
			name:     "when successful appends key and runs chown",
			username: "testuser",
			passwd:   testPasswdSSH,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chown", []string{"-R", "testuser:testuser", "/home/testuser/.ssh"}).
					Return("", nil)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)

				content := suite.readFile("/home/testuser/.ssh/authorized_keys")
				suite.Contains(content, testKey1Line)
			},
		},
		{
			name:     "when key already exists returns changed false",
			username: "testuser",
			passwd:   testPasswdSSH,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {
				suite.writeAuthorizedKeys("testuser", "/home/testuser", testKey1Line+"\n")
			},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.False(result.Changed)
			},
		},
		{
			name:     "when user not found returns error",
			username: "nonexistent",
			passwd:   testPasswdSSH,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS:   func() {},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name:     "when ssh dir and file are missing creates them",
			username: "testuser",
			passwd:   testPasswdSSH,
			key: user.SSHKey{
				RawLine: testKey2Line,
			},
			setupFS: func() {
				_ = suite.memFs.MkdirAll("/home/testuser", 0o755)
			},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chown", []string{"-R", "testuser:testuser", "/home/testuser/.ssh"}).
					Return("", nil)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)

				content := suite.readFile("/home/testuser/.ssh/authorized_keys")
				suite.Contains(content, testKey2Line)
			},
		},
		{
			name:       "when passwd file missing returns error",
			username:   "testuser",
			skipPasswd: true,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS:   func() {},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: add")
			},
		},
		{
			name:     "when raw line is malformed skips duplicate check",
			username: "testuser",
			passwd:   testPasswdSSH,
			key: user.SSHKey{
				RawLine: "malformed-single-field",
			},
			setupFS: func() {
				suite.writeAuthorizedKeys("testuser", "/home/testuser", testKey1Line+"\n")
			},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chown", []string{"-R", "testuser:testuser", "/home/testuser/.ssh"}).
					Return("", nil)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
		{
			name:       "when read file fails with non-NotExist error",
			username:   "testuser",
			skipPasswd: true,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()
				sshDir := "/home/testuser/.ssh"
				_ = baseFs.MkdirAll(sshDir, 0o700)
				af, _ := baseFs.Create(sshDir + "/authorized_keys")
				_, _ = af.Write([]byte(testKey2Line + "\n"))
				_ = af.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnReadFile {
							return fmt.Errorf("injected I/O error")
						}

						return nil
					})
			},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: add: read")
			},
		},
		{
			name:       "when mkdir fails",
			username:   "testuser",
			skipPasswd: true,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnMkdirAll {
							return fmt.Errorf("injected mkdir error")
						}

						return nil
					})
			},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: add: mkdir")
			},
		},
		{
			name:       "when open file fails for append",
			username:   "testuser",
			skipPasswd: true,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()

				openFileCalls := 0

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnOpenFile {
							openFileCalls++
							// 1st: userHomeDir Open("/etc/passwd")
							// 2nd: ReadFile's internal OpenFile (not-exist)
							// 3rd: OpenFile for append
							if openFileCalls > 2 {
								return fmt.Errorf("injected open error")
							}
						}

						return nil
					})
			},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: add: open")
			},
		},
		{
			name:       "when write fails",
			username:   "testuser",
			skipPasswd: true,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnFileWrite {
							return fmt.Errorf("injected write error")
						}

						return nil
					})
			},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: add: write")
			},
		},
		{
			name:       "when file close fails after write",
			username:   "testuser",
			skipPasswd: true,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnFileClose {
							return fmt.Errorf("injected close error")
						}

						return nil
					})
			},
			setupMock: func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: add")
			},
		},
		{
			name:     "when chown fails still returns changed true",
			username: "testuser",
			passwd:   testPasswdSSH,
			key: user.SSHKey{
				RawLine: testKey1Line,
			},
			setupFS: func() {},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chown", []string{"-R", "testuser:testuser", "/home/testuser/.ssh"}).
					Return("", errors.New("permission denied"))
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if !tc.skipPasswd {
				suite.writePasswd(tc.passwd)
			}
			tc.setupFS()
			tc.setupMock()

			result, err := suite.provider.AddKey(suite.ctx, tc.username, tc.key)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianSSHKeyPublicTestSuite) TestRemoveKey() {
	tests := []struct {
		name         string
		username     string
		passwd       string
		skipPasswd   bool
		fingerprint  string
		setupFS      func()
		validateFunc func(*user.SSHKeyResult, error)
	}{
		{
			name:        "when successful removes matching key",
			username:    "testuser",
			passwd:      testPasswdSSH,
			fingerprint: testKey1FP,
			setupFS: func() {
				suite.writeAuthorizedKeys(
					"testuser",
					"/home/testuser",
					testKey1Line+"\n"+testKey2Line+"\n",
				)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)

				content := suite.readFile("/home/testuser/.ssh/authorized_keys")
				suite.NotContains(content, testKey1Line)
				suite.Contains(content, testKey2Line)
			},
		},
		{
			name:        "when fingerprint not found returns changed false",
			username:    "testuser",
			passwd:      testPasswdSSH,
			fingerprint: "SHA256:nonexistent",
			setupFS: func() {
				suite.writeAuthorizedKeys(
					"testuser",
					"/home/testuser",
					testKey1Line+"\n",
				)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when user not found returns error",
			username:    "nonexistent",
			passwd:      testPasswdSSH,
			fingerprint: testKey1FP,
			setupFS:     func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name:        "when no authorized_keys file returns changed false",
			username:    "testuser",
			passwd:      testPasswdSSH,
			fingerprint: testKey1FP,
			setupFS:     func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.False(result.Changed)
			},
		},
		{
			name:        "when passwd file missing returns error",
			username:    "testuser",
			skipPasswd:  true,
			fingerprint: testKey1FP,
			setupFS:     func() {},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: remove")
			},
		},
		{
			name:        "when read file fails with non-NotExist error",
			username:    "testuser",
			skipPasswd:  true,
			fingerprint: testKey1FP,
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()
				sshDir := "/home/testuser/.ssh"
				_ = baseFs.MkdirAll(sshDir, 0o700)
				af, _ := baseFs.Create(sshDir + "/authorized_keys")
				_, _ = af.Write([]byte(testKey1Line + "\n"))
				_ = af.Close()

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						if fn == avfs.FnReadFile {
							return fmt.Errorf("injected I/O error")
						}

						return nil
					})
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: remove: read")
			},
		},
		{
			name:        "when write file fails on rewrite",
			username:    "testuser",
			skipPasswd:  true,
			fingerprint: testKey1FP,
			setupFS: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc", 0o755)
				f, _ := baseFs.Create("/etc/passwd")
				_, _ = f.Write([]byte(testPasswdSSH))
				_ = f.Close()
				sshDir := "/home/testuser/.ssh"
				_ = baseFs.MkdirAll(sshDir, 0o700)
				af, _ := baseFs.Create(sshDir + "/authorized_keys")
				_, _ = af.Write([]byte(testKey1Line + "\n"))
				_ = af.Close()

				openFileCalls := 0

				suite.provider = suite.newFailFSProvider(baseFs,
					func(_ avfs.VFSBase, fn avfs.FnVFS, _ *failfs.FailParam) error {
						// WriteFile uses OpenFile internally.
						// The first OpenFile is from ReadFile, the second is
						// from WriteFile (the rewrite). Fail the second.
						if fn == avfs.FnOpenFile {
							openFileCalls++
							// First OpenFile: userHomeDir Open("/etc/passwd")
							// Second OpenFile: ReadFile's internal OpenFile
							// Third OpenFile: WriteFile's internal OpenFile
							if openFileCalls > 2 {
								return fmt.Errorf("injected write error")
							}
						}

						return nil
					})
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "ssh key: remove: write")
			},
		},
		{
			name:        "when preserves comment lines and blank lines",
			username:    "testuser",
			passwd:      testPasswdSSH,
			fingerprint: testKey1FP,
			setupFS: func() {
				content := "# Header comment\n\n" + testKey1Line + "\n" + testKey2Line + "\n"
				suite.writeAuthorizedKeys("testuser", "/home/testuser", content)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)

				content := suite.readFile("/home/testuser/.ssh/authorized_keys")
				suite.Contains(content, "# Header comment")
				suite.Contains(content, testKey2Line)
				suite.NotContains(content, testKey1Line)
			},
		},
		{
			name:        "when file has single key after removal",
			username:    "testuser",
			passwd:      testPasswdSSH,
			fingerprint: testKey1FP,
			setupFS: func() {
				suite.writeAuthorizedKeys(
					"testuser",
					"/home/testuser",
					testKey1Line+"\n",
				)
			},
			validateFunc: func(result *user.SSHKeyResult, err error) {
				suite.NoError(err)
				suite.Require().NotNil(result)
				suite.True(result.Changed)

				content := suite.readFile("/home/testuser/.ssh/authorized_keys")
				suite.NotContains(content, testKey1Line)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if !tc.skipPasswd {
				suite.writePasswd(tc.passwd)
			}
			tc.setupFS()

			result, err := suite.provider.RemoveKey(
				suite.ctx,
				tc.username,
				tc.fingerprint,
			)

			tc.validateFunc(result, err)
		})
	}
}

func TestDebianSSHKeyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianSSHKeyPublicTestSuite))
}
