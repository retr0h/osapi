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
	"fmt"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/platform"
)

type PlatformPublicTestSuite struct {
	suite.Suite
}

func TestPlatformPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

func (suite *PlatformPublicTestSuite) TestDetect() {
	tests := []struct {
		name     string
		infoFn   func() (*host.InfoStat, error)
		expected string
	}{
		{
			name: "returns debian family when platform is Ubuntu",
			infoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{Platform: "Ubuntu"}, nil
			},
			expected: "debian",
		},
		{
			name: "returns debian family when platform is debian",
			infoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{Platform: "debian"}, nil
			},
			expected: "debian",
		},
		{
			name: "returns darwin when platform is empty and OS is darwin",
			infoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{Platform: "", OS: "darwin"}, nil
			},
			expected: "darwin",
		},
		{
			name: "returns centos for centos platform",
			infoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{Platform: "centos"}, nil
			},
			expected: "centos",
		},
		{
			name: "returns empty string when info is nil",
			infoFn: func() (*host.InfoStat, error) {
				return nil, fmt.Errorf("no host info")
			},
			expected: "",
		},
		{
			name: "returns empty string when platform and OS are empty",
			infoFn: func() (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			original := platform.HostInfoFn
			defer func() { platform.HostInfoFn = original }()

			platform.HostInfoFn = tc.infoFn
			result := platform.Detect()

			assert.Equal(suite.T(), tc.expected, result)
		})
	}
}
