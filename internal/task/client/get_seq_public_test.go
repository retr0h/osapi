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

	customerrors "github.com/retr0h/osapi/internal/errors"
	"github.com/retr0h/osapi/internal/task/client"
	"github.com/retr0h/osapi/internal/task/client/mocks"
	clienttest "github.com/retr0h/osapi/internal/task/client/testing"
)

type GetMessageBySeqPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller

	client *client.Client
}

func (suite *GetMessageBySeqPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())

	suite.client = clienttest.NewClientWithoutConnect()
}

func (suite *GetMessageBySeqPublicTestSuite) SetupSubTest() {}

func (suite *GetMessageBySeqPublicTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func (suite *GetMessageBySeqPublicTestSuite) TestGetMessageBySeq() {
	tests := []struct {
		name        string
		seq         uint64
		setupMock   func() *mocks.MockJetStream
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when GetMessageBySeq Ok",
			seq:  123,
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockStreamWithGet(suite.ctrl)

				return mock
			},
			want: &client.MessageItem{
				StreamSeq: 123,
				StoredAt:  mocks.FixedStoredAt,
				Data:      []byte("test message"),
			},
			wantErr: false,
		},
		{
			name: "when client.JS.Stream errors",
			seq:  123,
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewJetStreamWithStreamError(suite.ctrl)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when stream.GetMsg errors",
			seq:  123,
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockStreamWithGetMsgError(suite.ctrl)

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
		{
			name: "when GetMessageBySeq fails due to non-existent message",
			seq:  123,
			setupMock: func() *mocks.MockJetStream {
				mock := mocks.NewMockStreamWithMessageNotFound(suite.ctrl)

				return mock
			},
			wantErr:     true,
			wantErrType: customerrors.NewNotFoundError("no item found with ID 123"),
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			suite.client.JS = mock

			got, err := suite.client.GetMessageBySeq(context.Background(), tc.seq)
			if !tc.wantErr {
				assert.NoError(suite.T(), err)
				assert.Equal(suite.T(), tc.want, got)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}

			suite.ctrl.Finish()
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestGetMessageBySeqPublicTestSuite(t *testing.T) {
	suite.Run(t, new(GetMessageBySeqPublicTestSuite))
}
