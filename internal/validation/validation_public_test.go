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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/go-playground/validator/v10"

	"github.com/osapi-io/osapi/internal/validation"
)

type ValidationPublicTestSuite struct {
	suite.Suite
}

func (s *ValidationPublicTestSuite) TestStruct() {
	type testStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	tests := []struct {
		name         string
		input        any
		validateFunc func(string, bool)
	}{
		{
			name: "when valid struct",
			input: testStruct{
				Name:  "test",
				Email: "test@example.com",
			},
			validateFunc: func(_ string, ok bool) {
				s.Equal(true, ok)
			},
		},
		{
			name: "when missing required field",
			input: testStruct{
				Email: "test@example.com",
			},
			validateFunc: func(errMsg string, ok bool) {
				s.Equal(false, ok)
				s.Contains(errMsg, "Name")
				s.Contains(errMsg, "required")
			},
		},
		{
			name: "when invalid email",
			input: testStruct{
				Name:  "test",
				Email: "not-an-email",
			},
			validateFunc: func(errMsg string, ok bool) {
				s.Equal(false, ok)
				s.Contains(errMsg, "Email")
				s.Contains(errMsg, "email")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(validation.Struct(tt.input))
		})
	}
}

func (s *ValidationPublicTestSuite) TestVar() {
	tests := []struct {
		name         string
		field        any
		tag          string
		validateFunc func(string, bool)
	}{
		{
			name:  "when valid field",
			field: "hello",
			tag:   "required",
			validateFunc: func(_ string, ok bool) {
				s.Equal(true, ok)
			},
		},
		{
			name:  "when empty required field",
			field: "",
			tag:   "required",
			validateFunc: func(errMsg string, ok bool) {
				s.Equal(false, ok)
				s.Contains(errMsg, "required")
			},
		},
		{
			name:  "when invalid email",
			field: "not-an-email",
			tag:   "email",
			validateFunc: func(errMsg string, ok bool) {
				s.Equal(false, ok)
				s.Contains(errMsg, "email")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(validation.Var(tt.field, tt.tag))
		})
	}
}

func (s *ValidationPublicTestSuite) TestAlphanumOrFact() {
	tests := []struct {
		name         string
		field        string
		validateFunc func(bool)
	}{
		{
			name:  "when alphanumeric value",
			field: "eth0",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when fact reference",
			field: "@fact.interface.primary",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when fact custom reference",
			field: "@fact.custom.mykey",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when non-alphanum non-fact value",
			field: "eth-0!",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when empty value",
			field: "",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when partial fact prefix",
			field: "@fact",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when at-sign without fact",
			field: "@notfact.x",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when unknown fact key",
			field: "@fact.primary_interface",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when fact with bare custom prefix",
			field: "@fact.custom.",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, ok := validation.Var(tt.field, "required,alphanum_or_fact")
			tt.validateFunc(ok)
		})
	}
}

func (s *ValidationPublicTestSuite) TestIpOrFact() {
	tests := []struct {
		name         string
		field        string
		validateFunc func(bool)
	}{
		{
			name:  "when valid IPv4",
			field: "1.1.1.1",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when valid IPv6",
			field: "::1",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when fact reference",
			field: "@fact.custom.gateway",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when fact interface primary",
			field: "@fact.interface.primary",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name:  "when invalid address",
			field: "not-an-ip",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when empty value",
			field: "",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when partial fact prefix",
			field: "@fact",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when at-sign without fact",
			field: "@notfact.x",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when unknown fact key",
			field: "@fact.primary_interface",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name:  "when fact with bare custom prefix",
			field: "@fact.custom.",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, ok := validation.Var(tt.field, "required,ip_or_fact")
			tt.validateFunc(ok)
		})
	}
}

func (s *ValidationPublicTestSuite) TestCronSchedule() {
	tests := []struct {
		name         string
		field        string
		contains     []string
		validateFunc func(string, bool)
	}{
		// Valid expressions
		{
			name:  "when every minute",
			field: "* * * * *",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when daily at 2am",
			field: "0 2 * * *",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when every 5 minutes",
			field: "*/5 * * * *",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when weekdays at 9am",
			field: "0 9 * * 1-5",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when first of month at midnight",
			field: "0 0 1 * *",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when multiple hours",
			field: "0 2,14 * * *",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when range with step",
			field: "0-30/5 * * * *",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when month and day names",
			field: "0 0 * jan-mar mon",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		// Invalid expressions
		{
			name:  "when empty string",
			field: "",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when random text",
			field: "not a cron expression",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when too few fields",
			field: "* * *",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when too many fields (6 fields)",
			field: "* * * * * *",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when minute out of range",
			field: "60 * * * *",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when hour out of range",
			field: "0 25 * * *",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when day of month out of range",
			field: "0 0 32 * *",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when month out of range",
			field: "0 0 * 13 *",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when day of week out of range",
			field: "0 0 * * 8",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when invalid character",
			field: "0 0 * * abc",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when invalid expression shows hint in struct validation",
			field: "bad",
			contains: []string{
				"cron_schedule",
				"is not a valid cron expression",
			},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Contains(errMsg, "cron_schedule")
				s.Contains(errMsg, "is not a valid cron expression")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if len(tt.contains) > 0 {
				// Test through Struct() to verify hint formatting.
				type cronReq struct {
					Schedule string `validate:"required,cron_schedule"`
				}

				tt.validateFunc(validation.Struct(cronReq{Schedule: tt.field}))
			} else {
				tt.validateFunc(validation.Var(tt.field, "cron_schedule"))
			}
		})
	}
}

func (s *ValidationPublicTestSuite) TestGoDuration() {
	tests := []struct {
		name         string
		field        string
		contains     []string
		validateFunc func(string, bool)
	}{
		{
			name:  "when valid duration 30s passes",
			field: "30s",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when valid duration 5m passes",
			field: "5m",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when valid duration 1h passes",
			field: "1h",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when valid duration 720h passes",
			field: "720h",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when invalid duration 7d fails",
			field: "7d",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when invalid duration abc fails",
			field: "abc",
			validateFunc: func(_ string, ok bool) {
				s.False(ok)
			},
		},
		{
			name:  "when empty string passes with omitempty",
			field: "",
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when invalid duration shows hint in struct validation",
			field: "7d",
			contains: []string{
				"go_duration",
				"is not a valid Go duration",
			},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Contains(errMsg, "go_duration")
				s.Contains(errMsg, "is not a valid Go duration")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			if len(tt.contains) > 0 {
				// Test through Struct() to verify hint formatting.
				type durationReq struct {
					MaxAge string `validate:"required,go_duration"`
				}

				tt.validateFunc(validation.Struct(durationReq{MaxAge: tt.field}))
			} else if tt.field == "" {
				// Test empty string with omitempty — validation should pass.
				tt.validateFunc(validation.Var(tt.field, "omitempty,go_duration"))
			} else {
				tt.validateFunc(validation.Var(tt.field, "go_duration"))
			}
		})
	}
}

func (s *ValidationPublicTestSuite) TestAtLeastOneField() {
	type allPointers struct {
		Shell  *string
		Home   *string
		Groups *[]string
	}

	type withNonPointer struct {
		Name    string
		Shell   *string
		Enabled bool
	}

	type withSlice struct {
		Items []string
		Shell *string
	}

	type withMap struct {
		Labels map[string]string
		Shell  *string
	}

	type unexportedOnly struct {
		hidden *string //nolint:unused
	}

	str := "bash"
	groups := []string{"admin"}

	tests := []struct {
		name         string
		input        any
		validateFunc func(string, bool)
	}{
		{
			name:  "when one pointer field is non-nil",
			input: allPointers{Shell: &str},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when slice pointer field is non-nil",
			input: allPointers{Groups: &groups},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when all pointer fields are nil",
			input: allPointers{},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Equal("at least one field must be provided", errMsg)
			},
		},
		{
			name:  "when non-pointer field is non-zero",
			input: withNonPointer{Name: "test"},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when bool field is true",
			input: withNonPointer{Enabled: true},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when all fields are zero",
			input: withNonPointer{},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Equal("at least one field must be provided", errMsg)
			},
		},
		{
			name:  "when slice field is non-nil",
			input: withSlice{Items: []string{"a"}},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when slice field is nil",
			input: withSlice{},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Equal("at least one field must be provided", errMsg)
			},
		},
		{
			name:  "when map field is non-nil",
			input: withMap{Labels: map[string]string{"env": "dev"}},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when map field is nil",
			input: withMap{},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Equal("at least one field must be provided", errMsg)
			},
		},
		{
			name:  "when only unexported fields",
			input: unexportedOnly{},
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Equal("at least one field must be provided", errMsg)
			},
		},
		{
			name:  "when pointer to struct is passed",
			input: &allPointers{Shell: &str},
			validateFunc: func(_ string, ok bool) {
				s.True(ok)
			},
		},
		{
			name:  "when non-struct is passed",
			input: "not a struct",
			validateFunc: func(errMsg string, ok bool) {
				s.False(ok)
				s.Equal("expected struct", errMsg)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(validation.AtLeastOneField(tt.input))
		})
	}
}

func (s *ValidationPublicTestSuite) TestInstance() {
	tests := []struct {
		name         string
		validateFunc func(*validator.Validate)
	}{
		{
			name: "when returns shared validator instance",
			validateFunc: func(v *validator.Validate) {
				s.NotNil(v)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(validation.Instance())
		})
	}
}

func TestValidationPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidationPublicTestSuite))
}
