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
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/network/ping"
	"github.com/retr0h/osapi/internal/provider/network/ping/mocks"
)

type UbuntuDoPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (suite *UbuntuDoPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
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
		setupMock   func() *mocks.MockPinger
		address     string
		want        *ping.Result
		wantErr     bool
		wantErrType error
	}{
		{
			name:    "when Do Ok",
			address: "1.1.1.1",
			setupMock: func() *mocks.MockPinger {
				mock := mocks.NewDefaultMockPinger(suite.ctrl)

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
			name:    "when NewPingerFn errors",
			address: "invalid-address",
			setupMock: func() *mocks.MockPinger {
				return nil
			},
			wantErr:     true,
			wantErrType: fmt.Errorf("failed to initialize pinger"),
		},
		{
			name:    "when pinger.Run errors",
			address: "1.1.1.1",
			setupMock: func() *mocks.MockPinger {
				mock := mocks.NewPlainMockPinger(suite.ctrl)

				mocks.SetCommonExpectations(mock)
				mock.EXPECT().Run().Return(assert.AnError)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name:    "when ping operation times out",
			address: "1.1.1.1",
			setupMock: func() *mocks.MockPinger {
				mock := mocks.NewMockPinger(suite.ctrl)

				mocks.SetCommonExpectations(mock)
				mock.EXPECT().Run().DoAndReturn(func() error {
					time.Sleep(10 * time.Second)
					return nil
				})

				return mock
			},
			wantErr:     true,
			wantErrType: fmt.Errorf("ping operation timed out after 5s"),
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()

			ubuntu := ping.NewUbuntuProvider()
			if mock != nil {
				ubuntu.NewPingerFn = func(address string) (ping.Pinger, error) {
					return mock, nil
				}
			}

			got, err := ubuntu.Do(tc.address)

			if !tc.wantErr {
				suite.NoError(err)
				suite.Equal(tc.want, got)
			} else {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestUbuntuDoPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UbuntuDoPublicTestSuite))
}
