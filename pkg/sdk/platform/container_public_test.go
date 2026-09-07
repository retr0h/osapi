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

package platform_test

import (
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/platform"
)

type ContainerPublicTestSuite struct {
	suite.Suite
}

func (s *ContainerPublicTestSuite) TearDownSubTest() {
	platform.SetContainerFS(nil)
}

func (s *ContainerPublicTestSuite) TestIsContainer() {
	tests := []struct {
		name    string
		setupFS func()
		want    bool
	}{
		{
			name: "when /.dockerenv exists",
			setupFS: func() {
				fs := memfs.New()
				_ = fs.WriteFile("/.dockerenv", []byte(""), 0o644)
				platform.SetContainerFS(fs)
			},
			want: true,
		},
		{
			name: "when /.dockerenv does not exist",
			setupFS: func() {
				fs := memfs.New()
				platform.SetContainerFS(fs)
			},
			want: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.setupFS()

			got := platform.IsContainer()

			s.Equal(tc.want, got)
		})
	}
}

func TestContainerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ContainerPublicTestSuite))
}
