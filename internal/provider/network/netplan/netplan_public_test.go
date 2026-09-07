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

package netplan_test

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
	"github.com/osapi-io/osapi/internal/provider/network/netplan"
)

const (
	testHostname = "test-host"
	testPath     = "/etc/netplan/99-osapi-dns.yaml"
)

var testContent = []byte(
	"network:\n  ethernets:\n    eth0:\n      nameservers:\n        addresses:\n          - 8.8.8.8\n",
)

func testSHA() string {
	h := sha256.Sum256(testContent)

	return hex.EncodeToString(h[:])
}

type NetplanPublicTestSuite struct {
	suite.Suite

	ctrl        *gomock.Controller
	ctx         context.Context
	logger      *slog.Logger
	memFs       avfs.VFS
	mockStateKV *jobmocks.MockKeyValue
	mockExec    *execmocks.MockManager
}

func (suite *NetplanPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockStateKV = jobmocks.NewMockKeyValue(suite.ctrl)
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	_ = suite.memFs.MkdirAll("/etc/netplan", 0o755)
}

func (suite *NetplanPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *NetplanPublicTestSuite) TearDownSubTest() {
	netplan.ResetMarshalJSON()
}

func (suite *NetplanPublicTestSuite) TestApplyConfig() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func(bool, error)
	}{
		{
			name: "when new file deploys successfully",
			setup: func() {
				// KV Get returns not found (new file).
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// netplan generate succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				// netplan apply succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				// KV Put succeeds.
				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().NoError(err)
				suite.True(changed)

				// Verify file was written.
				data, readErr := suite.memFs.ReadFile(testPath)
				suite.Require().NoError(readErr)
				suite.Equal(testContent, data)
			},
		},
		{
			name: "when SHA matches and file exists (idempotent)",
			setup: func() {
				// Write the file so it exists on disk.
				_ = suite.memFs.WriteFile(testPath, testContent, 0o644)

				state := job.FileState{
					Path:       testPath,
					SHA256:     testSHA(),
					Mode:       "0644",
					DeployedAt: "2026-01-01T00:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().NoError(err)
				suite.False(changed)
			},
		},
		{
			name: "when SHA matches but file missing (rewrites)",
			setup: func() {
				// File does NOT exist on disk (not pre-created).
				state := job.FileState{
					Path:       testPath,
					SHA256:     testSHA(),
					Mode:       "0644",
					DeployedAt: "2026-01-01T00:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)

				// Proceeds to write, generate, apply, put.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().NoError(err)
				suite.True(changed)
			},
		},
		{
			name: "when netplan generate fails (rolls back file)",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", errors.New("invalid YAML"))
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan validate failed (file rolled back)")

				// Verify file was removed.
				_, statErr := suite.memFs.Stat(testPath)
				suite.Error(statErr)
			},
		},
		{
			name: "when netplan apply fails",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", errors.New("apply failed"))
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan apply:")
			},
		},
		{
			name: "when write file fails",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// Use failfs to block writes.
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc/netplan", 0o755)

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
				suite.memFs = vfs
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan apply: write file:")
			},
		},
		{
			name: "when mkdir fails",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				// Use failfs to block MkdirAll.
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
				suite.memFs = vfs
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan apply: create directory:")
			},
		},
		{
			name: "when state put fails",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("kv put failed"))
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan apply: update state:")
			},
		},
		{
			name: "when marshal state fails",
			setup: func() {
				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("not found"))

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"generate"}).
					Return("", nil)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				netplan.SetMarshalJSON(func(_ interface{}) ([]byte, error) {
					return nil, errors.New("marshal failure")
				})
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "marshal")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			changed, err := netplan.ApplyConfig(
				suite.ctx,
				suite.logger,
				suite.memFs,
				suite.mockStateKV,
				suite.mockExec,
				testHostname,
				testPath,
				testContent,
				nil,
			)

			tc.validateFunc(changed, err)
		})
	}
}

func (suite *NetplanPublicTestSuite) TestRemoveConfig() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func(bool, error)
	}{
		{
			name: "when file exists and removal succeeds",
			setup: func() {
				// Create the file on disk.
				_ = suite.memFs.WriteFile(testPath, testContent, 0o644)

				// netplan apply succeeds.
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				// KV state exists for undeploy marking.
				state := job.FileState{
					Path:       testPath,
					SHA256:     testSHA(),
					Mode:       "0644",
					DeployedAt: "2026-01-01T00:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)

				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(1), nil)
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().NoError(err)
				suite.True(changed)

				// Verify file was removed.
				_, statErr := suite.memFs.Stat(testPath)
				suite.Error(statErr)
			},
		},
		{
			name: "when file does not exist",
			setup: func() {
				// No file on disk — nothing to do.
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().NoError(err)
				suite.False(changed)
			},
		},
		{
			name: "when remove fails",
			setup: func() {
				// Create the file on disk, then use failfs to block Remove.
				baseFs := memfs.New()
				_ = baseFs.MkdirAll("/etc/netplan", 0o755)
				_ = baseFs.WriteFile(testPath, testContent, 0o644)

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
				suite.memFs = vfs
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan remove: remove file:")
			},
		},
		{
			name: "when netplan apply fails after remove",
			setup: func() {
				_ = suite.memFs.WriteFile(testPath, testContent, 0o644)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", errors.New("apply failed"))
			},
			validateFunc: func(changed bool, err error) {
				suite.Require().Error(err)
				suite.False(changed)
				suite.Contains(err.Error(), "netplan remove: apply:")
			},
		},
		{
			name: "when state KV put fails on undeploy (best-effort)",
			setup: func() {
				_ = suite.memFs.WriteFile(testPath, testContent, 0o644)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				state := job.FileState{
					Path:       testPath,
					SHA256:     testSHA(),
					Mode:       "0644",
					DeployedAt: "2026-01-01T00:00:00Z",
				}
				stateBytes, _ := json.Marshal(state)

				mockEntry := jobmocks.NewMockKeyValueEntry(suite.ctrl)
				mockEntry.EXPECT().Value().Return(stateBytes)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(mockEntry, nil)

				suite.mockStateKV.EXPECT().
					Put(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uint64(0), errors.New("kv put failed"))
			},
			validateFunc: func(changed bool, err error) {
				// Best-effort: should succeed despite KV error.
				suite.Require().NoError(err)
				suite.True(changed)
			},
		},
		{
			name: "when state KV get fails on undeploy (best-effort)",
			setup: func() {
				_ = suite.memFs.WriteFile(testPath, testContent, 0o644)

				suite.mockExec.EXPECT().
					RunPrivilegedCmd("netplan", []string{"apply"}).
					Return("", nil)

				suite.mockStateKV.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("kv get failed"))
			},
			validateFunc: func(changed bool, err error) {
				// Best-effort: should succeed despite KV error.
				suite.Require().NoError(err)
				suite.True(changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setup()

			changed, err := netplan.RemoveConfig(
				suite.ctx,
				suite.logger,
				suite.memFs,
				suite.mockStateKV,
				suite.mockExec,
				testHostname,
				testPath,
			)

			tc.validateFunc(changed, err)
		})
	}
}

func (suite *NetplanPublicTestSuite) TestComputeSHA256() {
	tests := []struct {
		name         string
		input        []byte
		validateFunc func(string)
	}{
		{
			name:  "when computing known hash",
			input: []byte("hello world"),
			validateFunc: func(result string) {
				h := sha256.Sum256([]byte("hello world"))
				expected := hex.EncodeToString(h[:])
				suite.Equal(expected, result)
			},
		},
		{
			name:  "when computing empty input hash",
			input: []byte(""),
			validateFunc: func(result string) {
				h := sha256.Sum256([]byte(""))
				expected := hex.EncodeToString(h[:])
				suite.Equal(expected, result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := netplan.ComputeSHA256(tc.input)

			tc.validateFunc(result)
		})
	}
}

func TestNetplanPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()

	suite.Run(t, new(NetplanPublicTestSuite))
}
