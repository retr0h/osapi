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

package container_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	container "github.com/osapi-io/osapi/internal/controller/api/node/docker"
	"github.com/osapi-io/osapi/internal/job"
)

type ConvertPublicTestSuite struct {
	suite.Suite
}

func (s *ConvertPublicTestSuite) TestPtrToSlice() {
	tests := []struct {
		name         string
		input        *[]string
		validateFunc func(result []string)
	}{
		{
			name:  "when nil pointer returns nil",
			input: nil,
			validateFunc: func(result []string) {
				s.Nil(result)
			},
		},
		{
			name:  "when non-nil pointer returns slice",
			input: &[]string{"a", "b", "c"},
			validateFunc: func(result []string) {
				s.Equal([]string{"a", "b", "c"}, result)
			},
		},
		{
			name:  "when pointer to empty slice returns empty slice",
			input: &[]string{},
			validateFunc: func(result []string) {
				s.NotNil(result)
				s.Empty(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportPtrToSlice(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestEnvSliceToMap() {
	tests := []struct {
		name         string
		input        *[]string
		validateFunc func(result map[string]string)
	}{
		{
			name:  "when nil returns nil",
			input: nil,
			validateFunc: func(result map[string]string) {
				s.Nil(result)
			},
		},
		{
			name:  "when valid KEY=VALUE pairs returns map",
			input: &[]string{"FOO=bar", "BAZ=qux"},
			validateFunc: func(result map[string]string) {
				s.Require().Len(result, 2)
				s.Equal("bar", result["FOO"])
				s.Equal("qux", result["BAZ"])
			},
		},
		{
			name:  "when value contains equals sign preserves it",
			input: &[]string{"KEY=val=ue"},
			validateFunc: func(result map[string]string) {
				s.Require().Len(result, 1)
				s.Equal("val=ue", result["KEY"])
			},
		},
		{
			name:  "when entry has no equals sign skips it",
			input: &[]string{"INVALID", "GOOD=value"},
			validateFunc: func(result map[string]string) {
				s.Require().Len(result, 1)
				s.Equal("value", result["GOOD"])
			},
		},
		{
			name:  "when empty slice returns empty map",
			input: &[]string{},
			validateFunc: func(result map[string]string) {
				s.NotNil(result)
				s.Empty(result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportEnvSliceToMap(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestParsePortMappings() {
	tests := []struct {
		name         string
		input        *[]string
		validateFunc func(result []job.PortMapping)
	}{
		{
			name:  "when nil returns nil",
			input: nil,
			validateFunc: func(result []job.PortMapping) {
				s.Nil(result)
			},
		},
		{
			name:  "when valid host:container pairs returns mappings",
			input: &[]string{"8080:80", "9090:90"},
			validateFunc: func(result []job.PortMapping) {
				s.Require().Len(result, 2)
				s.Equal(8080, result[0].Host)
				s.Equal(80, result[0].Container)
				s.Equal(9090, result[1].Host)
				s.Equal(90, result[1].Container)
			},
		},
		{
			name:  "when invalid port number skips entry",
			input: &[]string{"abc:80", "8080:xyz"},
			validateFunc: func(result []job.PortMapping) {
				s.Nil(result)
			},
		},
		{
			name:  "when entry has no colon skips it",
			input: &[]string{"8080"},
			validateFunc: func(result []job.PortMapping) {
				s.Nil(result)
			},
		},
		{
			name:  "when mixed valid and invalid returns only valid",
			input: &[]string{"8080:80", "bad:port", "9090:90"},
			validateFunc: func(result []job.PortMapping) {
				s.Require().Len(result, 2)
				s.Equal(8080, result[0].Host)
				s.Equal(80, result[0].Container)
				s.Equal(9090, result[1].Host)
				s.Equal(90, result[1].Container)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportParsePortMappings(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestParseVolumeMappings() {
	tests := []struct {
		name         string
		input        *[]string
		validateFunc func(result []job.VolumeMapping)
	}{
		{
			name:  "when nil returns nil",
			input: nil,
			validateFunc: func(result []job.VolumeMapping) {
				s.Nil(result)
			},
		},
		{
			name:  "when valid host:container pairs returns mappings",
			input: &[]string{"/host/path:/container/path", "/data:/app/data"},
			validateFunc: func(result []job.VolumeMapping) {
				s.Require().Len(result, 2)
				s.Equal("/host/path", result[0].Host)
				s.Equal("/container/path", result[0].Container)
				s.Equal("/data", result[1].Host)
				s.Equal("/app/data", result[1].Container)
			},
		},
		{
			name:  "when entry has no colon skips it",
			input: &[]string{"/no-colon-path"},
			validateFunc: func(result []job.VolumeMapping) {
				s.Nil(result)
			},
		},
		{
			name:  "when mixed valid and invalid returns only valid",
			input: &[]string{"/host:/container", "nocolon", "/data:/app"},
			validateFunc: func(result []job.VolumeMapping) {
				s.Require().Len(result, 2)
				s.Equal("/host", result[0].Host)
				s.Equal("/container", result[0].Container)
				s.Equal("/data", result[1].Host)
				s.Equal("/app", result[1].Container)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportParseVolumeMappings(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestPortMappingsToStrings() {
	tests := []struct {
		name  string
		input []struct {
			Host      int `json:"host"`
			Container int `json:"container"`
		}
		validateFunc func(result []string)
	}{
		{
			name:  "when empty slice returns nil",
			input: nil,
			validateFunc: func(result []string) {
				s.Nil(result)
			},
		},
		{
			name: "when non-empty slice returns formatted strings",
			input: []struct {
				Host      int `json:"host"`
				Container int `json:"container"`
			}{
				{Host: 8080, Container: 80},
				{Host: 9090, Container: 90},
			},
			validateFunc: func(result []string) {
				s.Require().Len(result, 2)
				s.Equal("8080:80", result[0])
				s.Equal("9090:90", result[1])
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportPortMappingsToStrings(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestVolumeMappingsToStrings() {
	tests := []struct {
		name  string
		input []struct {
			Host      string `json:"host"`
			Container string `json:"container"`
		}
		validateFunc func(result []string)
	}{
		{
			name:  "when empty slice returns nil",
			input: nil,
			validateFunc: func(result []string) {
				s.Nil(result)
			},
		},
		{
			name: "when non-empty slice returns formatted strings",
			input: []struct {
				Host      string `json:"host"`
				Container string `json:"container"`
			}{
				{Host: "/host/path", Container: "/container/path"},
				{Host: "/data", Container: "/app/data"},
			},
			validateFunc: func(result []string) {
				s.Require().Len(result, 2)
				s.Equal("/host/path:/container/path", result[0])
				s.Equal("/data:/app/data", result[1])
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportVolumeMappingsToStrings(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestNilIfEmptyStrSlice() {
	tests := []struct {
		name         string
		input        []string
		validateFunc func(result *[]string)
	}{
		{
			name:  "when nil slice returns nil",
			input: nil,
			validateFunc: func(result *[]string) {
				s.Nil(result)
			},
		},
		{
			name:  "when empty slice returns nil",
			input: []string{},
			validateFunc: func(result *[]string) {
				s.Nil(result)
			},
		},
		{
			name:  "when non-empty slice returns pointer",
			input: []string{"a", "b"},
			validateFunc: func(result *[]string) {
				s.Require().NotNil(result)
				s.Equal([]string{"a", "b"}, *result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportNilIfEmptyStrSlice(tt.input)
			tt.validateFunc(result)
		})
	}
}

func (s *ConvertPublicTestSuite) TestInt64PtrOrNil() {
	tests := []struct {
		name         string
		input        int64
		validateFunc func(result *int64)
	}{
		{
			name:  "when zero returns nil",
			input: 0,
			validateFunc: func(result *int64) {
				s.Nil(result)
			},
		},
		{
			name:  "when positive returns pointer",
			input: 42,
			validateFunc: func(result *int64) {
				s.Require().NotNil(result)
				s.Equal(int64(42), *result)
			},
		},
		{
			name:  "when negative returns pointer",
			input: -1,
			validateFunc: func(result *int64) {
				s.Require().NotNil(result)
				s.Equal(int64(-1), *result)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := container.ExportInt64PtrOrNil(tt.input)
			tt.validateFunc(result)
		})
	}
}

func TestConvertPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ConvertPublicTestSuite))
}
