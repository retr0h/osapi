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

package command_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/exec"
	execMocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/command"
)

type ShellPublicTestSuite struct {
	suite.Suite

	mockCtrl    *gomock.Controller
	mockExecMgr *execMocks.MockManager
	sut         *command.Executor
}

func (s *ShellPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockExecMgr = execMocks.NewMockManager(s.mockCtrl)
	s.sut = command.New(slog.Default(), s.mockExecMgr)
}

func (s *ShellPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ShellPublicTestSuite) TestShell() {
	tests := []struct {
		name         string
		params       command.ShellParams
		mockResult   *exec.CmdResult
		mockError    error
		validateFunc func(*command.Result, error)
	}{
		{
			name: "successful shell command",
			params: command.ShellParams{
				Command: "echo hello | tr a-z A-Z",
				Cwd:     "/tmp",
				Timeout: 30,
			},
			mockResult: &exec.CmdResult{
				Stdout:     "HELLO\n",
				Stderr:     "",
				ExitCode:   0,
				DurationMs: 15,
			},
			validateFunc: func(result *command.Result, err error) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal("HELLO\n", result.Stdout)
				s.Empty(result.Stderr)
				s.Equal(0, result.ExitCode)
				s.Equal(int64(15), result.DurationMs)
			},
		},
		{
			name: "shell command with non-zero exit code",
			params: command.ShellParams{
				Command: "exit 2",
			},
			mockResult: &exec.CmdResult{
				Stdout:     "",
				Stderr:     "",
				ExitCode:   2,
				DurationMs: 3,
			},
			validateFunc: func(result *command.Result, err error) {
				s.NoError(err)
				s.NotNil(result)
				s.Equal(2, result.ExitCode)
			},
		},
		{
			name: "shell execution error",
			params: command.ShellParams{
				Command: "bad command",
			},
			mockError: errors.New("shell failed"),
			validateFunc: func(result *command.Result, err error) {
				s.Error(err)
				s.Contains(err.Error(), "shell execution failed")
				s.Nil(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockExecMgr.EXPECT().
				RunCmdFull(
					"/bin/sh",
					[]string{"-c", tt.params.Command},
					tt.params.Cwd,
					tt.params.Timeout,
				).
				Return(tt.mockResult, tt.mockError)

			tt.validateFunc(s.sut.Shell(tt.params))
		})
	}
}

func TestShellPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShellPublicTestSuite))
}
