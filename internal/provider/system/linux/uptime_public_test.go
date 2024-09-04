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

package linux_test

import (
	"math"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system"
	"github.com/retr0h/osapi/internal/provider/system/linux"
)

type UptimePublicTestSuite struct {
	suite.Suite

	appFs      afero.Fs
	uptimeFile string
}

func (suite *UptimePublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.uptimeFile = "/proc/uptime"
}

func (suite *UptimePublicTestSuite) SetupSubTest() {
	// initializes a new afero.Fs in the table tests
	suite.SetupTest()
}

func (suite *UptimePublicTestSuite) TestGetUptime() {
	tests := []struct {
		name    string
		content []byte
		want    time.Duration
		wantErr bool
	}{
		{
			name:    "when uptime is valid",
			content: []byte(`350735.47 234388.90`),
			want:    time.Duration(int64(math.Trunc(350735.47))) * time.Second,
			wantErr: false,
		},
		{
			name:    "when missing uptimeFile",
			content: []byte{},
			want:    0,
			wantErr: true,
		},
		{
			name:    "when wrong format",
			content: []byte(" "),
			want:    0,
			wantErr: true,
		},
		{
			name:    "when cannot parse float",
			content: []byte("foo bar"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if len(tc.content) != 0 {
				_ = afero.WriteFile(suite.appFs, suite.uptimeFile, tc.content, 0o644)
			}

			sys := system.New(suite.appFs)
			sys.UptimeProvider = linux.NewOSUptimeProvider(suite.appFs)

			got, err := sys.GetUptime()

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUptimePublicTestSuite(t *testing.T) {
	suite.Run(t, new(UptimePublicTestSuite))
}