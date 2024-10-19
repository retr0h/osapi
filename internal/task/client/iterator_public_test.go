// Copyright (c) 2024 John Dewey

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
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/task/client"
	"github.com/retr0h/osapi/internal/task/client/mocks"
	clienttest "github.com/retr0h/osapi/internal/task/client/testing"
)

type GetMessageIteratorPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	client *client.Client
}

func (suite *GetMessageIteratorPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.client = clienttest.NewClientWithoutConnect()
}

func (suite *GetMessageIteratorPublicTestSuite) SetupSubTest() {}

func (suite *GetMessageIteratorPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *GetMessageIteratorPublicTestSuite) TestGetMessageIterator() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockJetStream
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetMessageIterator Ok",
			want: mocks.NewMockMessagesContext(suite.ctrl),
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockStreamWithMessageIterator(suite.ctrl)

				return mock
			},
			wantErr: false,
		},
		{
			name: "when client.JS.Stream errors",
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewJetStreamWithStreamError(suite.ctrl)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when stream.Consumer errors",
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockStreamWithConsumerError(suite.ctrl)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},

		{
			name: "when cons.Messages errors",
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockStreamWithMessageIteratorError(suite.ctrl)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			suite.client.JS = mock

			got, err := suite.client.GetMessageIterator(context.Background())

			if !tc.wantErr {
				suite.NoError(err)
				suite.Equal(tc.want, got)
			} else {
				suite.Error(err)
				suite.Contains(err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestGetMessageIteratorPublicTestSuite(t *testing.T) {
	suite.Run(t, new(GetMessageIteratorPublicTestSuite))
}
