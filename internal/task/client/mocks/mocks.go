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
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/retr0h/osapi/internal/task/client"
)

const (
	// fixedTime pinned time used by test suite.
	fixedTime = "2024-09-10T12:00:00Z"
)

var fixedStoredAt, _ = GetFixedTime()

// NewPlainMockManager creates a Mock without defaults.
func NewPlainMockManager(ctrl *gomock.Controller) *MockManager {
	return NewMockManager(ctrl)
}

// NewDefaultMockManager creates a Mock with defaults.
func NewDefaultMockManager(ctrl *gomock.Controller) *MockManager {
	mock := NewMockManager(ctrl)
	data := []byte("test message")

	mock.EXPECT().
		CountStreamMessages(context.Background()).
		Return(2, nil).
		AnyTimes()

	mock.EXPECT().
		DeleteMessageBySeq(context.Background(), uint64(123)).
		Return(nil).
		AnyTimes()

	mock.EXPECT().
		GetMessageBySeq(context.Background(), uint64(123)).
		Return(&client.MessageItem{
			StreamSeq: 123,
			StoredAt:  fixedStoredAt,
			Data:      data,
		}, nil).
		AnyTimes()

	mock.EXPECT().
		ListUndeliveredMessages(context.Background()).
		Return([]client.MessageItem{
			{
				StreamSeq: 10,
				StoredAt:  fixedStoredAt,
				Data:      []byte("test message 1"),
			},
			{
				StreamSeq: 11,
				StoredAt:  fixedStoredAt,
				Data:      []byte("test message 2"),
			},
		}, nil).
		AnyTimes()

	mock.EXPECT().
		PublishToStream(context.Background(), data).
		Return(uint64(1), nil).
		AnyTimes()

	mock.EXPECT().
		PublishToStream(context.Background(), gomock.Any()).
		Return(uint64(1), nil).
		AnyTimes()

	mock.EXPECT().
		GetMessageIterator(context.Background()).
		Return(nil, nil).
		AnyTimes()

	return mock
}

// NewPlainJetStream creates a Mock without defaults.
func NewPlainJetStream(ctrl *gomock.Controller) *MockJetStream {
	return NewMockJetStream(ctrl)
}

// NewDefaultJetStream creates a Mock with defaults.
func NewDefaultJetStream(ctrl *gomock.Controller) *MockJetStream {
	mock := NewMockJetStream(ctrl)
	seq := uint64(123)
	data := []byte("test message")

	mockStream := NewMockStream(ctrl)
	mockConsumer := NewMockConsumer(ctrl)
	mockIterator := NewMockMessagesContext(ctrl)

	mock.EXPECT().
		Stream(context.Background(), "TASKS").
		Return(mockStream, nil).
		AnyTimes()

	mockStream.EXPECT().
		GetMsg(context.Background(), seq).
		Return(&jetstream.RawStreamMsg{
			Sequence: seq,
			Time:     fixedStoredAt,
			Data:     data,
		}, nil).
		AnyTimes()

	mockStream.EXPECT().
		DeleteMsg(context.Background(), seq).
		Return(nil).
		AnyTimes()

	mock.EXPECT().
		Publish(context.Background(), "tasks", data).
		Return(&jetstream.PubAck{
			Sequence: seq,
		}, nil).
		AnyTimes()

	mockStream.EXPECT().
		Consumer(context.Background(), "osapi").
		Return(mockConsumer, nil).
		AnyTimes()

	mockConsumer.EXPECT().
		Messages().
		Return(mockIterator, nil).
		AnyTimes()

	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(&jetstream.ConsumerInfo{
			NumAckPending: 1,
			NumPending:    1,
		}, nil).
		AnyTimes()

	return mock
}

// NewConditionalJetstream creates a Mock based on conditions .
func NewConditionalJetstream(ctrl *gomock.Controller) *MockJetStream {
	mock := NewPlainJetStream(ctrl)
	mockStream := NewMockStream(ctrl)
	data := []byte("test message")

	mock.EXPECT().
		Stream(context.Background(), "TASKS").
		Return(mockStream, nil).
		AnyTimes()

	// stream.GetMsg erors -- 2
	mockStream.EXPECT().
		GetMsg(context.Background(), uint64(2)).
		Return(nil, assert.AnError).
		AnyTimes()

	// stream.GetMsg not found -- 3
	mockStream.EXPECT().
		GetMsg(context.Background(), uint64(3)).
		Return(nil, &jetstream.APIError{
			Code:        404,
			Description: "message not found",
		}).
		AnyTimes()

	// stream.DeleteMsg errors -- 4
	mockStream.EXPECT().
		GetMsg(context.Background(), uint64(4)).
		Return(&jetstream.RawStreamMsg{
			Sequence: 4,
			Time:     fixedStoredAt,
			Data:     data,
		}, nil).
		AnyTimes()
	mockStream.EXPECT().
		DeleteMsg(context.Background(), uint64(4)).
		Return(assert.AnError).
		AnyTimes()

	return mock
}

// NewStreamErrorJetstream creates a Mock wehre Stream errors.
func NewStreamErrorJetstream(ctrl *gomock.Controller) *MockJetStream {
	mock := NewPlainJetStream(ctrl)

	mock.EXPECT().
		Stream(context.Background(), "TASKS").
		Return(nil, assert.AnError)

	return mock
}

// GetFixedTime parses and returns the fixedTime constant as a time.Time value.
// If the parsing fails, it returns an error.
func GetFixedTime() (time.Time, error) {
	parsedTime, err := time.Parse(time.RFC3339Nano, fixedTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid fixed time format: %w", err)
	}
	return parsedTime, nil
}

// GetConsumerInfo returns an empty jetstream.ConsumerInfo.
func GetConsumerInfo() *jetstream.ConsumerInfo {
	return &jetstream.ConsumerInfo{}
}
