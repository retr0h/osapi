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
		name     string
		input    any
		wantOK   bool
		contains []string
	}{
		{
			name: "when valid struct",
			input: testStruct{
				Name:  "test",
				Email: "test@example.com",
			},
			wantOK: true,
		},
		{
			name: "when missing required field",
			input: testStruct{
				Email: "test@example.com",
			},
			wantOK:   false,
			contains: []string{"Name", "required"},
		},
		{
			name: "when invalid email",
			input: testStruct{
				Name:  "test",
				Email: "not-an-email",
			},
			wantOK:   false,
			contains: []string{"Email", "email"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
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

func (s *ValidationPublicTestSuite) TestVar() {
	tests := []struct {
		name     string
		field    any
		tag      string
		wantOK   bool
		contains []string
	}{
		{
			name:   "when valid field",
			field:  "hello",
			tag:    "required",
			wantOK: true,
		},
		{
			name:     "when empty required field",
			field:    "",
			tag:      "required",
			wantOK:   false,
			contains: []string{"required"},
		},
		{
			name:     "when invalid email",
			field:    "not-an-email",
			tag:      "email",
			wantOK:   false,
			contains: []string{"email"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			errMsg, ok := validation.Var(tt.field, tt.tag)
			s.Equal(tt.wantOK, ok)

			if !ok {
				for _, c := range tt.contains {
					s.Contains(errMsg, c)
				}
			}
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
		name     string
		field    string
		wantOK   bool
		contains []string
	}{
		// Valid expressions
		{
			name:   "when every minute",
			field:  "* * * * *",
			wantOK: true,
		},
		{
			name:   "when daily at 2am",
			field:  "0 2 * * *",
			wantOK: true,
		},
		{
			name:   "when every 5 minutes",
			field:  "*/5 * * * *",
			wantOK: true,
		},
		{
			name:   "when weekdays at 9am",
			field:  "0 9 * * 1-5",
			wantOK: true,
		},
		{
			name:   "when first of month at midnight",
			field:  "0 0 1 * *",
			wantOK: true,
		},
		{
			name:   "when multiple hours",
			field:  "0 2,14 * * *",
			wantOK: true,
		},
		{
			name:   "when range with step",
			field:  "0-30/5 * * * *",
			wantOK: true,
		},
		{
			name:   "when month and day names",
			field:  "0 0 * jan-mar mon",
			wantOK: true,
		},
		// Invalid expressions
		{
			name:   "when empty string",
			field:  "",
			wantOK: false,
		},
		{
			name:   "when random text",
			field:  "not a cron expression",
			wantOK: false,
		},
		{
			name:   "when too few fields",
			field:  "* * *",
			wantOK: false,
		},
		{
			name:   "when too many fields (6 fields)",
			field:  "* * * * * *",
			wantOK: false,
		},
		{
			name:   "when minute out of range",
			field:  "60 * * * *",
			wantOK: false,
		},
		{
			name:   "when hour out of range",
			field:  "0 25 * * *",
			wantOK: false,
		},
		{
			name:   "when day of month out of range",
			field:  "0 0 32 * *",
			wantOK: false,
		},
		{
			name:   "when month out of range",
			field:  "0 0 * 13 *",
			wantOK: false,
		},
		{
			name:   "when day of week out of range",
			field:  "0 0 * * 8",
			wantOK: false,
		},
		{
			name:   "when invalid character",
			field:  "0 0 * * abc",
			wantOK: false,
		},
		{
			name:   "when invalid expression shows hint in struct validation",
			field:  "bad",
			wantOK: false,
			contains: []string{
				"cron_schedule",
				"is not a valid cron expression",
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
				errMsg, ok := validation.Struct(cronReq{Schedule: tt.field})
				s.Equal(tt.wantOK, ok)
				for _, c := range tt.contains {
					s.Contains(errMsg, c)
				}
			} else {
				_, ok := validation.Var(tt.field, "cron_schedule")
				s.Equal(tt.wantOK, ok)
			}
		})
	}
}

func (s *ValidationPublicTestSuite) TestGoDuration() {
	tests := []struct {
		name     string
		field    string
		wantOK   bool
		contains []string
	}{
		{
			name:   "when valid duration 30s passes",
			field:  "30s",
			wantOK: true,
		},
		{
			name:   "when valid duration 5m passes",
			field:  "5m",
			wantOK: true,
		},
		{
			name:   "when valid duration 1h passes",
			field:  "1h",
			wantOK: true,
		},
		{
			name:   "when valid duration 720h passes",
			field:  "720h",
			wantOK: true,
		},
		{
			name:   "when invalid duration 7d fails",
			field:  "7d",
			wantOK: false,
		},
		{
			name:   "when invalid duration abc fails",
			field:  "abc",
			wantOK: false,
		},
		{
			name:   "when empty string passes with omitempty",
			field:  "",
			wantOK: true,
		},
		{
			name:  "when invalid duration shows hint in struct validation",
			field: "7d",
			contains: []string{
				"go_duration",
				"is not a valid Go duration",
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
				errMsg, ok := validation.Struct(durationReq{MaxAge: tt.field})
				s.False(ok)
				for _, c := range tt.contains {
					s.Contains(errMsg, c)
				}
			} else if tt.field == "" {
				// Test empty string with omitempty — validation should pass.
				_, ok := validation.Var(tt.field, "omitempty,go_duration")
				s.Equal(tt.wantOK, ok)
			} else {
				_, ok := validation.Var(tt.field, "go_duration")
				s.Equal(tt.wantOK, ok)
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
		name       string
		input      any
		wantOK     bool
		wantErrMsg string
	}{
		{
			name:   "when one pointer field is non-nil",
			input:  allPointers{Shell: &str},
			wantOK: true,
		},
		{
			name:   "when slice pointer field is non-nil",
			input:  allPointers{Groups: &groups},
			wantOK: true,
		},
		{
			name:       "when all pointer fields are nil",
			input:      allPointers{},
			wantOK:     false,
			wantErrMsg: "at least one field must be provided",
		},
		{
			name:   "when non-pointer field is non-zero",
			input:  withNonPointer{Name: "test"},
			wantOK: true,
		},
		{
			name:   "when bool field is true",
			input:  withNonPointer{Enabled: true},
			wantOK: true,
		},
		{
			name:       "when all fields are zero",
			input:      withNonPointer{},
			wantOK:     false,
			wantErrMsg: "at least one field must be provided",
		},
		{
			name:   "when slice field is non-nil",
			input:  withSlice{Items: []string{"a"}},
			wantOK: true,
		},
		{
			name:       "when slice field is nil",
			input:      withSlice{},
			wantOK:     false,
			wantErrMsg: "at least one field must be provided",
		},
		{
			name:   "when map field is non-nil",
			input:  withMap{Labels: map[string]string{"env": "dev"}},
			wantOK: true,
		},
		{
			name:       "when map field is nil",
			input:      withMap{},
			wantOK:     false,
			wantErrMsg: "at least one field must be provided",
		},
		{
			name:       "when only unexported fields",
			input:      unexportedOnly{},
			wantOK:     false,
			wantErrMsg: "at least one field must be provided",
		},
		{
			name:   "when pointer to struct is passed",
			input:  &allPointers{Shell: &str},
			wantOK: true,
		},
		{
			name:       "when non-struct is passed",
			input:      "not a struct",
			wantOK:     false,
			wantErrMsg: "expected struct",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			errMsg, ok := validation.AtLeastOneField(tt.input)
			s.Equal(tt.wantOK, ok)
			if !ok {
				s.Equal(tt.wantErrMsg, errMsg)
			}
		})
	}
}

func (s *ValidationPublicTestSuite) TestInstance() {
	tests := []struct {
		name string
	}{
		{
			name: "when returns shared validator instance",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			v := validation.Instance()
			s.NotNil(v)
		})
	}
}

func TestValidationPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ValidationPublicTestSuite))
}
