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

package network_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/network"
	"github.com/retr0h/osapi/internal/provider/network/mocks"
)

type UbuntuPingPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	appFs          afero.Fs
	resolvConfFile string
}

func (suite *UbuntuPingPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.appFs = afero.NewMemMapFs()
	suite.resolvConfFile = "/run/systemd/resolve/resolv.conf"
}

func (suite *UbuntuPingPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *UbuntuPingPublicTestSuite) TearDownTest() {}

func (suite *UbuntuPingPublicTestSuite) TestPingHost() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockProvider
		address     string
		want        *network.PingResult
		wantErr     bool
		wantErrType error
	}{
		{
			name:    "when PingHost Ok",
			address: "example.com",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewDefaultMockProvider(suite.ctrl)

				return mock
			},
			want: &network.PingResult{
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
			name:    "when PingHost returns an error",
			address: "example.com",
			setupMock: func() *mocks.MockProvider {
				mock := mocks.NewPlainMockProvider(suite.ctrl)
				mock.EXPECT().PingHost("example.com").Return(nil, assert.AnError)

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
			got, err := mock.PingHost(tc.address)

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
func TestUbuntuPingPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuPingPublicTestSuite))
}
