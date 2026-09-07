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

package process_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"syscall"
	"testing"

	gopsutil "github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/provider/node/process"
	"github.com/osapi-io/osapi/internal/provider/node/process/mocks"
)

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl         *gomock.Controller
	mockLister   *mocks.MockLister
	mockSignaler *mocks.MockSignaler
	provider     *process.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockLister = mocks.NewMockLister(suite.ctrl)
	suite.mockSignaler = mocks.NewMockSignaler(suite.ctrl)
	suite.provider = process.NewDebianProvider(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		suite.mockLister,
		suite.mockSignaler,
	)
}

func (suite *DebianPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianPublicTestSuite) TestList() {
	tests := []struct {
		name         string
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(result []process.Info)
	}{
		{
			name: "when successful returns process list",
			setupMock: func() {
				q1 := mocks.NewMockQuerier(suite.ctrl)
				q1.EXPECT().Name().Return("test-proc", nil)
				q1.EXPECT().Username().Return("root", nil)
				q1.EXPECT().Status().Return([]string{"running"}, nil)
				q1.EXPECT().CPUPercent().Return(1.5, nil)
				q1.EXPECT().MemoryPercent().Return(float32(2.3), nil)
				q1.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{RSS: 1024}, nil)
				q1.EXPECT().Cmdline().Return("/usr/bin/test", nil)
				q1.EXPECT().CreateTime().Return(int64(1735689600000), nil)

				q2 := mocks.NewMockQuerier(suite.ctrl)
				q2.EXPECT().Name().Return("other-proc", nil)
				q2.EXPECT().Username().Return("user", nil)
				q2.EXPECT().Status().Return([]string{"sleeping"}, nil)
				q2.EXPECT().CPUPercent().Return(0.5, nil)
				q2.EXPECT().MemoryPercent().Return(float32(1.0), nil)
				q2.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{RSS: 2048}, nil)
				q2.EXPECT().Cmdline().Return("/usr/bin/other", nil)
				q2.EXPECT().CreateTime().Return(int64(1735689600000), nil)

				suite.mockLister.EXPECT().Processes().Return([]process.Item{
					{PID: 1, Querier: q1},
					{PID: 2, Querier: q2},
				}, nil)
			},
			validateFunc: func(result []process.Info) {
				suite.Len(result, 2)
				suite.Equal(1, result[0].PID)
				suite.Equal("test-proc", result[0].Name)
				suite.Equal("root", result[0].User)
				suite.Equal("running", result[0].State)
				suite.Equal(1.5, result[0].CPUPercent)
				suite.InDelta(2.3, result[0].MemPercent, 0.01)
				suite.Equal(int64(1024), result[0].MemRSS)
				suite.Equal("/usr/bin/test", result[0].Command)
				suite.Equal(2, result[1].PID)
				suite.Equal("other-proc", result[1].Name)
			},
		},
		{
			name: "when listProcesses errors returns error",
			setupMock: func() {
				suite.mockLister.EXPECT().Processes().Return(nil, errors.New("cannot read /proc"))
			},
			wantErr:    true,
			wantErrMsg: "process: list: cannot read /proc",
		},
		{
			name: "when gather info errors skips process",
			setupMock: func() {
				q1 := mocks.NewMockQuerier(suite.ctrl)
				q1.EXPECT().Name().Return("ok-proc", nil)
				q1.EXPECT().Username().Return("root", nil)
				q1.EXPECT().Status().Return([]string{"running"}, nil)
				q1.EXPECT().CPUPercent().Return(0.0, nil)
				q1.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q1.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{}, nil)
				q1.EXPECT().Cmdline().Return("", nil)
				q1.EXPECT().CreateTime().Return(int64(0), nil)

				q2 := mocks.NewMockQuerier(suite.ctrl)
				q2.EXPECT().Name().Return("", errors.New("permission denied"))

				q3 := mocks.NewMockQuerier(suite.ctrl)
				q3.EXPECT().Name().Return("ok-proc", nil)
				q3.EXPECT().Username().Return("root", nil)
				q3.EXPECT().Status().Return([]string{"running"}, nil)
				q3.EXPECT().CPUPercent().Return(0.0, nil)
				q3.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q3.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{}, nil)
				q3.EXPECT().Cmdline().Return("", nil)
				q3.EXPECT().CreateTime().Return(int64(0), nil)

				suite.mockLister.EXPECT().Processes().Return([]process.Item{
					{PID: 1, Querier: q1},
					{PID: 2, Querier: q2},
					{PID: 3, Querier: q3},
				}, nil)
			},
			validateFunc: func(result []process.Info) {
				suite.Len(result, 2)
				suite.Equal(1, result[0].PID)
				suite.Equal(3, result[1].PID)
			},
		},
		{
			name: "when all processes error returns empty list",
			setupMock: func() {
				q1 := mocks.NewMockQuerier(suite.ctrl)
				q1.EXPECT().Name().Return("", errors.New("permission denied"))

				suite.mockLister.EXPECT().Processes().Return([]process.Item{
					{PID: 1, Querier: q1},
				}, nil)
			},
			validateFunc: func(result []process.Info) {
				suite.Empty(result)
			},
		},
		{
			name: "when status is empty returns empty state",
			setupMock: func() {
				q1 := mocks.NewMockQuerier(suite.ctrl)
				q1.EXPECT().Name().Return("proc", nil)
				q1.EXPECT().Username().Return("root", nil)
				q1.EXPECT().Status().Return([]string{}, nil)
				q1.EXPECT().CPUPercent().Return(0.0, nil)
				q1.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q1.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{}, nil)
				q1.EXPECT().Cmdline().Return("", nil)
				q1.EXPECT().CreateTime().Return(int64(0), nil)

				suite.mockLister.EXPECT().Processes().Return([]process.Item{
					{PID: 1, Querier: q1},
				}, nil)
			},
			validateFunc: func(result []process.Info) {
				suite.Len(result, 1)
				suite.Equal("", result[0].State)
			},
		},
		{
			name: "when MemoryInfo returns nil uses zero RSS",
			setupMock: func() {
				q1 := mocks.NewMockQuerier(suite.ctrl)
				q1.EXPECT().Name().Return("proc", nil)
				q1.EXPECT().Username().Return("root", nil)
				q1.EXPECT().Status().Return([]string{"running"}, nil)
				q1.EXPECT().CPUPercent().Return(0.0, nil)
				q1.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q1.EXPECT().MemoryInfo().Return(nil, nil)
				q1.EXPECT().Cmdline().Return("", nil)
				q1.EXPECT().CreateTime().Return(int64(0), nil)

				suite.mockLister.EXPECT().Processes().Return([]process.Item{
					{PID: 1, Querier: q1},
				}, nil)
			},
			validateFunc: func(result []process.Info) {
				suite.Len(result, 1)
				suite.Equal(int64(0), result[0].MemRSS)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.List(context.Background())

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		pid          int
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(result *process.Info)
	}{
		{
			name: "when successful returns process info",
			pid:  42,
			setupMock: func() {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("test-proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return([]string{"sleeping"}, nil)
				q.EXPECT().CPUPercent().Return(0.5, nil)
				q.EXPECT().MemoryPercent().Return(float32(1.2), nil)
				q.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{RSS: 2048}, nil)
				q.EXPECT().Cmdline().Return("/usr/bin/test --flag", nil)
				q.EXPECT().CreateTime().Return(int64(1735689600000), nil)

				suite.mockLister.EXPECT().NewProcess(int32(42)).Return(q, nil)
			},
			validateFunc: func(result *process.Info) {
				suite.Equal(42, result.PID)
				suite.Equal("test-proc", result.Name)
				suite.Equal("root", result.User)
				suite.Equal("sleeping", result.State)
				suite.Equal(0.5, result.CPUPercent)
				suite.InDelta(1.2, result.MemPercent, 0.01)
				suite.Equal(int64(2048), result.MemRSS)
				suite.Equal("/usr/bin/test --flag", result.Command)
			},
		},
		{
			name: "when pid not found returns error",
			pid:  99999,
			setupMock: func() {
				suite.mockLister.EXPECT().
					NewProcess(int32(99999)).
					Return(nil, errors.New("process not found"))
			},
			wantErr:    true,
			wantErrMsg: "process: get: process not found",
		},
		{
			name: "when gather info errors returns error",
			pid:  42,
			setupMock: func() {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("", errors.New("permission denied"))

				suite.mockLister.EXPECT().NewProcess(int32(42)).Return(q, nil)
			},
			wantErr:    true,
			wantErrMsg: "process: get: permission denied",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Get(context.Background(), tc.pid)

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			suite.NotNil(got)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestSignal() {
	tests := []struct {
		name         string
		pid          int
		signal       string
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(result *process.SignalResult)
	}{
		{
			name:   "when TERM signal succeeds",
			pid:    42,
			signal: "TERM",
			setupMock: func() {
				suite.mockSignaler.EXPECT().Kill(42, syscall.SIGTERM).Return(nil)
			},
			validateFunc: func(result *process.SignalResult) {
				suite.Equal(42, result.PID)
				suite.Equal("TERM", result.Signal)
				suite.True(result.Changed)
			},
		},
		{
			name:   "when KILL signal succeeds",
			pid:    42,
			signal: "KILL",
			setupMock: func() {
				suite.mockSignaler.EXPECT().Kill(42, syscall.SIGKILL).Return(nil)
			},
			validateFunc: func(result *process.SignalResult) {
				suite.Equal(42, result.PID)
				suite.Equal("KILL", result.Signal)
				suite.True(result.Changed)
			},
		},
		{
			name:       "when invalid signal returns error",
			pid:        42,
			signal:     "INVALID",
			setupMock:  func() {},
			wantErr:    true,
			wantErrMsg: `process: signal: invalid signal "INVALID"`,
		},
		{
			name:   "when process not found returns error",
			pid:    99999,
			signal: "TERM",
			setupMock: func() {
				suite.mockSignaler.EXPECT().Kill(99999, syscall.SIGTERM).Return(syscall.ESRCH)
			},
			wantErr:    true,
			wantErrMsg: "process: signal: process not found",
		},
		{
			name:   "when permission denied returns error",
			pid:    1,
			signal: "TERM",
			setupMock: func() {
				suite.mockSignaler.EXPECT().Kill(1, syscall.SIGTERM).Return(syscall.EPERM)
			},
			wantErr:    true,
			wantErrMsg: "process: signal: permission denied",
		},
		{
			name:   "when other error returns wrapped error",
			pid:    42,
			signal: "HUP",
			setupMock: func() {
				suite.mockSignaler.EXPECT().
					Kill(42, syscall.SIGHUP).
					Return(errors.New("unexpected error"))
			},
			wantErr:    true,
			wantErrMsg: "process: signal: unexpected error",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Signal(context.Background(), tc.pid, tc.signal)

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			suite.NotNil(got)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestGatherInfoErrors() {
	tests := []struct {
		name         string
		setupMock    func() *mocks.MockQuerier
		validateFunc func(any, error)
	}{
		{
			name: "when Username errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("", errors.New("user error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "user error")
			},
		},
		{
			name: "when Status errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return(nil, errors.New("status error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "status error")
			},
		},
		{
			name: "when CPUPercent errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return([]string{"running"}, nil)
				q.EXPECT().CPUPercent().Return(0.0, errors.New("cpu error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "cpu error")
			},
		},
		{
			name: "when MemoryPercent errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return([]string{"running"}, nil)
				q.EXPECT().CPUPercent().Return(0.0, nil)
				q.EXPECT().MemoryPercent().Return(float32(0.0), errors.New("mem percent error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "mem percent error")
			},
		},
		{
			name: "when MemoryInfo errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return([]string{"running"}, nil)
				q.EXPECT().CPUPercent().Return(0.0, nil)
				q.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q.EXPECT().MemoryInfo().Return(nil, errors.New("mem info error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "mem info error")
			},
		},
		{
			name: "when Cmdline errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return([]string{"running"}, nil)
				q.EXPECT().CPUPercent().Return(0.0, nil)
				q.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{}, nil)
				q.EXPECT().Cmdline().Return("", errors.New("cmdline error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "cmdline error")
			},
		},
		{
			name: "when CreateTime errors returns error",
			setupMock: func() *mocks.MockQuerier {
				q := mocks.NewMockQuerier(suite.ctrl)
				q.EXPECT().Name().Return("proc", nil)
				q.EXPECT().Username().Return("root", nil)
				q.EXPECT().Status().Return([]string{"running"}, nil)
				q.EXPECT().CPUPercent().Return(0.0, nil)
				q.EXPECT().MemoryPercent().Return(float32(0.0), nil)
				q.EXPECT().MemoryInfo().Return(&gopsutil.MemoryInfoStat{}, nil)
				q.EXPECT().Cmdline().Return("cmd", nil)
				q.EXPECT().CreateTime().Return(int64(0), errors.New("create time error"))

				return q
			},
			validateFunc: func(_ any, err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "create time error")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			q := tc.setupMock()

			suite.mockLister.EXPECT().NewProcess(int32(1)).Return(q, nil)

			tc.validateFunc(suite.provider.Get(context.Background(), 1))
		})
	}
}

// TestDefaultOSFunctions exercises the real OS-interaction wrappers
// against the current test process to ensure coverage of gopsutil calls.
func (suite *DebianPublicTestSuite) TestDefaultOSFunctions() {
	pid := os.Getpid()

	lister := process.NewGopsutilLister()
	signaler := process.NewSyscallSignaler()

	tests := []struct {
		name         string
		fn           func() error
		validateFunc func()
	}{
		{
			name: "when listing with real OS returns current process",
			fn: func() error {
				provider := process.NewDebianProvider(slog.Default(), lister, signaler)
				results, err := provider.List(context.Background())
				if err != nil {
					return err
				}

				found := false
				for _, p := range results {
					if p.PID == pid {
						found = true

						break
					}
				}
				suite.True(found, "current process not found in list")
				suite.NotEmpty(results)

				return nil
			},
		},
		{
			name: "when getting with real OS returns current process info",
			fn: func() error {
				provider := process.NewDebianProvider(slog.Default(), lister, signaler)
				info, err := provider.Get(context.Background(), pid)
				if err != nil {
					return err
				}

				suite.Equal(pid, info.PID)
				suite.NotEmpty(info.Name)
				suite.NotEmpty(info.StartTime)

				return nil
			},
		},
		{
			name: "when Kill with signal 0 checks process exists",
			fn: func() error {
				// Signal 0 doesn't kill -- it checks if the process exists.
				err := signaler.Kill(pid, syscall.Signal(0))
				suite.NoError(err)

				return nil
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			err := tc.fn()
			suite.NoError(err)
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
