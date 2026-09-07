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

type ExecPublicTestSuite struct {
	suite.Suite

	mockCtrl    *gomock.Controller
	mockExecMgr *execMocks.MockManager
	sut         *command.Executor
}

func (s *ExecPublicTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockExecMgr = execMocks.NewMockManager(s.mockCtrl)
	s.sut = command.New(slog.Default(), s.mockExecMgr)
}

func (s *ExecPublicTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ExecPublicTestSuite) TestExec() {
	tests := []struct {
		name          string
		params        command.ExecParams
		mockResult    *exec.CmdResult
		mockError     error
		expectError   bool
		errorContains string
		validate      func(*command.Result)
	}{
		{
			name: "successful execution",
			params: command.ExecParams{
				Command: "ls",
				Args:    []string{"-la"},
				Cwd:     "/tmp",
				Timeout: 30,
			},
			mockResult: &exec.CmdResult{
				Stdout:     "total 0\n",
				Stderr:     "",
				ExitCode:   0,
				DurationMs: 12,
			},
			validate: func(r *command.Result) {
				s.Equal("total 0\n", r.Stdout)
				s.Empty(r.Stderr)
				s.Equal(0, r.ExitCode)
				s.Equal(int64(12), r.DurationMs)
			},
		},
		{
			name: "execution with non-zero exit code",
			params: command.ExecParams{
				Command: "false",
			},
			mockResult: &exec.CmdResult{
				Stdout:     "",
				Stderr:     "error occurred\n",
				ExitCode:   1,
				DurationMs: 5,
			},
			validate: func(r *command.Result) {
				s.Equal(1, r.ExitCode)
				s.Equal("error occurred\n", r.Stderr)
			},
		},
		{
			name: "execution error",
			params: command.ExecParams{
				Command: "nonexistent",
			},
			mockError:     errors.New("command not found"),
			expectError:   true,
			errorContains: "command execution failed",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.mockExecMgr.EXPECT().
				RunCmdFull(
					tt.params.Command,
					tt.params.Args,
					tt.params.Cwd,
					tt.params.Timeout,
				).
				Return(tt.mockResult, tt.mockError)

			result, err := s.sut.Exec(tt.params)

			if tt.expectError {
				s.Error(err)
				s.Contains(err.Error(), tt.errorContains)
				s.Nil(result)
			} else {
				s.NoError(err)
				s.NotNil(result)
				if tt.validate != nil {
					tt.validate(result)
				}
			}
		})
	}
}

func TestExecPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ExecPublicTestSuite))
}
