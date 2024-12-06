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

package exec

import (
	"log/slog"
	"os/exec"
	"strings"
)

// New factory to create a new Exec instance.
func New(
	logger *slog.Logger,
) *Exec {
	return &Exec{
		logger: logger,
	}
}

// RunCmdImpl executes the provided command with the specified arguments and
// an optional working directory. It captures and logs the combined output
// (stdout and stderr) of the command.
func (e *Exec) RunCmdImpl(
	name string,
	args []string,
	cwd string,
) (string, error) {
	cmd := exec.Command(name, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	out, err := cmd.CombinedOutput()
	e.logger.Debug(
		"exec",
		slog.String("command", strings.Join(cmd.Args, " ")),
		slog.String("cwd", cwd),
		slog.String("output", string(out)),
		slog.Any("error", err),
	)
	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
