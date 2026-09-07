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

package agent_test

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/agent"
	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
)

type PreflightPublicTestSuite struct {
	suite.Suite

	mockCtrl    *gomock.Controller
	mockExecMgr *execmocks.MockManager
	logger      *slog.Logger
	tmpDir      string
}

func TestPreflightPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PreflightPublicTestSuite))
}

func (s *PreflightPublicTestSuite) SetupTest() {
	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	s.tmpDir = s.T().TempDir()
}

func (s *PreflightPublicTestSuite) SetupSubTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockExecMgr = execmocks.NewMockManager(s.mockCtrl)
}

func (s *PreflightPublicTestSuite) TearDownSubTest() {
	s.mockCtrl.Finish()
	agent.ResetProcStatusPath()
}

func (s *PreflightPublicTestSuite) TestCheckCapabilities() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func([]agent.PreflightResult)
	}{
		{
			name: "when all caps present",
			setup: func() {
				path := filepath.Join(s.tmpDir, "status_all_caps")
				content := "Name:\tosapi\nCapEff:\t000000000000102f\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult) {
				s.NotEmpty(results)
				for _, r := range results {
					s.True(r.Passed, "expected %s to pass", r.Name)
					s.Empty(r.Error)
				}
			},
		},
		{
			name: "when missing cap",
			setup: func() {
				path := filepath.Join(s.tmpDir, "status_no_caps")
				content := "Name:\tosapi\nCapEff:\t0000000000000000\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult) {
				s.NotEmpty(results)
				for _, r := range results {
					s.False(r.Passed, "expected %s to fail", r.Name)
					s.Contains(r.Error, "not in effective set")
				}
			},
		},
		{
			name: "when file not found",
			setup: func() {
				agent.SetProcStatusPath(filepath.Join(s.tmpDir, "nonexistent"))
			},
			validateFunc: func(results []agent.PreflightResult) {
				s.NotEmpty(results)
				for _, r := range results {
					s.False(r.Passed, "expected %s to fail", r.Name)
					s.Contains(r.Error, "failed to read capabilities")
				}
			},
		},
		{
			name: "when CapEff line has invalid hex",
			setup: func() {
				path := filepath.Join(s.tmpDir, "status_bad_hex")
				content := "Name:\tosapi\nCapEff:\tNOTHEX\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult) {
				s.NotEmpty(results)
				for _, r := range results {
					s.False(r.Passed, "expected %s to fail", r.Name)
					s.Contains(r.Error, "failed to read capabilities")
				}
			},
		},
		{
			name: "when scanner encounters read error",
			setup: func() {
				// Write a line longer than bufio.MaxScanTokenSize
				// (64 KiB) to trigger a scanner error before
				// reaching the CapEff line.
				path := filepath.Join(s.tmpDir, "status_long_line")
				longLine := make([]byte, 70000)
				for i := range longLine {
					longLine[i] = 'x'
				}
				content := string(longLine) + "\nCapEff:\t000000000000102f\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult) {
				s.NotEmpty(results)
				for _, r := range results {
					s.False(r.Passed, "expected %s to fail", r.Name)
					s.Contains(r.Error, "failed to read capabilities")
				}
			},
		},
		{
			name: "when CapEff line not present",
			setup: func() {
				path := filepath.Join(s.tmpDir, "status_no_capeff")
				content := "Name:\tosapi\nCapInh:\t0000000000000000\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult) {
				s.NotEmpty(results)
				for _, r := range results {
					s.False(r.Passed, "expected %s to fail", r.Name)
					s.Contains(r.Error, "failed to read capabilities")
				}
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setup()
			results := agent.ExportCheckCapabilities(s.logger)
			tc.validateFunc(results)
		})
	}
}

func (s *PreflightPublicTestSuite) TestRunPreflight() {
	tests := []struct {
		name         string
		setup        func()
		validateFunc func([]agent.PreflightResult, bool)
	}{
		{
			name: "when both checks pass",
			setup: func() {
				s.mockExecMgr.EXPECT().
					RunCmd("sudo", gomock.Any()).
					Return("/usr/bin/something", nil).
					AnyTimes()

				path := filepath.Join(s.tmpDir, "status_pass")
				content := "Name:\tosapi\nCapEff:\t000000000000102f\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult, allPassed bool) {
				s.True(allPassed)
				s.NotEmpty(results)
			},
		},
		{
			name: "when sudo check fails",
			setup: func() {
				s.mockExecMgr.EXPECT().
					RunCmd("sudo", gomock.Any()).
					Return("", fmt.Errorf("sudo failed")).
					AnyTimes()

				path := filepath.Join(s.tmpDir, "status_sudo_fail")
				content := "Name:\tosapi\nCapEff:\t000000000000102f\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult, allPassed bool) {
				s.False(allPassed)
				s.NotEmpty(results)
			},
		},
		{
			name: "when caps check fails",
			setup: func() {
				s.mockExecMgr.EXPECT().
					RunCmd("sudo", gomock.Any()).
					Return("/usr/bin/something", nil).
					AnyTimes()

				path := filepath.Join(s.tmpDir, "status_fail")
				content := "Name:\tosapi\nCapEff:\t0000000000000000\n"
				err := os.WriteFile(path, []byte(content), 0o644)
				s.Require().NoError(err)
				agent.SetProcStatusPath(path)
			},
			validateFunc: func(results []agent.PreflightResult, allPassed bool) {
				s.False(allPassed)
				s.NotEmpty(results)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setup()
			results, allPassed := agent.RunPreflight(
				s.logger,
				s.mockExecMgr,
			)
			tc.validateFunc(results, allPassed)
		})
	}
}
