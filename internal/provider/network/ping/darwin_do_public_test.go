// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
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

	probing "github.com/prometheus-community/pro-bing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/provider/network/ping"
	"github.com/osapi-io/osapi/internal/provider/network/ping/mocks"
)

type DarwinDoPublicTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
}

func (suite *DarwinDoPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
}

func (suite *DarwinDoPublicTestSuite) SetupSubTest() {
	suite.SetupTest()
}

func (suite *DarwinDoPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *DarwinDoPublicTestSuite) TestDo() {
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
				mock := mocks.NewPlainMockPinger(suite.ctrl)

				mock.EXPECT().SetCount(3)
				mock.EXPECT().SetPrivileged(true)
				mock.EXPECT().Run().Return(nil)
				mock.EXPECT().Statistics().Return(&probing.Statistics{
					PacketsSent: 3,
					PacketsRecv: 3,
					PacketLoss:  0,
					MinRtt:      10 * time.Millisecond,
					AvgRtt:      15 * time.Millisecond,
					MaxRtt:      20 * time.Millisecond,
				})

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

				mock.EXPECT().SetCount(3)
				mock.EXPECT().SetPrivileged(true)
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

				mock.EXPECT().SetCount(3)
				mock.EXPECT().SetPrivileged(true)
				mock.EXPECT().Run().DoAndReturn(func() error {
					time.Sleep(10 * time.Second)
					return nil
				})
				// The goroutine may call Statistics() after timeout
				mock.EXPECT().Statistics().Return(&probing.Statistics{}).AnyTimes()

				return mock
			},
			wantErr:     true,
			wantErrType: fmt.Errorf("ping operation timed out after 5s"),
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()

			darwin := ping.NewDarwinProvider()
			if mock != nil {
				darwin.NewPingerFn = func(_ string) (ping.Pinger, error) {
					return mock, nil
				}
			}

			got, err := darwin.Do(tc.address)

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

func (suite *DarwinDoPublicTestSuite) TestNewDarwinProvider() {
	tests := []struct {
		name         string
		address      string
		wantErr      bool
		validateFunc func(p ping.Pinger)
	}{
		{
			name:    "when valid address creates pinger",
			address: "127.0.0.1",
			validateFunc: func(p ping.Pinger) {
				suite.NotNil(p)
			},
		},
		{
			name:    "when invalid address returns error",
			address: "invalid-address",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			darwin := ping.NewDarwinProvider()

			pinger, err := darwin.NewPingerFn(tc.address)

			if tc.wantErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				if tc.validateFunc != nil {
					tc.validateFunc(pinger)
				}
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDarwinDoPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DarwinDoPublicTestSuite))
}
