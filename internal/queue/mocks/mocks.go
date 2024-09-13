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
	"fmt"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/retr0h/osapi/internal/queue"
)

const (
	// fixedTime pinned time used by test suite.
	fixedTime     = "2024-09-10T12:00:00Z"
	itemBody      = "test body"
	itemMessageID = "message-id"
	itemReceived  = 5
)

// NewPlainMockManager creates a Mock without defaults.
func NewPlainMockManager(ctrl *gomock.Controller) *MockManager {
	return NewMockManager(ctrl)
}

// NewDefaultMockManager creates a Mock with defaultas.
func NewDefaultMockManager(ctrl *gomock.Controller) *MockManager {
	mock := NewMockManager(ctrl)

	fixedCreated, _ := GetFixedTime()
	timeout := fixedCreated.Add(time.Hour)
	updated := fixedCreated.Add(time.Minute)

	// Set up default expectations for the mock methods

	mock.EXPECT().GetAll(gomock.Any(), gomock.Any(), gomock.Any()).Return([]queue.Item{
		{
			Body:     itemBody,
			ID:       itemMessageID,
			Received: itemReceived,
			Created:  fixedCreated,
			Timeout:  timeout,
			Updated:  updated,
		},
	}, nil).AnyTimes()

	mock.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(&queue.Item{
		Body:     itemBody,
		ID:       itemMessageID,
		Received: itemReceived,
		Created:  fixedCreated,
		Timeout:  timeout,
		Updated:  updated,
	}, nil).AnyTimes()

	mock.EXPECT().Count(gomock.Any()).Return(10, nil).AnyTimes()

	mock.EXPECT().DeleteByID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	mock.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

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
