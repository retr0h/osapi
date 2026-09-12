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

	"github.com/osapi-io/osapi/internal/provider/file"
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
	mock := NewPlainMockProvider(ctrl)

	mock.EXPECT().Deploy(gomock.Any(), gomock.Any()).Return(&file.DeployResult{
		Changed: true,
		SHA256:  "abc123def456",
		Path:    "/etc/mock/file.conf",
	}, nil).AnyTimes()

	mock.EXPECT().Status(gomock.Any(), gomock.Any()).Return(&file.StatusResult{
		Path:   "/etc/mock/file.conf",
		Status: "in-sync",
		SHA256: "abc123def456",
	}, nil).AnyTimes()

	return mock
}
