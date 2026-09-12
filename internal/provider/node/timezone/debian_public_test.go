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

package timezone_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/timezone"
)

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl     *gomock.Controller
	logger   *slog.Logger
	mockExec *execmocks.MockManager
	provider *timezone.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	suite.provider = timezone.NewDebianProvider(
		suite.logger,
		suite.mockExec,
	)
}

func (suite *DebianPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DebianPublicTestSuite) TestGet() {
	tests := []struct {
		name         string
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(*timezone.Info)
	}{
		{
			name: "when successful returns timezone info",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("America/New_York\n", nil)
				suite.mockExec.EXPECT().
					RunCmd("date", []string{"+%:z"}).
					Return("-04:00\n", nil)
			},
			validateFunc: func(info *timezone.Info) {
				suite.Equal("America/New_York", info.Timezone)
				suite.Equal("-04:00", info.UTCOffset)
			},
		},
		{
			name: "when timedatectl fails returns error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("", errors.New("command not found"))
			},
			wantErr:    true,
			wantErrMsg: "timezone: timedatectl show: command not found",
		},
		{
			name: "when date command fails returns error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("America/New_York\n", nil)
				suite.mockExec.EXPECT().
					RunCmd("date", []string{"+%:z"}).
					Return("", errors.New("date failed"))
			},
			wantErr:    true,
			wantErrMsg: "timezone: date offset: date failed",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Get(context.Background())

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			suite.NotNil(got)

			if tc.validateFunc != nil {
				tc.validateFunc(got)
			}
		})
	}
}

func (suite *DebianPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		timezone     string
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(*timezone.UpdateResult)
	}{
		{
			name:     "when timezone changed returns changed true",
			timezone: "Europe/London",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("America/New_York\n", nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("timedatectl", []string{"set-timezone", "Europe/London"}).
					Return("", nil)
			},
			validateFunc: func(r *timezone.UpdateResult) {
				suite.True(r.Changed)
				suite.Equal("Europe/London", r.Timezone)
			},
		},
		{
			name:     "when timezone unchanged returns changed false",
			timezone: "America/New_York",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("America/New_York\n", nil)
			},
			validateFunc: func(r *timezone.UpdateResult) {
				suite.False(r.Changed)
				suite.Equal("America/New_York", r.Timezone)
			},
		},
		{
			name:     "when get current timezone fails returns error",
			timezone: "Europe/London",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("", errors.New("command not found"))
			},
			wantErr:    true,
			wantErrMsg: "timezone: timedatectl show: command not found",
		},
		{
			name:     "when set-timezone fails returns error",
			timezone: "Europe/London",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("timedatectl", []string{"show", "-p", "Timezone", "--value"}).
					Return("America/New_York\n", nil)
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("timedatectl", []string{"set-timezone", "Europe/London"}).
					Return("", errors.New("permission denied"))
			},
			wantErr:    true,
			wantErrMsg: "timezone: set-timezone: permission denied",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Update(context.Background(), tc.timezone)

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			suite.NotNil(got)

			if tc.validateFunc != nil {
				tc.validateFunc(got)
			}
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
