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

package client_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
	"github.com/osapi-io/osapi/pkg/sdk/client/gen"
)

type CheckErrorPublicTestSuite struct {
	suite.Suite
}

func (suite *CheckErrorPublicTestSuite) TestCheckError() {
	tests := []struct {
		name         string
		statusCode   int
		validateFunc func(error)
	}{
		{
			name:       "when status is 200",
			statusCode: 200,
			validateFunc: func(err error) {
				suite.NoError(err)
			},
		},
		{
			name:       "when status is 201",
			statusCode: 201,
			validateFunc: func(err error) {
				suite.NoError(err)
			},
		},
		{
			name:       "when status is 202",
			statusCode: 202,
			validateFunc: func(err error) {
				suite.NoError(err)
			},
		},
		{
			name:       "when status is 204",
			statusCode: 204,
			validateFunc: func(err error) {
				suite.NoError(err)
			},
		},
		{
			name:       "when status is 400",
			statusCode: 400,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(400, target.StatusCode)
			},
		},
		{
			name:       "when status is 401",
			statusCode: 401,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(401, target.StatusCode)
			},
		},
		{
			name:       "when status is 403",
			statusCode: 403,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(403, target.StatusCode)
			},
		},
		{
			name:       "when status is 404",
			statusCode: 404,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(404, target.StatusCode)
			},
		},
		{
			name:       "when status is 409",
			statusCode: 409,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.ConflictError
				suite.True(errors.As(err, &target))
				suite.Equal(409, target.StatusCode)
			},
		},
		{
			name:       "when status is 500",
			statusCode: 500,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(500, target.StatusCode)
			},
		},
		{
			name:       "when status is 503",
			statusCode: 503,
			validateFunc: func(err error) {
				suite.Error(err)
				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(503, target.StatusCode)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			err := client.ExportCheckError(tc.statusCode)
			tc.validateFunc(err)
		})
	}
}

func (suite *CheckErrorPublicTestSuite) TestCheckErrorMessages() {
	tests := []struct {
		name         string
		statusCode   int
		responses    []*gen.ErrorResponse
		validateFunc func(error)
	}{
		{
			name:       "when error response contains a message",
			statusCode: 400,
			responses: func() []*gen.ErrorResponse {
				msg := "field 'name' is required"
				return []*gen.ErrorResponse{{Error: &msg}}
			}(),
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "field 'name' is required")
			},
		},
		{
			name:       "when all responses are nil",
			statusCode: 400,
			responses:  []*gen.ErrorResponse{nil, nil},
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "unexpected status 400")
			},
		},
		{
			name:       "when no responses are provided",
			statusCode: 500,
			responses:  nil,
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "unexpected status 500")
			},
		},
		{
			name:       "when response has nil Error field",
			statusCode: 404,
			responses:  []*gen.ErrorResponse{{Error: nil}},
			validateFunc: func(err error) {
				suite.Error(err)
				suite.Contains(err.Error(), "unexpected status 404")
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			err := client.ExportCheckError(tc.statusCode, tc.responses...)
			tc.validateFunc(err)
		})
	}
}

func TestCheckErrorPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CheckErrorPublicTestSuite))
}
