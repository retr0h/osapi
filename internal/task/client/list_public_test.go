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

type ListUndeliveredMessagesPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	client *client.Client
}

func (suite *ListUndeliveredMessagesPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.client = clienttest.NewClientWithoutConnect()
}

func (suite *ListUndeliveredMessagesPublicTestSuite) SetupSubTest() {}

func (suite *ListUndeliveredMessagesPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *ListUndeliveredMessagesPublicTestSuite) TestListUndeliveredMessages() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockJetStream
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when ListUndeliveredMessages Ok",
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockJetStreamWithEphemeralConsumer(suite.ctrl)

				return mock
			},
			want: []client.MessageItem{
				{
					StreamSeq: 1,
					StoredAt:  mocks.FixedStoredAt,
					Data:      []byte("test message"),
				},
				{
					StreamSeq: 2,
					StoredAt:  mocks.FixedStoredAt,
					Data:      []byte("test message"),
				},
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
				return mocks.NewMockStreamWithConsumerError(suite.ctrl)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when Consumer.Info errors",
			setupMock: func() *mocks.MockJetStream {
				return mocks.NewMockJetStreamWithConsumerInfoError(suite.ctrl)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when strem.CreateOrUpdateConsumer errors",
			setupMock: func() *mocks.MockJetStream {
				return mocks.NewMockJetStreamWithCreateConsumerError(suite.ctrl)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when ephemeralConsumer.Fetch errors",
			setupMock: func() *mocks.MockJetStream {
				return mocks.NewMockJetStreamWithConsumerAndFetchError(suite.ctrl)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when msg.Metadata errors",
			setupMock: func() *mocks.MockJetStream {
				return mocks.NewMockJetStreamWithMetadataError(suite.ctrl)
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			suite.client.JS = mock

			got, err := suite.client.ListUndeliveredMessages(context.Background())

			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestListUndeliveredMessagesPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ListUndeliveredMessagesPublicTestSuite))
}