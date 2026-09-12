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

package disk_test

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider/node/disk"
)

type DiskPublicTestSuite struct {
	suite.Suite
}

func (suite *DiskPublicTestSuite) SetupTest() {}

func (suite *DiskPublicTestSuite) TearDownTest() {}

func (suite *DiskPublicTestSuite) TestIsPermissionError() {
	tests := []struct {
		name         string
		err          error
		validateFunc func(bool)
	}{
		{
			name: "when error is EACCES wrapped in os.PathError",
			err: &os.PathError{
				Op:   "open",
				Path: "/restricted",
				Err:  syscall.EACCES,
			},
			validateFunc: func(got bool) {
				suite.Equal(true, got)
			},
		},
		{
			name: "when error is a different syscall errno",
			err: &os.PathError{
				Op:   "open",
				Path: "/file",
				Err:  syscall.EPERM, // Not EACCES.
			},
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
		{
			name: "when error is not a PathError",
			err:  errors.New("some other error"),
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
		{
			name: "when PathError contains non-syscall error",
			err: &os.PathError{
				Op:   "open",
				Path: "/file",
				Err:  errors.New("some random error"),
			},
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
		{
			name: "when error is nil",
			err:  nil,
			validateFunc: func(got bool) {
				suite.Equal(false, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			got := disk.ExportIsPermissionError(tc.err)

			tc.validateFunc(got)
		})
	}
}

func TestDiskPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DiskPublicTestSuite))
}
