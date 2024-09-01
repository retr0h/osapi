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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/manager/system"
	"github.com/retr0h/osapi/internal/manager/system/linux"
)

// TODO(retr0h): table driven

type LoadPublicTestSuite struct {
	suite.Suite

	appFs       afero.Fs
	loadAvgFile string
}

func (suite *LoadPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.loadAvgFile = "/proc/loadavg"
}

func (suite *LoadPublicTestSuite) SetupSubTest() {
	// initializes a new afero.Fs in the table tests
	suite.SetupTest()
}

func (suite *LoadPublicTestSuite) TearDownTest() {}

func (suite *LoadPublicTestSuite) TestGetHostnameOk() {
	tests := []struct {
		name    string
		content []byte
		want    [3]float64
		wantErr bool
	}{
		{
			name:    "when loadavg is valid",
			content: []byte("0.45 0.30 0.25 2/150 12345"),
			want:    [3]float64{0.45, 0.30, 0.25},
			wantErr: false,
		},
		{
			name:    "when missing loadAvgFile",
			content: []byte{},
			want:    [3]float64{},
			wantErr: true,
		},
		{
			name:    "when wrong format",
			content: []byte(" "),
			want:    [3]float64{},
			wantErr: true,
		},
		{
			name:    "when cannot parse 1m",
			content: []byte("invalid 0.30 0.25 2/150 12345"),
			want:    [3]float64{},
			wantErr: true,
		},
		{
			name:    "when cannot parse 5m",
			content: []byte("0.45 invalid 0.25 2/150 12345"),
			want:    [3]float64{},
			wantErr: true,
		},
		{
			name:    "when cannot parse 15m",
			content: []byte("0.30 0.30 invalid 2/150 12345"),
			want:    [3]float64{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if len(tc.content) != 0 {
				_ = afero.WriteFile(suite.appFs, suite.loadAvgFile, tc.content, 0o644)
			}

			sys := system.New(suite.appFs)
			sys.LoadProvider = linux.NewOSLoadProvider(suite.appFs)

			got, err := sys.GetLoadAverage()

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
func TestLoadPublicTestSuite(t *testing.T) {
	suite.Run(t, new(LoadPublicTestSuite))
}
