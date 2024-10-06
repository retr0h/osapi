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
	// "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/retr0h/osapi/internal/task/client/mocks"
)

type DeleteMessageBySeqPublicTestSuite struct {
	suite.Suite
	ctrl *gomock.Controller
}

func (suite *DeleteMessageBySeqPublicTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
}

func (suite *DeleteMessageBySeqPublicTestSuite) SetupSubTest() {}

func (suite *DeleteMessageBySeqPublicTestSuite) TearDownTest() {}

func (suite *DeleteMessageBySeqPublicTestSuite) TestDeleteMessageBySeq() {
	tests := []struct {
		name        string
		setupMock   func() *mocks.MockManager
		want        interface{}
		wantErr     bool
		wantErrType error
	}{
		{
			name: "when DeleteMessageBySeq Ok",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewDefaultMockManager(suite.ctrl)

				return mock
			},
			wantErr: false,
		},
		{
			name: "when DeleteMessageBySeq errors",
			setupMock: func() *mocks.MockManager {
				mock := mocks.NewPlainMockManager(suite.ctrl)
				mock.EXPECT().
					DeleteMessageBySeq(context.Background(), "TASKS", uint64(123)).
					Return(assert.AnError).
					AnyTimes()

				return mock
			},
			wantErr:     true,
			wantErrType: assert.AnError,
		},
	}
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			mock := tc.setupMock()
			err := mock.DeleteMessageBySeq(context.Background(), "TASKS", 123)
			if !tc.wantErr {
				assert.NoError(suite.T(), err)
			} else {
				assert.Error(suite.T(), err)
				assert.Contains(suite.T(), err.Error(), tc.wantErrType.Error())
			}
		})
	}
}

// In order for `go test` to run this suite, we need to create
// a normal test function and pass our suite to suite.Run.
func TestDeleteMessageBySeqPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteMessageBySeqPublicTestSuite))
}
