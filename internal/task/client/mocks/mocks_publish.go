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

// NewJetStreamWithPublish creates a JetStream mock that simulates a successful publish.
func NewJetStreamWithPublish(ctrl *gomock.Controller) *MockJetStream {
	mock := NewPlainJetStream(ctrl)

	mock.EXPECT().
		Publish(context.Background(), SubjectName, msgData).
		Return(&jetstream.PubAck{
			Sequence: msgSeq,
		}, nil).
		AnyTimes()

	return mock
}

// NewJetStreamWithPublishError creates a JetStream mock that simulates publish errors.
func NewJetStreamWithPublishError(ctrl *gomock.Controller) *MockJetStream {
	mock := NewPlainJetStream(ctrl)

	mock.EXPECT().
		Publish(context.Background(), SubjectName, msgData).
		Return(nil, assert.AnError).
		Times(1)

	return mock
}
