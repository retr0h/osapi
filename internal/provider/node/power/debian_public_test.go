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

package power_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/power"
)

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl     *gomock.Controller
	logger   *slog.Logger
	mockExec *execmocks.MockManager
	provider *power.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	suite.provider = power.NewDebianProvider(
		suite.logger,
		suite.mockExec,
	)
}

func (suite *DebianPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianPublicTestSuite) TestReboot() {
	tests := []struct {
		name         string
		opts         power.Opts
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(r *power.Result)
	}{
		{
			name: "when successful with no opts uses default delay",
			opts: power.Opts{},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-r", "+1"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("reboot", r.Action)
				suite.Equal(60, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when delay greater than 60 converts to minutes",
			opts: power.Opts{Delay: 180},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-r", "+3"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("reboot", r.Action)
				suite.Equal(180, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when delay less than 60 clamps to 1 minute",
			opts: power.Opts{Delay: 30},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-r", "+1"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("reboot", r.Action)
				suite.Equal(60, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when message provided passes to shutdown command",
			opts: power.Opts{Message: "maintenance reboot"},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-r", "+1", "maintenance reboot"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("reboot", r.Action)
				suite.Equal(60, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when delay and message provided uses both",
			opts: power.Opts{Delay: 300, Message: "scheduled reboot"},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-r", "+5", "scheduled reboot"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("reboot", r.Action)
				suite.Equal(300, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when exec error returns error",
			opts: power.Opts{},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-r", "+1"}).
					Return("", errors.New("permission denied"))
			},
			wantErr:    true,
			wantErrMsg: "power: reboot: permission denied",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Reboot(context.Background(), tc.opts)

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

func (suite *DebianPublicTestSuite) TestShutdown() {
	tests := []struct {
		name         string
		opts         power.Opts
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(r *power.Result)
	}{
		{
			name: "when successful with no opts uses default delay",
			opts: power.Opts{},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-h", "+1"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("shutdown", r.Action)
				suite.Equal(60, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when delay greater than 60 converts to minutes",
			opts: power.Opts{Delay: 180},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-h", "+3"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("shutdown", r.Action)
				suite.Equal(180, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when delay less than 60 clamps to 1 minute",
			opts: power.Opts{Delay: 30},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-h", "+1"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("shutdown", r.Action)
				suite.Equal(60, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when message provided passes to shutdown command",
			opts: power.Opts{Message: "maintenance shutdown"},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-h", "+1", "maintenance shutdown"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("shutdown", r.Action)
				suite.Equal(60, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when delay and message provided uses both",
			opts: power.Opts{Delay: 300, Message: "scheduled shutdown"},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-h", "+5", "scheduled shutdown"}).
					Return("", nil)
			},
			validateFunc: func(r *power.Result) {
				suite.Equal("shutdown", r.Action)
				suite.Equal(300, r.Delay)
				suite.True(r.Changed)
			},
		},
		{
			name: "when exec error returns error",
			opts: power.Opts{},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("shutdown", []string{"-h", "+1"}).
					Return("", errors.New("permission denied"))
			},
			wantErr:    true,
			wantErrMsg: "power: shutdown: permission denied",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Shutdown(context.Background(), tc.opts)

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

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDebianPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DebianPublicTestSuite))
}
