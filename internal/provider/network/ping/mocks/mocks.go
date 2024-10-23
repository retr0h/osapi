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

package mocks

import (
	"time"

	"github.com/golang/mock/gomock"
	probing "github.com/prometheus-community/pro-bing"

	"github.com/retr0h/osapi/internal/provider/network/ping"
)

// NewPlainMockProvider creates a Mock without defaults.
func NewPlainMockProvider(ctrl *gomock.Controller) *MockProvider {
	return NewMockProvider(ctrl)
}

// NewDefaultMockProvider creates a Mock with defaults.
func NewDefaultMockProvider(ctrl *gomock.Controller) *MockProvider {
	mock := NewPlainMockProvider(ctrl)

	result := &ping.Result{
		PacketsSent:     3,
		PacketsReceived: 3,
		PacketLoss:      0,
		MinRTT:          10 * time.Millisecond,
		AvgRTT:          15 * time.Millisecond,
		MaxRTT:          20 * time.Millisecond,
	}

	mock.EXPECT().Do("1.1.1.1").Return(result, nil).AnyTimes()

	mock.EXPECT().Do("2001:4860:4860::8888").Return(result, nil).AnyTimes()

	return mock
}

// NewPlainMockPinger creates a pinger Mock without defaults.
func NewPlainMockPinger(ctrl *gomock.Controller) *MockPinger {
	return NewMockPinger(ctrl)
}

// NewDefaultMockPinger creates a pinger Mock with defaults.
func NewDefaultMockPinger(ctrl *gomock.Controller) *MockPinger {
	mock := NewPlainMockPinger(ctrl)

	SetCommonExpectations(mock)
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
}

// SetCommonExpectations sets the common expectations on the mock pinger.
func SetCommonExpectations(mock *MockPinger) {
	mock.EXPECT().SetCount(3)
	mock.EXPECT().SetPrivileged(false)
}
