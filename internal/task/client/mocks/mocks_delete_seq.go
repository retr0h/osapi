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

package mocks

import (
	"context"

	"github.com/golang/mock/gomock"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
)

// NewMockStreamWithDeleteMessageSetup sets up a stream with message retrieval and deletion.
func NewMockStreamWithDeleteMessageSetup(
	ctrl *gomock.Controller,
	getMsgErr error,
) (*MockJetStream, *MockStream) {
	mockJetStream, mockStream, _ := NewMockStreamWithConsumer(ctrl)

	mockStream.EXPECT().
		GetMsg(context.Background(), msgSeq).
		Return(&jetstream.RawStreamMsg{
			Sequence: msgSeq,
			Time:     FixedStoredAt,
			Data:     msgData,
		}, getMsgErr).
		AnyTimes()

	return mockJetStream, mockStream
}

// NewMockStreamWithDeleteError creates a mock where message deletion fails.
func NewMockStreamWithDeleteError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, mockStream := NewMockStreamWithDeleteMessageSetup(ctrl, nil)

	mockJetStream.EXPECT().
		Stream(context.Background(), StreamName).
		Return(mockStream, nil).
		Times(1)

	mockStream.EXPECT().
		DeleteMsg(context.Background(), msgSeq).
		Return(assert.AnError).
		Times(1)

	return mockJetStream
}

// NewMockStreamWithDeleteMessageNotFound creates a mock where message retrieval fails with 404.
func NewMockStreamWithDeleteMessageNotFound(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, _ := NewMockStreamWithDeleteMessageSetup(ctrl, &jetstream.APIError{
		Code:        404,
		Description: "message not found",
	})

	return mockJetStream
}

// NewMockStreamWithDelete creates a mock where deletion succeeds.
func NewMockStreamWithDelete(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, mockStream := NewMockStreamWithDeleteMessageSetup(ctrl, nil)

	mockJetStream.EXPECT().
		Stream(context.Background(), StreamName).
		Return(mockStream, nil).
		Times(1)

	mockStream.EXPECT().
		DeleteMsg(context.Background(), msgSeq).
		Return(nil).
		Times(1)

	return mockJetStream
}

// NewMockStreamWithDeleteGetMsgError creates a mock where GetMessageBySeq returns a generic error.
func NewMockStreamWithDeleteGetMsgError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, _ := NewMockStreamWithDeleteMessageSetup(ctrl, assert.AnError)

	return mockJetStream
}

// NewMockStreamWithDeleteStreamError creates a mock where Stream returns a generic error.
func NewMockStreamWithDeleteStreamError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, _ := NewMockStreamWithDeleteMessageSetup(ctrl, nil)

	mockJetStream.EXPECT().
		Stream(context.Background(), StreamName).
		Return(nil, assert.AnError).
		AnyTimes()

	return mockJetStream
}
