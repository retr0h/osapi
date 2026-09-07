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

	"github.com/osapi-io/osapi/internal/exec"
)

type RunCmdPublicTestSuite struct {
	suite.Suite

	logger *slog.Logger
}

func (suite *RunCmdPublicTestSuite) SetupTest() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (suite *RunCmdPublicTestSuite) TearDownTest() {}

func (suite *RunCmdPublicTestSuite) TestRunCmd() {
	tests := []struct {
		name         string
		command      string
		args         []string
		validateFunc func(any, error)
	}{
		{
			name:    "Valid command with no arguments",
			command: "ls",
			args:    []string{},
			validateFunc: func(output any, err error) {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(output)
			},
		},
		{
			name:    "Valid command with no arguments and working dir",
			command: "ls",
			args:    []string{},
			validateFunc: func(output any, err error) {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(output)
			},
		},
		{
			name:    "Valid command with output",
			command: "echo",
			args:    []string{"-n", "foo"},
			validateFunc: func(output any, err error) {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(output)
			},
		},
		{
			name:    "Invalid command",
			command: "invalid",
			args:    []string{"foo"},
			validateFunc: func(_ any, err error) {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), "not found")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			em := exec.New(suite.logger, false)

			tc.validateFunc(em.RunCmd(tc.command, tc.args))
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestRunCmdPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RunCmdPublicTestSuite))
}
