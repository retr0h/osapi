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

package provider_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/provider"
)

type FactsPublicTestSuite struct {
	suite.Suite
}

func (suite *FactsPublicTestSuite) SetupTest() {}

func (suite *FactsPublicTestSuite) TearDownTest() {}

func (suite *FactsPublicTestSuite) TestSetFactsFunc() {
	tests := []struct {
		name         string
		factsFn      provider.FactsFunc
		validateFunc func(map[string]any)
	}{
		{
			name: "when sets the facts function",
			factsFn: func() map[string]any {
				return map[string]any{"cpu_count": 4}
			},
			validateFunc: func(got map[string]any) {
				suite.Equal(map[string]any{"cpu_count": 4}, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fa := &provider.FactsAware{}
			fa.SetFactsFunc(tc.factsFn)

			got := fa.Facts()
			tc.validateFunc(got)
		})
	}
}

func (suite *FactsPublicTestSuite) TestFacts() {
	tests := []struct {
		name         string
		factsFn      provider.FactsFunc
		setFacts     bool
		validateFunc func(map[string]any)
	}{
		{
			name:     "when factsFn is nil returns nil",
			setFacts: false,
			validateFunc: func(got map[string]any) {
				suite.Equal(map[string]any(nil), got)
			},
		},
		{
			name: "when factsFn is set returns facts",
			factsFn: func() map[string]any {
				return map[string]any{"cpu_count": 4}
			},
			setFacts: true,
			validateFunc: func(got map[string]any) {
				suite.Equal(map[string]any{"cpu_count": 4}, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			fa := &provider.FactsAware{}
			if tc.setFacts {
				fa.SetFactsFunc(tc.factsFn)
			}

			got := fa.Facts()
			tc.validateFunc(got)
		})
	}
}

// testFactsProvider is a helper type that embeds FactsAware to test WireProviderFacts.
type testFactsProvider struct {
	provider.FactsAware
}

func (suite *FactsPublicTestSuite) TestWireProviderFacts() {
	tests := []struct {
		name         string
		providers    []any
		checkIdx     int
		validateFunc func(map[string]any)
	}{
		{
			name: "when wires facts to implementing providers",
			providers: []any{
				&testFactsProvider{},
			},
			checkIdx: 0,
			validateFunc: func(got map[string]any) {
				suite.Equal(map[string]any{"os": "linux"}, got)
			},
		},
		{
			name: "when skips non-implementing providers",
			providers: []any{
				"not-a-provider",
				&testFactsProvider{},
			},
			checkIdx: 1,
			validateFunc: func(got map[string]any) {
				suite.Equal(map[string]any{"os": "linux"}, got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			factsFn := func() map[string]any {
				return map[string]any{"os": "linux"}
			}

			suite.NotPanics(func() {
				provider.WireProviderFacts(factsFn, tc.providers...)
			})

			p, ok := tc.providers[tc.checkIdx].(*testFactsProvider)
			suite.Require().True(ok)

			got := p.Facts()
			tc.validateFunc(got)
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestFactsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FactsPublicTestSuite))
}
