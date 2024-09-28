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
	"github.com/golang/mock/gomock"

	"github.com/retr0h/osapi/internal/provider/system"
)

// NewPlainMockProvider creates a Mock without defaults.
func NewPlainMockProvider(ctrl *gomock.Controller) *MockProvider {
	return NewMockProvider(ctrl)
}

// NewDefaultMockProvider creates a Mock with defaults.
func NewDefaultMockProvider(ctrl *gomock.Controller) *MockProvider {
	mock := NewMockProvider(ctrl)

	mock.EXPECT().GetLocalDiskStats().Return([]system.DiskUsageStats{
		{
			Name:  "/dev/disk1",
			Total: 500000000000,
			Used:  250000000000,
			Free:  250000000000,
		},
	}, nil).AnyTimes()

	return mock
}
