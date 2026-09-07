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

package pki_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/agent/pki"
)

type TypesPublicTestSuite struct {
	suite.Suite
}

func (suite *TypesPublicTestSuite) TestSubjectConstants() {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "EnrollRequestSuffix has correct value",
			constant: pki.EnrollRequestSuffix,
			expected: "enroll.request",
		},
		{
			name:     "EnrollResponsePrefix has correct value",
			constant: pki.EnrollResponsePrefix,
			expected: "enroll.response",
		},
		{
			name:     "PKIRotateSuffix has correct value",
			constant: pki.PKIRotateSuffix,
			expected: "pki.rotate",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			assert.Equal(suite.T(), tc.expected, tc.constant)
		})
	}
}

func (suite *TypesPublicTestSuite) TestEnrollmentStateValues() {
	tests := []struct {
		name     string
		state    pki.EnrollmentState
		expected string
	}{
		{
			name:     "StateUnregistered",
			state:    pki.StateUnregistered,
			expected: "unregistered",
		},
		{
			name:     "StatePending",
			state:    pki.StatePending,
			expected: "pending",
		},
		{
			name:     "StateAccepted",
			state:    pki.StateAccepted,
			expected: "accepted",
		},
		{
			name:     "StateRejected",
			state:    pki.StateRejected,
			expected: "rejected",
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			assert.Equal(suite.T(), tc.expected, string(tc.state))
		})
	}
}

func TestTypesPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(TypesPublicTestSuite))
}
