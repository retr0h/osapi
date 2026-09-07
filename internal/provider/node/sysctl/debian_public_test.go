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

package sysctl_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/osapi-io/osapi/internal/job"
	jobmocks "github.com/osapi-io/osapi/internal/job/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/sysctl"
)

const testHostname = "test-host"

// managedStateJSON returns a JSON-encoded FileState with sysctl metadata.
func managedStateJSON(
	key string,
	value string,
	path string,
) []byte {
	state := job.FileState{
		Path:       path,
		SHA256:     "abc123",
		Mode:       "0644",
		DeployedAt: "2026-01-01T00:00:00Z",
		Metadata: map[string]string{
			"key":   key,
			"value": value,
		},
	}

	b, _ := json.Marshal(state)

	return b
}

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl        *gomock.Controller
	logger      *slog.Logger
	memFs       avfs.VFS
	mockStateKV *jobmocks.MockKeyValue
	mockExec    *execmocks.MockManager
	provider    *sysctl.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockStateKV = jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	_ = suite.memFs.MkdirAll("/etc/sysctl.d", 0o755)

	suite.provider = sysctl.NewDebianProvider(
		suite.logger,
		suite.memFs,
		suite.mockStateKV,
		suite.mockExec,
		testHostname,
	)
}

func (suite *DebianPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianPublicTestSuite) TearDownSubTest() {
	sysctl.ResetMarshalJSON()
}

func (suite *DebianPublicTestSuite) TestCreate() {
	tests := []struct {
		name         string
		entry        sysctl.Entry
		setup        func()
		validateFunc func(*sysctl.CreateResult, error)
	}{
		{
			name: "when deploy succeeds",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				// First Get in Create (check if already managed) => not found.
				// Second Get in deploy (idempotency check) => not found.
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found")).
					Times(2)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sysctl", []string{"-p", "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf"}).
					Return("", nil)
			},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("net.ipv4.ip_forward", result.Key)
				suite.True(result.Changed)

				// Verify file was written.
				content, readErr := suite.memFs.ReadFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				suite.NoError(readErr)
				suite.Equal("net.ipv4.ip_forward = 1\n", string(content))
			},
		},
		{
			name: "when key already managed returns unchanged",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				stateBytes := managedStateJSON(
					"net.ipv4.ip_forward",
					"0",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.NotNil(result)
				suite.Equal("net.ipv4.ip_forward", result.Key)
				suite.False(result.Changed)
			},
		},
		{
			name: "when key is empty",
			entry: sysctl.Entry{
				Key:   "",
				Value: "1",
			},
			setup: func() {},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "key must not be empty")
			},
		},
		{
			name: "when value is empty",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "",
			},
			setup: func() {},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "value must not be empty")
			},
		},
		{
			name: "when previously undeployed allows create",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				content := []byte("net.ipv4.ip_forward = 1\n")
				state := job.FileState{
					Path:         "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					SHA256:       computeTestSHA256(content),
					Mode:         "0644",
					DeployedAt:   "2026-01-01T00:00:00Z",
					UndeployedAt: "2026-02-01T00:00:00Z",
					Metadata: map[string]string{
						"key":   "net.ipv4.ip_forward",
						"value": "1",
					},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes).Times(2)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil).
					Times(2)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sysctl", gomock.Any()).
					Return("", nil)
			},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.True(result.Changed)
			},
		},
		{
			name: "when state put fails",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found")).
					Times(2)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("kv put error"))
			},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "update state")
			},
		},
		{
			name: "when marshal state fails",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found")).
					Times(2)
				sysctl.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, errors.New("marshal error")
				})
			},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "marshal state")
			},
		},
		{
			name: "when sysctl apply fails it still succeeds",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found")).
					Times(2)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sysctl", gomock.Any()).
					Return("", errors.New("sysctl failed"))
			},
			validateFunc: func(
				result *sysctl.CreateResult,
				err error,
			) {
				suite.NoError(err)
				suite.True(result.Changed)
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
		entry        sysctl.Entry
		setup        func()
		validateFunc func(*sysctl.UpdateResult, error)
	}{
		{
			name: "when update succeeds",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "0",
			},
			setup: func() {
				stateBytes := managedStateJSON(
					"net.ipv4.ip_forward",
					"1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes).Times(2)

				// First Get in Update (check managed), second Get in deploy (idempotency).
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil).
					Times(2)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sysctl", []string{"-p", "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf"}).
					Return("", nil)
			},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("net.ipv4.ip_forward", result.Key)
				suite.True(result.Changed)
			},
		},
		{
			name: "when key not managed returns error",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "0",
			},
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not managed")
			},
		},
		{
			name: "when key was undeployed returns error",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "0",
			},
			setup: func() {
				content := []byte("net.ipv4.ip_forward = 1\n")
				state := job.FileState{
					Path:         "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					SHA256:       computeTestSHA256(content),
					Mode:         "0644",
					DeployedAt:   "2026-01-01T00:00:00Z",
					UndeployedAt: "2026-02-01T00:00:00Z",
					Metadata: map[string]string{
						"key":   "net.ipv4.ip_forward",
						"value": "1",
					},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "not managed")
			},
		},
		{
			name: "when key is empty",
			entry: sysctl.Entry{
				Key:   "",
				Value: "1",
			},
			setup: func() {},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "key must not be empty")
			},
		},
		{
			name: "when value is empty",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "",
			},
			setup: func() {},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "value must not be empty")
			},
		},
		{
			name: "when content unchanged returns not changed",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "1",
			},
			setup: func() {
				// Write the file to disk so Stat succeeds.
				content := []byte("net.ipv4.ip_forward = 1\n")
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					content,
					0o644,
				)

				// Create state with matching SHA.
				state := job.FileState{
					Path:       "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					SHA256:     computeTestSHA256(content),
					Mode:       "0644",
					DeployedAt: "2026-01-01T00:00:00Z",
					Metadata: map[string]string{
						"key":   "net.ipv4.ip_forward",
						"value": "1",
					},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes).Times(2)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil).
					Times(2)
			},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("net.ipv4.ip_forward", result.Key)
				suite.False(result.Changed)
			},
		},
		{
			name: "when deploy MkdirAll fails returns error",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "0",
			},
			setup: func() {
				stateBytes := managedStateJSON(
					"net.ipv4.ip_forward",
					"1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				// First Get in Update (check managed), second Get in deploy (idempotency).
				mockEntry.EXPECT().Value().Return(stateBytes).Times(2)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil).
					Times(2)

				baseFs := memfs.New()
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnMkdirAll {
						return errors.New("mkdir failed")
					}

					return nil
				})
				suite.provider = sysctl.NewDebianProvider(
					suite.logger,
					vfs,
					suite.mockStateKV,
					suite.mockExec,
					testHostname,
				)
			},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "create directory")
			},
		},
		{
			name: "when deploy WriteFile fails returns error",
			entry: sysctl.Entry{
				Key:   "net.ipv4.ip_forward",
				Value: "0",
			},
			setup: func() {
				stateBytes := managedStateJSON(
					"net.ipv4.ip_forward",
					"1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				// First Get in Update (check managed), second Get in deploy (idempotency).
				mockEntry.EXPECT().Value().Return(stateBytes).Times(2)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil).
					Times(2)

				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc/sysctl.d", 0o755)
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnOpenFile {
						return errors.New("write failed")
					}

					return nil
				})
				suite.provider = sysctl.NewDebianProvider(
					suite.logger,
					vfs,
					suite.mockStateKV,
					suite.mockExec,
					testHostname,
				)
			},
			validateFunc: func(
				result *sysctl.UpdateResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "write file")
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
		key          string
		setup        func()
		validateFunc func(*sysctl.DeleteResult, error)
	}{
		{
			name: "when delete succeeds",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sysctl", []string{"--system"}).
					Return("", nil)
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("net.ipv4.ip_forward", result.Key)
				suite.True(result.Changed)
			},
		},
		{
			name:  "when key is empty",
			key:   "",
			setup: func() {},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "key must not be empty")
			},
		},
		{
			name: "when not managed returns not changed",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.False(result.Changed)
			},
		},
		{
			name: "when state unmarshal fails returns not changed",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.False(result.Changed)
			},
		},
		{
			name: "when already undeployed returns not changed",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				state := job.FileState{
					Path:         "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					SHA256:       "abc123",
					DeployedAt:   "2026-01-01T00:00:00Z",
					UndeployedAt: "2026-02-01T00:00:00Z",
					Metadata: map[string]string{
						"key":   "net.ipv4.ip_forward",
						"value": "1",
					},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.False(result.Changed)
			},
		},
		{
			name: "when file not on disk but state exists",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				// No file on disk.
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				// No sysctl --system call since changed is false.
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.False(result.Changed)
			},
		},
		{
			name: "when Remove fails returns error",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc/sysctl.d", 0o755)
				_ = baseFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnRemove {
						return errors.New("remove failed")
					}

					return nil
				})
				suite.provider = sysctl.NewDebianProvider(
					suite.logger,
					vfs,
					suite.mockStateKV,
					suite.mockExec,
					testHostname,
				)
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "remove file")
			},
		},
		{
			name: "when state put fails",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("kv put error"))
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "update state")
			},
		},
		{
			name: "when marshal state fails",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				sysctl.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, errors.New("marshal error")
				})
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.Error(err)
				suite.Nil(result)
				suite.Contains(err.Error(), "marshal state")
			},
		},
		{
			name: "when sysctl reload fails it still succeeds",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("sysctl", []string{"--system"}).
					Return("", errors.New("sysctl failed"))
			},
			validateFunc: func(
				result *sysctl.DeleteResult,
				err error,
			) {
				suite.NoError(err)
				suite.True(result.Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			result, err := suite.provider.Delete(context.Background(), tc.key)

			tc.validateFunc(result, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func([]sysctl.Entry, error)
	}{
		{
			name: "when managed entries exist",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)

				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockExec.EXPECT().
					RunCmd("sysctl", []string{"-n", "net.ipv4.ip_forward"}).
					Return("1\n", nil)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Len(entries, 1)
				suite.Equal("net.ipv4.ip_forward", entries[0].Key)
				suite.Equal("1", entries[0].Value)
			},
		},
		{
			name: "when non-osapi files are skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/99-custom.conf",
					[]byte("custom = 1\n"),
					0o644,
				)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when subdirectories are skipped",
			setup: func() {
				_ = suite.memFs.MkdirAll("/etc/sysctl.d/osapi-subdir", 0o755)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when sysctl.d dir read fails",
			setup: func() {
				badFs := memfs.New()
				suite.provider = sysctl.NewDebianProvider(
					suite.logger,
					badFs,
					suite.mockStateKV,
					suite.mockExec,
					testHostname,
				)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.Error(err)
				suite.Nil(entries)
				suite.Contains(err.Error(), "list sysctl entries")
			},
		},
		{
			name: "when state KV lookup fails entry is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("kv error"))
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when state unmarshal fails entry is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when entry is undeployed it is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)

				state := job.FileState{
					Path:         "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					SHA256:       "abc123",
					DeployedAt:   "2026-01-01T00:00:00Z",
					UndeployedAt: "2026-02-01T00:00:00Z",
					Metadata: map[string]string{
						"key":   "net.ipv4.ip_forward",
						"value": "1",
					},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when metadata has no key field entry is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-something.conf",
					[]byte("something = 1\n"),
					0o644,
				)

				state := job.FileState{
					Path:       "/etc/sysctl.d/osapi-something.conf",
					SHA256:     "abc123",
					DeployedAt: "2026-01-01T00:00:00Z",
					Metadata:   map[string]string{"other": "field"},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when metadata is nil entry is skipped",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-something.conf",
					[]byte("something = 1\n"),
					0o644,
				)

				state := job.FileState{
					Path:       "/etc/sysctl.d/osapi-something.conf",
					SHA256:     "abc123",
					DeployedAt: "2026-01-01T00:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
			},
		},
		{
			name: "when sysctl runtime read fails value from state is used",
			setup: func() {
				_ = suite.memFs.WriteFile(
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					[]byte("net.ipv4.ip_forward = 1\n"),
					0o644,
				)

				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockExec.EXPECT().
					RunCmd("sysctl", []string{"-n", "net.ipv4.ip_forward"}).
					Return("", errors.New("sysctl not found"))
			},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Len(entries, 1)
				suite.Equal("1", entries[0].Value)
			},
		},
		{
			name:  "when no entries exist",
			setup: func() {},
			validateFunc: func(
				entries []sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Empty(entries)
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

func (suite *DebianPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		key          string
		setup        func()
		validateFunc func(*sysctl.Entry, error)
	}{
		{
			name: "when managed entry found",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockExec.EXPECT().
					RunCmd("sysctl", []string{"-n", "net.ipv4.ip_forward"}).
					Return("1\n", nil)
			},
			validateFunc: func(
				entry *sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("net.ipv4.ip_forward", entry.Key)
				suite.Equal("1", entry.Value)
			},
		},
		{
			name:  "when key is empty",
			key:   "",
			setup: func() {},
			validateFunc: func(
				entry *sysctl.Entry,
				err error,
			) {
				suite.Error(err)
				suite.Nil(entry)
				suite.Contains(err.Error(), "key must not be empty")
			},
		},
		{
			name: "when not found in KV",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))
			},
			validateFunc: func(
				entry *sysctl.Entry,
				err error,
			) {
				suite.Error(err)
				suite.Nil(entry)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name: "when state unmarshal fails",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return([]byte("not-json"))

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entry *sysctl.Entry,
				err error,
			) {
				suite.Error(err)
				suite.Nil(entry)
				suite.Contains(err.Error(), "failed to read state")
			},
		},
		{
			name: "when entry is undeployed",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				state := job.FileState{
					Path:         "/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
					SHA256:       "abc123",
					DeployedAt:   "2026-01-01T00:00:00Z",
					UndeployedAt: "2026-02-01T00:00:00Z",
					Metadata: map[string]string{
						"key":   "net.ipv4.ip_forward",
						"value": "1",
					},
				}
				stateBytes, _ := json.Marshal(state)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(
				entry *sysctl.Entry,
				err error,
			) {
				suite.Error(err)
				suite.Nil(entry)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name: "when sysctl runtime read fails value from state is used",
			key:  "net.ipv4.ip_forward",
			setup: func() {
				stateData := managedStateJSON(
					"net.ipv4.ip_forward", "1",
					"/etc/sysctl.d/osapi-net.ipv4.ip_forward.conf",
				)
				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateData)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
				suite.mockExec.EXPECT().
					RunCmd("sysctl", []string{"-n", "net.ipv4.ip_forward"}).
					Return("", errors.New("sysctl not found"))
			},
			validateFunc: func(
				entry *sysctl.Entry,
				err error,
			) {
				suite.NoError(err)
				suite.Equal("1", entry.Value)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			entry, err := suite.provider.Get(context.Background(), tc.key)

			tc.validateFunc(entry, err)
		})
	}
}

// computeTestSHA256 returns the hex-encoded SHA-256 hash of the given data.
// Test-only helper that mirrors the provider's internal function.
func computeTestSHA256(
	data []byte,
) string {
	h := sha256.Sum256(data)

	return hex.EncodeToString(h[:])
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianPublicTestSuite))
}
