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

package ntp_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/failfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	execmocks "github.com/osapi-io/osapi/internal/exec/mocks"
	"github.com/osapi-io/osapi/internal/provider/node/ntp"
)

const (
	trackingOutput = `Reference ID    : A29FC801 (time.cloudflare.com)
Stratum         : 3
Ref time (UTC)  : Sun Mar 30 22:30:45 2026
System time     : 0.000001834 seconds slow of NTP time
Last offset     : -0.000002341 seconds
RMS offset      : 0.000015432 seconds
Frequency       : 12.345 ppm fast
Residual freq   : -0.001 ppm
Skew            : 0.123 ppm
Root delay      : 0.025432100 seconds
Root dispersion : 0.000123456 seconds
Update interval : 64.5 seconds
Leap status     : Normal`

	sourcesOutput = `^,*,time.cloudflare.com,3,6,377,45,+0.000012345[+0.000012345],+/-0.012345678
^,+,ntp.ubuntu.com,2,6,377,32,+0.000023456[+0.000023456],+/-0.023456789`

	sourcesFile = "/etc/chrony/sources.d/osapi-ntp.sources"
	sourcesDir  = "/etc/chrony/sources.d"
)

type DebianPublicTestSuite struct {
	suite.Suite

	ctrl     *gomock.Controller
	logger   *slog.Logger
	memFs    avfs.VFS
	mockExec *execmocks.MockManager
	provider *ntp.Debian
}

func (suite *DebianPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	suite.memFs = memfs.New()
	suite.mockExec = execmocks.NewMockManager(suite.ctrl)

	suite.provider = ntp.NewDebianProvider(
		suite.logger,
		suite.memFs,
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
		validateFunc func(*ntp.Status, error)
	}{
		{
			name: "when successful returns status with servers",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"tracking"}).
					Return(trackingOutput, nil)
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"sources", "-c"}).
					Return(sourcesOutput, nil)
			},
			validateFunc: func(got *ntp.Status, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Synchronized)
				suite.Equal(3, got.Stratum)
				suite.Equal("-0.000001834s", got.Offset)
				suite.Equal("time.cloudflare.com", got.CurrentSource)
				suite.Equal([]string{"time.cloudflare.com", "ntp.ubuntu.com"}, got.Servers)
			},
		},
		{
			name: "when tracking not synchronized",
			setupMock: func() {
				output := `Reference ID    : 00000000 ()
Stratum         : 0
Ref time (UTC)  : Thu Jan 01 00:00:00 1970
System time     : 0.000000000 seconds fast of NTP time
Last offset     : +0.000000000 seconds
RMS offset      : 0.000000000 seconds
Frequency       : 0.000 ppm slow
Residual freq   : +0.000 ppm
Skew            : 0.000 ppm
Root delay      : 1.000000000 seconds
Root dispersion : 1.000000000 seconds
Update interval : 0.0 seconds
Leap status     : Not synchronised`
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"tracking"}).
					Return(output, nil)
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"sources", "-c"}).
					Return("", nil)
			},
			validateFunc: func(got *ntp.Status, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.False(got.Synchronized)
				suite.Equal(0, got.Stratum)
				suite.Equal("+0.000000000s", got.Offset)
				suite.Empty(got.CurrentSource)
				suite.Empty(got.Servers)
			},
		},
		{
			name: "when chronyc tracking fails returns error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"tracking"}).
					Return("", errors.New("command not found"))
			},
			validateFunc: func(got *ntp.Status, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: chronyc tracking: command not found")
			},
		},
		{
			name: "when chronyc sources fails returns error",
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"tracking"}).
					Return(trackingOutput, nil)
				suite.mockExec.EXPECT().
					RunCmd("chronyc", []string{"sources", "-c"}).
					Return("", errors.New("connection refused"))
			},
			validateFunc: func(got *ntp.Status, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: chronyc sources: connection refused")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupMock()

			got, err := suite.provider.Get(context.Background())
			tc.validateFunc(got, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestCreate() {
	tests := []struct {
		name         string
		config       ntp.Config
		setupFs      func()
		setupMock    func()
		validateFunc func(*ntp.CreateResult, error)
	}{
		{
			name: "when successful creates config file",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org", "1.pool.ntp.org"},
			},
			setupFs: func() {},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chronyc", []string{"reload", "sources"}).
					Return("", nil)
			},
			validateFunc: func(got *ntp.CreateResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Changed)

				content, readErr := suite.memFs.ReadFile(sourcesFile)
				suite.NoError(readErr)
				suite.Equal(
					"server 0.pool.ntp.org iburst\nserver 1.pool.ntp.org iburst\n",
					string(content),
				)
			},
		},
		{
			name: "when config already managed returns unchanged",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org"},
			},
			setupFs: func() {
				_ = suite.memFs.MkdirAll(sourcesDir, 0o755)
				_ = suite.memFs.WriteFile(sourcesFile, []byte("existing"), 0o644)
			},
			setupMock: func() {},
			validateFunc: func(got *ntp.CreateResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.False(got.Changed)
			},
		},
		{
			name: "when mkdir fails returns error",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org"},
			},
			setupFs: func() {
				baseFs := memfs.New()
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnMkdirAll {
						return errors.New("permission denied")
					}

					return nil
				})
				suite.memFs = vfs
				suite.provider = ntp.NewDebianProvider(
					suite.logger,
					suite.memFs,
					suite.mockExec,
				)
			},
			setupMock: func() {},
			validateFunc: func(got *ntp.CreateResult, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: create directory: permission denied")
			},
		},
		{
			name: "when write fails returns error",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org"},
			},
			setupFs: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll(sourcesDir, 0o755)
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnOpenFile {
						return errors.New("disk full")
					}

					return nil
				})
				suite.memFs = vfs
				suite.provider = ntp.NewDebianProvider(
					suite.logger,
					suite.memFs,
					suite.mockExec,
				)
			},
			setupMock: func() {},
			validateFunc: func(got *ntp.CreateResult, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: write file: disk full")
			},
		},
		{
			name: "when reload fails logs warning but succeeds",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org"},
			},
			setupFs: func() {},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chronyc", []string{"reload", "sources"}).
					Return("", errors.New("chronyc not running"))
			},
			validateFunc: func(got *ntp.CreateResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupFs()
			tc.setupMock()

			got, err := suite.provider.Create(context.Background(), tc.config)
			tc.validateFunc(got, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestUpdate() {
	tests := []struct {
		name         string
		config       ntp.Config
		setupFs      func()
		setupMock    func()
		validateFunc func(*ntp.UpdateResult, error)
	}{
		{
			name: "when content changed updates file",
			config: ntp.Config{
				Servers: []string{"time.google.com", "time.cloudflare.com"},
			},
			setupFs: func() {
				_ = suite.memFs.MkdirAll(sourcesDir, 0o755)
				_ = suite.memFs.WriteFile(
					sourcesFile,
					[]byte("server 0.pool.ntp.org iburst\n"),
					0o644,
				)
			},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chronyc", []string{"reload", "sources"}).
					Return("", nil)
			},
			validateFunc: func(got *ntp.UpdateResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Changed)

				content, readErr := suite.memFs.ReadFile(sourcesFile)
				suite.NoError(readErr)
				suite.Equal(
					"server time.google.com iburst\nserver time.cloudflare.com iburst\n",
					string(content),
				)
			},
		},
		{
			name: "when content unchanged returns not changed",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org"},
			},
			setupFs: func() {
				_ = suite.memFs.MkdirAll(sourcesDir, 0o755)
				_ = suite.memFs.WriteFile(
					sourcesFile,
					[]byte("server 0.pool.ntp.org iburst\n"),
					0o644,
				)
			},
			setupMock: func() {},
			validateFunc: func(got *ntp.UpdateResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.False(got.Changed)
			},
		},
		{
			name: "when config not managed returns error",
			config: ntp.Config{
				Servers: []string{"0.pool.ntp.org"},
			},
			setupFs:   func() {},
			setupMock: func() {},
			validateFunc: func(got *ntp.UpdateResult, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: config not managed")
			},
		},
		{
			name: "when write fails returns error",
			config: ntp.Config{
				Servers: []string{"time.google.com"},
			},
			setupFs: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll(sourcesDir, 0o755)
				_ = baseFs.WriteFile(
					sourcesFile,
					[]byte("server 0.pool.ntp.org iburst\n"),
					0o644,
				)
				vfs := failfs.New(baseFs)
				openCount := 0
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnOpenFile {
						openCount++
						// First OpenFile call is ReadFile in Update;
						// second is the WriteFile we want to fail.
						if openCount > 1 {
							return errors.New("disk full")
						}
					}

					return nil
				})
				suite.memFs = vfs
				suite.provider = ntp.NewDebianProvider(
					suite.logger,
					suite.memFs,
					suite.mockExec,
				)
			},
			setupMock: func() {},
			validateFunc: func(got *ntp.UpdateResult, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: write file: disk full")
			},
		},
		{
			name: "when reload fails logs warning but succeeds",
			config: ntp.Config{
				Servers: []string{"time.google.com"},
			},
			setupFs: func() {
				_ = suite.memFs.MkdirAll(sourcesDir, 0o755)
				_ = suite.memFs.WriteFile(
					sourcesFile,
					[]byte("server 0.pool.ntp.org iburst\n"),
					0o644,
				)
			},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chronyc", []string{"reload", "sources"}).
					Return("", errors.New("chronyc not running"))
			},
			validateFunc: func(got *ntp.UpdateResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupFs()
			tc.setupMock()

			got, err := suite.provider.Update(context.Background(), tc.config)
			tc.validateFunc(got, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestDelete() {
	tests := []struct {
		name         string
		setupFs      func()
		setupMock    func()
		validateFunc func(*ntp.DeleteResult, error)
	}{
		{
			name: "when successful removes config file",
			setupFs: func() {
				_ = suite.memFs.MkdirAll(sourcesDir, 0o755)
				_ = suite.memFs.WriteFile(
					sourcesFile,
					[]byte("server 0.pool.ntp.org iburst\n"),
					0o644,
				)
			},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chronyc", []string{"reload", "sources"}).
					Return("", nil)
			},
			validateFunc: func(got *ntp.DeleteResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Changed)

				_, statErr := suite.memFs.Stat(sourcesFile)
				suite.Error(statErr)
			},
		},
		{
			name:      "when config not managed returns error",
			setupFs:   func() {},
			setupMock: func() {},
			validateFunc: func(got *ntp.DeleteResult, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: config not managed")
			},
		},
		{
			name: "when remove fails returns error",
			setupFs: func() {
				baseFs := memfs.New()
				_ = baseFs.MkdirAll(sourcesDir, 0o755)
				_ = baseFs.WriteFile(sourcesFile, []byte("server 0.pool.ntp.org iburst\n"), 0o644)
				vfs := failfs.New(baseFs)
				_ = vfs.SetFailFunc(func(
					_ avfs.VFSBase,
					fn avfs.FnVFS,
					_ *failfs.FailParam,
				) error {
					if fn == avfs.FnRemove {
						return errors.New("permission denied")
					}

					return nil
				})
				suite.memFs = vfs
				suite.provider = ntp.NewDebianProvider(
					suite.logger,
					suite.memFs,
					suite.mockExec,
				)
			},
			setupMock: func() {},
			validateFunc: func(got *ntp.DeleteResult, err error) {
				suite.Require().Error(err)
				suite.Nil(got)
				suite.Contains(err.Error(), "ntp: remove file: permission denied")
			},
		},
		{
			name: "when reload fails logs warning but succeeds",
			setupFs: func() {
				_ = suite.memFs.MkdirAll(sourcesDir, 0o755)
				_ = suite.memFs.WriteFile(
					sourcesFile,
					[]byte("server 0.pool.ntp.org iburst\n"),
					0o644,
				)
			},
			setupMock: func() {
				suite.mockExec.EXPECT().
					RunPrivilegedCmd("chronyc", []string{"reload", "sources"}).
					Return("", errors.New("chronyc not running"))
			},
			validateFunc: func(got *ntp.DeleteResult, err error) {
				suite.Require().NoError(err)
				suite.Require().NotNil(got)
				suite.True(got.Changed)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.setupFs()
			tc.setupMock()

			got, err := suite.provider.Delete(context.Background())
			tc.validateFunc(got, err)
		})
	}
}

func (suite *DebianPublicTestSuite) TestParseTracking() {
	tests := []struct {
		name         string
		input        string
		validateFunc func(*ntp.Status)
	}{
		{
			name:  "when full tracking output parses all fields",
			input: trackingOutput,
			validateFunc: func(s *ntp.Status) {
				suite.True(s.Synchronized)
				suite.Equal(3, s.Stratum)
				suite.Equal("-0.000001834s", s.Offset)
				suite.Equal("time.cloudflare.com", s.CurrentSource)
			},
		},
		{
			name:  "when empty input returns zero status",
			input: "",
			validateFunc: func(s *ntp.Status) {
				suite.False(s.Synchronized)
				suite.Equal(0, s.Stratum)
				suite.Empty(s.Offset)
				suite.Empty(s.CurrentSource)
			},
		},
		{
			name:  "when malformed lines are skipped",
			input: "no colon here\nStratum         : 4\n",
			validateFunc: func(s *ntp.Status) {
				suite.Equal(4, s.Stratum)
			},
		},
		{
			name:  "when stratum is not a number defaults to zero",
			input: "Stratum         : abc\n",
			validateFunc: func(s *ntp.Status) {
				suite.Equal(0, s.Stratum)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := ntp.ParseTracking(tc.input)

			suite.NotNil(got)

			if tc.validateFunc != nil {
				tc.validateFunc(got)
			}
		})
	}
}

func (suite *DebianPublicTestSuite) TestParseSources() {
	tests := []struct {
		name         string
		input        string
		validateFunc func([]string)
	}{
		{
			name:  "when valid CSV extracts server names",
			input: sourcesOutput,
			validateFunc: func(got []string) {
				suite.Equal([]string{"time.cloudflare.com", "ntp.ubuntu.com"}, got)
			},
		},
		{
			name:  "when empty input returns nil",
			input: "",
			validateFunc: func(got []string) {
				suite.Equal([]string(nil), got)
			},
		},
		{
			name:  "when line has fewer than 3 fields skips it",
			input: "^,*\n^,+,valid.server,2,6\n",
			validateFunc: func(got []string) {
				suite.Equal([]string{"valid.server"}, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := ntp.ParseSources(tc.input)

			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestParseOffset() {
	tests := []struct {
		name         string
		input        string
		validateFunc func(string)
	}{
		{
			name:  "when fast returns positive offset",
			input: "0.000003422 seconds fast of NTP time",
			validateFunc: func(got string) {
				suite.Equal("+0.000003422s", got)
			},
		},
		{
			name:  "when slow returns negative offset",
			input: "0.000001834 seconds slow of NTP time",
			validateFunc: func(got string) {
				suite.Equal("-0.000001834s", got)
			},
		},
		{
			name:  "when too few fields returns empty",
			input: "0.0 seconds",
			validateFunc: func(got string) {
				suite.Equal("", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := ntp.ParseOffset(tc.input)

			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestParseReferenceID() {
	tests := []struct {
		name         string
		input        string
		validateFunc func(string)
	}{
		{
			name:  "when parens present extracts hostname",
			input: "A29FC801 (time.cloudflare.com)",
			validateFunc: func(got string) {
				suite.Equal("time.cloudflare.com", got)
			},
		},
		{
			name:  "when empty parens returns empty",
			input: "00000000 ()",
			validateFunc: func(got string) {
				suite.Equal("", got)
			},
		},
		{
			name:  "when no parens returns empty",
			input: "A29FC801",
			validateFunc: func(got string) {
				suite.Equal("", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := ntp.ParseReferenceID(tc.input)

			tc.validateFunc(got)
		})
	}
}

func (suite *DebianPublicTestSuite) TestGenerateContent() {
	tests := []struct {
		name         string
		servers      []string
		validateFunc func(string)
	}{
		{
			name:    "when multiple servers generates correct content",
			servers: []string{"0.pool.ntp.org", "1.pool.ntp.org"},
			validateFunc: func(got string) {
				suite.Equal("server 0.pool.ntp.org iburst\nserver 1.pool.ntp.org iburst\n", got)
			},
		},
		{
			name:    "when single server generates one line",
			servers: []string{"time.google.com"},
			validateFunc: func(got string) {
				suite.Equal("server time.google.com iburst\n", got)
			},
		},
		{
			name:    "when empty servers generates empty content",
			servers: []string{},
			validateFunc: func(got string) {
				suite.Equal("", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := ntp.GenerateContent(tc.servers)

			tc.validateFunc(string(got))
		})
	}
}

func (suite *DebianPublicTestSuite) TestComputeSHA256() {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "when given data returns consistent hash",
			data: []byte("server 0.pool.ntp.org iburst\n"),
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got1 := ntp.ComputeSHA256(tc.data)
			got2 := ntp.ComputeSHA256(tc.data)

			suite.Equal(got1, got2)
			suite.Len(got1, 64)
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
