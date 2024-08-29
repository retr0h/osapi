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

package system_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/api"
	"github.com/retr0h/osapi/internal/api/system"
	systemGen "github.com/retr0h/osapi/internal/api/system/gen"
	"github.com/retr0h/osapi/internal/config"
)

type SystemStatusIntegrationTestSuite struct {
	suite.Suite

	appFs     afero.Fs
	appConfig config.Config
	logger    *slog.Logger
}

func (suite *SystemStatusIntegrationTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.appConfig = config.Config{}
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	osRelease := `
NAME="Ubuntu"
VERSION_ID="22.04"
`
	_ = afero.WriteFile(suite.appFs, "/etc/os-release", []byte(osRelease), 0o644)
}

func (suite *SystemStatusIntegrationTestSuite) TestGetSystemStatusOk() {
	procUptime := "350735.47 234388.90"
	_ = afero.WriteFile(suite.appFs, "/proc/uptime", []byte(procUptime), 0o644)
	procHostname := "test-hostname"
	_ = afero.WriteFile(suite.appFs, "/proc/sys/kernel/hostname", []byte(procHostname), 0o644)

	a := api.New(suite.appFs, suite.appConfig, suite.logger)
	systemGen.RegisterHandlers(a, system.New(suite.appFs))

	// Create a new request to the /system/status endpoint
	req := httptest.NewRequest(http.MethodGet, "/system/status", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	a.ServeHTTP(rec, req)

	assert.Equal(suite.T(), http.StatusOK, rec.Code)
	want := `{
"disk": {
    "free": 0,
    "total": 0,
    "used": 0
},
"hostname": "test-hostname",
"load_average": {
    "15min": 0,
    "1min": 0,
    "5min": 0
},
"memory": {
    "free": 0,
    "total": 0,
    "used": 0
},
"uptime": "4 days 1 hours 25 minutes"
}`
	assert.JSONEq(suite.T(), want, rec.Body.String())
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestSystemStatusIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SystemStatusIntegrationTestSuite))
}
