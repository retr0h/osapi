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

// NewMockConsumerWithInfo sets up the consumer to return default info.
func NewMockConsumerWithInfo(mockConsumer *MockConsumer) {
	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(&jetstream.ConsumerInfo{
			NumAckPending: 1,
			NumPending:    1,
		}, nil).
		AnyTimes()
}

// NewJetStreamWithCount creates a JetStream mock with Count functionality.
func NewJetStreamWithCount(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, _, mockConsumer := NewMockStreamWithConsumer(ctrl)
	NewMockConsumerWithInfo(mockConsumer)

	return mockJetStream
}

// NewMockConsumerWithCountInfoError sets up a mock consumer that returns an Info error.
func NewMockConsumerWithCountInfoError(ctrl *gomock.Controller) *MockJetStream {
	mockJetStream, _, mockConsumer := NewMockStreamWithConsumer(ctrl)

	mockConsumer.EXPECT().
		Info(context.Background()).
		Return(nil, assert.AnError).
		Times(1)

	return mockJetStream
}
