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
		name         string
		constant     string
		validateFunc func(string)
	}{
		{
			name:     "EnrollRequestSuffix has correct value",
			constant: pki.EnrollRequestSuffix,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "enroll.request", got)
			},
		},
		{
			name:     "EnrollResponsePrefix has correct value",
			constant: pki.EnrollResponsePrefix,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "enroll.response", got)
			},
		},
		{
			name:     "PKIRotateSuffix has correct value",
			constant: pki.PKIRotateSuffix,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "pki.rotate", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(tc.constant)
		})
	}
}

func (suite *TypesPublicTestSuite) TestEnrollmentStateValues() {
	tests := []struct {
		name         string
		state        pki.EnrollmentState
		validateFunc func(string)
	}{
		{
			name:  "StateUnregistered",
			state: pki.StateUnregistered,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "unregistered", got)
			},
		},
		{
			name:  "StatePending",
			state: pki.StatePending,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "pending", got)
			},
		},
		{
			name:  "StateAccepted",
			state: pki.StateAccepted,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "accepted", got)
			},
		},
		{
			name:  "StateRejected",
			state: pki.StateRejected,
			validateFunc: func(got string) {
				assert.Equal(suite.T(), "rejected", got)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(string(tc.state))
		})
	}
}

func TestTypesPublicTestSuite(
	t *testing.T,
) {
	t.Parallel()
	suite.Run(t, new(TypesPublicTestSuite))
}
