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

	"go.uber.org/mock/gomock"

	"github.com/osapi-io/osapi/internal/provider/node/host"
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

	mock.EXPECT().GetUptime().Return(time.Hour*5, nil).AnyTimes()
	mock.EXPECT().GetHostname().Return("default-hostname", nil).AnyTimes()

	mock.EXPECT().GetOSInfo().Return(&host.Result{
		Distribution: "Ubuntu",
		Version:      "24.04",
	}, nil).AnyTimes()

	mock.EXPECT().GetArchitecture().Return("amd64", nil).AnyTimes()
	mock.EXPECT().GetKernelVersion().Return("5.15.0-91-generic", nil).AnyTimes()
	mock.EXPECT().GetFQDN().Return("default-hostname.local", nil).AnyTimes()
	mock.EXPECT().GetCPUCount().Return(4, nil).AnyTimes()
	mock.EXPECT().GetServiceManager().Return("systemd", nil).AnyTimes()
	mock.EXPECT().GetPackageManager().Return("apt", nil).AnyTimes()

	return mock
}
