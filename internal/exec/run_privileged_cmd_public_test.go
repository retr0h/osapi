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

package exec_test

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/exec"
	"github.com/osapi-io/osapi/internal/exec/mocks"
)

type RunPrivilegedCmdPublicTestSuite struct {
	suite.Suite

	ctrl         *gomock.Controller
	mockExecutor *mocks.MockCommandExecutor
	logger       *slog.Logger
}

func (s *RunPrivilegedCmdPublicTestSuite) SetupTest() {
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *RunPrivilegedCmdPublicTestSuite) SetupSubTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockExecutor = mocks.NewMockCommandExecutor(s.ctrl)
}

func (s *RunPrivilegedCmdPublicTestSuite) TearDownSubTest() {
	s.ctrl.Finish()
}

func (s *RunPrivilegedCmdPublicTestSuite) TestRunPrivilegedCmd() {
	tests := []struct {
		name         string
		sudo         bool
		command      string
		args         []string
		setupMock    func()
		validateFunc func(string, error)
	}{
		{
			name:    "without sudo runs command directly",
			sudo:    false,
			command: "echo",
			args:    []string{"-n", "hello"},
			setupMock: func() {
				s.mockExecutor.EXPECT().
					Execute("echo", []string{"-n", "hello"}, "").
					Return("hello", nil)
			},
			validateFunc: func(output string, err error) {
				s.NoError(err)
				s.Equal("hello", output)
			},
		},
		{
			name:    "with sudo prepends sudo to args",
			sudo:    true,
			command: "echo",
			args:    []string{"-n", "hello"},
			setupMock: func() {
				s.mockExecutor.EXPECT().
					Execute("sudo", []string{"echo", "-n", "hello"}, "").
					Return("hello", nil)
			},
			validateFunc: func(output string, err error) {
				s.NoError(err)
				s.Equal("hello", output)
			},
		},
		{
			name:    "with sudo no args",
			sudo:    true,
			command: "systemctl",
			args:    nil,
			setupMock: func() {
				s.mockExecutor.EXPECT().
					Execute("sudo", []string{"systemctl"}, "").
					Return("", nil)
			},
			validateFunc: func(output string, err error) {
				s.NoError(err)
				s.Equal("", output)
			},
		},
		{
			name:    "executor error propagates",
			sudo:    false,
			command: "nonexistent",
			args:    []string{},
			setupMock: func() {
				s.mockExecutor.EXPECT().
					Execute("nonexistent", []string{}, "").
					Return("", fmt.Errorf("command not found"))
			},
			validateFunc: func(_ string, err error) {
				s.Error(err)
				s.Contains(err.Error(), "command not found")
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupMock()

			em := exec.New(s.logger, tc.sudo)
			exec.SetExecutor(em, s.mockExecutor)

			output, err := em.RunPrivilegedCmd(tc.command, tc.args)

			tc.validateFunc(output, err)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestRunPrivilegedCmdPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RunPrivilegedCmdPublicTestSuite))
}
