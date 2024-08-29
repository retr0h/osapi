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

	"github.com/retr0h/osapi/internal/manager/system"
	"github.com/retr0h/osapi/internal/manager/system/linux"
)

type UptimePublicTestSuite struct {
	suite.Suite

	appFs      afero.Fs
	uptimePath string
}

func (suite *UptimePublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.uptimePath = "/proc/uptime"
}

func (suite *UptimePublicTestSuite) TestGetUptimeOk() {
	procUptime := "350735.47 234388.90"
	_ = afero.WriteFile(suite.appFs, suite.uptimePath, []byte(procUptime), 0o644)

	sys := system.New(suite.appFs)
	sys.UptimeProvider = linux.NewOSUptimeProvider(suite.appFs)

	got, err := sys.GetUptime()

	assert.NoError(suite.T(), err)
	want := time.Duration(int64(math.Trunc(350735.47))) * time.Second
	assert.Equal(suite.T(), want, got)
}

func (suite *UptimePublicTestSuite) TestGetUptimeReturnsError() {
	sys := system.New(suite.appFs)
	sys.UptimeProvider = linux.NewOSUptimeProvider(suite.appFs)

	_, err := sys.GetUptime()

	assert.Error(suite.T(), err)
}

func (suite *UptimePublicTestSuite) TestGetUptimeReturnsErrorWhenWrongFormat() {
	procUptime := ""
	_ = afero.WriteFile(suite.appFs, suite.uptimePath, []byte(procUptime), 0o644)

	sys := system.New(suite.appFs)
	sys.UptimeProvider = linux.NewOSUptimeProvider(suite.appFs)

	_, err := sys.GetUptime()

	assert.Error(suite.T(), err)
}

func (suite *UptimePublicTestSuite) TestGetUptimeReturnsErrorWhenCannotParseFloat() {
	procUptime := "foo bar"
	_ = afero.WriteFile(suite.appFs, suite.uptimePath, []byte(procUptime), 0o644)

	sys := system.New(suite.appFs)
	sys.UptimeProvider = linux.NewOSUptimeProvider(suite.appFs)

	_, err := sys.GetUptime()

	assert.Error(suite.T(), err)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUptimePublicTestSuite(t *testing.T) {
	suite.Run(t, new(UptimePublicTestSuite))
}
