// Copyright (c) 2026 John Dewey

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
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/provider/network/netinfo"
)

// NewPlainMockProvider creates a Mock without defaults.
func NewPlainMockProvider(
	ctrl *gomock.Controller,
) *MockProvider {
	return NewMockProvider(ctrl)
}

// NewDefaultMockProvider creates a Mock with defaults.
func NewDefaultMockProvider(
	ctrl *gomock.Controller,
) *MockProvider {
	mock := NewMockProvider(ctrl)

	mock.EXPECT().GetInterfaces().Return([]netinfo.InterfaceResult{
		{
			Name:   "eth0",
			IPv4:   "192.168.1.10",
			IPv6:   "fe80::1",
			MAC:    "00:11:22:33:44:55",
			Family: "dual",
		},
	}, nil).AnyTimes()

	mock.EXPECT().GetRoutes().Return([]netinfo.RouteResult{
		{
			Destination: "0.0.0.0",
			Gateway:     "192.168.1.1",
			Interface:   "eth0",
			Mask:        "/0",
			Metric:      100,
		},
	}, nil).AnyTimes()

	mock.EXPECT().GetPrimaryInterface().Return("eth0", nil).AnyTimes()

	return mock
}
