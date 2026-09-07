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

package facts_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/facts"
)

type KeysPublicTestSuite struct {
	suite.Suite
}

func (s *KeysPublicTestSuite) TestBuiltInKeys() {
	tests := []struct {
		name         string
		validateFunc func([]string)
	}{
		{
			name: "when called returns all six built-in keys",
			validateFunc: func(keys []string) {
				s.Len(keys, 6)
				s.Contains(keys, facts.KeyInterfacePrimary)
				s.Contains(keys, facts.KeyHostname)
				s.Contains(keys, facts.KeyArch)
				s.Contains(keys, facts.KeyKernel)
				s.Contains(keys, facts.KeyFQDN)
				s.Contains(keys, facts.KeyContainerized)
			},
		},
		{
			name: "when called twice returns independent slices",
			validateFunc: func(_ []string) {
				a := facts.BuiltInKeys()
				b := facts.BuiltInKeys()
				a[0] = "mutated"
				s.NotEqual(a[0], b[0])
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			keys := facts.BuiltInKeys()
			tt.validateFunc(keys)
		})
	}
}

func (s *KeysPublicTestSuite) TestIsKnownKey() {
	tests := []struct {
		name         string
		key          string
		validateFunc func(bool)
	}{
		{
			name: "when interface.primary",
			key:  facts.KeyInterfacePrimary,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when hostname",
			key:  facts.KeyHostname,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when arch",
			key:  facts.KeyArch,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when kernel",
			key:  facts.KeyKernel,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when fqdn",
			key:  facts.KeyFQDN,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when containerized",
			key:  facts.KeyContainerized,
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when valid custom key",
			key:  "custom.gateway",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when valid custom key with dots",
			key:  "custom.network.gateway",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when custom prefix only",
			key:  "custom.",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when empty string",
			key:  "",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when unknown key",
			key:  "unknown",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when partial match",
			key:  "host",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when not fact prefix",
			key:  "@notfact.x",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(facts.IsKnownKey(tt.key))
		})
	}
}

func (s *KeysPublicTestSuite) TestIsCustomKey() {
	tests := []struct {
		name         string
		key          string
		validateFunc func(bool)
	}{
		{
			name: "when valid custom key",
			key:  "custom.gateway",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when valid custom key with nested dots",
			key:  "custom.network.primary.gateway",
			validateFunc: func(got bool) {
				s.Equal(true, got)
			},
		},
		{
			name: "when custom prefix only",
			key:  "custom.",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when empty string",
			key:  "",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when built-in key",
			key:  "hostname",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
		{
			name: "when partial custom prefix",
			key:  "custo",
			validateFunc: func(got bool) {
				s.Equal(false, got)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(facts.IsCustomKey(tt.key))
		})
	}
}

func TestKeysPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(KeysPublicTestSuite))
}
