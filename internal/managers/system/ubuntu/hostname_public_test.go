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

package ubuntu_test

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/managers/system"
	"github.com/retr0h/osapi/internal/managers/system/ubuntu/mocks"
)

type UbuntuPublicTestSuite struct {
	suite.Suite

	appFs afero.Fs
	ctrl  *gomock.Controller
}

func (suite *UbuntuPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.ctrl = gomock.NewController(suite.T())
}

func (suite *UbuntuPublicTestSuite) TearDownTest() {
	defer suite.ctrl.Finish()
}

func (suite *UbuntuPublicTestSuite) TestGetHostnameOk() {
	mockHostnameProvider := mocks.NewMockHostnameProvider(suite.ctrl)
	mockHostnameProvider.EXPECT().GetHostname().Return("test-hostname", nil)

	sys := system.New(suite.appFs)
	sys.HostnameProvider = mockHostnameProvider

	got, err := sys.GetHostname()

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-hostname", got)
}

func (suite *UbuntuPublicTestSuite) TestGetHostnameReturnsError() {
	mockHostnameProvider := mocks.NewMockHostnameProvider(suite.ctrl)
	mockHostnameProvider.EXPECT().GetHostname().Return("", errors.New("unable to get hostname"))

	sys := system.New(suite.appFs)
	sys.HostnameProvider = mockHostnameProvider

	_, err := sys.GetHostname()

	assert.Error(suite.T(), err)
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuPublicTestSuite))
}
