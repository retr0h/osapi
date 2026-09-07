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

package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/cli"
)

type RawOutputPublicTestSuite struct {
	suite.Suite
}

func TestRawOutputPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(RawOutputPublicTestSuite))
}

func (suite *RawOutputPublicTestSuite) TestPrintRawOutputPlain() {
	tests := []struct {
		name         string
		results      []cli.RawResult
		showStdout   bool
		showStderr   bool
		wantOut      string
		validateFunc func(string)
	}{
		{
			name: "when single host stdout only prefixes with hostname",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "file1\nfile2\n", Stderr: ""},
			},
			showStdout: true,
			showStderr: true,
			wantOut:    "[server1] file1\n[server1] file2\n",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "", got)
			},
		},
		{
			name: "when single host stderr only prints to stderr",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "", Stderr: "permission denied\n"},
			},
			showStdout: true,
			showStderr: true,
			wantOut:    "",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "[server1] permission denied\n", got)
			},
		},
		{
			name: "when single host both streams prints each",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "output\n", Stderr: "warning\n"},
			},
			showStdout: true,
			showStderr: true,
			wantOut:    "[server1] output\n",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "[server1] warning\n", got)
			},
		},
		{
			name: "when content has embedded empty lines preserves them",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "line1\n\nline3\n"},
			},
			showStdout: true,
			showStderr: true,
			wantOut:    "[server1] line1\n[server1] \n[server1] line3\n",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "", got)
			},
		},
		{
			name:       "when single host empty output prints nothing",
			results:    []cli.RawResult{{Hostname: "server1"}},
			showStdout: true,
			showStderr: true,
			wantOut:    "",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "", got)
			},
		},
		{
			name: "when multi host stdout prefixed with hostname",
			results: []cli.RawResult{
				{Hostname: "web-01", Stdout: "file1\nfile2\n"},
				{Hostname: "web-02", Stdout: "file3\n"},
			},
			showStdout: true,
			showStderr: false,
			wantOut:    "[web-01] file1\n[web-01] file2\n[web-02] file3\n",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "", got)
			},
		},
		{
			name: "when multi host stderr prefixed with hostname",
			results: []cli.RawResult{
				{Hostname: "web-01", Stderr: "err1\n"},
				{Hostname: "web-02", Stderr: "err2\n"},
			},
			showStdout: false,
			showStderr: true,
			wantOut:    "",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "[web-01] err1\n[web-02] err2\n", got)
			},
		},
		{
			name: "when showStdout false suppresses stdout",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "output\n", Stderr: "warning\n"},
			},
			showStdout: false,
			showStderr: true,
			wantOut:    "",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "[server1] warning\n", got)
			},
		},
		{
			name: "when showStderr false suppresses stderr",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "output\n", Stderr: "warning\n"},
			},
			showStdout: true,
			showStderr: false,
			wantOut:    "[server1] output\n",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "", got)
			},
		},
		{
			name: "when both false prints nothing",
			results: []cli.RawResult{
				{Hostname: "server1", Stdout: "output\n", Stderr: "warning\n"},
			},
			showStdout: false,
			showStderr: false,
			wantOut:    "",
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			var stdout, stderr bytes.Buffer

			cli.PrintRawOutputPlain(&stdout, &stderr, tc.results, tc.showStdout, tc.showStderr)

			assert.Equal(suite.T(), tc.wantOut, stdout.String())
			tc.validateFunc(stderr.String())
		})
	}
}

func (suite *RawOutputPublicTestSuite) TestPrintRawOutput() {
	results := []cli.RawResult{
		{Hostname: "server1", Stdout: "hello\n", Stderr: "warn\n"},
	}

	var stdout, stderr bytes.Buffer
	cli.PrintRawOutput(&stdout, &stderr, results, true, true)

	assert.Contains(suite.T(), stdout.String(), "server1")
	assert.Contains(suite.T(), stdout.String(), "hello")
	assert.Contains(suite.T(), stderr.String(), "server1")
	assert.Contains(suite.T(), stderr.String(), "warn")
}

func (suite *RawOutputPublicTestSuite) TestMaxExitCode() {
	tests := []struct {
		name         string
		results      []cli.RawResult
		validateFunc func(int)
	}{
		{
			name:    "when empty results returns zero",
			results: []cli.RawResult{},
			validateFunc: func(got int) {
				assert.Equal(suite.T(), 0, got)
			},
		},
		{
			name: "when all zero returns zero",
			results: []cli.RawResult{
				{Hostname: "s1", ExitCode: 0},
				{Hostname: "s2", ExitCode: 0},
			},
			validateFunc: func(got int) {
				assert.Equal(suite.T(), 0, got)
			},
		},
		{
			name: "when mixed returns highest",
			results: []cli.RawResult{
				{Hostname: "s1", ExitCode: 0},
				{Hostname: "s2", ExitCode: 2},
				{Hostname: "s3", ExitCode: 1},
			},
			validateFunc: func(got int) {
				assert.Equal(suite.T(), 2, got)
			},
		},
		{
			name: "when single non-zero returns it",
			results: []cli.RawResult{
				{Hostname: "s1", ExitCode: 127},
			},
			validateFunc: func(got int) {
				assert.Equal(suite.T(), 127, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(cli.MaxExitCode(tc.results))
		})
	}
}
