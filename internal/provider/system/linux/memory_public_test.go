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

package linux_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/provider/system"
	"github.com/retr0h/osapi/internal/provider/system/linux"
)

type MemoryPublicTestSuite struct {
	suite.Suite

	appFs       afero.Fs
	memInfoFile string
}

func (suite *MemoryPublicTestSuite) SetupTest() {
	suite.appFs = afero.NewMemMapFs()
	suite.memInfoFile = "/proc/meminfo"
}

func (suite *MemoryPublicTestSuite) SetupSubTest() {
	// initializes a new afero.Fs in the table tests
	suite.SetupTest()
}

func (suite *MemoryPublicTestSuite) TearDownTest() {}

func (suite *MemoryPublicTestSuite) TestGetHostname() {
	tests := []struct {
		name    string
		content []byte
		want    []uint64
		wantErr bool
	}{
		{
			name: "when meminfo is valid",
			content: []byte(`
MemTotal:       16384 kB
MemFree:        8192 kB
Cached:         2048 kB`),
			want:    []uint64{16384 * 1024, 8192 * 1024, (16384 - 8192 - 2048) * 1024},
			wantErr: false,
		},
		{
			name:    "when missing memInfoFile",
			content: []byte{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "when cannot parse uint64 handles underflow is a highly unlikely edge-case",
			content: []byte(`
MemTotal:       abcdef kB
MemFree:        8192 kB
Cached:         2048 kB`),
			want:    []uint64{0 * 1024, 8192 * 1024, 0 * 1024},
			wantErr: false,
		},
		{
			name: "when cannot parse uint64 defaults to 0",
			content: []byte(`
MemTotal:       16384 kB
MemFree:        abcd kB
Cached:         2048 kB`),
			want:    []uint64{16384 * 1024, 0 * 1024, (16384 - 0 - 2048) * 1024},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			if len(tc.content) != 0 {
				_ = afero.WriteFile(suite.appFs, suite.memInfoFile, tc.content, 0o644)
			}

			sys := system.New(suite.appFs)
			sys.MemoryProvider = linux.NewOSMemoryProvider(suite.appFs)

			got, err := sys.GetMemory()

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestMemoryPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryPublicTestSuite))
}
