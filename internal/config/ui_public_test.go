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

package config_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/config"
)

type UIPublicTestSuite struct {
	suite.Suite
}

func (s *UIPublicTestSuite) TestUIEnabled() {
	trueVal := true
	falseVal := false

	tests := []struct {
		name         string
		cfg          config.UIConfig
		validateFunc func(bool)
	}{
		{
			name: "defaults to true when Enabled is nil",
			cfg:  config.UIConfig{},
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "returns true when Enabled is explicitly true",
			cfg:  config.UIConfig{Enabled: &trueVal},
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "returns false when Enabled is explicitly false",
			cfg:  config.UIConfig{Enabled: &falseVal},
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(tc.cfg.UIEnabled())
		})
	}
}

func TestUIPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UIPublicTestSuite))
}
