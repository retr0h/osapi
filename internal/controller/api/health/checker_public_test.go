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

package health_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/controller/api/health"
)

type CheckerPublicTestSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *CheckerPublicTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *CheckerPublicTestSuite) TestCheckHealth() {
	tests := []struct {
		name      string
		checker   *health.NATSChecker
		expectErr bool
		errMsg    string
	}{
		{
			name: "all checks pass",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return nil },
			},
			expectErr: false,
		},
		{
			name: "NATS check fails",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats error") },
				KVCheck:   func() error { return nil },
			},
			expectErr: true,
			errMsg:    "nats error",
		},
		{
			name: "KV check fails",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
				KVCheck:   func() error { return fmt.Errorf("kv error") },
			},
			expectErr: true,
			errMsg:    "kv error",
		},
		{
			name: "both checks fail",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats error") },
				KVCheck:   func() error { return fmt.Errorf("kv error") },
			},
			expectErr: true,
			errMsg:    "nats error",
		},
		{
			name:      "nil checks pass",
			checker:   &health.NATSChecker{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.checker.CheckHealth(s.ctx)

			if tt.expectErr {
				s.Error(err)
				s.Contains(err.Error(), tt.errMsg)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *CheckerPublicTestSuite) TestCheckNATS() {
	tests := []struct {
		name      string
		checker   *health.NATSChecker
		expectErr bool
	}{
		{
			name: "NATS check passes",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return nil },
			},
			expectErr: false,
		},
		{
			name: "NATS check fails",
			checker: &health.NATSChecker{
				NATSCheck: func() error { return fmt.Errorf("nats error") },
			},
			expectErr: true,
		},
		{
			name:      "nil NATS check passes",
			checker:   &health.NATSChecker{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.checker.CheckNATS()

			if tt.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *CheckerPublicTestSuite) TestCheckKV() {
	tests := []struct {
		name      string
		checker   *health.NATSChecker
		expectErr bool
	}{
		{
			name: "KV check passes",
			checker: &health.NATSChecker{
				KVCheck: func() error { return nil },
			},
			expectErr: false,
		},
		{
			name: "KV check fails",
			checker: &health.NATSChecker{
				KVCheck: func() error { return fmt.Errorf("kv error") },
			},
			expectErr: true,
		},
		{
			name:      "nil KV check passes",
			checker:   &health.NATSChecker{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := tt.checker.CheckKV()

			if tt.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
			}
		})
	}
}

func TestCheckerPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CheckerPublicTestSuite))
}
