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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/retr0h/osapi/internal/task/client"
)

const (
	fixedTime = "2024-09-10T12:00:00Z"

	// ConsumerName is the durable name of the NATS JetStream consumer.
	ConsumerName = "osapi"
	// StreamName represents the NATS JetStream stream to consume from.
	StreamName = "TASKS"
	// SubjectName represents the NATS JetStream subject within the "TASKS" stream.
	SubjectName = "tasks"
)

var (
	// FixedStoredAt used throughout test suite for fixed time assertions.
	FixedStoredAt = GetFixedTime()
	msgData       = []byte("test message")
	msgSeq        = uint64(123)
)

// NewPlainMockManager creates a Mock without defaults.
func NewPlainMockManager(ctrl *gomock.Controller) *MockManager {
	return NewMockManager(ctrl)
}

// NewDefaultMockManager creates a Mock with defaults.
func NewDefaultMockManager(ctrl *gomock.Controller) *MockManager {
	mock := NewMockManager(ctrl)

	mock.EXPECT().
		CountStreamMessages(context.Background()).
		Return(2, nil).
		AnyTimes()

	mock.EXPECT().
		DeleteMessageBySeq(context.Background(), msgSeq).
		Return(nil).
		AnyTimes()

	mock.EXPECT().
		GetMessageBySeq(context.Background(), msgSeq).
		Return(&client.MessageItem{
			StreamSeq: msgSeq,
			StoredAt:  FixedStoredAt,
			Data:      msgData,
		}, nil).
		AnyTimes()

	mock.EXPECT().
		ListUndeliveredMessages(context.Background()).
		Return([]client.MessageItem{
			{
				StreamSeq: 10,
				StoredAt:  FixedStoredAt,
				Data:      []byte("test message 1"),
			},
			{
				StreamSeq: 11,
				StoredAt:  FixedStoredAt,
				Data:      []byte("test message 2"),
			},
		}, nil).
		AnyTimes()

	mock.EXPECT().
		PublishToStream(context.Background(), msgData).
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

// NewPlainJetStream creates a jetstream Mock without defaults.
func NewPlainJetStream(ctrl *gomock.Controller) *MockJetStream {
	return NewMockJetStream(ctrl)
}

// NewJetStreamWithStreamError creates a jetstream Mock with Stream error.
func NewJetStreamWithStreamError(ctrl *gomock.Controller) *MockJetStream {
	mock := NewMockJetStream(ctrl)

	mock.EXPECT().
		Stream(context.Background(), StreamName).
		Return(nil, assert.AnError).
		AnyTimes()

	return mock
}

// NewMockStreamWithConsumer sets up a mock stream with a consumer.
func NewMockStreamWithConsumer(
	ctrl *gomock.Controller,
) (*MockJetStream, *MockStream, *MockConsumer) {
	mockJetStream := NewMockJetStream(ctrl)
	mockStream := NewMockStream(ctrl)
	mockConsumer := NewMockConsumer(ctrl)

	mockJetStream.EXPECT().
		Stream(context.Background(), StreamName).
		Return(mockStream, nil).
		// AnyTimes()
		Times(1)

	mockStream.EXPECT().
		Consumer(context.Background(), ConsumerName).
		Return(mockConsumer, nil).
		AnyTimes()

	return mockJetStream, mockStream, mockConsumer
}

// NewMockStreamWithConsumerError sets up a mock stream that returns a consumer error.
func NewMockStreamWithConsumerError(
	ctrl *gomock.Controller,
) *MockJetStream {
	mockJetStream := NewMockJetStream(ctrl)
	mockStream := NewMockStream(ctrl)

	mockJetStream.EXPECT().
		Stream(context.Background(), StreamName).
		Return(mockStream, nil).
		AnyTimes()

	mockStream.EXPECT().
		Consumer(context.Background(), ConsumerName).
		Return(nil, assert.AnError).
		Times(1)

	return mockJetStream
}

// GetFixedTime parses and returns the fixedTime constant as a time.Time value.
func GetFixedTime() time.Time {
	parsedTime, _ := time.Parse(time.RFC3339Nano, fixedTime)

	return parsedTime
}
