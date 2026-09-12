// Copyright (c) 2026 John Dewey

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

package notify_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/osapi/internal/controller/notify"
)

type LogNotifierPublicTestSuite struct {
	suite.Suite

	notifier *notify.LogNotifier
}

func (s *LogNotifierPublicTestSuite) SetupTest() {
	s.notifier = notify.NewLogNotifier(slog.Default())
}

func (s *LogNotifierPublicTestSuite) TestNotify() {
	tests := []struct {
		name         string
		event        notify.ConditionEvent
		validateFunc func(error)
	}{
		{
			name: "logs fired event",
			event: notify.ConditionEvent{
				ComponentType: "agent",
				Hostname:      "web-01",
				Condition:     "MemoryPressure",
				Active:        true,
				Reason:        "memory usage above threshold",
				Timestamp:     time.Now(),
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
		{
			name: "logs resolved event",
			event: notify.ConditionEvent{
				ComponentType: "agent",
				Hostname:      "web-01",
				Condition:     "MemoryPressure",
				Active:        false,
				Reason:        "memory usage returned to normal",
				Timestamp:     time.Now(),
			},
			validateFunc: func(err error) {
				s.NoError(err)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(s.notifier.Notify(context.Background(), tt.event))
		})
	}
}

func TestLogNotifierPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LogNotifierPublicTestSuite))
}
