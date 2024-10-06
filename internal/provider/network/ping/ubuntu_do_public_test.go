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

package ping_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/network/ping"
	"github.com/retr0h/osapi/internal/provider/network/ping/mocks"
)

type UbuntuDoPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appFs          afero.Fs
	resolvConfFile string
}

func (suite *UbuntuDoPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appFs = afero.NewMemMapFs()
	suite.resolvConfFile = "/run/systemd/resolve/resolv.conf"
}

func (suite *UbuntuDoPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *UbuntuDoPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *UbuntuDoPublicTestSuite) TestDo() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockProvider
		address     string
		want        *ping.Result
		wantErr     bool
		wantErrType error
	}{
		{
			name:    "when Do Ok",
			address: "1.1.1.1",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			want: &ping.Result{
				PacketsSent:     3,
				PacketsReceived: 3,
				PacketLoss:      0,
				MinRTT:          10 * time.Millisecond,
				AvgRTT:          15 * time.Millisecond,
				MaxRTT:          20 * time.Millisecond,
			},
			wantErr: false,
		},
		{
			name:    "when Do errors",
			address: "1.1.1.1",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().Do("1.1.1.1").Return(nil, assert.AnError)

				return mock
			},
			want:        nil,
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			got, err := mock.Do(tc.address)

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuDoPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuDoPublicTestSuite))
}
