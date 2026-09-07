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

package validation_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/validation"
)

type TargetPublicTestSuite struct {
	suite.Suite
}

type targetInput struct {
	Target string `validate:"required,valid_target"`
}

func (s *TargetPublicTestSuite) TestValidTarget() {
	tests := []struct {
		name         string
		setupLister  func()
		input        targetInput
		wantOK       bool
		contains     []string
		validateFunc func()
	}{
		{
			name: "when target is _any",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "_any"},
			wantOK: true,
		},
		{
			name: "when target is _all",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "_all"},
			wantOK: true,
		},
		{
			name: "when target is _all and all agents pending",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", State: "Pending"},
							{Hostname: "server2", State: "Pending"},
						}, nil
					},
				)
			},
			input:    targetInput{Target: "_all"},
			wantOK:   false,
			contains: []string{"all agents are pending PKI enrollment"},
		},
		{
			name: "when target is _any and all agents pending",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", State: "Pending"},
						}, nil
					},
				)
			},
			input:    targetInput{Target: "_any"},
			wantOK:   false,
			contains: []string{"all agents are pending PKI enrollment"},
		},
		{
			name: "when target is _all and lister returns error",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return nil, fmt.Errorf("nats down")
					},
				)
			},
			input:  targetInput{Target: "_all"},
			wantOK: true, // error → can't determine pending, allow through
		},
		{
			name: "when target is _all and no agents registered",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{}, nil
					},
				)
			},
			input:  targetInput{Target: "_all"},
			wantOK: true, // no agents → can't determine pending, allow through
		},
		{
			name: "when target is _all and lister is nil",
			setupLister: func() {
				validation.RegisterTargetValidator(nil)
			},
			input:  targetInput{Target: "_all"},
			wantOK: true, // nil lister → can't determine pending, allow through
		},
		{
			name: "when target is _all with mix of pending and ready agents",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", State: "Pending"},
							{Hostname: "server2", State: "Ready"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "_all"},
			wantOK: true,
		},
		{
			name: "when target is a label with exact match",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "group:web"},
			wantOK: true,
		},
		{
			name: "when target is a label with hierarchical prefix match",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{
								Hostname: "server1",
								Labels:   map[string]string{"group": "web.dev.us-east"},
							},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "group:web.dev"},
			wantOK: true,
		},
		{
			name: "when target label does not match any agent",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "env:prod"},
			wantOK: false,
		},
		{
			name: "when label has empty key",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: ":value"},
			wantOK: false,
		},
		{
			name: "when label has empty value",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "key:"},
			wantOK: false,
		},
		{
			name: "when label is malformed colons",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: ":::"},
			wantOK: false,
		},
		{
			name: "when label key has invalid characters",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "gr@up:web"},
			wantOK: false,
		},
		{
			name: "when label value segment has invalid characters",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "group:web/dev"},
			wantOK: false,
		},
		{
			name: "when target is a known agent hostname",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:  targetInput{Target: "server1"},
			wantOK: true,
		},
		{
			name: "when target matches a pending agent returns false with pending message",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "pending-host", State: "Pending"},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:    targetInput{Target: "pending-host"},
			wantOK:   false,
			contains: []string{"valid_target", "pending PKI enrollment"},
		},
		{
			name: "when target is an unknown hostname",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{Hostname: "server1", Labels: map[string]string{"group": "web"}},
							{Hostname: "server2"},
						}, nil
					},
				)
			},
			input:    targetInput{Target: "nonexistent"},
			wantOK:   false,
			contains: []string{"valid_target", "target agent", "nonexistent", "not found"},
		},
		{
			name: "when lister returns error for hostname",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return nil, fmt.Errorf("nats connection failed")
					},
				)
			},
			input:  targetInput{Target: "server1"},
			wantOK: false,
		},
		{
			name: "when lister returns error for label",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return nil, fmt.Errorf("nats connection failed")
					},
				)
			},
			input:  targetInput{Target: "group:web"},
			wantOK: false,
		},
		{
			name: "when lister is nil",
			setupLister: func() {
				validation.RegisterTargetValidator(nil)
			},
			input:  targetInput{Target: "server1"},
			wantOK: false,
		},
		{
			name: "when same target validated twice uses cache",
			validateFunc: func() {
				callCount := 0
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						callCount++

						return []validation.AgentTarget{
							{Hostname: "server1"},
						}, nil
					},
				)

				// First call populates cache.
				_, ok := validation.Struct(targetInput{Target: "server1"})
				s.True(ok)
				s.Equal(1, callCount)

				// Second call should use cache, not call lister again.
				_, ok = validation.Struct(targetInput{Target: "server1"})
				s.True(ok)
				s.Equal(1, callCount)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.validateFunc != nil {
				tt.validateFunc()

				return
			}

			tt.setupLister()

			errMsg, ok := validation.Struct(tt.input)
			s.Equal(tt.wantOK, ok)

			if !ok {
				for _, c := range tt.contains {
					s.Contains(errMsg, c)
				}
			}
		})
	}
}

func (s *TargetPublicTestSuite) TestResolveTarget() {
	tests := []struct {
		name         string
		setupLister  func()
		target       string
		validateFunc func(string)
	}{
		{
			name:   "when target is _any returns unchanged",
			target: "_any",
			validateFunc: func(got string) {
				s.Equal("_any", got)
			},
		},
		{
			name:   "when target is _all returns unchanged",
			target: "_all",
			validateFunc: func(got string) {
				s.Equal("_all", got)
			},
		},
		{
			name:   "when target is label returns unchanged",
			target: "group:web",
			validateFunc: func(got string) {
				s.Equal("group:web", got)
			},
		},
		{
			name: "when target is hostname resolves to machine ID",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{MachineID: "abc123", Hostname: "web-01"},
						}, nil
					},
				)
			},
			target: "web-01",
			validateFunc: func(got string) {
				s.Equal("abc123", got)
			},
		},
		{
			name: "when target is machine ID returns unchanged",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{
							{MachineID: "abc123", Hostname: "web-01"},
						}, nil
					},
				)
			},
			target: "abc123",
			validateFunc: func(got string) {
				s.Equal("abc123", got)
			},
		},
		{
			name: "when target not found returns unchanged",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return []validation.AgentTarget{}, nil
					},
				)
			},
			target: "unknown",
			validateFunc: func(got string) {
				s.Equal("unknown", got)
			},
		},
		{
			name: "when lister errors returns unchanged",
			setupLister: func() {
				validation.RegisterTargetValidator(
					func(_ context.Context) ([]validation.AgentTarget, error) {
						return nil, fmt.Errorf("nats down")
					},
				)
			},
			target: "web-01",
			validateFunc: func(got string) {
				s.Equal("web-01", got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.setupLister != nil {
				tt.setupLister()
			}
			got := validation.ResolveTarget(tt.target)
			tt.validateFunc(got)
		})
	}
}

func (s *TargetPublicTestSuite) TestValidTargetMatchesMachineID() {
	validation.RegisterTargetValidator(
		func(_ context.Context) ([]validation.AgentTarget, error) {
			return []validation.AgentTarget{
				{MachineID: "abc123-def456", Hostname: "web-01"},
			}, nil
		},
	)

	// Machine ID should pass validation
	_, ok := validation.Struct(targetInput{Target: "abc123-def456"})
	s.True(ok)
}

func TestTargetPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TargetPublicTestSuite))
}
