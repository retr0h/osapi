// Copyright (c) 2024 John Dewey

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
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/exec"
)

type RunCmdDirPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
}

func (suite *RunCmdDirPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *RunCmdDirPublicTestSuite) TearDownTest() {}

func (suite *RunCmdDirPublicTestSuite) TestRunCmd() {
	tests := []struct {
		name          string
		command       string
		args          []string
		cwd           string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid command with no arguments",
			command:     "ls",
			args:        []string{},
			expectError: false,
		},
		{
			name:        "Valid command with no arguments and working dir",
			command:     "ls",
			args:        []string{},
			cwd:         "/tmp",
			expectError: false,
		},
		{
			name:        "Valid command with output",
			command:     "echo",
			args:        []string{"-n", "foo"},
			cwd:         "/tmp",
			expectError: false,
		},
		{
			name:          "Invalid command",
			command:       "invalid",
			args:          []string{"foo"},
			cwd:           "/tmp",
			expectError:   true,
			errorContains: "not found",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			em := exec.New(suite.logger)

			output, err := em.RunCmdInDir(tc.command, tc.args, tc.cwd)

			if tc.expectError {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errorContains)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(output)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestRunCmdDirPublicTestSuite(t *testing.T) {
	suite.Run(t, new(RunCmdDirPublicTestSuite))
}
