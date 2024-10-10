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

	"github.com/retr0h/osapi/internal/task/client"
)

const (
	// fixedTime pinned time used by test suite.
	fixedTime = "2024-09-10T12:00:00Z"
)

// NewPlainMockManager creates a Mock without defaults.
func NewPlainMockManager(ctrl *gomock.Controller) *MockManager {
	return NewMockManager(ctrl)
}

// NewDefaultMockManager creates a Mock with defaults.
func NewDefaultMockManager(ctrl *gomock.Controller) *MockManager {
	mock := NewMockManager(ctrl)

	fixedStoredAt, _ := GetFixedTime()

	mock.EXPECT().
		CountStreamMessages(context.Background()).
		Return(2, nil).
		AnyTimes()

	mock.EXPECT().
		DeleteMessageBySeq(context.Background(), uint64(123)).
		Return(nil).
		AnyTimes()

	mockMessageItem := &client.MessageItem{
		StreamSeq: 123,
		StoredAt:  fixedStoredAt,
		Data:      []byte("test message"),
	}

	mock.EXPECT().
		GetMessageBySeq(context.Background(), uint64(123)).
		Return(mockMessageItem, nil).
		AnyTimes()

	mockMessageItems := []client.MessageItem{
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
	}

	mock.EXPECT().
		GetAllPaginatedMessages(context.Background(), 2, 0).
		Return(mockMessageItems, nil).
		AnyTimes()

		// NOTE(retr0h): May remove this expectation in favor of one below
	mock.EXPECT().
		PublishToStream(context.Background(), []byte("test message")).
		Return(uint64(1), nil).
		AnyTimes()

	mock.EXPECT().
		PublishToStream(context.Background(), gomock.Any()).
		Return(uint64(1), nil).
		AnyTimes()

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
