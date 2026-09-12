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
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/pkg/sdk/client"
)

type ErrorsPublicTestSuite struct {
	suite.Suite
}

func (suite *ErrorsPublicTestSuite) TestErrorFormat() {
	tests := []struct {
		name         string
		err          error
		validateFunc func(error)
	}{
		{
			name: "when APIError formats correctly",
			err: &client.APIError{
				StatusCode: 500,
				Message:    "something went wrong",
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 500): something went wrong",
					err.Error(),
				)
			},
		},
		{
			name: "when AuthError formats correctly",
			err: &client.AuthError{
				APIError: client.APIError{
					StatusCode: 401,
					Message:    "unauthorized",
				},
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 401): unauthorized",
					err.Error(),
				)
			},
		},
		{
			name: "when NotFoundError formats correctly",
			err: &client.NotFoundError{
				APIError: client.APIError{
					StatusCode: 404,
					Message:    "resource not found",
				},
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 404): resource not found",
					err.Error(),
				)
			},
		},
		{
			name: "when ValidationError formats correctly",
			err: &client.ValidationError{
				APIError: client.APIError{
					StatusCode: 400,
					Message:    "invalid input",
				},
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 400): invalid input",
					err.Error(),
				)
			},
		},
		{
			name: "when ServerError formats correctly",
			err: &client.ServerError{
				APIError: client.APIError{
					StatusCode: 500,
					Message:    "internal server error",
				},
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 500): internal server error",
					err.Error(),
				)
			},
		},
		{
			name: "when ConflictError formats correctly",
			err: &client.ConflictError{
				APIError: client.APIError{
					StatusCode: 409,
					Message:    "already draining",
				},
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 409): already draining",
					err.Error(),
				)
			},
		},
		{
			name: "when UnexpectedStatusError formats correctly",
			err: &client.UnexpectedStatusError{
				APIError: client.APIError{
					StatusCode: 418,
					Message:    "unexpected status",
				},
			},
			validateFunc: func(err error) {
				suite.Equal(
					"api error (status 418): unexpected status",
					err.Error(),
				)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(tc.err)
		})
	}
}

func (suite *ErrorsPublicTestSuite) TestErrorsAsUnwrap() {
	tests := []struct {
		name         string
		err          error
		validateFunc func(error)
	}{
		{
			name: "when AuthError is unwrapped via errors.As",
			err: fmt.Errorf("wrapped: %w", &client.AuthError{
				APIError: client.APIError{
					StatusCode: 403,
					Message:    "forbidden",
				},
			}),
			validateFunc: func(err error) {
				var target *client.AuthError
				suite.True(errors.As(err, &target))
				suite.Equal(403, target.StatusCode)
				suite.Equal("forbidden", target.Message)
			},
		},
		{
			name: "when NotFoundError is unwrapped via errors.As",
			err: fmt.Errorf("wrapped: %w", &client.NotFoundError{
				APIError: client.APIError{
					StatusCode: 404,
					Message:    "not found",
				},
			}),
			validateFunc: func(err error) {
				var target *client.NotFoundError
				suite.True(errors.As(err, &target))
				suite.Equal(404, target.StatusCode)
				suite.Equal("not found", target.Message)
			},
		},
		{
			name: "when ValidationError is unwrapped via errors.As",
			err: fmt.Errorf("wrapped: %w", &client.ValidationError{
				APIError: client.APIError{
					StatusCode: 400,
					Message:    "bad request",
				},
			}),
			validateFunc: func(err error) {
				var target *client.ValidationError
				suite.True(errors.As(err, &target))
				suite.Equal(400, target.StatusCode)
				suite.Equal("bad request", target.Message)
			},
		},
		{
			name: "when ServerError is unwrapped via errors.As",
			err: fmt.Errorf("wrapped: %w", &client.ServerError{
				APIError: client.APIError{
					StatusCode: 500,
					Message:    "server failure",
				},
			}),
			validateFunc: func(err error) {
				var target *client.ServerError
				suite.True(errors.As(err, &target))
				suite.Equal(500, target.StatusCode)
				suite.Equal("server failure", target.Message)
			},
		},
		{
			name: "when ConflictError is unwrapped via errors.As",
			err: fmt.Errorf("wrapped: %w", &client.ConflictError{
				APIError: client.APIError{
					StatusCode: 409,
					Message:    "already draining",
				},
			}),
			validateFunc: func(err error) {
				var target *client.ConflictError
				suite.True(errors.As(err, &target))
				suite.Equal(409, target.StatusCode)
				suite.Equal("already draining", target.Message)
			},
		},
		{
			name: "when UnexpectedStatusError is unwrapped via errors.As",
			err: fmt.Errorf("wrapped: %w", &client.UnexpectedStatusError{
				APIError: client.APIError{
					StatusCode: 502,
					Message:    "bad gateway",
				},
			}),
			validateFunc: func(err error) {
				var target *client.UnexpectedStatusError
				suite.True(errors.As(err, &target))
				suite.Equal(502, target.StatusCode)
				suite.Equal("bad gateway", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(tc.err)
		})
	}
}

func (suite *ErrorsPublicTestSuite) TestErrorsAsAPIError() {
	tests := []struct {
		name         string
		err          error
		validateFunc func(error)
	}{
		{
			name: "when AuthError is matchable as APIError",
			err: fmt.Errorf("wrapped: %w", &client.AuthError{
				APIError: client.APIError{
					StatusCode: 401,
					Message:    "unauthorized",
				},
			}),
			validateFunc: func(err error) {
				var target *client.APIError
				suite.True(errors.As(err, &target))
				suite.Equal(401, target.StatusCode)
				suite.Equal("unauthorized", target.Message)
			},
		},
		{
			name: "when NotFoundError is matchable as APIError",
			err: fmt.Errorf("wrapped: %w", &client.NotFoundError{
				APIError: client.APIError{
					StatusCode: 404,
					Message:    "not found",
				},
			}),
			validateFunc: func(err error) {
				var target *client.APIError
				suite.True(errors.As(err, &target))
				suite.Equal(404, target.StatusCode)
				suite.Equal("not found", target.Message)
			},
		},
		{
			name: "when ValidationError is matchable as APIError",
			err: fmt.Errorf("wrapped: %w", &client.ValidationError{
				APIError: client.APIError{
					StatusCode: 400,
					Message:    "invalid",
				},
			}),
			validateFunc: func(err error) {
				var target *client.APIError
				suite.True(errors.As(err, &target))
				suite.Equal(400, target.StatusCode)
				suite.Equal("invalid", target.Message)
			},
		},
		{
			name: "when ServerError is matchable as APIError",
			err: fmt.Errorf("wrapped: %w", &client.ServerError{
				APIError: client.APIError{
					StatusCode: 500,
					Message:    "internal error",
				},
			}),
			validateFunc: func(err error) {
				var target *client.APIError
				suite.True(errors.As(err, &target))
				suite.Equal(500, target.StatusCode)
				suite.Equal("internal error", target.Message)
			},
		},
		{
			name: "when ConflictError is matchable as APIError",
			err: fmt.Errorf("wrapped: %w", &client.ConflictError{
				APIError: client.APIError{
					StatusCode: 409,
					Message:    "conflict",
				},
			}),
			validateFunc: func(err error) {
				var target *client.APIError
				suite.True(errors.As(err, &target))
				suite.Equal(409, target.StatusCode)
				suite.Equal("conflict", target.Message)
			},
		},
		{
			name: "when UnexpectedStatusError is matchable as APIError",
			err: fmt.Errorf("wrapped: %w", &client.UnexpectedStatusError{
				APIError: client.APIError{
					StatusCode: 418,
					Message:    "teapot",
				},
			}),
			validateFunc: func(err error) {
				var target *client.APIError
				suite.True(errors.As(err, &target))
				suite.Equal(418, target.StatusCode)
				suite.Equal("teapot", target.Message)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.validateFunc(tc.err)
		})
	}
}

func TestErrorsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ErrorsPublicTestSuite))
}
