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

// NewMockJetStreamWithEphemeralConsumer sets up a mock with an ephemeral consumer.
func NewMockJetStreamWithEphemeralConsumer(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, mockStream, mockConsumer := NewMockStreamWithConsumer(ctrl)

	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(&jetstream.ConsumerInfo{
			NumAckPending: 1,
			NumPending:    1,
			Delivered:     jetstream.SequenceInfo{Stream: 2},
		}, nil).
		Times(1)

	mockEphemeralConsumer := NewMockConsumer(ctrl)
	mockStream.EXPECT().
		CreateOrUpdateConsumer(gomock.Any(), gomock.Any()).
		Return(mockEphemeralConsumer, nil).
		Times(1)

	mockFetchResult1 := NewMockMessageBatch(ctrl)
	mockFetchResult2 := NewMockMessageBatch(ctrl)

	msgChan1 := make(chan jetstream.Msg, 1)
	msgChan2 := make(chan jetstream.Msg, 1)

	mockMsg1 := NewMockMsg(ctrl)
	mockMsg2 := NewMockMsg(ctrl)

	mockMsg1.EXPECT().
		Metadata().
		Return(&jetstream.MsgMetadata{
			Sequence:  jetstream.SequencePair{Stream: 1},
			Timestamp: FixedStoredAt,
		}, nil).AnyTimes()
	mockMsg1.EXPECT().
		Data().
		Return(msgData).AnyTimes()

	mockMsg2.EXPECT().
		Metadata().
		Return(&jetstream.MsgMetadata{
			Sequence:  jetstream.SequencePair{Stream: 2},
			Timestamp: FixedStoredAt,
		}, nil).AnyTimes()
	mockMsg2.EXPECT().
		Data().
		Return(msgData).AnyTimes()

	msgChan1 <- mockMsg1
	close(msgChan1)

	msgChan2 <- mockMsg2
	close(msgChan2)

	mockFetchResult1.EXPECT().
		Messages().
		Return(msgChan1).AnyTimes()

	mockFetchResult2.EXPECT().
		Messages().
		Return(msgChan2).AnyTimes()

	gomock.InOrder(
		mockEphemeralConsumer.EXPECT().
			Fetch(1, gomock.Any()).
			Return(mockFetchResult1, nil).
			Times(1),
		mockEphemeralConsumer.EXPECT().
			Fetch(1, gomock.Any()).
			Return(mockFetchResult2, nil).
			Times(1),
	)

	return mockJetStream
}

// NewMockJetStreamWithConsumerAndFetchError sets up a mock where ephemeralConsumer.Fetch fails.
func NewMockJetStreamWithConsumerAndFetchError(ctrl *gomock.Controller) *MockJetStream {
	// Reuse the existing helper to set up the stream and consumer.
	mockJetStream, mockStream, mockConsumer := NewMockStreamWithConsumer(ctrl)

	// Set up expectations for the consumer info call.
	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(&jetstream.ConsumerInfo{
			NumAckPending: 1,
			NumPending:    1,
			Delivered:     jetstream.SequenceInfo{Stream: 2},
		}, nil).
		AnyTimes()

	// Create an ephemeral consumer and simulate its `Fetch` method failing.
	mockEphemeralConsumer := NewMockConsumer(ctrl)
	mockStream.EXPECT().
		CreateOrUpdateConsumer(gomock.Any(), gomock.Any()).
		Return(mockEphemeralConsumer, nil).
		Times(1)

	mockEphemeralConsumer.EXPECT().
		Fetch(1, gomock.Any()).
		Return(nil, assert.AnError). // Simulate Fetch error
		AnyTimes()

	return mockJetStream
}

// NewMockJetStreamWithConsumerInfoError creates a JetStream mock where Consumer.Info() returns an error.
func NewMockJetStreamWithConsumerInfoError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, _, mockConsumer := NewMockStreamWithConsumer(ctrl)

	mockConsumer.EXPECT().
		Info(gomock.Any()).
		Return(nil, assert.AnError).
		Times(1)

	return mockJetStream
}

// NewMockJetStreamWithCreateConsumerError sets up a mock where CreateOrUpdateConsumer fails.
func NewMockJetStreamWithCreateConsumerError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, mockStream, mockConsumer := NewMockStreamWithConsumer(ctrl)

	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(&jetstream.ConsumerInfo{
			NumAckPending: 1,
			NumPending:    1,
		}, nil).
		AnyTimes()

	mockStream.EXPECT().
		CreateOrUpdateConsumer(gomock.Any(), gomock.Any()).
		Return(nil, assert.AnError).
		Times(1)

	return mockJetStream
}

// NewMockFetchResult creates a mock fetch result with messages.
func NewMockFetchResult(ctrl *gomock.Controller) *MockMessageBatch {
	mockFetchResult := NewMockMessageBatch(ctrl)

	msgChan := make(chan jetstream.Msg, 2)
	mockMsg1 := NewMockMsg(ctrl)
	mockMsg1.EXPECT().
		Metadata().
		Return(&jetstream.MsgMetadata{
			Sequence:  jetstream.SequencePair{Stream: 1},
			Timestamp: FixedStoredAt,
		}, nil).AnyTimes()
	mockMsg1.EXPECT().Data().Return(msgData).AnyTimes()

	mockMsg2 := NewMockMsg(ctrl)
	mockMsg2.EXPECT().
		Metadata().
		Return(&jetstream.MsgMetadata{
			Sequence:  jetstream.SequencePair{Stream: 2},
			Timestamp: FixedStoredAt,
		}, nil).AnyTimes()
	mockMsg2.EXPECT().Data().Return(msgData).AnyTimes()

	msgChan <- mockMsg1
	msgChan <- mockMsg2
	close(msgChan)

	mockFetchResult.EXPECT().
		Messages().
		Return(msgChan).
		AnyTimes()

	return mockFetchResult
}

// NewMockJetStreamWithMetadataError sets up a mock where metadata retrieval fails.
func NewMockJetStreamWithMetadataError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, mockStream, mockConsumer := NewMockStreamWithConsumer(ctrl)

	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(&jetstream.ConsumerInfo{
			NumAckPending: 1,
			NumPending:    1,
			Delivered:     jetstream.SequenceInfo{Stream: 2},
		}, nil).
		Times(1)

	mockEphemeralConsumer := NewMockConsumer(ctrl)
	mockStream.EXPECT().
		CreateOrUpdateConsumer(gomock.Any(), gomock.Any()).
		Return(mockEphemeralConsumer, nil).
		Times(1)

	mockFetchResult := NewMockMessageBatch(ctrl)
	msgChan := make(chan jetstream.Msg, 1)
	mockMsg := NewMockMsg(ctrl)

	mockMsg.EXPECT().
		Metadata().
		Return(nil, assert.AnError).
		AnyTimes()

	mockMsg.EXPECT().
		Data().
		Return(msgData).
		AnyTimes()

	msgChan <- mockMsg
	close(msgChan)

	mockFetchResult.EXPECT().
		Messages().
		Return(msgChan).
		AnyTimes()

	mockEphemeralConsumer.EXPECT().
		Fetch(1, gomock.Any()).
		Return(mockFetchResult, nil).
		AnyTimes()

	return mockJetStream
}
