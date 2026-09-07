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

package log_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	oslog "github.com/osapi-io/osapi/internal/provider/node/log"
)

// singleEntry is a valid journalctl JSON line used across tests.
const singleEntry = `{"__REALTIME_TIMESTAMP":"1711929045123456","SYSLOG_IDENTIFIER":"nginx","PRIORITY":"6","MESSAGE":"Started nginx","_PID":"1234","_HOSTNAME":"web-01"}`

// twoEntries is two valid journalctl JSON lines.
const twoEntries = `{"__REALTIME_TIMESTAMP":"1711929045123456","SYSLOG_IDENTIFIER":"nginx","PRIORITY":"6","MESSAGE":"Started nginx","_PID":"1234","_HOSTNAME":"web-01"}
{"__REALTIME_TIMESTAMP":"1711929046000000","SYSLOG_IDENTIFIER":"sshd","PRIORITY":"5","MESSAGE":"Accepted key","_PID":"5678","_HOSTNAME":"web-01"}`

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl        *gomock.Controller
	mockManager *execmocks.MockManager
	provider    *oslog.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockManager = execmocks.NewMockManager(suite.ctrl)
	suite.provider = oslog.NewDebianProvider(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		suite.mockManager,
	)
}

func (suite *DebianPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DebianPublicTestSuite) TestQuery() {
	tests := []struct {
		name         string
		opts         oslog.QueryOpts
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(result []oslog.Entry)
	}{
		{
			name: "when default query returns entries",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--output=json", "-n", "100"}).
					Return(twoEntries, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 2)
				suite.Equal("nginx", result[0].Unit)
				suite.Equal("info", result[0].Priority)
				suite.Equal("Started nginx", result[0].Message)
				suite.Equal(1234, result[0].PID)
				suite.Equal("web-01", result[0].Hostname)
				suite.NotEmpty(result[0].Timestamp)
				suite.Equal("sshd", result[1].Unit)
				suite.Equal("notice", result[1].Priority)
			},
		},
		{
			name: "when all options set uses correct args",
			opts: oslog.QueryOpts{
				Lines:    50,
				Since:    "1 hour ago",
				Priority: "err",
			},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--output=json", "--since", "1 hour ago", "--priority", "err", "-n", "50"}).
					Return(singleEntry, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
				suite.Equal("nginx", result[0].Unit)
			},
		},
		{
			name: "when custom lines only uses correct args",
			opts: oslog.QueryOpts{Lines: 25},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--output=json", "-n", "25"}).
					Return(singleEntry, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
			},
		},
		{
			name: "when exec errors returns error",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", gomock.Any()).
					Return("", errors.New("journalctl not found"))
			},
			wantErr:    true,
			wantErrMsg: "log: query: journalctl not found",
		},
		{
			name: "when empty output returns empty slice",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", gomock.Any()).
					Return("", nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Empty(result)
			},
		},
		{
			name: "when malformed JSON line is skipped",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", gomock.Any()).
					Return("not-valid-json\n"+singleEntry, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
				suite.Equal("nginx", result[0].Unit)
			},
		},
		{
			name: "when entry has empty timestamp",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", gomock.Any()).
					Return(`{"__REALTIME_TIMESTAMP":"","SYSLOG_IDENTIFIER":"test","PRIORITY":"6","MESSAGE":"hello"}`, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
				suite.Equal("", result[0].Timestamp)
			},
		},
		{
			name: "when entry has non-numeric timestamp",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", gomock.Any()).
					Return(`{"__REALTIME_TIMESTAMP":"not-a-number","SYSLOG_IDENTIFIER":"test","PRIORITY":"6","MESSAGE":"hello"}`, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
				suite.Equal("not-a-number", result[0].Timestamp)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Query(context.Background(), tc.opts)

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestQueryUnit() {
	tests := []struct {
		name         string
		unit         string
		opts         oslog.QueryOpts
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(result []oslog.Entry)
	}{
		{
			name: "when default unit query returns entries",
			unit: "nginx.service",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--output=json", "-u", "nginx.service", "-n", "100"}).
					Return(singleEntry, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
				suite.Equal("nginx", result[0].Unit)
				suite.Equal("info", result[0].Priority)
				suite.Equal("Started nginx", result[0].Message)
				suite.Equal(1234, result[0].PID)
				suite.Equal("web-01", result[0].Hostname)
			},
		},
		{
			name: "when all options set uses correct args",
			unit: "sshd.service",
			opts: oslog.QueryOpts{
				Lines:    20,
				Since:    "30 minutes ago",
				Priority: "warning",
			},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--output=json", "-u", "sshd.service", "--since", "30 minutes ago", "--priority", "warning", "-n", "20"}).
					Return(singleEntry, nil)
			},
			validateFunc: func(result []oslog.Entry) {
				suite.Len(result, 1)
			},
		},
		{
			name: "when exec errors returns error",
			unit: "nginx.service",
			opts: oslog.QueryOpts{},
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", gomock.Any()).
					Return("", errors.New("journalctl failed"))
			},
			wantErr:    true,
			wantErrMsg: "log: query unit: journalctl failed",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.QueryUnit(context.Background(), tc.unit, tc.opts)

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestListSources() {
	tests := []struct {
		name         string
		setupMock    func()
		wantErr      bool
		wantErrMsg   string
		validateFunc func(result []string)
	}{
		{
			name: "when sources returned sorted list",
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--field=SYSLOG_IDENTIFIER"}).
					Return("sshd\nnginx\ncron\n", nil)
			},
			validateFunc: func(result []string) {
				suite.Equal([]string{"cron", "nginx", "sshd"}, result)
			},
		},
		{
			name: "when exec errors returns error",
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--field=SYSLOG_IDENTIFIER"}).
					Return("", errors.New("journalctl not found"))
			},
			wantErr:    true,
			wantErrMsg: "log: list sources: journalctl not found",
		},
		{
			name: "when empty output returns nil",
			setupMock: func() {
				suite.mockManager.EXPECT().
					RunCmd("journalctl", []string{"--field=SYSLOG_IDENTIFIER"}).
					Return("", nil)
			},
			validateFunc: func(result []string) {
				suite.Nil(result)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.ListSources(context.Background())

			if tc.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrMsg)
				suite.Nil(got)

				return
			}

			suite.NoError(err)
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
